package cnfg_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golift.io/cnfg"
)

// TimeX uses two environment variables to multiply a duration.
type TimeX struct {
	time.Duration
}

// This is a test to make sure our struct satisfies the interface.
var _ cnfg.ENVUnmarshaler = (*TimeX)(nil)

type AppConfig struct {
	Name    string `xml:"name"`
	Special TimeX  `xml:"in"`
}

func (t *TimeX) UnmarshalENV(tag, val string) error {
	xTag := tag + "_X"

	xString, ok := os.LookupEnv(xTag)
	if !ok {
		xString = "1"
	}

	multiplier, err := strconv.Atoi(xString)
	if err != nil {
		return fmt.Errorf("multiplier invalid %s: %w", xTag, err)
	}

	t.Duration, err = time.ParseDuration(val)
	if err != nil {
		return fmt.Errorf("duration invalid %s: %w", tag, err)
	}

	t.Duration *= time.Duration(multiplier)

	return nil
}

// This simple example shows how you may use the ENVUnmarshaler interface.
// This shows how to use two environment variables to set one custom value.
func ExampleENVUnmarshaler() {
	config := &AppConfig{}

	_ = os.Setenv("APP_IN", "5m")
	_ = os.Setenv("APP_IN_X", "10")
	_ = os.Setenv("APP_NAME", "myApp")

	_, err := cnfg.UnmarshalENV(config, "APP")
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s starts in %v", config.Name, config.Special)
	// Output: myApp starts in 50m0s
}

func TestUnmarshalText(t *testing.T) {
	t.Parallel()

	d := cnfg.Duration{Duration: time.Minute + time.Second}
	b, err := d.MarshalText()

	require.NoError(t, err, "this method must not return an error")
	assert.Equal(t, []byte("1m1s"), b)
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		D cnfg.Duration `json:"d"`
	}

	d := testStruct{D: cnfg.Duration{Duration: time.Hour + time.Minute}}
	b, err := json.Marshal(d)

	require.NoError(t, err, "this method must not return an error")
	assert.JSONEq(t, `{"d":"1h1m"}`, string(b))
}

func TestString(t *testing.T) {
	t.Parallel()

	testDur := cnfg.Duration{Duration: time.Hour + time.Minute}
	assert.Equal(t, "1h1m", testDur.String())
	assert.Equal(t, "1h1m", fmt.Sprint(testDur))

	testDur = cnfg.Duration{Duration: time.Hour}
	assert.Equal(t, "1h", testDur.String())
	assert.Equal(t, "1h", fmt.Sprint(testDur))
}
