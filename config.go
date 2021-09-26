package cnfg

import (
	"encoding"
	"encoding/json"
	"fmt"
	"time"
)

// ENVTag is the tag to look for on struct members. You may choose to use a custom
// tag by creating an &ENV{} struct with a different Tag. "env" is popular, but I
// chose "xml" because the nouns are generally singular, and those look good as
// env variables. "xml" is also convenient because it's brief and doesn't add yet
// another struct tag. Those lines can get long quickly.
const ENVTag = "xml"

// LevelSeparator is used to separate the names from different struct levels.
// This is hard coded here and cannot be changed or modified.
const LevelSeparator = "_"

// ENVUnmarshaler allows custom unmarshaling on a custom type.
// If your type implements this, it will be called and the logic stops there.
type ENVUnmarshaler interface {
	UnmarshalENV(tag, envval string) error
}

// ENVMarshaler allows marshaling custom types into env variables.
type ENVMarshaler interface {
	MarshalENV(tag string) (map[string]string, error)
}

// ENV allows you to parse environment variables using an object instead
// of global state. This package allows using the default ENVTag from global
// state, or you can pass in your own using this struct. See the UnmarshalENV
// function (it's 1 line) for an example of how to use this.
type ENV struct {
	Tag string // Struct tag name.
	Pfx string // ENV var prefix.
}

// Satify goconst.
const (
	TypeINT     = "int"
	TypeINT8    = "int8"
	TypeINT16   = "int16"
	TypeINT32   = "int32"
	TypeINT64   = "int64"
	TypeUINT    = "uint"
	TypeUINT8   = "uint8"
	TypeUINT16  = "uint16"
	TypeUINT32  = "uint32"
	TypeUINT64  = "uint64"
	TypeSTR     = "string"
	TypeFloat64 = "float64"
	TypeFloat32 = "float32"
	TypeBool    = "bool"
	TypeError   = "error"
	TypeDur     = "time.Duration"
)

const (
	base10 = 10
	bits8  = 8
	bits16 = 16
	bits32 = 32
	bits64 = 64
)

// The following is only used in tests, and perhaps externally.

// Duration is useful if you need to load a time Duration from a config file into
// your application. Use the config.Duration type to support automatic unmarshal
// from all sources. If you do not use a config file, do not use this type because
// the environment unmarshaler supports time.Duration natively.
type Duration struct{ time.Duration }

// UnmarshalText parses a duration type from a config file. This method works
// with the Duration type to allow unmarshaling of durations from files and
// env variables in the same struct. You won't generally call this directly.
func (d *Duration) UnmarshalText(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(b))

	if err != nil {
		return fmt.Errorf("parsing duration '%s': %w", b, err)
	}

	return nil
}

// MarshalText returns the string representation of a Duration. ie. 1m32s.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// MarshalJSON returns the string representation of a Duration for JSON. ie. "1m32s".
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Duration.String() + `"`), nil
}

// Make sure our struct satisfies the interface it's for.
var (
	_ encoding.TextUnmarshaler = (*Duration)(nil)
	_ encoding.TextMarshaler   = (*Duration)(nil)
	_ json.Marshaler           = (*Duration)(nil)
)
