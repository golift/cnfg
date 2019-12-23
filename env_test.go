package cnfg

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/* >95% code coverage. */

func TestBrokenENV(t *testing.T) {
	type testBroken struct {
		Broke []interface{} `xml:"broke"`
	}

	type testBroken2 struct {
		Broke map[interface{}]string `xml:"broke"`
	}

	type testBroken3 struct {
		Broke map[string]interface{} `xml:"broke"`
	}

	os.Setenv("TEST_BROKE_0", "f00")
	os.Setenv("TEST_BROKE_broke", "foo")

	a := assert.New(t)
	c := &testBroken{}
	ok, err := UnmarshalENV(c, "TEST")

	a.NotNil(err, "an error must be returned for an unsupported type")
	a.False(ok)

	c2 := &testBroken2{}
	ok, err = UnmarshalENV(c2, "TEST")

	a.NotNil(err, "an error must be returned for an unsupported map type")
	a.False(ok)

	c3 := &testBroken3{}
	ok, err = UnmarshalENV(c3, "TEST")

	a.NotNil(err, "an error must be returned for an unsupported map type")
	a.False(ok)
}

func TestUnmarshalENVerrors(t *testing.T) {
	a := assert.New(t)

	type tester struct {
		unexpd map[string]string
		Works  map[string]string `xml:"works"`
		Rad    map[string][]int  `xml:"yup"`
		Error  error             `xml:"error"`
	}

	os.Setenv("YO_WORKS_foostring", "fooval")
	os.Setenv("YO_WORKS_foo2string", "foo2val")
	os.Setenv("YO_YUP_server99_0", "128")
	os.Setenv("YO_YUP_server99_1", "129")
	os.Setenv("YO_YUP_server99_2", "130")
	os.Setenv("YO_YUP_server100_0", "256")
	os.Setenv("YO_ERROR", "this is an error")

	c := tester{}
	ok, err := UnmarshalENV(&c, "YO")

	a.Nil(err, "maps are supported and must not produce an error")
	a.True(ok)
	a.Nil(c.unexpd)
	a.Equal("fooval", c.Works["foostring"])
	a.Equal("foo2val", c.Works["foo2string"])
	a.Equal([]int{128, 129, 130}, c.Rad["server99"])
	a.Equal([]int{256}, c.Rad["server100"])
	a.Equal(fmt.Errorf("this is an error"), c.Error)

	type tester2 struct {
		NotBroken  []map[string]string  `xml:"broken"`
		NotBroken2 []*map[string]string `xml:"broken2"`
		NotBroken3 []map[int]int        `xml:"broken3"`
		HasStuff   []map[string]string  `xml:"stuff"`
	}

	os.Setenv("MORE_BROKEN", "value")
	os.Setenv("MORE_BROKEN_0_freesauce", "at-charlies")
	os.Setenv("MORE_BROKEN2_0_freesoup", "at-daves")
	os.Setenv("MORE_STUFF_0_freesoda", "not-at-pops")
	os.Setenv("MORE_STUFF_0_freetime", "at-pops")
	os.Setenv("MORE_STUFF_0_a", "")

	c2 := tester2{HasStuff: []map[string]string{{"freesoda": "at-pops"}, {"a": "v"}}}
	ok, err = UnmarshalENV(&c2, "MORE")

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

func TestUnmarshalENV(t *testing.T) {
	// do not run this in parallel with other tests that change environment variables
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}
	ok, err := UnmarshalENV(c, "PRE")

	a.Nil(err, "there must not be an error when parsing no variables")
	a.False(ok, "there are no environment variables set, so ok should be false")
	testThingENV(a)
	testOscureENV(a)
	testSpecialENV(a)

	f := true
	g := &f
	_, err = UnmarshalENV(g, "OOO")
	a.NotNil(err, "unmarshaling a non-struct pointer must produce an error")
}

func testThingENV(a *assert.Assertions) {
	os.Clearenv()
	os.Setenv("PRE_PSLICE_0_BOOL", "true")
	os.Setenv("PRE_PSLICE_0_FLOAT", "123.4567")

	os.Setenv("PRE_SSLICE_0_STRING", "foo")
	os.Setenv("PRE_SSLICE_0_INT", "123")

	os.Setenv("PRE_STRUCT_BOOL", "false")
	os.Setenv("PRE_PSTRUCT_STRING", "foo2")

	c := &testStruct{}

	ok, err := UnmarshalENV(c, "PRE")
	a.True(ok, "ok must be true since things must be parsed")
	testUnmarshalFileValues(a, c, err, "testThingENV")
	// do it again, and we should get the same result
	ok, err = UnmarshalENV(c, "PRE")
	a.True(ok, "ok must be true since things must be parsed")
	testUnmarshalFileValues(a, c, err, "testThingENV")
}

func testOscureENV(a *assert.Assertions) {
	type testObscure struct {
		FloatSlice []float32        `xml:"is"`
		UintSliceP []*uint16        `xml:"uis"`
		Weirdo     *[]int           `xml:"psi"`
		Wut        *[]testSubConfig `xml:"wut"`
	}

	os.Clearenv()
	os.Setenv("OB_IS_0", "-5")
	os.Setenv("OB_IS_1", "8")

	os.Setenv("OB_UIS_0", "12")
	os.Setenv("OB_UIS_1", "22")

	os.Setenv("OB_PSI_0", "-1")
	os.Setenv("OB_WUT_0_BOOL", "true")

	c := &testObscure{}
	testit := func() {
		ok, err := UnmarshalENV(c, "OB")
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

func testSpecialENV(a *assert.Assertions) {
	type testSpecial struct {
		Dur  time.Duration    `xml:"dur"`
		CDur Duration         `xml:"cdur"`
		Time time.Time        `xml:"time"`
		Durs *[]time.Duration `xml:"durs"`
		Sub  *struct {
			URL url.URL `xml:"url"`
			IP  net.IP  `xml:"ip"`
		} `xml:"sub"`
	}

	os.Clearenv()
	os.Setenv("TEST_DUR", "1m")
	os.Setenv("TEST_CDUR", "1s")
	os.Setenv("TEST_TIME", "2019-12-18T00:35:49+08:00")
	os.Setenv("TEST_SUB_URL", "https://golift.io/cnfg?rad=true")
	os.Setenv("TEST_SUB_IP", "123.45.67.89")

	c := &testSpecial{}
	ok, err := (&ENV{Pfx: "TEST"}).Unmarshal(c)

	a.True(ok, "ok must be true since things must be parsed")
	a.Nil(err)
	a.Equal(time.Minute, c.Dur)
	a.Equal(time.Second, c.CDur.Duration)
	a.Equal("golift.io", c.Sub.URL.Host, "the url wasn't parsed properly")
	a.Equal("123.45.67.89", c.Sub.IP.String(), "the IP wasn't parsed properly")
	a.Nil(c.Durs)

	os.Setenv("TEST_TIME", "not a real time")

	c = &testSpecial{}
	ok, err = (&ENV{Pfx: "TEST"}).Unmarshal(c)

	a.False(ok, "cannot parse an invalid time")
	a.NotNil(err, "cannot parse an invalid time")
}
