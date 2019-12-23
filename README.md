# golift.io/cnfg

[![Go Report Card](https://goreportcard.com/badge/golift.io/cnfg)](https://goreportcard.com/report/golift.io/cnfg)


Procedures for parsing config files and environment variables into data structures.
Works much like `json.Unmarshal`
Short explanation on how the env variable mapping works below.
See [GoDoc](https://godoc.org/golift.io/cnfg) for several working examples and
further explanation of how maps and slices can be accessed with shell env vars.

Supports all base types, including slices, maps, slices of maps, maps of slices,
pointers of slices to maps of slices full of ints, strings, floats and the like!

Please open an issue if you run into a bug or an unsupported type.

Better documentation is needed. Most of it is in [GoDoc](https://godoc.org/golift.io/cnfg).
This package is **full featured** for environment variable parsing!

```
type Shelter struct {
	Title  string    `xml:"title"`
	Sym    float64   `xml:"sym"`
	People []*Person `xml:"people"`
	Dogs   []*Dog    `xml:"dogs"`
}

type Person struct {
	Name    string `xml:"name"`
	Present bool   `xml:"present"`
	Age     int    `xml:"age"`
	ID      int64  `xml:"id"`
}

type Dog struct {
	Name    string
	Elapsed config.Duration
	Owners  []string
}

type Config struct {
	*Shelter `xml:"shelter"`
}
```
The above struct can be configured with the following environment variables,
assuming you set `prefix := "APP"` when you call `UnmarshalENV()`. Slices use env
vars with numbers in them, starting at 0 and going to infinity, or the last env
var provided + 1, whichever comes first. It just works. The `...` and `++` indicate
that those parameters belong to slices, and many items may be appended or overridden.
```
APP_SHELTER_TITLE
APP_SHELTER_SYM
APP_SHELTER_PEOPLE_0_NAME
APP_SHELTER_PEOPLE_0_PRESENT
APP_SHELTER_PEOPLE_0_AGE
APP_SHELTER_PEOPLE_0_ID
APP_SHELTER_PEOPLE_1_NAME
...
APP_SHELTER_PEOPLE_10_ID ++

APP_SHELTER_DOGS_0_NAME
APP_SHELTER_DOGS_0_ELAPSED
APP_SHELTER_DOGS_0_OWNERS_0
...
APP_SHELTER_DOGS_0_OWNERS_10 ++

APP_SHELTER_DOGS_1_NAME
APP_SHELTER_DOGS_1_ELAPSED
APP_SHELTER_DOGS_1_OWNERS_0
APP_SHELTER_DOGS_1_OWNERS_1 ++
```
If you passed in the `Shelter` struct instead of `Config`, all the of the `SHELTER_`
portions of the tags would be omitted. You can also set which struct tag to use by
creating an `&ENV{}` pointer and setting `Tag` and/or `Pfx` . `Tag` defaults to
`"xml"`, but you could set it to `"env"` and make custom names for env variables.
The env var prefix `Pfx` is optional, but recommended.
