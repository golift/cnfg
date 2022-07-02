package cnfg //nolint:testpackage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInt(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	for _, t := range []interface{}{int(0), int8(8), int16(16), int32(32), int64(64)} {
		i, err := parseInt(t, fmt.Sprintf("%d", t))

		assert.EqualValues(t, i)
		assert.Nil(err)
	}
}

func TestParseByteSlice(t *testing.T) { //nolint:paralleltest
	assert := assert.New(t)

	type test struct {
		F []byte `xml:"bytes,delenv"` //nolint:staticcheck
	}

	t.Setenv("D_BYTES", "byte slice incoming")

	testStruct := &test{}
	ok, err := UnmarshalENV(testStruct, "D")

	assert.True(ok)
	assert.Nil(err)
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

	for _, t := range []interface{}{uint(0), uint16(16), uint32(32), uint64(64)} {
		err := parseUint(theField, t, "1")

		assert.EqualValues(1, embeddedInt.F)
		assert.Nil(err)
	}

	type test2 struct {
		F byte
	}

	testStruct := &test2{}
	theField = reflect.ValueOf(testStruct).Elem().Field(0)

	err := parseUint(theField, uint8(0), "11")
	assert.NotNil(err, "must return an error when more than one byte is provided")

	err = parseUint(theField, uint8(0), "f")
	assert.Nil(err, "must not return an error when only one byte is provided")
	assert.Equal(byte('f'), testStruct.F)

	err = parseUint(theField, uint8(0), "")
	assert.Nil(err, "must not return an error when only no bytes are provided")
	assert.Equal(uint8(0), testStruct.F)
}

// make sure we don't panic when trying to interface something we can't.
func TestParseInterfaceError(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type F uint64

	ok, err := (&parser{}).Interface(reflect.ValueOf(F(0)), "", "", false)
	assert.False(ok, "unaddressable value must return false")
	assert.Nil(err, "unaddressable value must return nil")
}
