package cnfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalMap(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	pairs := map[string]string{
		"FOO": "bar",
		"BAZ": "yup",
	}

	type mapTester struct {
		Foo string `xml:"foo"`
		Baz string `xml:"baz"`
	}

	i := mapTester{}
	ok, err := UnmarshalMap(pairs, &i)

	a.Nil(err)
	a.True(ok)
	a.EqualValues("bar", i.Foo)

	ok, err = UnmarshalMap(pairs, i)

	a.False(ok)
	a.NotNil(err, "must have an error when attempting unmarshal to non-pointer")

	ok, err = (&ENV{}).UnmarshalMap(pairs, &i)
	a.True(ok)
	a.Nil(err)
}
