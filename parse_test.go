package cnfg_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"golift.io/cnfg"
)

func TestParseInt(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	for _, t := range []string{cnfg.TypeINT, cnfg.TypeINT8, cnfg.TypeINT16, cnfg.TypeINT32, cnfg.TypeINT64} {
		i, err := cnfg.ParseInt(t, "1")

		a.Equal(int64(1), i)
		a.Nil(err)
	}
}

func TestParseByteSlice(t *testing.T) {
	a := assert.New(t)

	type test struct {
		F []byte `xml:"bytes,delenv"`
	}

	os.Setenv("D_BYTES", "byte slice incoming")

	f := &test{}
	ok, err := cnfg.UnmarshalENV(f, "D")

	a.True(ok)
	a.Nil(err)
	a.Equal("byte slice incoming", string(f.F))
}

func TestParseUint(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	type test struct {
		F uint64
	}

	f := &test{}
	g := reflect.ValueOf(f).Elem().Field(0)

	for _, t := range []string{cnfg.TypeUINT, cnfg.TypeUINT16, cnfg.TypeUINT32, cnfg.TypeUINT64} {
		err := cnfg.ParseUint(g, t, "1")

		a.Equal(uint64(1), f.F)
		a.Nil(err)
	}

	type test2 struct {
		F byte
	}

	f2 := &test2{}
	g = reflect.ValueOf(f2).Elem().Field(0)

	err := cnfg.ParseUint(g, cnfg.TypeUINT8, "11")
	a.NotNil(err, "must return an error when more than one byte is provided")

	err = cnfg.ParseUint(g, cnfg.TypeUINT8, "f")
	a.Nil(err, "must not return an error when only one byte is provided")
	a.Equal(byte('f'), f2.F)

	err = cnfg.ParseUint(g, cnfg.TypeUINT8, "")
	a.Nil(err, "must not return an error when only no bytes are provided")
	a.Equal(uint8(0), f2.F)
}

/*
// make sure we don't panic when trying to interface something we can't.
func TestParseInterfaceError(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	type F uint64

	ok, err := (&cnfg.Parser{}).Interface(reflect.ValueOf(F(0)), "", "")

	a.Nil(err, "unaddressable value must return nil")
	a.False(ok, "unaddressable value must return false")
}
*/
