package cnfg

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInt(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	for _, val := range []interface{}{int(0), int8(8), int16(16), int32(32), int64(64)} {
		i, err := parseInt(val, fmt.Sprintf("%d", val))

		require.NoError(t, err)
		assert.EqualValues(val, i)
	}
}

func TestParseByteSlice(t *testing.T) { //nolint:paralleltest
	assert := assert.New(t)

	type test struct {
		F []byte `xml:"bytes,delenv"`
	}

	t.Setenv("D_BYTES", "byte slice incoming")

	testStruct := &test{}
	ok, err := UnmarshalENV(testStruct, "D")

	assert.True(ok)
	require.NoError(t, err)
	assert.Equal("byte slice incoming", string(testStruct.F))
}

func TestParseUint(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type test struct {
		F uint64
	}

	embeddedInt := &test{}
	theField := reflect.ValueOf(embeddedInt).Elem().Field(0)

	for _, val := range []interface{}{uint(0), uint16(16), uint32(32), uint64(64)} {
		err := parseUint(theField, val, "1")
		require.NoError(t, err)
		assert.EqualValues(1, embeddedInt.F)
	}

	type test2 struct {
		F byte
	}

	testStruct := &test2{}
	theField = reflect.ValueOf(testStruct).Elem().Field(0)

	err := parseUint(theField, uint8(0), "11")
	require.Error(t, err, "must return an error when more than one byte is provided")

	err = parseUint(theField, uint8(0), "f")
	require.NoError(t, err, "must not return an error when only one byte is provided")
	assert.Equal(byte('f'), testStruct.F)

	err = parseUint(theField, uint8(0), "")
	require.NoError(t, err, "must not return an error when only no bytes are provided")
	assert.Equal(uint8(0), testStruct.F)
}

// make sure we don't panic when trying to interface something we can't.
func TestParseInterfaceError(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type F uint64

	ok, err := (&parser{}).Interface(reflect.ValueOf(F(0)), "", "", false)
	require.NoError(t, err, "unaddressable value must return nil")
	assert.False(ok, "unaddressable value must return false")
}
