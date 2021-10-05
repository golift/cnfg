package cnfg_test

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golift.io/cnfg"
)

type MarshalTest struct {
	Name  string            `xml:"name,omitempty"`
	Pass  string            `xml:"pass,omitempty"`
	IP    net.IP            `xml:"ip,omitempty"`
	Smap  map[string]string `xml:"smap,omitempty"`
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
	// not setting Name2 creates empty variables (no omitempty)
	Name2        string `xml:""` // non-anonymous struct members will use their name if no struct tag name.
	Ignore       string `xml:"-"`
	*MarshalTest        // anonymous struct memebrs do not have their names exposed.
}

func marshalTestData() (*MarshalTest, int) {
	//nolint:goerr113
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
			Name: "subtest",                                                    // 24
			Err:  fmt.Errorf("this long error is here 2 line up the comments"), // 25
		}, // + 3 more from marshalTest2.Name2. That puts the total var count at 28
	}, 28 // set the count here.
}

func TestDeconStruct(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	data, count := marshalTestData()
	pairs, err := cnfg.MarshalENV(data, "PFX")

	a.Nil(err)
	a.Equal(data.Name, pairs["PFX_NAME"])
	a.Equal(data.Pass, pairs["PFX_PASS"])
	a.Equal(data.IP.String(), pairs["PFX_IP"])
	a.Equal(data.Smap["blue"], pairs["PFX_SMAP_blue"])
	a.Equal(data.Smap["red"], pairs["PFX_SMAP_red"])
	a.Equal(data.List[0], pairs["PFX_LIST_0"])
	a.Equal(data.List[1], pairs["PFX_LIST_1"])
	a.Equal(string(data.Byte), pairs["PFX_BYTE"])
	a.Equal(fmt.Sprintf("%d", data.Uint), pairs["PFX_UINT"])
	a.Equal(fmt.Sprintf("%d", data.Un16), pairs["PFX_UN16"])
	a.Equal(fmt.Sprintf("%d", data.Un32), pairs["PFX_UN32"])
	a.Equal(fmt.Sprintf("%d", data.Un64), pairs["PFX_UN64"])
	a.Equal(data.Dur.String(), pairs["PFX_DUR"])
	a.Equal(fmt.Sprintf("%d", data.Int), pairs["PFX_INT"])
	a.Equal(fmt.Sprintf("%d", data.In8), pairs["PFX_IN8"])
	a.Equal(fmt.Sprintf("%d", data.In16), pairs["PFX_IN16"])
	a.Equal(fmt.Sprintf("%d", data.In32), pairs["PFX_IN32"])
	a.Equal(fmt.Sprintf("%d", data.In64), pairs["PFX_IN64"])
	a.Equal("true", pairs["PFX_BOOL"])
	a.Equal(data.Test2.MarshalTest.Name, pairs["PFX_TEST2_NAME"])
	a.Equal(data.Test2.Name, pairs["PFX_TEST2_NAME"])
	a.Equal("32.32", pairs["PFX_FL32"])
	a.Equal("64.64", pairs["PFX_FL64"])
	a.Equal(data.Test.Name, pairs["PFX_TEST_NAME"])
	a.Equal(data.Test.Err.Error(), pairs["PFX_TEST_ERR"])
	a.Equal(count, len(pairs),
		fmt.Sprintf("%d variables are created in marshalTestData, update as more tests are added.", count))

	for _, v := range pairs.Quoted() {
		// fmt.Println(v)
		p := strings.Split(v, "=")
		a.Equal(`"`+pairs[p[0]]+`"`, p[1], "returned Slice() value is wrong")
	}
}
