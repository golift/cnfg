package cnfgfile

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"golift.io/cnfg"
)

type testStruct struct {
	PointerSlice  []*testSubConfig `json:"pslice" xml:"pslice" yaml:"pslice" toml:"pslice"`
	StructSlice   []testSubConfig  `json:"sslice" xml:"sslice" yaml:"sslice" toml:"sslice"`
	Struct        testSubConfig    `json:"struct" xml:"struct" yaml:"struct" toml:"struct"`
	PointerStruct *testSubConfig   `json:"pstruct" xml:"pstruct" yaml:"pstruct" toml:"pstruct"`
	// These dont get targeted during unmarhsal (not in the files).
	PointerSlice2  []*testSubConfig `json:"pslice2" xml:"pslice2" yaml:"pslice2" toml:"pslice2"`
	StructSlice2   []testSubConfig  `json:"sslice2" xml:"sslice2" yaml:"sslice2" toml:"sslice2"`
	Struct2        testSubConfig    `json:"struct2" xml:"struct2" yaml:"struct2" toml:"struct2"`
	PointerStruct2 *testSubConfig   `json:"pstruct2" xml:"pstruct2" yaml:"pstruct2" toml:"pstruct2"`
}

type testSubConfig struct {
	Bool    bool     `json:"bool" xml:"bool" yaml:"bool" toml:"bool"`
	Int     int64    `json:"int" xml:"int" yaml:"int" toml:"int"`
	StringP *string  `json:"string" xml:"string" yaml:"string" toml:"string"`
	FloatP  *float64 `json:"float" xml:"float" yaml:"float" toml:"float"`
}

func testUnmarshalValues(a *assert.Assertions, c *testStruct, err error, from string) {
	from += " "

	a.Nil(err, "there should not be an error reading the test file")
	// PointerSlice
	a.Equal(1, len(c.PointerSlice), from+"pointerslice is too short")
	a.EqualValues(true, c.PointerSlice[0].Bool, from+"the boolean was true")
	a.EqualValues(123.4567, *c.PointerSlice[0].FloatP, from+"the float64 was set to 123.4567")
	a.EqualValues(0, c.PointerSlice[0].Int, from+"int was not set so should be zero")
	a.Nil(c.PointerSlice[0].StringP, from+"the string pointer was not set so should remain nil")

	// StructSlice
	a.Equal(1, len(c.StructSlice), from+"pointerslice is too short")
	a.EqualValues(false, c.StructSlice[0].Bool, from+"the boolean was missing and should be false")
	a.Nil(c.StructSlice[0].FloatP, from+"the float64 was missing and should be nil")
	a.EqualValues(123, c.StructSlice[0].Int, from+"int was set to 123")
	a.EqualValues("foo", *c.StructSlice[0].StringP, from+"the string was set to foo")

	// Struct
	a.EqualValues(false, c.Struct.Bool, from+"the boolean was false and should be false")
	a.Nil(c.Struct.FloatP, from+"the float64 was missing and should be nil")
	a.EqualValues(0, c.Struct.Int, from+"int was not set and must be 0")
	a.Nil(c.Struct.StringP, from+"the string was missing and should be nil")

	// PointerStruct
	a.NotNil(c.PointerStruct, from+"the pointer struct has values and must not be nil")
	a.EqualValues(false, c.PointerStruct.Bool, from+"the boolean was missing and should be false")
	a.Nil(c.PointerStruct.FloatP, from+"the float64 was missing and should be nil")
	a.EqualValues(0, c.PointerStruct.Int, from+"int was not set and must be 0")
	a.EqualValues("foo2", *c.PointerStruct.StringP, from+"the string was set to foo2")

	// PointerSlice2
	a.Equal(0, len(c.PointerSlice2), from+"pointerslice2 is too long")
	// StructSlice2
	a.Equal(0, len(c.StructSlice2), from+"structslice2 is too long")
	// Struct2
	a.EqualValues(false, c.Struct2.Bool, from+"this must be zero value")
	a.Nil(c.Struct2.FloatP, from+"this must be zero value")
	a.EqualValues(0, c.Struct2.Int, from+"this must be zero value")
	a.Nil(c.Struct2.StringP, from+"this must be zero value")
	// PointerStruct2
	a.Nil(c.PointerStruct2, from+"pointer struct 2 must be nil")
}

func TestUnmarshalErrors(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}

	err := Unmarshal(c, "/etc/passwd")
	a.NotNil(err, "there should be an error parsing a password file")

	err = Unmarshal(c, "no file here")
	a.NotNil(err, "there should be an error parsing a missing file")

	err = Unmarshal(c)
	a.NotNil(err, "there should be an error parsing a nil file")
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}
	err := Unmarshal(c, "tests/config.json")
	testUnmarshalValues(a, c, err, "TestUnmarshalJSON")
}

func TestUnmarshalXML(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}

	err := Unmarshal(c, "tests/config.xml")
	testUnmarshalValues(a, c, err, "TestUnmarshalXML")
}

func TestUnmarshalYAML(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}

	err := Unmarshal(c, "tests/config.yaml")
	testUnmarshalValues(a, c, err, "TestUnmarshalYAML")
}

func TestUnmarshalTOML(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}

	err := Unmarshal(c, "tests/config.toml")
	testUnmarshalValues(a, c, err, "TestUnmarshalTOML")
}

// The cnfgfile.Unmarshal() procedure can be used in place of: xml.Unmarshal,
// json.Unmarshal, yaml.Unmarshal and toml.Unmarshal. This procedure also reads
// in the provided file, so you don't need to do any of the io work beforehand.
// Using this procedure in your app allows your consumers to a use a config file
// format of their choosing. Very cool stuff when you consider _that file_ could
// just be a config file for a larger project.
func ExampleUnmarshal() {
	// Recommend adding tags for each type to your struct members. Provide full compatibility.
	type Config struct {
		Interval cnfg.Duration `json:"interval" xml:"interval" toml:"interval" yaml:"interval"`
		Location string        `json:"location" xml:"location" toml:"location" yaml:"location"`
		Provided bool          `json:"provided" xml:"provided" toml:"provided" yaml:"provided"`
	}

	// Create a test file with some test data to unmarshal.
	// YAML is just an example, you can use any supported format.
	yaml := []byte("---\ninterval: 5m\nlocation: Earth\nprovided: true")
	path := "/tmp/path_to_config.yaml"

	err := ioutil.WriteFile(path, yaml, 0600)
	if err != nil {
		panic(err)
	}

	// Start with an empty config. Or set some defaults beforehand.
	c := &Config{}

	// Simply pass in your config file. If it contains ".yaml" it will be parsed as YAML.
	// Same for ".xml" and ".json". If the file has none of these extensions it is parsed
	// as TOML. Meaning if you name your config "config.conf" it needs ot be TOML formatted.
	err = Unmarshal(c, path)
	if err != nil {
		panic(err)
	}

	fmt.Printf("interval: %v, location: %v, provided: %v", c.Interval, c.Location, c.Provided)
	// Output: interval: 5m0s, location: Earth, provided: true
}
