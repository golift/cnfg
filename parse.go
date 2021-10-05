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

/* This file contains all the logic to parse a data structure
   using reflection tags from a map of keys and values. */

type parser struct {
	Tag  string // struct tag to look for on struct members
	Vals Pairs  // pairs of env variables (saved at start)
}

// Struct does most of the heavy lifting. Called every time a struct is encountered.
// The entire process begins here. It's very recursive.
func (p *parser) Struct(field reflect.Value, prefix string) (bool, error) {
	var exitOk bool

	t := field.Type().Elem()
	for i := 0; i < t.NumField(); i++ { // Loop each struct member
		tagval := strings.Split(t.Field(i).Tag.Get(p.Tag), ",")
		shorttag := strings.ToUpper(tagval[0]) // like "NAME" or "TIMEOUT"

		if !field.Elem().Field(i).CanSet() || shorttag == "-" {
			continue // This _only_ works with reflection tags.
		}

		delenv := false

		for i := 1; i < len(tagval); i++ {
			if tagval[i] == "delenv" {
				delenv = true
			}
		}

		tag := strings.Trim(strings.Join([]string{prefix, shorttag}, LevelSeparator), LevelSeparator) // PFX_NAME, PFX_TIMEOUT
		envval, ok := p.Vals[tag]                                                                     // see if it exists

		//		log.Print("tag ", tag, " = ", envval)
		exists, err := p.Anything(field.Elem().Field(i), tag, envval, ok, delenv)
		if err != nil {
			return false, err
		} else if exists {
			exitOk = true
		}
	}

	return exitOk, nil
}

func (p *parser) Anything(field reflect.Value, tag, envval string, force, delenv bool) (bool, error) {
	//	log.Println("Anything", envval, tag, field.Kind(), field.Type(), field.Interface())
	if exists, err := p.Interface(field, tag, envval); err != nil {
		return false, err
	} else if exists {
		return true, nil
	}

	switch field.Kind() { // nolint: exhaustive
	case reflect.Ptr:
		return p.Pointer(field, tag, envval, delenv)
	case reflect.Struct:
		return p.Struct(field.Addr(), tag)
	case reflect.Slice:
		return p.Slice(field, tag, delenv)
	case reflect.Map:
		return p.Map(field, tag, delenv)
	default:
		if delenv {
			os.Unsetenv(tag) // delete it if it was requested in the env tag.
		}

		if !force && envval == "" {
			return false, nil
		}

		return p.Member(field, tag, envval)
	}
}

func (p *parser) Pointer(field reflect.Value, tag, envval string, delenv bool) (ok bool, err error) {
	value := reflect.New(field.Type().Elem())
	if field.Elem().CanAddr() {
		// if the pointer already has a value, copy it instead of use the new one.
		value = field.Elem().Addr()
	}

	// Pass the non-pointer element back into the start.
	ok, err = p.Anything(value.Elem(), tag, envval, false, delenv)
	if ok {
		// overwrite the pointer only if something was parsed.
		field.Set(value)
	}

	return ok, err
}

func (p *parser) Interface(field reflect.Value, tag, envval string) (bool, error) {
	if !field.CanAddr() || !field.Addr().CanInterface() {
		return false, nil
	}

	if v, ok := field.Addr().Interface().(ENVUnmarshaler); ok {
		// Custom unmarshaler can proceed even if envval is empty. It may produce new envvals...
		if err := v.UnmarshalENV(tag, envval); err != nil {
			return false, fmt.Errorf("UnmarshalENV interface: %w", err)
		}

		return true, nil
	}

	if envval == "" {
		return false, nil
	}

	if v, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
		if err := v.UnmarshalText([]byte(envval)); err != nil {
			return false, fmt.Errorf("UnmarshalText interface: %w", err)
		}

		return true, nil
	}

	// We may want to gate this with a config option or something.
	// time.Time does not like this but url.URL does. Placing this
	// _after_ TextUnmarshaler fixed the time.Time bug, so it's "ok"
	if v, ok := field.Addr().Interface().(encoding.BinaryUnmarshaler); ok {
		if err := v.UnmarshalBinary([]byte(envval)); err != nil {
			return false, fmt.Errorf("UnmarshalBinary interface: %w", err)
		}

		return true, nil
	}

	return false, nil
}

