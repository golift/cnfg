package cnfg

import (
	"encoding"
	"errors"
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
	Low  bool   // allow lowercase variables?
	Tag  string // struct tag to look for on struct members
	Vals Pairs  // pairs of env variables (saved at start)
}

// Struct does most of the heavy lifting. Called every time a struct is encountered.
// The entire process begins here. It's very recursive.
func (p *parser) Struct(field reflect.Value, prefix string) (bool, error) {
	var exitOk bool

	t := field.Type().Elem()
	for idx := 0; idx < t.NumField(); idx++ { // Loop each struct member
		tagval := strings.Split(t.Field(idx).Tag.Get(p.Tag), ",")
		shorttag := tagval[0]

		if !p.Low {
			shorttag = strings.ToUpper(tagval[0]) // like "NAME" or "TIMEOUT"
		}

		if !field.Elem().Field(idx).CanSet() || shorttag == "-" {
			continue // This _only_ works with reflection tags.
		}

		delenv := false

		for i := 1; i < len(tagval); i++ {
			if tagval[i] == "delenv" {
				delenv = true
			}
		}

		tag := strings.Trim(strings.Join([]string{prefix, shorttag}, LevelSeparator), LevelSeparator) // PFX_NAME, PFX_TIMEOUT
		envval, found := p.Vals[tag]                                                                  // see if it exists

		//		log.Print("tag ", tag, " = ", envval)
		exists, err := p.Anything(field.Elem().Field(idx), tag, envval, found, delenv)
		if err != nil {
			return false, err
		} else if exists {
			exitOk = true
		}
	}

	return exitOk, nil
}

//nolint:cyclop
func (p *parser) Anything(field reflect.Value, tag, envval string, force, delenv bool) (bool, error) {
	//	log.Println("Anything", envval, tag, field.Kind(), field.Type(), field.Interface())
	if exists, err := p.Interface(field, tag, envval, force); err != nil {
		return false, err
	} else if exists {
		return true, nil
	}

	switch field.Kind() {
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
			_ = os.Unsetenv(tag) // delete it if it was requested in the env tag.
		}

		if !force && envval == "" {
			return false, nil
		}

		return p.Member(field, tag, envval, force)
	}
}

func (p *parser) Pointer(field reflect.Value, tag, envval string, delenv bool) (bool, error) {
	value := reflect.New(field.Type().Elem())
	if field.Elem().CanAddr() {
		// if the pointer already has a value, copy it instead of use the new one.
		value = field.Elem().Addr()
	}

	// Pass the non-pointer element back into the start.
	found, err := p.Anything(value.Elem(), tag, envval, false, delenv)
	if found {
		// overwrite the pointer only if something was parsed.
		field.Set(value)
	}

	return found, err
}

