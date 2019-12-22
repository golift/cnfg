package cnfg

import (
	"fmt"
	"os"
	"reflect"
)

// UnmarshalENV copies environment variables into configuration values.
// This is useful for Docker users that find it easier to pass ENV variables
// than a specific configuration file. Uses reflection to find struct tags.
func UnmarshalENV(i interface{}, prefix string) (bool, error) {
	return (&ENV{Pfx: prefix, Tag: ENVTag}).UnmarshalENV(i)
}

// UnmarshalENV parses and processes environment variables into the provided
// interface. Uses the Prefix and Tag name from the &ENV{} struct values.
func (e *ENV) UnmarshalENV(i interface{}) (bool, error) {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("can only unmarshal ENV into pointer to struct")
	}

	// Save the current environment.
	e.pairs = MapEnvPairs(e.Pfx, os.Environ())

	if e.Tag == "" {
		e.Tag = ENVTag
	}

	return e.parseStruct(value, e.Pfx)
}
