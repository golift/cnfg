// Package config provide basic procedures to parse a config file into a struct,
// and more powerfully, parse a slew of environment variables into the same or
// a different struct. These two procedures can be used one after the other in
// either order (the latter overrides the former).
// As of now, this software is still very new and lacks great examples. It is in
// use in "production" but hasn't had a lot of different use cases applied to it.
//
// If this package interests you, pull requests and feature requests are welcomed!
//
// I consider this package the pinacle example of how to configure a (small) Go application from a file.
// You can put your configuration into any file format: XML, YAML, JSON, TOML, and you can override any
// struct member using an environment variable. As it is now, the (env) code lacks map{} and *pointer
// support, but pretty much any other base type and nested member is supported. Adding more/the rest will
// happen in time. I created this package because I got tired of writing custom env parser code for every
// app I make. This simplifies all the heavy lifting and I don't even have to think about it now.
package config

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	yaml "gopkg.in/yaml.v2"
)

// ENVTag is the tag to look for on struct members. "json" is default.
var ENVTag = "json"

// ENVUnmarshaler allows custom unmarshaling on a custom type.
// If your type implements this, it will be called.
type ENVUnmarshaler interface {
	UnmarshalENV(tag, envval string) error
}

// Duration is used to UnmarshalTOML into a time.Duration value. This is for convenience.
// The environment parser also supports native time.Duration.
// This is most useful for config file parsing, that doesn't support native time.Duration.
type Duration struct{ time.Duration }

// UnmarshalText parses a duration type from a config file.
func (d *Duration) UnmarshalText(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(b))
	return
}

// ParseFile parses a configuration file (of any format) into a config struct.
func ParseFile(c interface{}, configFile string) error {
	switch buf, err := ioutil.ReadFile(configFile); {
	case err != nil:
		return err
	case strings.Contains(configFile, ".json"):
		return json.Unmarshal(buf, c)
	case strings.Contains(configFile, ".xml"):
		return xml.Unmarshal(buf, c)
	case strings.Contains(configFile, ".yaml"):
		return yaml.Unmarshal(buf, c)
	default:
		return toml.Unmarshal(buf, c)
	}
}

// ParseENV copies environment variables into configuration values.
// This is useful for Docker users that find it easier to pass ENV variables
// than a specific configuration file. Uses reflection to find struct tags.
// This method uses the json struct tag member to match environment variables.
// Use a custom tag name by changing "json" below, but that's overkill for this app.
func ParseENV(c interface{}, prefix string) (bool, error) {
	return parseStruct(reflect.ValueOf(c), reflect.TypeOf(c).Elem(), prefix)
}

func parseStruct(v reflect.Value, t reflect.Type, prefix string) (bool, error) {
	var exitOk, exists bool

	var err error

	for i := 0; i < t.NumField(); i++ { // Loop each struct member
		tag := strings.Split(strings.ToUpper(t.Field(i).Tag.Get(ENVTag)), ",")[0]
		ntag := prefix + "_" + tag

		switch field := v.Elem().Field(i); {
		case field.Kind() == reflect.Struct:
			exists, err = parseStruct(field.Addr(), field.Type(), ntag)
			if err != nil {
				return false, err
			}
		case field.Kind() == reflect.Slice:
			exists, err = parseSlice(field, ntag)
			if err != nil {
				return false, err
			}
		default:
			envval, ok := os.LookupEnv(ntag)
			if !ok || tag == "" {
				break // switch
			}

			exists = true

			if err = parseMember(field, ntag, envval); err != nil {
				return false, err
			}
		}

		if exists {
			exitOk = true
		}
	}

	return exitOk, nil
}

func parseMember(field reflect.Value, tag, envval string) error {
	log.Println(field.Type().String())
	// Reflect and update the u.Config struct member at position i.
	switch field.Type().String() {
	// Handle each member type appropriately (differently).
	case "string":
		// This is a reflect package method to update a struct member by index.
		field.SetString(envval)
	case "int":
		val, err := strconv.Atoi(envval)
		if err != nil {
			return fmt.Errorf("%s: %v", tag, err)
		}

		field.Set(reflect.ValueOf(val))
	case "float64":
		val, err := strconv.ParseFloat(envval, 64)
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
	case "config.Duration":
		val, err := time.ParseDuration(envval)
		if err != nil {
			return fmt.Errorf("%s: %v", tag, err)
		}

		field.Set(reflect.ValueOf(Duration{val}))
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

		return fmt.Errorf("%s: unsupported type: %v (val: %s) - Make a PR! :)", tag, field.Type(), envval)
	}

	return nil
}

func parseSlice(field reflect.Value, tag string) (bool, error) {
	if field.IsNil() {
		field.Set(reflect.MakeSlice(field.Type(), 0, 0))
	}

	/*	XXX: deal with a slice of pointers? hm.
		if field.Type().Elem().Kind() == reflect.Ptr {
				field = field.Elem().......stuff.
			}
	*/

	switch k := field.Type().Elem().Kind(); k {
	default:
		return false, fmt.Errorf("unsupported slice type: %v", k)
	case reflect.Ptr:
		return false, fmt.Errorf("slices of pointers don't work yet")
	case reflect.Map:
		return false, fmt.Errorf("maps don't work yet")
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

FORLOOP:
	for i := 0; ; i++ {
		ntag := tag + "_" + strconv.Itoa(i)
		value := reflect.New(field.Type().Elem())

		switch exists, err := parseStruct(value, field.Type().Elem(), ntag); {
		case err != nil:
			return false, err

		case !exists && i > field.Len():
			// We've checked all possible ENV var's up to slice count + 1, stop there if it's empty.
			break FORLOOP

		case exists:
			ok = true

			if i >= field.Len() {
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

	for i := 0; ; i++ {
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
			return ok, err
		}

		if i >= field.Len() {
			// The position in the ENV var is > slice size, so append.
			field.Set(reflect.Append(field, value))
			continue // check for the next slice member env var.
		}

		// The position in the ENV var exists! Overwrite slice index directly.
		field.Index(i).Set(value)
	}

	return ok, nil
}
