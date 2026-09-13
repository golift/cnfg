package cnfg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeelMapKey(t *testing.T) {
	t.Parallel()

	type folder struct {
		Path         string   `xml:"path"`
		ExtractPath  string   `xml:"extract_path"`
		DeleteAfter  string   `xml:"delete_after"`
		ExcludePaths []string `xml:"exclude_path"`
		Paths        []string `xml:"paths"`
	}

	type nested struct {
		Name string `xml:"name"`
		Dogs []struct {
			Name string `xml:"name"`
		} `xml:"dogs"`
	}

	type dog struct {
		Name string `xml:"name"`
	}

	folderType := reflect.TypeFor[folder]()
	nestedType := reflect.TypeFor[nested]()
	dogsType := reflect.TypeFor[[]dog]()
	intsType := reflect.TypeFor[[]int]()
	mapsType := reflect.TypeFor[map[string]string]()
	stringType := reflect.TypeFor[string]()
	ptrFolder := reflect.TypeFor[*folder]()

	tests := []struct {
		name      string
		remainder string
		typ       reflect.Type
		key       string
		field     string
		ok        bool
		low       bool
	}{
		{
			name: "extract_path wins over path", remainder: "watch_EXTRACT_PATH",
			typ: folderType, key: "watch", field: "EXTRACT_PATH", ok: true,
		},
		{
			name: "path", remainder: "watch_PATH",
			typ: folderType, key: "watch", field: "PATH", ok: true,
		},
		{
			name: "underscore key", remainder: "starrs_stripes_PATH",
			typ: folderType, key: "starrs_stripes", field: "PATH", ok: true,
		},
		{
			name: "index key and slice index", remainder: "0_PATHS_0",
			typ: folderType, key: "0", field: "PATHS", ok: true,
		},
		{
			name: "exclude_path slice", remainder: "watch_EXCLUDE_PATH_0",
			typ: folderType, key: "watch", field: "EXCLUDE_PATH", ok: true,
		},
		{
			name: "nested slice field", remainder: "a_DOGS_0_NAME",
			typ: nestedType, key: "a", field: "DOGS", ok: true,
		},
		{
			name: "slice of struct walks index", remainder: "key_0_NAME",
			typ: dogsType, key: "key", field: "NAME", ok: true,
		},
		{
			name: "slice of scalar index", remainder: "key_0",
			typ: intsType, key: "key", ok: true,
		},
		{
			name: "unmatched slice leftover", remainder: "key_nope",
			typ: dogsType,
		},
		{
			name: "nested map first token", remainder: "outer_inner_key",
			typ: mapsType, key: "outer", ok: true,
		},
		{
			name: "nested map empty first token", remainder: "_x",
			typ: mapsType,
		},
		{
			name: "scalar whole remainder", remainder: "db_primary",
			typ: stringType, key: "db_primary", ok: true,
		},
		{
			name: "pointer deref", remainder: "watch_PATH",
			typ: ptrFolder, key: "watch", field: "PATH", ok: true,
		},
		{
			name: "low mode keeps tag case", remainder: "watch_extract_path",
			typ: folderType, key: "watch", field: "extract_path", ok: true, low: true,
		},
		{
			name: "slice remainder is only an index", remainder: "0",
			typ: intsType,
		},
		{name: "empty", remainder: "", typ: folderType},
		{
			name: "unmatched leftover is the key", remainder: "watch_NOTAFIELD",
			typ: folderType, key: "watch_NOTAFIELD", ok: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			key, field, ok := peelMapKey(test.remainder, test.typ, ENVTag, test.low)
			assert.Equal(t, test.ok, ok)
			assert.Equal(t, test.key, key)
			assert.Equal(t, test.field, field)
		})
	}
}
