// Package config provide basic procedures to parse a config file into a struct,
// and more powerfully, parse a slew of environment variables into the same or
// a different struct. These two procedures can be used one after the other in
// either order (the latter overrides the former).
// As of now, this software is still very new and lacks great examples. It is in
// use in "production" but hasn't had a lot of different use cases applied to it.
//
// If this package interests you, pull requests and feature requests are welcomed!
//
// I consider this package the pinacle example of how to configure small Go applications from a file.
// You can put your configuration into any file format: XML, YAML, JSON, TOML, and you can override
// any struct member using an environment variable. As it is now, the (env) code lacks map{} support
// but pretty much any other base type and nested member is supported. Adding more/the rest will
// happen in time. I created this package because I got tired of writing custom env parser code for
// every app I make. This simplifies all the heavy lifting and I don't even have to think about it now.
package config

import (
	"encoding"
	"time"
)

// ENVTag is the tag to look for on struct members. "json" is default.
var ENVTag = "json"

// IgnoreUnknown controls the error returned by ParseENV when you try to parse
// unsupported types, like maps. As more types are added this becomes less of an issue.
// Setting this to true supresses the error.
var IgnoreUnknown bool

// ENVUnmarshaler allows custom unmarshaling on a custom type.
// If your type implements this, it will be called.
type ENVUnmarshaler interface {
	UnmarshalENV(tag, envval string) error
}

// Duration is useful if you need to load a time Duration from a config file into
// your application. Use the config.Duration type to support automatic unmarshal
// from all sources. If you do not use a config file, do not use this type because
// the envrionment unmarshaler supports time.Duration natively.
type Duration struct{ time.Duration }

// UnmarshalText parses a duration type from a config file.
func (d *Duration) UnmarshalText(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(b))
	return
}

var _ encoding.TextUnmarshaler = (*Duration)(nil)
