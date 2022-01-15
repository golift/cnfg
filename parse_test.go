package cnfg //nolint:testpackage

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInt(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	for _, t := range []string{typeINT, typeINT8, typeINT16, typeINT32, typeINT64} {
		i, err := parseInt(t, "1")

		assert.Equal(int64(1), i)
		assert.Nil(err)
	}
}

func TestParseByteSlice(t *testing.T) { //nolint:paralleltest
	assert := assert.New(t)

	type test struct {
		F []byte `xml:"bytes,delenv"` //nolint:staticcheck
	}

	t.Setenv("D_BYTES", "byte slice incoming")

	test1val := &test{}
	ok, err := UnmarshalENV(test1val, "D")

	assert.True(ok)
	assert.Nil(err)
	assert.Equal("byte slice incoming", string(test1val.F))
}

func TestParseUint(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type test struct {
		F uint64
	}

	config := &test{}
	field := reflect.ValueOf(config).Elem().Field(0)

	for _, t := range []string{typeUINT, typeUINT16, typeUINT32, typeUINT64} {
		err := parseUint(field, t, "1")

		assert.Equal(uint64(1), config.F)
		assert.Nil(err)
	}

	type test2 struct {
		F byte
	}

	test2val := &test2{}
	field = reflect.ValueOf(test2val).Elem().Field(0)

	err := parseUint(field, typeUINT8, "11")
	assert.NotNil(err, "must return an error when more than one byte is provided")

	err = parseUint(field, typeUINT8, "f")
	assert.Nil(err, "must not return an error when only one byte is provided")
	assert.Equal(byte('f'), test2val.F)

	err = parseUint(field, typeUINT8, "")
	assert.Nil(err, "must not return an error when only no bytes are provided")
	assert.Equal(uint8(0), test2val.F)
}

/*
// make sure we don't panic when trying to interface something we can't.
func TestParseInterfaceError(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	type F uint64

	ok, err := (&Parser{}).Interface(reflect.ValueOf(F(0)), "", "")

	assert.Nil(err, "unaddressable value must return nil")
	assert.False(ok, "unaddressable value must return false")
}
*/
