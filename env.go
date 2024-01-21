package cnfg

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Custom errors this package may produce.
var (
	ErrUnsupported      = fmt.Errorf("unsupported type, please report this if this type should be supported")
	ErrInvalidByte      = fmt.Errorf("invalid byte")
	ErrInvalidInterface = fmt.Errorf("can only unmarshal ENV into pointer to struct")
)

// UnmarshalENV copies environment variables into configuration values.
// This is useful for Docker users that find it easier to pass ENV variables
// than a specific configuration file. Uses reflection to find struct tags.
func UnmarshalENV(i interface{}, prefixes ...string) (bool, error) {
	return (&ENV{Pfx: strings.Join(prefixes, LevelSeparator), Tag: ENVTag}).Unmarshal(i)
}

// Unmarshal parses and processes environment variables into the provided
// interface. Uses the Prefix and Tag name from the &ENV{} struct values.
func (e *ENV) Unmarshal(i interface{}) (bool, error) {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return false, ErrInvalidInterface
	}

	if e.Tag == "" {
		e.Tag = ENVTag
	}

	// Save the current environment.
	parse := &parser{Low: e.Low, Tag: e.Tag, Vals: MapEnvPairs(e.Pfx, os.Environ())}

	return parse.Struct(value, e.Pfx)
}

// MarshalENV turns a data structure into an environment variable.
// The resulting slice can be copied into exec.Command.Env.
// Prefix is optional, and will prefix returned variables.
func MarshalENV(i interface{}, prefix string) (Pairs, error) {
	return (&ENV{Pfx: prefix, Tag: ENVTag}).Marshal(i)
}

// Marshal converts deconstructs a data structure into environment variable pairs.
func (e *ENV) Marshal(i interface{}) (Pairs, error) {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return nil, ErrInvalidInterface
	}

	if e.Tag == "" {
		e.Tag = ENVTag
	}

	unparse := &unparser{Low: e.Low, Tag: e.Tag}

	pairs, err := unparse.DeconStruct(value, e.Pfx)
	if err != nil {
		return nil, err
	}

	return pairs, nil
}
