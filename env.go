package config

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

	var err error

	t := field.Type().Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()

		// Make a memory location for the nil pointer, and un-nil it.
		if field = field.Elem(); field.IsNil() && field.CanSet() {
			field.Set(reflect.New(field.Type().Elem()))
		}
	}

	for i := 0; i < t.NumField(); i++ { // Loop each struct member
		exists := false
		shorttag := strings.Split(strings.ToUpper(t.Field(i).Tag.Get(ENVTag)), ",")[0]
		fulltag := prefix + "_" + shorttag
		envval, ok := os.LookupEnv(fulltag)
		subfield := field.Elem().Field(i)
		//		log.Println(t, subfield.Type(), " ===> ", fulltag, " ===> ", envval, field.Kind(), field)

		if exists, err = checkInterface(subfield, envval, fulltag); err != nil {
			return false, err
		} else if exists {
			continue
		}

		switch subfield.Kind() {
		case reflect.Ptr:
			exists, err = parsePointer(subfield, fulltag, envval)
		case reflect.Struct:
			exists, err = parseStruct(subfield.Addr(), fulltag)
		case reflect.Slice:
			exists, err = parseSlice(subfield, fulltag)
		case reflect.Map:
			if !IgnoreUnknown {
				err = fmt.Errorf("maps don't work")
			}
		default:
			if !ok || shorttag == "" || shorttag == "-" {
				break // switch
			}

			exists, err = parseMember(subfield, fulltag, envval)
		}

		if err != nil {
			return false, err
		}

		if exists {
			exitOk = true
		}
	}

	return exitOk, nil
}

func parsePointer(field reflect.Value, tag, envval string) (ok bool, err error) {
	value := reflect.Value{}

	switch field.Type().Elem().Kind() {
	case reflect.Struct:
		value = reflect.New(field.Type().Elem())
		ok, err = parseStruct(value, tag)
	case reflect.Slice:
		// don't do this. a pointer to a slice? uhg.
		value = reflect.New(field.Type().Elem())
		ok, err = parseSlice(value.Elem(), tag)
	default:
		if strings.HasSuffix(tag, "_") || envval == "" {
			return false, nil
		}

		value = reflect.Indirect(reflect.New(field.Type().Elem()))
		ok, err = parseMember(value, tag, envval)
		value = value.Addr()
	}

	if ok && field.CanSet() {
		field.Set(value)
	}

	return ok, err
}

func checkInterface(field reflect.Value, envval, tag string) (bool, error) {
	if !field.CanInterface() {
		return false, nil
	}

	if v, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
		if envval == "" {
			return false, nil
		}

		if err := v.UnmarshalText([]byte(envval)); err != nil {
			return false, err
		}

		return true, nil
	}

	if v, ok := field.Addr().Interface().(ENVUnmarshaler); ok {
		// Custom unmarshaler can proceed even if envval is empty. It may produce new envvals...
		if err := v.UnmarshalENV(tag, envval); err != nil {
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

	// log.Println("found", tag, envval)
	switch fieldType := field.Type().String(); fieldType {
	// Handle each member type appropriately (differently).
	case typeSTR:
		// SetString is a reflect package method to update a struct member by index.
		field.SetString(envval)
	case typeUINT, typeUINT8, typeUINT16, typeUINT32, typeUINT64:
		val := uint64(0)
		val, err = parseUint(fieldType, envval)
		field.SetUint(val)
	case typeINT, typeINT8, typeINT16, typeINT32, typeINT64:
		val := int64(0)
		val, err = parseInt(fieldType, envval)
		field.SetInt(val)
	case typeFloat64:
		val := float64(0)
		val, err = strconv.ParseFloat(envval, 64)
		field.SetFloat(val)
	case typeFloat32:
		val := float64(0)
		val, err = strconv.ParseFloat(envval, 32)
		field.SetFloat(val)
	case typeDur:
		val := time.Duration(0)
		val, err = time.ParseDuration(envval)
		field.Set(reflect.ValueOf(val))
	case typeBool:
		val := false
		val, err = strconv.ParseBool(envval)
		field.SetBool(val)
	default:
		ok := false
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
	if !field.CanSet() {
		return false, nil
	}

	if field.IsNil() {
		field.Set(reflect.MakeSlice(field.Type(), 0, 0))
	}

	switch k := field.Type().Elem(); k.Kind() {
	case reflect.Ptr:
		if k.Elem().Kind() == reflect.Struct {
			return parseStructSlice(field, tag)
		}

		return parseMemberSlice(field, tag)
	case reflect.Struct:
		return parseStructSlice(field, tag)
	case reflect.String, reflect.Float32, reflect.Float64, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint16, reflect.Uint64:
		return parseMemberSlice(field, tag)
	case reflect.Map:
		fallthrough
	default:
		if IgnoreUnknown {
			return false, nil
		}

		return false, fmt.Errorf("unsupported slice type: %v %v", k, k.Elem().Kind())
	}
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
		isPtr := field.Type().Elem().Kind() == reflect.Ptr

		if !exists {
			continue // only work with env var data that exists.
		}

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