// Member parses non-struct, non-slice struct-member types.
func (p *parser) Member(field reflect.Value, tag, envval string) (bool, error) {
	var err error

	switch fieldType := field.Type().String(); fieldType {
	// Handle each member type appropriately (differently).
	case typeString:
		// SetString is a reflect package method to update a struct member by index.
		field.SetString(envval)
	case typeUINT, typeUINT8, typeUINT16, typeUINT32, typeUINT64:
		err = parseUint(field, fieldType, envval)
	case typeINT, typeINT8, typeINT16, typeINT32, typeINT64:
		var val int64

		val, err = parseInt(fieldType, envval)
		field.SetInt(val)
	case typeFloat64:
		var val float64

		val, err = strconv.ParseFloat(envval, bits64)
		field.SetFloat(val)
	case typeFloat32:
		var val float64

		val, err = strconv.ParseFloat(envval, bits32)
		field.SetFloat(val)
	case typeDur:
		var val time.Duration

		val, err = time.ParseDuration(envval)
		field.Set(reflect.ValueOf(val))
	case typeBool:
		var val bool

		val, err = strconv.ParseBool(envval)
		field.SetBool(val)
	case typeError: // lul
		field.Set(reflect.ValueOf(fmt.Errorf(envval))) // nolint: goerr113
	default:
		var ok bool

		if ok, err = p.Interface(field, tag, envval); err == nil && !ok {
			err = fmt.Errorf("%w: %v (val: %s)", ErrUnsupported, field.Type(), envval)
		}
	}

	if err != nil {
		return false, fmt.Errorf("%s: %w", tag, err)
	}

	return true, nil
}

func (p *parser) Slice(field reflect.Value, tag string, delenv bool) (ok bool, err error) {
	value := field
	reflect.Copy(value, field)

	// slice of bytes works differently than any other slice type.
	if value.Type().String() == "[]uint8" {
		envval, exists := p.Vals[tag]
		ok = exists

		value.SetBytes([]byte(envval))
	} else {
		ok, err = p.SliceValue(value, tag, delenv)
	}

	if delenv {
		os.Unsetenv(tag) // delete it if it was requested in the env tag.
	}

	if ok {
		field.Set(value) // Overwrite the slice.
	}

	return ok, err
}

func (p *parser) SliceValue(field reflect.Value, tag string, delenv bool) (bool, error) {
	var ok bool

	total := field.Len()
	for i := 0; i <= total; i++ {
		ntag := strings.Join([]string{tag, strconv.Itoa(i)}, LevelSeparator)
		envval, exists := p.Vals[ntag]

		if delenv {
			os.Unsetenv(ntag) // delete it if it was requested in the env tag.
		}

		// Start with a blank value for this item
		value := reflect.Indirect(reflect.New(field.Type().Elem()))
		if i < field.Len() {
			// Use the passed in value if it exists.
			value = reflect.Indirect(field.Index(i).Addr())
		}

		if exists, err := p.Anything(value, ntag, envval, exists, delenv); err != nil {
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

func (p *parser) Map(field reflect.Value, tag string, delenv bool) (bool, error) {
	var ok bool

	vals := p.Vals.Get(tag) // key=val, ... (prefix stripped)
	if len(vals) < 1 {
		return false, nil
	}

	if field.IsNil() {
		field.Set(reflect.MakeMap(field.Type()))
	}

	for k, v := range vals {
		if delenv {
			os.Unsetenv(k)
		}

		// Maps have 2 types. The index and the value. First, parse the index into its type.
		keyval := reflect.Indirect(reflect.New(field.Type().Key()))
		if _, err := p.Anything(keyval, tag, k, true, delenv); err != nil {
			return false, err
		}

		if v == "" {
			// a blank env value was provided, set the field to nil.
			ok = true

			field.SetMapIndex(keyval, reflect.Value{})

			continue
		}

		// And now parse the second type: the value.
		valval := reflect.Indirect(reflect.New(field.Type().Elem()))

		exists, err := p.Anything(valval, strings.Join([]string{tag, k}, LevelSeparator), v, true, delenv)
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

// parseUint parses an unsigned integer from a string as specific size.
func parseUint(field reflect.Value, intType, envval string) error {
	var (
		err error
		val uint64
	)

	switch intType {
	default:
		val, err = strconv.ParseUint(envval, base10, 0)
	case typeUINT8:
		// this crap is to support byte and []byte
		switch len(envval) {
		case 0:
			field.Set(reflect.ValueOf(uint8(0)))

			return nil
		case 1:
			field.Set(reflect.ValueOf(envval[0]))

			return nil
		default:
			return fmt.Errorf("%w: %s", ErrInvalidByte, envval)
		}
	case typeUINT16:
		val, err = strconv.ParseUint(envval, base10, bits16)
	case typeUINT32:
		val, err = strconv.ParseUint(envval, base10, bits32)
	case typeUINT64:
		val, err = strconv.ParseUint(envval, base10, bits64)
	}

	if err != nil {
		return fmt.Errorf("parsing integer: %w", err)
	}

	field.SetUint(val)

	return nil
}

// parseInt parses an integer from a string as specific size.
func parseInt(intType, envval string) (i int64, err error) {
	switch intType {
	default:
		i, err = strconv.ParseInt(envval, base10, 0)
	case typeINT8:
		i, err = strconv.ParseInt(envval, base10, bits8)
	case typeINT16:
		i, err = strconv.ParseInt(envval, base10, bits16)
	case typeINT32:
		i, err = strconv.ParseInt(envval, base10, bits32)
	case typeINT64:
		i, err = strconv.ParseInt(envval, base10, bits64)
	}

	if err != nil {
		return i, fmt.Errorf("parsing integer: %w", err)
	}

	return i, nil
}
