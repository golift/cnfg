package cnfg

import (
	"errors"
	"os"
	"reflect"
	"strings"
)

// Custom errors this package may produce.
var (
	ErrUnsupported      = errors.New("unsupported type, please report this if this type should be supported")
	ErrInvalidByte      = errors.New("invalid byte")
	ErrInvalidInterface = errors.New("can only unmarshal ENV into pointer to struct")
)

// UnmarshalENV copies environment variables into configuration values.
// This is useful for Docker users that find it easier to pass ENV variables
// than a specific configuration file. Uses reflection to find struct tags.
func UnmarshalENV(i any, prefixes ...string) (bool, error) {
	return (&ENV{Pfx: strings.Join(prefixes, LevelSeparator), Tag: ENVTag}).Unmarshal(i)
}

// ParseENV is UnmarshalENV plus a Result (which env names set fields).
func ParseENV(i any, prefixes ...string) (Result, error) {
	return (&ENV{Pfx: strings.Join(prefixes, LevelSeparator), Tag: ENVTag}).Parse(i)
}

// Unmarshal parses and processes environment variables into the provided
// interface. Uses the Prefix and Tag name from the &ENV{} struct values.
func (e *ENV) Unmarshal(i any) (bool, error) {
	res, err := e.Parse(i)
	return res.Ok, err
}

// Parse is Unmarshal plus a Result so callers can inspect which env names applied.
func (e *ENV) Parse(i any) (Result, error) {
	return e.parsePairs(MapEnvPairs(e.Pfx, os.Environ()), i)
}

func (e *ENV) parsePairs(pairs Pairs, i any) (Result, error) {
	res := Result{Used: Pairs{}}
	value := reflect.ValueOf(i)

	if value.Kind() != reflect.Pointer || value.Elem().Kind() != reflect.Struct {
		return res, ErrInvalidInterface
	}

	if e.Tag == "" {
		e.Tag = ENVTag
	}

	parse := &parser{Low: e.Low, Tag: e.Tag, Vals: pairs}

	ok, err := parse.Struct(value, e.Pfx)
	res.Ok = ok

	if parse.Used != nil {
		res.Used = parse.Used
	}

	return res, err
}

// MarshalENV turns a data structure into an environment variable.
// The resulting slice can be copied into exec.Command.Env.
// Prefix is optional, and will prefix returned variables.
func MarshalENV(i any, prefix string) (Pairs, error) {
	return (&ENV{Pfx: prefix, Tag: ENVTag}).Marshal(i)
}

// Marshal deconstructs a data structure into environment variable pairs.
func (e *ENV) Marshal(i any) (Pairs, error) {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Pointer || value.Elem().Kind() != reflect.Struct {
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
