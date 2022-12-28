// Package cnfg provides procedures to parse a slew of environment variables
// into a struct pointer.
//
// Use this package if your app uses a config file and you want to allow users to change
// or override the configurations in that file with environment variables. You can also
// use this app just to parse environment variables; handling a config file is entirely
// optional. Every member type is supported. If I missed one, please open an issue.
// If you need a non-base type supported (net.IP works already) please open an issue.
// New types are extremely easy to add. If this package interests you, pull requests
// and feature requests are welcomed!
//
// I consider this package the pinnacle example of how to configure Go applications from a file.
// You can put your configuration into any file format: XML, YAML, JSON, TOML, and you can override
// any struct member using an environment variable. I created this package because I got tired of
// writing custom env parser code for every app I make. This simplifies all the heavy lifting and I
// don't even have to think about it now. I hope you enjoy using this simplification as much as I do!
package cnfg