//nolint:cyclop // it's complicated, and probably not worth breaking up.
func (p *parser) Interface(field reflect.Value, tag, envval string, force bool) (bool, error) {
	if !field.CanAddr() || !field.Addr().CanInterface() {
		return false, nil
	}

	if envval == "" && !force {
		return false, nil
	}

	if v, ok := field.Addr().Interface().(ENVUnmarshaler); ok {
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
func (p *parser) Member(field reflect.Value, tag, envval string, force bool) (bool, error) { //nolint:cyclop
	var err error

	// Errors cannot be type-switched from reflection for some reason.
	if field.Type().String() == "error" {
		field.Set(reflect.ValueOf(errors.New(envval))) //nolint:err113

		return true, nil
	}

	switch fieldType := field.Interface().(type) {
	// Handle each member type appropriately (differently).
	case string:
		// SetString is a reflect package method to update a struct member by index.
		field.SetString(envval)
	case uint, uint8, uint16, uint32, uint64:
		err = parseUint(field, fieldType, envval)
	case int, int8, int16, int32, int64:
		var val int64

		val, err = parseInt(fieldType, envval)
		field.SetInt(val)
	case float64:
		var val float64

		val, err = strconv.ParseFloat(envval, bits64)
		field.SetFloat(val)
	case float32:
		var val float64

		val, err = strconv.ParseFloat(envval, bits32)
		field.SetFloat(val)
	case time.Duration:
		var val time.Duration

		val, err = time.ParseDuration(envval)
		field.Set(reflect.ValueOf(val))
	case bool:
		var val bool

		val, err = strconv.ParseBool(envval)
		field.SetBool(val)
	default:
		return p.customMember(field, tag, envval, force)
	}

	if err != nil {
		return false, fmt.Errorf("%s: %w", tag, err)
	}

	return true, nil
}

func (p *parser) customMember(field reflect.Value, tag, envval string, force bool) (bool, error) { //nolint:cyclop
	// Check if this type matches an unmarshaller interface.
	found, err := p.Interface(field, tag, envval, force)
	if err != nil || found {
		return found, err
	}

	switch kind := field.Kind(); kind {
	// Handle each member type appropriately (differently).
	case reflect.String:
		// SetString is a reflect package method to update a struct member by index.
		field.SetString(envval)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		err = parseUint(field, field.Interface(), envval)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var val int64

		val, err = parseInt(field.Interface(), envval)
		field.SetInt(val)
	case reflect.Float64:
		var val float64

		val, err = strconv.ParseFloat(envval, bits64)
		field.SetFloat(val)
	case reflect.Float32:
		var val float64

		val, err = strconv.ParseFloat(envval, bits32)
		field.SetFloat(val)
	case reflect.Bool:
		var val bool

		val, err = strconv.ParseBool(envval)
		field.SetBool(val)
	default:
		return false, fmt.Errorf("%w: type: %T, kind: %v, value: %s", ErrUnsupported, field.Interface(), field.Kind(), envval)
	}

	if err != nil {
		return false, fmt.Errorf("%s: %w", tag, err)
	}

	return true, nil
}

func (p *parser) Slice(field reflect.Value, tag string, delenv bool) (bool, error) {
	value := field
	reflect.Copy(value, field)

	var (
		found bool
		err   error
	)

	// slice of bytes works differently than any other slice type.
	if value.Type().String() == "[]uint8" {
		envval, exists := p.Vals[tag]
		found = exists

		value.SetBytes([]byte(envval))
	} else {
		found, err = p.SliceValue(value, tag, delenv)
	}

	if delenv {
		_ = os.Unsetenv(tag) // delete it if it was requested in the env tag.
	}

	if found {
		field.Set(value) // Overwrite the slice.
	}

	return found, err
}

func (p *parser) SliceValue(field reflect.Value, tag string, delenv bool) (bool, error) {
	var found bool

	total := field.Len()
	for idx := 0; idx <= total; idx++ {
		ntag := strings.Join([]string{tag, strconv.Itoa(idx)}, LevelSeparator)
		envval, exists := p.Vals[ntag]

		if delenv {
			_ = os.Unsetenv(ntag) // delete it if it was requested in the env tag.
		}

		// Start with a blank value for this item
		value := reflect.Indirect(reflect.New(field.Type().Elem()))
		if idx < field.Len() {
			// Use the passed in value if it exists.
			value = reflect.Indirect(field.Index(idx).Addr())
		}

		if exists, err := p.Anything(value, ntag, envval, exists, delenv); err != nil {
			return false, err
		} else if !exists {
			continue
		}

		found = true

		if idx >= field.Len() {
			total++ // do one more iteration.

			// The position in the ENV var is > slice size, so append.
			field.Set(reflect.Append(field, value))

			continue
		}

		// The position in the ENV var exists! Overwrite slice index directly.
		field.Index(idx).Set(value)
	}

	return found, nil
}

func (p *parser) Map(field reflect.Value, tag string, delenv bool) (bool, error) {
	var found bool

	vals := p.Vals.Get(tag) // key=val, ... (prefix stripped)
	if len(vals) < 1 {
		return false, nil
	}

	if field.IsNil() {
		field.Set(reflect.MakeMap(field.Type()))
	}

	for key, val := range vals {
		if delenv {
			_ = os.Unsetenv(key)
		}

		// Maps have 2 types. The index and the value. First, parse the index into its type.
		keyval := reflect.Indirect(reflect.New(field.Type().Key()))
		if _, err := p.Anything(keyval, tag, key, true, delenv); err != nil {
			return false, err
		}

		if val == "" {
			// a blank env value was provided, set the field to nil.
			found = true

			field.SetMapIndex(keyval, reflect.Value{})

			continue
		}

		// And now parse the second type: the value.
		valval := reflect.Indirect(reflect.New(field.Type().Elem()))

		exists, err := p.Anything(valval, strings.Join([]string{tag, key}, LevelSeparator), val, true, delenv)
		if err != nil {
			return false, err
		}

		if exists {
			found = true
		}

		field.SetMapIndex(keyval, valval)
	}

	return found, nil
}

// parseUint parses an unsigned integer from a string as specific size.
func parseUint(field reflect.Value, intType any, envval string) error {
	var (
		err error
		val uint64
	)

	switch intType.(type) {
	default:
		val, err = strconv.ParseUint(envval, base10, 0)
	case uint8:
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
	case uint16:
		val, err = strconv.ParseUint(envval, base10, bits16)
	case uint32:
		val, err = strconv.ParseUint(envval, base10, bits32)
	case uint64:
		val, err = strconv.ParseUint(envval, base10, bits64)
	}

	if err != nil {
		return fmt.Errorf("parsing integer: %w", err)
	}

	field.SetUint(val)

	return nil
}

// parseInt parses an integer from a string as specific size.
func parseInt(intType any, envval string) (int64, error) {
	var (
		out int64
		err error
	)

	switch intType.(type) {
	default:
		out, err = strconv.ParseInt(envval, base10, 0)
	case int8:
		out, err = strconv.ParseInt(envval, base10, bits8)
	case int16:
		out, err = strconv.ParseInt(envval, base10, bits16)
	case int32:
		out, err = strconv.ParseInt(envval, base10, bits32)
	case int64:
		out, err = strconv.ParseInt(envval, base10, bits64)
	}

	if err != nil {
		return out, fmt.Errorf("parsing integer: %w", err)
	}

	return out, nil
}
