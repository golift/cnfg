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
	return parseStruct(reflect.ValueOf(c), reflect.TypeOf(c).Elem(), prefix)
}

// parseStruct does most of the heavy lifting. Called every time a struct is encountered.
func parseStruct(field reflect.Value, t reflect.Type, prefix string) (bool, error) {
	var exitOk, exists bool

	var err error

	if t.Kind() == reflect.Ptr {
		t = t.Elem()

		// Make a memory location for the nil pointer, and un-nil it.
		if field = field.Elem(); field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
	}

	for i := 0; i < t.NumField(); i++ { // Loop each struct member
		shorttag := strings.Split(strings.ToUpper(t.Field(i).Tag.Get(ENVTag)), ",")[0]
		fulltag := prefix + "_" + shorttag
		envval, ok := os.LookupEnv(fulltag)
		subfield := field.Elem().Field(i)
		// log.Println(t, subfield.Type(), " ===> ", ntag, " ===> ", envval, field.Kind(), field)

		if exists, err := checkInterface(subfield, envval, fulltag); err != nil {
			return false, err
		} else if exists {
			continue
		}

		switch subfield.Kind() {
		case reflect.Ptr:
			subfield = subfield.Elem()
			if subfield.Kind() == reflect.Struct {
				exists, err = parseStruct(subfield.Addr(), subfield.Type(), fulltag)
			}

			// don't do this. a pointer to a slice? uhg.
			if subfield.Kind() == reflect.Slice {
				exists, err = parseSlice(subfield, fulltag)
			}

			if err != nil {
				return false, err
			}
		case reflect.Struct:
			exists, err = parseStruct(subfield.Addr(), subfield.Type(), fulltag)
			if err != nil {
				return false, err
			}
		case reflect.Slice:
			exists, err = parseSlice(subfield, fulltag)
			if err != nil {
				return false, err
			}
		default:
			if !ok || shorttag == "" || shorttag == "-" {
				break // switch
			}

			exists = true

			if err = parseMember(subfield, fulltag, envval); err != nil {
				return false, err
			}
		}

		if exists {
			exitOk = true
		}
	}

	return exitOk, nil
}

func checkInterface(field reflect.Value, envval, tag string) (bool, error) {
	if !field.CanInterface() {
		return false, nil
	}

	if v, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
		if err := v.UnmarshalText([]byte(envval)); err != nil {
			return false, err
		}

		return true, nil
	}

	if v, ok := field.Addr().Interface().(ENVUnmarshaler); ok {
		if err := v.UnmarshalENV(tag, envval); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

// parseMember parses non-struct, non-slice struct-member types.
func parseMember(field reflect.Value, tag, envval string) error {
	// log.Println("found", tag, envval)
	switch fieldType := field.Type().String(); fieldType {
	// Handle each member type appropriately (differently).
	case "string":
		// SetString is a reflect package method to update a struct member by index.
		field.SetString(envval)
	case "uint", "uint8", "uint16", "uint32", "uint64":
		val, err := parseUint(fieldType, envval)
		if err != nil {
			return fmt.Errorf("%s: %v", tag, err)
		}

		field.SetUint(val)
	case "int", "int8", "int16", "int32", "int64":
		val, err := parseInt(fieldType, envval)
		if err != nil {
			return fmt.Errorf("%s: %v", tag, err)
		}

		field.SetInt(val)
	case "float64":
		val, err := strconv.ParseFloat(envval, 64)
		if err != nil {
			return fmt.Errorf("%s: %v", tag, err)
		}

		field.SetFloat(val)
	case "float32":
		val, err := strconv.ParseFloat(envval, 32)
		if err != nil {
			return fmt.Errorf("%s: %v", tag, err)
		}

		field.SetFloat(val)
	case "time.Duration":
		val, err := time.ParseDuration(envval)
		if err != nil {
			return fmt.Errorf("%s: %v", tag, err)
		}

		field.Set(reflect.ValueOf(val))
	case "bool":
		val, err := strconv.ParseBool(envval)
		if err != nil {
			return fmt.Errorf("%s: %v", tag, err)
		}

		field.SetBool(val)
	default:
		if v, ok := field.Interface().(encoding.TextUnmarshaler); ok {
			return v.UnmarshalText([]byte(envval))
		}

		if v, ok := field.Interface().(ENVUnmarshaler); ok {
			return v.UnmarshalENV(tag, envval)
		}

		return fmt.Errorf("%s: unsupported type: %v (val: %s) - please report this", tag, field.Type(), envval)
	}

	return nil
}

func parseUint(intType, envval string) (uint64, error) {
	switch intType {
	default:
		return strconv.ParseUint(envval, 10, 0)
	case "int8":
		return strconv.ParseUint(envval, 10, 8)
	case "int16":
		return strconv.ParseUint(envval, 10, 16)
	case "int32":
		return strconv.ParseUint(envval, 10, 32)
	case "int64":
		return strconv.ParseUint(envval, 10, 64)
	}
}

func parseInt(intType, envval string) (int64, error) {
	switch intType {
	default:
		return strconv.ParseInt(envval, 10, 0)
	case "int8":
		return strconv.ParseInt(envval, 10, 8)
	case "int16":
		return strconv.ParseInt(envval, 10, 16)
	case "int32":
		return strconv.ParseInt(envval, 10, 32)
	case "int64":
		return strconv.ParseInt(envval, 10, 64)
	}
}

func parseSlice(field reflect.Value, tag string) (bool, error) {
	if field.IsNil() {
		field.Set(reflect.MakeSlice(field.Type(), 0, 0))
	}

	switch k := field.Type().Elem(); k.Kind() {
	case reflect.Map:
		fallthrough
	default:
		if IgnoreUnknown {
			return false, nil
		}

		return false, fmt.Errorf("unsupported slice type: %v", k)
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
	}
}

func parseStructSlice(field reflect.Value, tag string) (bool, error) {
	var ok bool

	total := field.Len()

FORLOOP:
	for i := 0; i <= total; i++ {
		ntag := tag + "_" + strconv.Itoa(i)
		value := reflect.New(field.Type().Elem())
		if i < field.Len() {
			value = field.Index(i).Addr()
		}
		switch exists, err := parseStruct(value, field.Type().Elem(), ntag); {
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

	if field.IsNil() {
		field.Set(field.Elem())
	}

	total := field.Len()
	for i := 0; i <= total; i++ {
		ntag := tag + "_" + strconv.Itoa(i)
		envval, exists := os.LookupEnv(ntag)

		if !exists && i > field.Len() {
			// We've checked all possible ENV var's up to slice count + 1, stop there if it's empty.
			break
		} else if !exists {
			continue // only work with env var data that exists.
		}

		// This makes an empty value we _set_ with parseMemebr()
		value := reflect.Indirect(reflect.New(field.Type().Elem()))
		ok = true // the slice exists because it has at least 1 member.

		if err := parseMember(value, ntag, envval); err != nil {
			return false, err
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
