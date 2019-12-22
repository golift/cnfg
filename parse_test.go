package cnfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInt(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	for _, t := range []string{typeUINT, typeUINT8, typeUINT16, typeUINT32, typeUINT64} {
		i, err := parseUint(t, "1")

		a.Equal(uint64(1), i)
		a.Nil(err)
	}

	for _, t := range []string{typeINT, typeINT8, typeINT16, typeINT32, typeINT64} {
		i, err := parseInt(t, "1")

		a.Equal(int64(1), i)
		a.Nil(err)
	}
}
