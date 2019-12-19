package cnfg

import (
	"encoding"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ParseENV copies environment variables into configuration values.
// This is useful for Docker users that find it easier to pass ENV variables
// than a specific configuration file. Uses reflection to find struct tags.
func ParseENV(c interface{}, prefix string) (bool, error) {
	return parseStruct(reflect.ValueOf(c), prefix)
}

// parseStruct does most of the heavy lifting. Called every time a struct is encountered.
func parseStruct(field reflect.Value, prefix string) (bool, error) {
	var exitOk bool

	newfield, t := getFieldType(field)
	for i := 0; i < t.NumField(); i++ { // Loop each struct member
		shorttag := strings.Split(strings.ToUpper(t.Field(i).Tag.Get(ENVTag)), ",")[0]
		if shorttag == "-" {
			continue
		}

		tag := prefix + "_" + shorttag
		envval, ok := os.LookupEnv(tag)

		if exists, err := parseAnything(newfield.Elem().Field(i), tag, envval, ok); err != nil {
			return false, err
		} else if exists {
			exitOk = true
		}
	}

	return exitOk, nil
}

func parseAnything(field reflect.Value, tag, envval string, force bool) (bool, error) {
	//	log.Println("anything", envval, tag, field.Kind(), field.Type())
	if exists, err := checkInterface(field, envval, tag); err != nil {
		return false, err
	} else if exists {
		return true, nil
	}

	switch field.Kind() {
	case reflect.Ptr:
		return parsePointer(field, tag, envval)
	case reflect.Struct:
		return parseStruct(field.Addr(), tag)
	case reflect.Slice:
		return parseSlice(field, tag)
	case reflect.Map:
		return parseMap(field, tag)
	default:
		if !force && envval == "" {
			return false, nil
		}

		return parseMember(field, tag, envval)
	}
}

func getFieldType(field reflect.Value) (reflect.Value, reflect.Type) {
	t := field.Type().Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()

		// Make a memory location for the nil pointer, and un-nil it.
		if field = field.Elem(); field.IsNil() && field.CanSet() {
			field.Set(reflect.New(field.Type().Elem()))
		}
	}

	return field, t
}

func parsePointer(field reflect.Value, tag, envval string) (ok bool, err error) {
	value := reflect.New(field.Type().Elem()).Elem()

	ok, err = parseAnything(value, tag, envval, false)
	if ok && field.CanSet() {
		field.Set(value.Addr())
	}

	return ok, err
}

