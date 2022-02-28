package cnfg_test

import (
	"errors"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golift.io/cnfg"
)

//nolint:staticcheck
type testStruct struct {
	PointerSlice  []*testSubConfig `json:"pslice" xml:"pslice" yaml:"pslice" toml:"pslice"`
	StructSlice   []testSubConfig  `json:"sslice" xml:"sslice" yaml:"sslice" toml:"sslice"`
	Struct        testSubConfig    `json:"struct" xml:"struct" yaml:"struct" toml:"struct"`
	PointerStruct *testSubConfig   `json:"pstruct" xml:"pstruct" yaml:"pstruct" toml:"pstruct"`
	// These dont get targeted during unmarhsal (not in the files).
	PointerSlice2  []*testSubConfig `json:"pslice2" xml:"pslice2,delenv" yaml:"pslice2" toml:"pslice2"`
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

// A few tests hit this.
func testUnmarshalFileValues(assert *assert.Assertions, c *testStruct, err error, from string) { //nolint:varnamelen
	from += " "

	assert.Nil(err, "there should not be an error reading the test file")
	// PointerSlice
	assert.Equal(1, len(c.PointerSlice), from+"pointerslice is too short")
	assert.EqualValues(true, c.PointerSlice[0].Bool, from+"the boolean was true")
	assert.EqualValues(123.4567, *c.PointerSlice[0].FloatP, from+"the float64 was set to 123.4567")
	assert.EqualValues(0, c.PointerSlice[0].Int, from+"int was not set so should be zero")
	assert.Nil(c.PointerSlice[0].StringP, from+"the string pointer was not set so should remain nil")

	// StructSlice
	assert.Equal(1, len(c.StructSlice), from+"pointerslice is too short")
	assert.EqualValues(false, c.StructSlice[0].Bool, from+"the boolean was missing and should be false")
	assert.Nil(c.StructSlice[0].FloatP, from+"the float64 was missing and should be nil")
	assert.EqualValues(123, c.StructSlice[0].Int, from+"int was set to 123")
	assert.EqualValues("foo", *c.StructSlice[0].StringP, from+"the string was set to foo")

	// Struct
	assert.EqualValues(false, c.Struct.Bool, from+"the boolean was false and should be false")
	assert.Nil(c.Struct.FloatP, from+"the float64 was missing and should be nil")
	assert.EqualValues(0, c.Struct.Int, from+"int was not set and must be 0")
	assert.Nil(c.Struct.StringP, from+"the string was missing and should be nil")

	// PointerStruct
	assert.NotNil(c.PointerStruct, from+"the pointer struct has values and must not be nil")
	assert.EqualValues(false, c.PointerStruct.Bool, from+"the boolean was missing and should be false")
	assert.Nil(c.PointerStruct.FloatP, from+"the float64 was missing and should be nil")
	assert.EqualValues(0, c.PointerStruct.Int, from+"int was not set and must be 0")
	assert.EqualValues("foo2", *c.PointerStruct.StringP, from+"the string was set to foo2")

	// PointerSlice2
	assert.Equal(0, len(c.PointerSlice2), from+"pointerslice2 is too long")
	// StructSlice2
	assert.Equal(0, len(c.StructSlice2), from+"structslice2 is too long")
	// Struct2
	assert.EqualValues(false, c.Struct2.Bool, from+"this must be zero value")
	assert.Nil(c.Struct2.FloatP, from+"this must be zero value")
	assert.EqualValues(0, c.Struct2.Int, from+"this must be zero value")
	assert.Nil(c.Struct2.StringP, from+"this must be zero value")
	// PointerStruct2
	assert.Nil(c.PointerStruct2, from+"pointer struct 2 must be nil")
}

func TestBrokenENV(t *testing.T) { //nolint:paralleltest // cannot parallel env vars.
	type testBroken struct {
		Broke []interface{} `xml:"broke"`
	}

	type testBroken2 struct {
		Broke map[interface{}]string `xml:"broke"`
	}

	type testBroken3 struct {
		Broke map[string]interface{} `xml:"broke"`
	}

	t.Setenv("TEST_BROKE_0", "f00")
	t.Setenv("TEST_BROKE_broke", "foo")

	assert := assert.New(t)
	c := &testBroken{}
	ook, err := cnfg.UnmarshalENV(c, "TEST")
	assert.False(ook)
	assert.NotNil(err, "an error must be returned for an unsupported type")

	c2 := &testBroken2{}
	ook, err = cnfg.UnmarshalENV(c2, "TEST")

	assert.NotNil(err, "an error must be returned for an unsupported map type")
	assert.False(ook)

	c3 := &testBroken3{}
	ook, err = cnfg.UnmarshalENV(c3, "TEST")

	assert.NotNil(err, "an error must be returned for an unsupported map type")
	assert.False(ook)
}

func TestUnmarshalENVerrors(t *testing.T) { //nolint:paralleltest // cannot parallel env vars.
	assert := assert.New(t)

	type tester struct {
		unexpd map[string]string
		Works  map[string]string `xml:"works,delenv"` //nolint:staticcheck
		Rad    map[string][]int  `xml:"yup"`
		Error  error             `xml:"error"`
	}

	t.Setenv("YO_WORKS_foostring", "fooval")
	t.Setenv("YO_WORKS_foo2string", "foo2val")
	t.Setenv("YO_YUP_server99_0", "128")
	t.Setenv("YO_YUP_server99_1", "129")
	t.Setenv("YO_YUP_server99_2", "130")
	t.Setenv("YO_YUP_server100_0", "256")
	t.Setenv("YO_ERROR", "this is an error")

	testStruct1 := tester{}
	ook, err := cnfg.UnmarshalENV(&testStruct1, "YO")
	assert.True(ook)
	assert.Nil(err, "maps are supported and must not produce an error")
	assert.Empty(os.Getenv("YO_WORKS_foo2string"), "delenv must delete the environment variable")
	assert.Empty(os.Getenv("YO_WORKS_foostring"), "delenv must delete the environment variable")
	assert.Nil(testStruct1.unexpd)
	assert.Equal("fooval", testStruct1.Works["foostring"])
	assert.Equal("foo2val", testStruct1.Works["foo2string"])
	assert.Equal([]int{128, 129, 130}, testStruct1.Rad["server99"])
	assert.Equal([]int{256}, testStruct1.Rad["server100"])
	assert.Equal(errors.New("this is an error"), testStruct1.Error) // nolint: goerr113

	type tester2 struct {
		NotBroken  []map[string]string  `xml:"broken"`
		NotBroken2 []*map[string]string `xml:"broken2"`
		NotBroken3 []map[int]int        `xml:"broken3"`
		HasStuff   []map[string]string  `xml:"stuff"`
	}

	t.Setenv("MORE_BROKEN", "value")
	t.Setenv("MORE_BROKEN_0_freesauce", "at-charlies")
	t.Setenv("MORE_BROKEN2_0_freesoup", "at-daves")
	t.Setenv("MORE_STUFF_0_freesoda", "not-at-pops")
	t.Setenv("MORE_STUFF_0_freetime", "at-pops")
	t.Setenv("MORE_STUFF_0_a", "")

	testStruct := tester2{HasStuff: []map[string]string{{"freesoda": "at-pops"}, {"a": "v"}}}
	ok, err := cnfg.UnmarshalENV(&testStruct, "MORE")
	assert.True(ok)
	assert.Nil(err, "map slices are supported and must not produce an error")

	f := *testStruct.NotBroken2[0]
	assert.EqualValues("at-charlies", testStruct.NotBroken[0]["freesauce"])
	assert.EqualValues("at-daves", f["freesoup"])
	assert.EqualValues("not-at-pops", testStruct.HasStuff[0]["freesoda"])
	assert.EqualValues("at-pops", testStruct.HasStuff[0]["freetime"])
	assert.EqualValues("", testStruct.HasStuff[0]["a"], "the empty map value must be set when the env var is empty")
	assert.Nil(testStruct.NotBroken3, "a nil map without overrides must remain nil")
}

// do not run this in parallel with other tests that change environment variables.
func TestUnmarshalENV(t *testing.T) { //nolint:paralleltest // cannot parallel env vars.
	assert := assert.New(t)
	c := &testStruct{}
	ok, err := cnfg.UnmarshalENV(c, "PRE")

	assert.Nil(err, "there must not be an error when parsing no variables")
	assert.False(ok, "there are no environment variables set, so ok should be false")
	testThingENV(t, assert)
	testOscureENV(t, assert)
	testSpecialENV(t, assert)

	f := true
	g := &f
	_, err = cnfg.UnmarshalENV(g, "OOO")
	assert.NotNil(err, "unmarshaling a non-struct pointer must produce an error")
}

func testThingENV(t *testing.T, assert *assert.Assertions) {
	t.Helper()
	os.Clearenv()
	t.Setenv("PRE_PSLICE_0_BOOL", "true")
	t.Setenv("PRE_PSLICE_0_FLOAT", "123.4567")

	t.Setenv("PRE_SSLICE_0_STRING", "foo")
	t.Setenv("PRE_SSLICE_0_INT", "123")

	t.Setenv("PRE_STRUCT_BOOL", "false")
	t.Setenv("PRE_PSTRUCT_STRING", "foo2")

	testStruct := &testStruct{}

	ok, err := cnfg.UnmarshalENV(testStruct, "PRE")
	assert.True(ok, "ok must be true since things must be parsed")
	testUnmarshalFileValues(assert, testStruct, err, "testThingENV")
	// do it again, and we should get the same result
	ok, err = cnfg.UnmarshalENV(testStruct, "PRE")
	assert.True(ok, "ok must be true since things must be parsed")
	testUnmarshalFileValues(assert, testStruct, err, "testThingENV")
}

func testOscureENV(t *testing.T, assert *assert.Assertions) {
	t.Helper()

	type testObscure struct {
		FloatSlice []float32        `xml:"is"`
		UintSliceP []*uint16        `xml:"uis"`
		Weirdo     *[]int           `xml:"psi"`
		Wut        *[]testSubConfig `xml:"wut"`
	}

	os.Clearenv()
	t.Setenv("OB_IS_0", "-5")
	t.Setenv("OB_IS_1", "8")

	t.Setenv("OB_UIS_0", "12")
	t.Setenv("OB_UIS_1", "22")

	t.Setenv("OB_PSI_0", "-1")
	t.Setenv("OB_WUT_0_BOOL", "true")

	config := &testObscure{}
	testit := func() {
		ok, err := cnfg.UnmarshalENV(config, "OB")
		assert.True(ok, "ok must be true since things must be parsed")
		assert.Nil(err)

		assert.EqualValues(2, len(config.FloatSlice))
		assert.EqualValues(-5, config.FloatSlice[0])
		assert.EqualValues(8, config.FloatSlice[1])

		assert.EqualValues(2, len(config.UintSliceP))
		assert.EqualValues(12, *config.UintSliceP[0])
		assert.EqualValues(22, *config.UintSliceP[1])

		assert.NotNil(config.Weirdo)
		assert.NotNil(config.Wut)

		weirdo := *config.Weirdo
		wut := *config.Wut

		assert.EqualValues(1, len(weirdo))
		assert.EqualValues(-1, weirdo[0])
		assert.EqualValues(1, len(wut))
		assert.True(wut[0].Bool)
	}

	testit()
	testit() // twice to make sure it's idempotent
}

func testSpecialENV(t *testing.T, assert *assert.Assertions) {
	t.Helper()

	type testSpecial struct {
		Dur  time.Duration    `xml:"dur"`
		CDur cnfg.Duration    `xml:"cdur"`
		Time time.Time        `xml:"time"`
		Durs *[]time.Duration `xml:"durs"`
		Sub  *struct {
			URL url.URL `xml:"url"`
			IP  net.IP  `xml:"ip"`
		} `xml:"sub"`
	}

	os.Clearenv()
	t.Setenv("TEST_DUR", "1m")
	t.Setenv("TEST_CDUR", "1s")
	t.Setenv("TEST_TIME", "2019-12-18T00:35:49+08:00")
	t.Setenv("TEST_SUB_URL", "https://golift.io/cnfg?rad=true")
	t.Setenv("TEST_SUB_IP", "123.45.67.89")

	config := &testSpecial{}
	ook, err := (&cnfg.ENV{Pfx: "TEST"}).Unmarshal(config)
	assert.True(ook, "ok must be true since things must be parsed")
	assert.Nil(err)
	assert.Equal(time.Minute, config.Dur)
	assert.Equal(time.Second, config.CDur.Duration)
	assert.Equal("golift.io", config.Sub.URL.Host, "the url wasn't parsed properly")
	assert.Equal("123.45.67.89", config.Sub.IP.String(), "the IP wasn't parsed properly")
	assert.Nil(config.Durs)

	t.Setenv("TEST_TIME", "not a real time")

	config = &testSpecial{}
	ook, err = (&cnfg.ENV{Pfx: "TEST"}).Unmarshal(config)

	assert.False(ook, "cannot parse an invalid time")
	assert.NotNil(err, "cannot parse an invalid time")
}
