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
	value := reflect.ValueOf(c)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("must provide pointer to struct")
	}

	return parseStruct(value, prefix)
}

// parseStruct does most of the heavy lifting. Called every time a struct is encountered.
func parseStruct(field reflect.Value, prefix string) (bool, error) {
	var exitOk bool

	t := field.Type().Elem()
	for i := 0; i < t.NumField(); i++ { // Loop each struct member
		shorttag := strings.Split(strings.ToUpper(t.Field(i).Tag.Get(ENVTag)), ",")[0]
		if !field.Elem().Field(i).CanSet() || shorttag == "-" || shorttag == "" {
			continue // This _only_ works with reflection tags.
		}

		tag := prefix + "_" + shorttag
		envval, ok := os.LookupEnv(tag)

		if exists, err := parseAnything(field.Elem().Field(i), tag, envval, ok); err != nil {
			return false, err
		} else if exists {
			exitOk = true
		}
	}

	return exitOk, nil
}

func parseAnything(field reflect.Value, tag, envval string, force bool) (bool, error) {
	// 	log.Println("parseAnything", envval, tag, field.Kind(), field.Type(), field.Interface())
	if exists, err := checkInterface(field, tag, envval); err != nil {
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

func parsePointer(field reflect.Value, tag, envval string) (ok bool, err error) {
	value := reflect.New(field.Type().Elem())
	if field.Elem().CanAddr() {
		// if the pointer already has a value, copy it instead of use the new one.
		value = field.Elem().Addr()
	}

	// Pass the non-pointer element back into the start.
	ok, err = parseAnything(value.Elem(), tag, envval, false)
	if ok {
		// overwrite the pointer only if something was parsed.
		field.Set(value)
	}

	return ok, err
}

func checkInterface(field reflect.Value, tag, envval string) (bool, error) {
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

		if ok, err = checkInterface(field, tag, envval); err == nil && !ok {
			err = fmt.Errorf("unsupported type: %v (val: %s) - please report this if you think this type should be supported", field.Type(), envval)
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

	reflect.Copy(value, field)

	ok, err := parseSliceValue(value, tag)
	if ok {
		field.Set(value) // Overwrite the slice.
	}

	return ok, err
}

func parseSliceValue(field reflect.Value, tag string) (bool, error) {
	var ok bool

	total := field.Len()
	for i := 0; i <= total; i++ {
		ntag := tag + "_" + strconv.Itoa(i)
		envval, exists := os.LookupEnv(ntag)

		// Start with a blank value for this item
		value := reflect.Indirect(reflect.New(field.Type().Elem()))
		if i < field.Len() {
			// Use the passed in value if it exists.
			value = reflect.Indirect(field.Index(i).Addr())
		}

		if exists, err := parseAnything(value, ntag, envval, exists); err != nil {
			return false, err
		} else if !exists {
			continue
		}

		ok = true

		if i >= field.Len() {
			total++ // do one more iteration.

			// The position in the ENV var is > slice size, so append.
			field.Set(reflect.Append(field, value))

			continue
		}

		// The position in the ENV var exists! Overwrite slice index directly.
		field.Index(i).Set(value)
	}

	return ok, nil
}

func parseMap(field reflect.Value, tag string) (bool, error) {
	var ok bool

	vals := getMapVals(tag)
	if len(vals) < 1 {
		return false, nil
	}

	if field.IsNil() {
		field.Set(reflect.MakeMap(field.Type()))
	}

	for k, v := range vals {
		keyval := reflect.Indirect(reflect.New(field.Type().Key()))

		if _, err := parseAnything(keyval, tag, k, true); err != nil {
			return false, err
		}

		if v == "" {
			// a blank env value was provided, set the field to nil.
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
