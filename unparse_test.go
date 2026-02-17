package cnfg_test

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golift.io/cnfg"
)

const base10 = 10

type MarshalTest struct {
	Name  string            `xml:"name,omitempty"`
	Pass  string            `xml:"pass,omitempty"`
	IP    net.IP            `xml:"ip,omitempty"`
	Smap  map[string]string `xml:"smap,omitempty"`
	Imap  map[string]any    `xml:"imap,omitempty"`
	List  []string          `xml:"list,omitempty"`
	Byte  []byte            `xml:"byte,omitempty"`
	Dur   time.Duration     `xml:"dur,omitempty"`
	Time  time.Time         `xml:"time,omitempty"`
	Err   error             `xml:"err,omitempty"`
	Bool  bool              `xml:"bool,omitempty"`
	Uint  uint8             `xml:"uint,omitempty"`
	Un16  uint16            `xml:"un16,omitempty"`
	Un32  uint32            `xml:"un32,omitempty"`
	Un64  uint64            `xml:"un64,omitempty"`
	Int   int               `xml:"int,omitempty"`
	In8   int8              `xml:"in8,omitempty"`
	In16  int16             `xml:"in16,omitempty"`
	In32  int32             `xml:"in32,omitempty"`
	In64  int64             `xml:"in64,omitempty"`
	Fl32  float32           `xml:"fl32,omitempty"`
	Fl64  float64           `xml:"fl64,omitempty"`
	Test2 marshalTest2      `xml:"test2"`
	Test  *MarshalTest      `xml:"test"`
}

type marshalTest2 struct {
	*MarshalTest // anonymous struct memebrs do not have their names exposed.

	// not setting Name2 creates empty variables (no omitempty)
	Name2  string `xml:""` // non-anonymous struct members will use their name if no struct tag name.
	Ignore string `xml:"-"`
}

func marshalTestData() (*MarshalTest, int) {
	return &MarshalTest{
		Name:  "my name is golift",                                           // 1
		Pass:  "supersecret",                                                 // 2
		IP:    net.ParseIP("127.0.0.1"),                                      // 3
		Smap:  map[string]string{"blue": "sky", "red": "lava"},               // 4, 5
		List:  []string{"humble", "beginnings"},                              // 6, 7
		Byte:  []byte("some bytes for dinner"),                               // 8
		Uint:  1,                                                             // 9
		Un16:  16,                                                            // 10
		Un32:  32,                                                            // 11
		Un64:  64,                                                            // 12
		Time:  time.Now(),                                                    // 13
		Dur:   15 * time.Minute,                                              // 14
		Int:   1,                                                             // 15
		In8:   8,                                                             // 16
		In16:  16,                                                            // 17
		In32:  32,                                                            // 18
		In64:  64,                                                            // 19
		Bool:  true,                                                          // 20
		Fl32:  32.32,                                                         // 21
		Fl64:  64.64,                                                         // 22
		Test2: marshalTest2{MarshalTest: &MarshalTest{Name: "supersubname"}}, // 23
		Test: &MarshalTest{
			Name: "subtest",                                                  // 24
			Err:  errors.New("this error is here to line up the comments->"), // 25
		}, Imap: map[string]any{
			"orange":  "sunset",  // 26
			"pink":    "sunrise", // 27
			"counter": 8967,      // 28
			"floater": 3.1415926, // 29
		}, // + 3 more from marshalTest2.Name2. That puts the total var count at 32.
	}, 32 // set the count here.
}

func TestDeconStruct(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	data, count := marshalTestData()
	pairs, err := cnfg.MarshalENV(data, "PFX")

	require.NoError(t, err)
	assert.Equal(data.Name, pairs["PFX_NAME"])
	assert.Equal(data.Pass, pairs["PFX_PASS"])
	assert.Equal(data.IP.String(), pairs["PFX_IP"])
	assert.Equal(data.Smap["blue"], pairs["PFX_SMAP_blue"])
	assert.Equal(data.Smap["red"], pairs["PFX_SMAP_red"])
	assert.Equal(data.Imap["orange"], pairs["PFX_IMAP_orange"])
	assert.Equal(data.Imap["pink"], pairs["PFX_IMAP_pink"])
	assert.Equal(fmt.Sprint(data.Imap["counter"]), pairs["PFX_IMAP_counter"])
	assert.Equal(fmt.Sprint(data.Imap["floater"]), pairs["PFX_IMAP_floater"])
	assert.Equal(data.List[0], pairs["PFX_LIST_0"])
	assert.Equal(data.List[1], pairs["PFX_LIST_1"])
	assert.Equal(string(data.Byte), pairs["PFX_BYTE"])
	assert.Equal(strconv.FormatUint(uint64(data.Uint), base10), pairs["PFX_UINT"])
	assert.Equal(strconv.FormatUint(uint64(data.Un16), base10), pairs["PFX_UN16"])
	assert.Equal(strconv.FormatUint(uint64(data.Un32), base10), pairs["PFX_UN32"])
	assert.Equal(strconv.FormatUint(data.Un64, base10), pairs["PFX_UN64"])
	assert.Equal(data.Dur.String(), pairs["PFX_DUR"])
	assert.Equal(strconv.FormatInt(int64(data.Int), base10), pairs["PFX_INT"])
	assert.Equal(strconv.FormatInt(int64(data.In8), base10), pairs["PFX_IN8"])
	assert.Equal(strconv.FormatInt(int64(data.In16), base10), pairs["PFX_IN16"])
	assert.Equal(strconv.FormatInt(int64(data.In32), base10), pairs["PFX_IN32"])
	assert.Equal(strconv.FormatInt(data.In64, base10), pairs["PFX_IN64"])
	assert.Equal("true", pairs["PFX_BOOL"])
	assert.Equal(data.Test2.MarshalTest.Name, pairs["PFX_TEST2_NAME"])
	assert.Equal(data.Test2.Name, pairs["PFX_TEST2_NAME"])
	assert.Equal("32.32", pairs["PFX_FL32"])
	assert.Equal("64.64", pairs["PFX_FL64"])
	assert.Equal(data.Test.Name, pairs["PFX_TEST_NAME"])
	assert.Equal(data.Test.Err.Error(), pairs["PFX_TEST_ERR"])
	assert.Len(pairs, count,
		"%d variables are created in marshalTestData, update as more tests are added.", count)

	for _, v := range pairs.Quoted() {
		// fmt.Println(v)
		p := strings.Split(v, "=")
		assert.Equal(`"`+pairs[p[0]]+`"`, p[1], "returned Slice() value is wrong")
	}
}
