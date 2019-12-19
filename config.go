package config

import (
	"encoding"
	"time"
)

// ENVTag is the tag to look for on struct members. You may choose to use a custom
// tag by changing this. "env" is popular, "json" is default and "xml" works well
// because XML tags are usually singular. Singular variable names work well for
// bash environment variables.
var ENVTag = "json"

// IgnoreUnknown controls the error returned by ParseENV when you try to parse
// unsupported types. As more types are added this becomes less of an issue.
// Most types are supported, but if you run into an unexpected error, set this to true.
// Setting this to true suppresses the error.
var IgnoreUnknown bool

// ENVUnmarshaler allows custom unmarshaling on a custom type.
// If your type implements this, it will be called and the logic stops there.
type ENVUnmarshaler interface {
	UnmarshalENV(tag, envval string) error
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
