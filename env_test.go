package cnfg_test

import (
	"fmt"
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
func testUnmarshalFileValues(a *assert.Assertions, c *testStruct, err error, from string) {
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

	a := assert.New(t)
	c := &testBroken{}
	ok, err := cnfg.UnmarshalENV(c, "TEST")

	a.NotNil(err, "an error must be returned for an unsupported type")
	a.False(ok)

	c2 := &testBroken2{}
	ok, err = cnfg.UnmarshalENV(c2, "TEST")

	a.NotNil(err, "an error must be returned for an unsupported map type")
	a.False(ok)

	c3 := &testBroken3{}
	ok, err = cnfg.UnmarshalENV(c3, "TEST")

	a.NotNil(err, "an error must be returned for an unsupported map type")
	a.False(ok)
}

func TestUnmarshalENVerrors(t *testing.T) { //nolint:paralleltest // cannot parallel env vars.
	a := assert.New(t)

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

	c := tester{}
	ok, err := cnfg.UnmarshalENV(&c, "YO")

	a.Nil(err, "maps are supported and must not produce an error")
	a.Empty(os.Getenv("YO_WORKS_foo2string"), "delenv must delete the environment variable")
	a.Empty(os.Getenv("YO_WORKS_foostring"), "delenv must delete the environment variable")
	a.True(ok)
	a.Nil(c.unexpd)
	a.Equal("fooval", c.Works["foostring"])
	a.Equal("foo2val", c.Works["foo2string"])
	a.Equal([]int{128, 129, 130}, c.Rad["server99"])
	a.Equal([]int{256}, c.Rad["server100"])
	a.Equal(fmt.Errorf("this is an error"), c.Error) // nolint: goerr113

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

	c2 := tester2{HasStuff: []map[string]string{{"freesoda": "at-pops"}, {"a": "v"}}}
	ok, err = cnfg.UnmarshalENV(&c2, "MORE")

	a.Nil(err, "map slices are supported and must not produce an error")
	a.True(ok)

	f := *c2.NotBroken2[0]
	a.EqualValues("at-charlies", c2.NotBroken[0]["freesauce"])
	a.EqualValues("at-daves", f["freesoup"])
	a.EqualValues("not-at-pops", c2.HasStuff[0]["freesoda"])
	a.EqualValues("at-pops", c2.HasStuff[0]["freetime"])
	a.EqualValues("", c2.HasStuff[0]["a"], "the empty map value must be set when the env var is empty")
	a.Nil(c2.NotBroken3, "a nil map without overrides must remain nil")
}

// do not run this in parallel with other tests that change environment variables.
func TestUnmarshalENV(t *testing.T) { //nolint:paralleltest // cannot parallel env vars.
	a := assert.New(t)
	c := &testStruct{}
	ok, err := cnfg.UnmarshalENV(c, "PRE")

	a.Nil(err, "there must not be an error when parsing no variables")
	a.False(ok, "there are no environment variables set, so ok should be false")
	testThingENV(t, a)
	testOscureENV(t, a)
	testSpecialENV(t, a)

	f := true
	g := &f
	_, err = cnfg.UnmarshalENV(g, "OOO")
	a.NotNil(err, "unmarshaling a non-struct pointer must produce an error")
}

func testThingENV(t *testing.T, a *assert.Assertions) {
	t.Helper()
	os.Clearenv()
	t.Setenv("PRE_PSLICE_0_BOOL", "true")
	t.Setenv("PRE_PSLICE_0_FLOAT", "123.4567")

	t.Setenv("PRE_SSLICE_0_STRING", "foo")
	t.Setenv("PRE_SSLICE_0_INT", "123")

	t.Setenv("PRE_STRUCT_BOOL", "false")
	t.Setenv("PRE_PSTRUCT_STRING", "foo2")

	c := &testStruct{}

	ok, err := cnfg.UnmarshalENV(c, "PRE")
	a.True(ok, "ok must be true since things must be parsed")
	testUnmarshalFileValues(a, c, err, "testThingENV")
	// do it again, and we should get the same result
	ok, err = cnfg.UnmarshalENV(c, "PRE")
	a.True(ok, "ok must be true since things must be parsed")
	testUnmarshalFileValues(a, c, err, "testThingENV")
}

func testOscureENV(t *testing.T, a *assert.Assertions) {
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

	c := &testObscure{}
	testit := func() {
		ok, err := cnfg.UnmarshalENV(c, "OB")
		a.True(ok, "ok must be true since things must be parsed")
		a.Nil(err)

		a.EqualValues(2, len(c.FloatSlice))
		a.EqualValues(-5, c.FloatSlice[0])
		a.EqualValues(8, c.FloatSlice[1])

		a.EqualValues(2, len(c.UintSliceP))
		a.EqualValues(12, *c.UintSliceP[0])
		a.EqualValues(22, *c.UintSliceP[1])

		a.NotNil(c.Weirdo)
		a.NotNil(c.Wut)

		weirdo := *c.Weirdo
		wut := *c.Wut

		a.EqualValues(1, len(weirdo))
		a.EqualValues(-1, weirdo[0])
		a.EqualValues(1, len(wut))
		a.True(wut[0].Bool)
	}

	testit()
	testit() // twice to make sure it's idempotent
}

func testSpecialENV(t *testing.T, a *assert.Assertions) {
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

	c := &testSpecial{}
	ok, err := (&cnfg.ENV{Pfx: "TEST"}).Unmarshal(c)

	a.True(ok, "ok must be true since things must be parsed")
	a.Nil(err)
	a.Equal(time.Minute, c.Dur)
	a.Equal(time.Second, c.CDur.Duration)
	a.Equal("golift.io", c.Sub.URL.Host, "the url wasn't parsed properly")
	a.Equal("123.45.67.89", c.Sub.IP.String(), "the IP wasn't parsed properly")
	a.Nil(c.Durs)

	t.Setenv("TEST_TIME", "not a real time")

	c = &testSpecial{}
	ok, err = (&cnfg.ENV{Pfx: "TEST"}).Unmarshal(c)

	a.False(ok, "cannot parse an invalid time")
	a.NotNil(err, "cannot parse an invalid time")
}
