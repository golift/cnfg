# golift.io/cnfg

[![Go Report Card](https://goreportcard.com/badge/golift.io/cnfg)](https://goreportcard.com/report/golift.io/cnfg)


Procedures for parsing configs files and environment variables into data structures.

Quick explanation on how the env variable mapping works below.
See [GODOC](https://godoc.org/golift.io/cnfg) for a working code example.

Supports almost every possible type. Please open an issue if you run into a bug
or an unsupported type.

```
type Shelter struct {
	Title  string    `json:"title"`
	Sym    float64   `json:"sym"`
	People []*Person `json:"people"`
	Dogs   []*Dog    `json:"dogs"`
}

type Person struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	Age     int    `json:"age"`
	ID      int64  `json:"id"`
}

type Dog struct {
	Name    string
	Elapsed config.Duration
	Owners  []string
}

type Config struct {
	*Shelter `json:"shelter"`
}
```
The above struct can be configured with the following environment variables,
assuming you set `prefix := "APP"` when you call `ParseENV()`. Slices use env
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
portions of the tags would be omitted. You can also set which struct tag to use with
`config.ENVTag` - it defaults to `"json"`, but you could set it to `"env"` and make
custom names for env variables.