func checkInterface(field reflect.Value, envval, tag string) (bool, error) {
	if !field.CanAddr() || !field.Addr().CanInterface() {
		return false, nil
	}

	if v, ok := field.Addr().Interface().(ENVUnmarshaler); ok {
		// Custom unmarshaler can proceed even if envval is empty. It may produce new envvals...
		if err := v.UnmarshalENV(tag, envval); err != nil {
			return false, err
		}

		return true, nil
	}

	if envval == "" {
		return false, nil
	}

	if v, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
		if err := v.UnmarshalText([]byte(envval)); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

// parseMember parses non-struct, non-slice struct-member types.
func parseMember(field reflect.Value, tag, envval string) (bool, error) {
	var err error

	if !field.CanSet() {
		return false, nil
	}

	switch fieldType := field.Type().String(); fieldType {
	// Handle each member type appropriately (differently).
	case typeSTR:
		// SetString is a reflect package method to update a struct member by index.
		field.SetString(envval)
	case typeUINT, typeUINT8, typeUINT16, typeUINT32, typeUINT64:
		var val uint64

		val, err = parseUint(fieldType, envval)
		field.SetUint(val)
	case typeINT, typeINT8, typeINT16, typeINT32, typeINT64:
		var val int64

		val, err = parseInt(fieldType, envval)
		field.SetInt(val)
	case typeFloat64:
		var val float64

		val, err = strconv.ParseFloat(envval, 64)
		field.SetFloat(val)
	case typeFloat32:
		var val float64

		val, err = strconv.ParseFloat(envval, 32)
		field.SetFloat(val)
	case typeDur:
		var val time.Duration

		val, err = time.ParseDuration(envval)
		field.Set(reflect.ValueOf(val))
	case typeBool:
		var val bool

		val, err = strconv.ParseBool(envval)
		field.SetBool(val)
	default:
		var ok bool

		if ok, err = checkInterface(field, envval, tag); err == nil && !ok {
			err = fmt.Errorf("unsupported type: %v (val: %s) - please report this", field.Type(), envval)
		}
	}

	if err != nil {
		return false, fmt.Errorf("%s: %v", tag, err)
	}

	return true, nil
}

func parseUint(intType, envval string) (uint64, error) {
	switch intType {
	default:
		return strconv.ParseUint(envval, 10, 0)
	case typeUINT8:
		return strconv.ParseUint(envval, 10, 8)
	case typeUINT16:
		return strconv.ParseUint(envval, 10, 16)
	case typeUINT32:
		return strconv.ParseUint(envval, 10, 32)
	case typeUINT64:
		return strconv.ParseUint(envval, 10, 64)
	}
}

func parseInt(intType, envval string) (int64, error) {
	switch intType {
	default:
		return strconv.ParseInt(envval, 10, 0)
	case typeINT8:
		return strconv.ParseInt(envval, 10, 8)
	case typeINT16:
		return strconv.ParseInt(envval, 10, 16)
	case typeINT32:
		return strconv.ParseInt(envval, 10, 32)
	case typeINT64:
		return strconv.ParseInt(envval, 10, 64)
	}
}

func parseSlice(field reflect.Value, tag string) (bool, error) {
	value := field
	if !value.CanSet() {
		return false, nil
	}

	reflect.Copy(value, field)

	ok, err := parseSliceValue(value, tag)
	if ok {
		field.Set(value) // Overwrite the slice.
	}

	return ok, err
}

func parseSliceValue(field reflect.Value, tag string) (bool, error) {
	switch k := field.Type().Elem(); k.Kind() {
	case reflect.Ptr:
		switch k.Elem().Kind() {
		case reflect.Struct:
			return parseStructSlice(field, tag)
		case reflect.Map:
			return parseMapSlice(field, tag)
		default:
			return parseMemberSlice(field, tag)
		}
	case reflect.Struct:
		return parseStructSlice(field, tag)
	case reflect.String, reflect.Float32, reflect.Float64, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint16, reflect.Uint64:
		return parseMemberSlice(field, tag)
	case reflect.Map:
		return parseMapSlice(field, tag)
	default:
		if IgnoreUnknown {
			return false, nil
		}

		return false, fmt.Errorf("unsupported slice type: %v %v", k, k.Kind())
	}
}

func parseMapSlice(field reflect.Value, tag string) (bool, error) {
	var ok bool

	if !field.CanSet() {
		return false, nil
	}

	total := field.Len()
	isPtr := field.Type().Elem().Kind() == reflect.Ptr

FORLOOP:
	for i := 0; i <= total; i++ {
		ntag := tag + "_" + strconv.Itoa(i)

		value := reflect.Indirect(reflect.New(field.Type().Elem()))

		if i < field.Len() {
			value = reflect.Indirect(field.Index(i).Addr())
		}

		if isPtr {
			value = reflect.New(value.Type().Elem()).Elem()
		}

		exists, err := parseMap(value, ntag)
		if err != nil {
			return false, err
		}

		if !exists {
			continue FORLOOP
		}

		ok = true

		if isPtr {
			value = value.Addr()
		}

		if i >= field.Len() {
			total++ // do one more iteration.
			// The position in the ENV var is > slice size, so append.
			field.Set(reflect.Append(field, value))
			continue FORLOOP
		}

		// The position in the ENV var exists! Overwrite slice index directly.
		field.Index(i).Set(reflect.Indirect(value))
	}

	return ok, nil
}

func parseMap(field reflect.Value, tag string) (bool, error) {
	var ok bool

	if !field.CanSet() {
		return false, nil
	}

	vals := getMapVals(tag)
	if len(vals) < 1 {
		return false, nil
	}

	if field.IsNil() {
		field.Set(reflect.MakeMap(field.Type()))
	}

	for k, v := range vals {
		keyval := reflect.Indirect(reflect.New(field.Type().Key()))

		_, err := parseAnything(keyval, tag, k, true)
		if err != nil {
			return false, err
		}

		if v == "" {
			ok = true

			field.SetMapIndex(keyval, reflect.Value{})

			continue
		}

		valval := reflect.Indirect(reflect.New(field.Type().Elem()))

		exists, err := parseAnything(valval, tag+"_"+k, v, true)
		if err != nil {
			return false, err
		}

		if exists {
			ok = true
		}

		field.SetMapIndex(keyval, valval)
	}

	return ok, nil
}

func getMapVals(tag string) map[string]string {
	m := make(map[string]string)

	for _, pair := range os.Environ() {
		split := strings.SplitN(pair, "=", 2)
		if len(split) != 2 {
			continue
		}

		fulltag := split[0]
		value := split[1]
		shorttag := strings.TrimPrefix(fulltag, tag+"_")
		shorttag = strings.Split(shorttag, "_")[0]

		if strings.HasPrefix(fulltag, tag+"_") {
			m[shorttag] = value
		}
	}

	return m
}

func parseStructSlice(field reflect.Value, tag string) (bool, error) {
	var ok bool

	if !field.CanSet() {
		return false, nil
	}

	total := field.Len()
FORLOOP:
	for i := 0; i <= total; i++ {
		ntag := tag + "_" + strconv.Itoa(i)
		value := reflect.New(field.Type().Elem())

		if i < field.Len() {
			value = field.Index(i).Addr()
		}

		switch exists, err := parseStruct(value, ntag); {
		case err != nil:
			return false, err
		case exists:
			ok = true

			if i >= field.Len() {
				total++ // do one more iteration.
				// The position in the ENV var is > slice size, so append.
				field.Set(reflect.Append(field, reflect.Indirect(value)))
				continue FORLOOP
			}

			// The position in the ENV var exists! Overwrite slice index directly.
			field.Index(i).Set(reflect.Indirect(value))
		}
	}

	return ok, nil
}

func parseMemberSlice(field reflect.Value, tag string) (bool, error) {
	var ok bool

	if !field.CanSet() {
		return false, nil
	}

	total := field.Len()
	for i := 0; i <= total; i++ {
		ntag := tag + "_" + strconv.Itoa(i)
		envval, exists := os.LookupEnv(ntag)

		if !exists {
			continue // only work with env var data that exists.
		}

		isPtr := field.Type().Elem().Kind() == reflect.Ptr
		ok = true // the slice exists because it has at least 1 member.

		// This makes an empty value we _set_ with parseMemebr()
		value := reflect.Indirect(reflect.New(field.Type().Elem()))
		if isPtr {
			value = reflect.New(value.Type().Elem()).Elem()
		}

		if _, err := parseMember(value, ntag, envval); err != nil {
			return false, err
		}

		if isPtr {
			value = value.Addr()
		}

		if i >= field.Len() {
			total++ // do one more loop iteration
			// The position in the ENV var is > slice size, so append.
			field.Set(reflect.Append(field, value))

			continue // check for the next slice member env var.
		}

		// The position in the ENV var exists! Overwrite slice index directly.
		field.Index(i).Set(value)
	}

	return ok, nil
}
