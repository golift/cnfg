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
