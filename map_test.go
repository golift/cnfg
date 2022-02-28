package cnfg_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golift.io/cnfg"
)

func TestUnmarshalMap(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	pairs := map[string]string{
		"FOO": "bar",
		"BAZ": "yup",
	}

	type mapTester struct {
		Foo string `xml:"foo"`
		Baz string `xml:"baz"`
	}

	testConfig := mapTester{}
	found, err := cnfg.UnmarshalMap(pairs, &testConfig)
	assert.True(found)
	assert.Nil(err)
	assert.EqualValues("bar", testConfig.Foo)

	found, err = cnfg.UnmarshalMap(pairs, testConfig)
	assert.False(found)
	assert.NotNil(err, "must have an error when attempting unmarshal to non-pointer")

	found, err = (&cnfg.ENV{}).UnmarshalMap(pairs, &testConfig)
	assert.True(found)
	assert.Nil(err)
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
	testConfig := &myConfig{}

	// Generally you'd use MapEnvPairs() to create a map from a slice of []string.
	// You can also get your data from any other source, as long as it can be
	// formatted into a map.
	// The important part is formatting the map keys correctly. The struct tag names
	// are always upcased, but nested struct member maps are not. They can be any case.
	// Each nested struct is appended to the parent name(s) with an underscore _.
	// Slices (except byte slices) are accessed by their position, beginning with 0.
	pairs := make(cnfg.Pairs)
	pairs["ENVKEY"] = "some env value"
	pairs["ENVKEY2"] = "some other env value"
	pairs["NESTED_SUBSLICE_0"] = "first slice value"
	pairs["NESTED_SUBMAP_mapKey"] = "first map key value"

	ok, err := cnfg.UnmarshalMap(pairs, testConfig)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ok: %v, key: %v, key2: %v\n", ok, testConfig.Key, testConfig.Key2)
	fmt.Println("map:", testConfig.Nested.SubMap)
	fmt.Println("slice:", testConfig.Nested.SubSlice)
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
	pairs := cnfg.MapEnvPairs("TESTAPP", os.Environ())
	for k, v := range pairs {
		fmt.Println(k, v)
	}

	// This is the magic offered by this method.
	pairs["TESTAPP_ENVKEY3"] = "add (or overwrite) a third value in code"
	config := &myConfig{}

	// We have to use &ENV{} to set a custom prefix, and change the struct tag.
	ok, err := (&cnfg.ENV{Pfx: "TESTAPP", Tag: "env"}).UnmarshalMap(pairs, config)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ok: %v, key: %v, key2: %v, key3: %v\n", ok, config.Key, config.Key2, config.Key3)
	// Unordered Output: TESTAPP_ENVKEY some env value
	// TESTAPP_ENVKEY2 some other env value
	// ok: true, key: some env value, key2: some other env value, key3: add (or overwrite) a third value in code
}
