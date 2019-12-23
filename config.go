package cnfg

import (
	"encoding"
	"time"
)

// ENVTag is the tag to look for on struct members. You may choose to use a custom
// tag by creating an &ENV{} struct with a different Tag. "env" is popular, but I
// chose "xml" because the nouns are generally singular, and those look good as
// env variables. "xml" is also convenient because it's brief and doesn't add yet
// another struct tag. Those lines can get long quickly.
const ENVTag = "xml"

// ENVUnmarshaler allows custom unmarshaling on a custom type.
// If your type implements this, it will be called and the logic stops there.
type ENVUnmarshaler interface {
	UnmarshalENV(tag, envval string) error
}

// ENV allows you to parse environment variables using an object instead
// of global state. This package allows using the default ENVTag from global
// state, or you can pass in your own using this struct. See the UnmarshalENV
// function (it's 1 line) for an example of how to use this.
type ENV struct {
	Tag string // Struct tag name.
	Pfx string // ENV var prefix.
}

type parser struct {
	Tag  string // struct tag to look for on struct values
	Vals Pairs  // pairs of env variables (saved at start)
}

// satify goconst
const (
	typeINT     = "int"
	typeINT8    = "int8"
	typeINT16   = "int16"
	typeINT32   = "int32"
	typeINT64   = "int64"
	typeUINT    = "uint"
	typeUINT8   = "uint8"
	typeUINT16  = "uint16"
	typeUINT32  = "uint32"
	typeUINT64  = "uint64"
	typeSTR     = "string"
	typeFloat64 = "float64"
	typeFloat32 = "float32"
	typeBool    = "bool"
	typeError   = "error"
	typeDur     = "time.Duration"
)

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
	return
}

// Make sure our struct satisfies the interface it's for.
var _ encoding.TextUnmarshaler = (*Duration)(nil)
