package cnfg

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalMap(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	pairs := map[string]string{
		"FOO": "bar",
		"BAZ": "yup",
	}

	type mapTester struct {
		Foo string `xml:"foo"`
		Baz string `xml:"baz"`
	}

	i := mapTester{}
	ok, err := UnmarshalMap(pairs, &i)

	a.Nil(err)
	a.True(ok)
	a.EqualValues("bar", i.Foo)

	ok, err = UnmarshalMap(pairs, i)

	a.False(ok)
	a.NotNil(err, "must have an error when attempting unmarshal to non-pointer")

	ok, err = (&ENV{}).UnmarshalMap(pairs, &i)
	a.True(ok)
	a.Nil(err)
}

func ExampleUnmarshalMap() {
	type myConfig struct {
		Key    string `xml:"envkey"`
		Key2   string `xml:"envkey2"`
		Nested struct {
			SubSlice []string          `xml:"subslice"`
			SubMap   map[string]string `xml:"submap"`
		} `xml:"nested"`
	}

	// Create a pointer to unmarshal your map into.
	i := &myConfig{}

	// Generally you'd use MapEnvPairs() to create a map from a slice of []string.
	// You can also get your data from any other source, as long as it can be
	// formatted into a map.
	// The important part is formatting the map keys correctly. The struct tag names
	// are always upcased, but nested struct member maps are not. They can be any case.
	// Each nested struct is appended to the parent name(s) with an underscore _.
	// Slices (except byte slices) are accessed by their position, beginning with 0.
	pairs := make(Pairs)
	pairs["ENVKEY"] = "some env value"
	pairs["ENVKEY2"] = "some other env value"
	pairs["NESTED_SUBSLICE_0"] = "first slice value"
	pairs["NESTED_SUBMAP_mapKey"] = "first map key value"

	ok, err := UnmarshalMap(pairs, i)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ok: %v, key: %v, key2: %v\n", ok, i.Key, i.Key2)
	fmt.Println("map:", i.Nested.SubMap)
	fmt.Println("slice:", i.Nested.SubSlice)
	// Output: ok: true, key: some env value, key2: some other env value
	// map: map[mapKey:first map key value]
	// slice: [first slice value]
}

// MapEnvPairs can be used when you want to inspect or modify the environment
// variable values before unmarshaling them.
func ExampleMapEnvPairs() {
	type myConfig struct {
		Key  string `env:"envkey"`
		Key2 string `env:"envkey2"`
		Key3 string `env:"envkey3"`
	}

	os.Setenv("TESTAPP_ENVKEY", "some env value")
	os.Setenv("TESTAPP_ENVKEY2", "some other env value")

	// Create pairs from the current environment.
	// Only consider environment variables that begin with "TESTAPP"
	pairs := MapEnvPairs("TESTAPP", os.Environ())
	for k, v := range pairs {
		fmt.Println(k, v)
	}

	// This is the magic offered by this method.
	pairs["TESTAPP_ENVKEY3"] = "add (or overwrite) a third value in code"
	i := &myConfig{}

	// We have to use &ENV{} to set a custom prefix, and change the struct tag.
	ok, err := (&ENV{Pfx: "TESTAPP", Tag: "env"}).UnmarshalMap(pairs, i)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ok: %v, key: %v, key2: %v, key3: %v\n", ok, i.Key, i.Key2, i.Key3)
	// Unordered Output: TESTAPP_ENVKEY some env value
	// TESTAPP_ENVKEY2 some other env value
	// ok: true, key: some env value, key2: some other env value, key3: add (or overwrite) a third value in code
}
