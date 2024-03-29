package cnfg

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

/* This file contains the methods that convert a struct into environment variables. */

type unparser struct {
	Low bool   // Allow lowercase values in env variable names.
	Tag string // struct tag to look for on struct members
}

func (p *unparser) DeconStruct(field reflect.Value, prefix string) (Pairs, error) { //nolint:cyclop
	output := Pairs{}

	element := field.Type().Elem()
	for idx := 0; idx < element.NumField(); idx++ { // Loop each struct member
		tagval := strings.Split(element.Field(idx).Tag.Get(p.Tag), ",")
		tag := tagval[0]

		if !p.Low {
			tag = strings.ToUpper(tagval[0]) // like "NAME" or "TIMEOUT"
		}

		if !field.Elem().Field(idx).CanSet() || tag == "-" {
			continue
		}

		if tag == "" && !element.Field(idx).Anonymous {
			tag = element.Field(idx).Name
			if !p.Low {
				tag = strings.ToUpper(element.Field(idx).Name)
			}
		}

		tag = strings.Trim(strings.Join([]string{prefix, tag}, LevelSeparator), LevelSeparator)
		omitempty := false

		for i := 1; i < len(tagval); i++ {
			if tagval[i] == "omitempty" {
				omitempty = true
			}
		}

		o, err := p.Anything(field.Elem().Field(idx), tag, omitempty)
		if err != nil {
			return nil, err
		}

		output.Merge(o)
	}

	return output, nil
}

func (p *unparser) Anything(field reflect.Value, tag string, omitempty bool) (Pairs, error) { //nolint:cyclop
	if field.IsZero() && omitempty {
		return Pairs{}, nil
	}

	output, exists, err := p.Interface(field, tag, omitempty)
	if err != nil || exists {
		return output, err
	}

	switch field.Kind() {
	case reflect.Ptr:
		if !field.Elem().CanAddr() {
			return output, nil
		}

		// Pass the non-pointer element back into the start.
		return p.Anything(field.Elem().Addr().Elem(), tag, omitempty)
	case reflect.Struct:
		return p.DeconStruct(field.Addr(), tag)
	case reflect.Slice:
		return p.Slice(field, tag, omitempty)
	case reflect.Map:
		return p.Map(field, tag, omitempty)
	default:
		return p.Member(field, tag, omitempty)
	}
}

func (p *unparser) Interface(field reflect.Value, tag string, omitempty bool) (Pairs, bool, error) {
	output := Pairs{}

	if !field.CanAddr() || !field.Addr().CanInterface() {
		pairs, err := p.Member(reflect.ValueOf(field.Interface()), tag, omitempty)
		return pairs, true, err
	}

	if v, ok := field.Addr().Interface().(ENVMarshaler); ok {
		// Custom unmarshaler can proceed even if envval is empty. It may produce new envvals...
		o, err := v.MarshalENV(tag)
		if err != nil {
			return output, false, fmt.Errorf("MarshallENV interface: %w", err)
		}

		output.Merge(o)

		return output, true, nil
	}

	if tag == "" {
		return output, false, nil
	}

	if v, ok := field.Addr().Interface().(encoding.TextMarshaler); ok {
		text, err := v.MarshalText()
		if err != nil {
			return output, false, fmt.Errorf("MarshalText interface: %w", err)
		}

		output.Set(tag, string(text))

		return output, true, nil
	}

	if v, ok := field.Addr().Interface().(encoding.BinaryMarshaler); ok {
		bin, err := v.MarshalBinary()
		if err != nil {
			return output, false, fmt.Errorf("MarshalText interface: %w", err)
		}

		output.Set(tag, string(bin))

		return output, true, nil
	}

	return output, false, nil
}

// Member parses non-struct, non-slice struct-member types.
func (p *unparser) Member(field reflect.Value, tag string, omitempty bool) (Pairs, error) { //nolint:cyclop
	output := Pairs{}

	switch val := field.Interface().(type) {
	// Handle each member type appropriately (differently).
	case error:
		if err, _ := field.Interface().(error); err != nil {
			output.Set(tag, err.Error())
		}
	case string:
		output.Set(tag, field.String())
	case uint:
		output.Set(tag, strconv.FormatUint(uint64(val), base10))
	case uint8:
		output.Set(tag, strconv.FormatUint(uint64(val), base10))
	case uint16:
		output.Set(tag, strconv.FormatUint(uint64(val), base10))
	case uint32:
		output.Set(tag, strconv.FormatUint(uint64(val), base10))
	case uint64:
		output.Set(tag, strconv.FormatUint(val, base10))
	case int:
		output.Set(tag, strconv.FormatInt(int64(val), base10))
	case int8:
		output.Set(tag, strconv.FormatInt(int64(val), base10))
	case int16:
		output.Set(tag, strconv.FormatInt(int64(val), base10))
	case int32:
		output.Set(tag, strconv.FormatInt(int64(val), base10))
	case int64:
		output.Set(tag, strconv.FormatInt(val, base10))
	case float64:
		output.Set(tag, strconv.FormatFloat(val, 'f', -1, bits64))
	case float32:
		output.Set(tag, strconv.FormatFloat(float64(val), 'f', -1, bits32))
	case time.Duration:
		output.Set(tag, (time.Duration(field.Int()) * time.Nanosecond).String())
	case bool:
		output.Set(tag, strconv.FormatBool(field.Bool()))
	default:
		return p.kindMember(field, tag, omitempty)
	}

	return output, nil
}

func (p *unparser) kindMember(field reflect.Value, tag string, _ bool) (Pairs, error) {
	output := Pairs{}

	switch field.Kind() {
	case reflect.String:
		output.Set(tag, field.String())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, _ := field.Interface().(uint64)
		output.Set(tag, strconv.FormatUint(val, base10))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, _ := field.Interface().(int64)
		output.Set(tag, strconv.FormatInt(val, base10))
	case reflect.Float64:
		val, _ := field.Interface().(float64)
		output.Set(tag, strconv.FormatFloat(val, 'f', -1, bits64))
	case reflect.Float32:
		val, _ := field.Interface().(float32)
		output.Set(tag, strconv.FormatFloat(float64(val), 'f', -1, bits32))
	case reflect.Bool:
		output.Set(tag, strconv.FormatBool(field.Bool()))
	}

	return output, nil
}

func (p *unparser) Slice(field reflect.Value, tag string, omitempty bool) (Pairs, error) {
	output := Pairs{}

	// slice of bytes works differently than any other slice type.
	if field.Type().String() == "[]uint8" {
		output.Set(tag, string(field.Bytes()))

		return output, nil
	}

	return p.SliceValue(field, tag, omitempty)
}

func (p *unparser) SliceValue(field reflect.Value, tag string, omitempty bool) (Pairs, error) {
	output := Pairs{}

	total := field.Len()
	for i := 0; i < total; i++ {
		ntag := strings.Join([]string{tag, strconv.Itoa(i)}, LevelSeparator)
		value := reflect.Indirect(field.Index(i).Addr())

		o, err := p.Anything(value, ntag, omitempty)
		if err != nil {
			return output, err
		}

		output.Merge(o)
	}

	return output, nil
}

func (p *unparser) Map(field reflect.Value, tag string, omitempty bool) (Pairs, error) {
	output := Pairs{}

	for i := field.MapRange(); i.Next(); {
		ntag := fmt.Sprintf("%s%s%v", tag, LevelSeparator, i.Key())

		o, err := p.Anything(i.Value(), ntag, omitempty)
		if err != nil {
			return output, err
		}

		output.Merge(o)
	}

	return output, nil
}
