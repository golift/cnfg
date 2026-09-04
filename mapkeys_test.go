package cnfg_test

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golift.io/cnfg"
)

const (
	testPermAPI  = "api"
	testPermGUI  = "gui"
	testStats    = "stats"
	testRoleRead = "read_only"
)

func TestPairsGetRequiresChildPrefix(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	pairs := cnfg.Pairs{
		"UN_WEBSERVER_ROLES":                     "not-a-key",
		"UN_WEBSERVER_ROLES_stats_PERMISSIONS_0": testPermAPI,
	}

	got := pairs.Get("UN_WEBSERVER_ROLES")

	assert.NotContains(got, "UN", "the exact prefix must not become a map key")
	assert.NotContains(got, "UN_WEBSERVER_ROLES")
	assert.Equal(testPermAPI, got[testStats])
}

func TestUnmarshalMapScalarKeyWithUnderscore(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type cfg struct {
		Hosts map[string]string `xml:"hosts"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"HOSTS_db_primary": "sql.home.lan",
	}, config)

	require.NoError(t, err)
	assert.True(ok)
	assert.Equal("sql.home.lan", config.Hosts["db_primary"])
	assert.NotContains(config.Hosts, "db")
}

func TestUnmarshalENVSliceKeyWithUnderscore(t *testing.T) {
	assert := assert.New(t)

	t.Setenv("YO_YUP_server_99_0", "128")
	t.Setenv("YO_YUP_server_99_1", "129")
	t.Setenv("YO_YUP_plain_0", "7")

	type cfg struct {
		Rad map[string][]int `xml:"yup"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalENV(config, "YO")

	require.NoError(t, err)
	assert.True(ok)
	assert.Equal([]int{128, 129}, config.Rad["server_99"])
	assert.Equal([]int{7}, config.Rad["plain"])
	assert.NotContains(config.Rad, "server")
}

func TestUnmarshalENVStructMapRoles(t *testing.T) {
	assert := assert.New(t)

	t.Setenv("UN_WEBSERVER_ROLES", "must-not-become-a-key")
	t.Setenv("UN_WEBSERVER_ROLES_stats_PERMISSIONS_0", testPermAPI)
	t.Setenv("UN_WEBSERVER_ROLES_stats_PERMISSIONS_1", testPermGUI)
	t.Setenv("UN_WEBSERVER_ROLES_read_only_PERMISSIONS_0", testStats)

	type role struct {
		Permissions []string `xml:"permissions"`
	}

	type cfg struct {
		Roles map[string]role `xml:"roles"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalENV(config, "UN_WEBSERVER")

	require.NoError(t, err)
	assert.True(ok)
	require.Contains(t, config.Roles, testStats)
	require.Contains(t, config.Roles, testRoleRead)
	assert.Equal([]string{testPermAPI, testPermGUI}, config.Roles[testStats].Permissions)
	assert.Equal([]string{testStats}, config.Roles[testRoleRead].Permissions)
	assert.NotContains(config.Roles, "UN")
	assert.NotContains(config.Roles, "read")
}

func TestUnmarshalENVIgnoresUnmatchedStructLeftover(t *testing.T) {
	assert := assert.New(t)

	t.Setenv("APP_ROLES_stats_NOTAFIELD_0", "nope")

	type role struct {
		Permissions []string `xml:"permissions"`
	}

	type cfg struct {
		Roles map[string]role `xml:"roles"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalENV(config, "APP")

	require.NoError(t, err)
	assert.False(ok)
	assert.Empty(config.Roles)
}

func TestUnmarshalENVEmptyMapValueClears(t *testing.T) {
	assert := assert.New(t)

	t.Setenv("APP_HOSTS_db_primary", "")

	type cfg struct {
		Hosts map[string]string `xml:"hosts"`
	}

	config := &cfg{Hosts: map[string]string{"db_primary": "old"}}
	ok, err := cnfg.UnmarshalENV(config, "APP")

	require.NoError(t, err)
	assert.True(ok)
	assert.Empty(config.Hosts["db_primary"])
}

func TestUnmarshalENVDelenvUnsetsFullNames(t *testing.T) {
	assert := assert.New(t)

	t.Setenv("foo_bar", "sentinel")
	t.Setenv("YO_WORKS", "bare-prefix")
	t.Setenv("YO_WORKS_foo_bar", "fooval")
	t.Setenv("YO_WORKS_plain", "plainval")

	type cfg struct {
		Works map[string]string `xml:"works,delenv"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalENV(config, "YO")

	require.NoError(t, err)
	assert.True(ok)
	assert.Equal("fooval", config.Works["foo_bar"])
	assert.Equal("plainval", config.Works["plain"])
	assert.Empty(os.Getenv("YO_WORKS_foo_bar"))
	assert.Empty(os.Getenv("YO_WORKS_plain"))
	assert.Empty(os.Getenv("YO_WORKS"), "delenv must also unset the bare map prefix")
	assert.Equal("sentinel", os.Getenv("foo_bar"), "delenv must not unset the bare map key")
}

func TestMarshalENVRoundTripStructMap(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type role struct {
		Permissions []string `xml:"permissions"`
	}

	type cfg struct {
		Roles map[string]role `xml:"roles"`
	}

	src := &cfg{Roles: map[string]role{
		testStats:    {Permissions: []string{testPermAPI, testPermGUI}},
		testRoleRead: {Permissions: []string{testStats}},
	}}

	pairs, err := cnfg.MarshalENV(src, "UN_WEBSERVER")
	require.NoError(t, err)
	assert.Equal(testPermAPI, pairs["UN_WEBSERVER_ROLES_stats_PERMISSIONS_0"])
	assert.Equal(testPermGUI, pairs["UN_WEBSERVER_ROLES_stats_PERMISSIONS_1"])
	assert.Equal(testStats, pairs["UN_WEBSERVER_ROLES_read_only_PERMISSIONS_0"])

	dst := &cfg{}
	ok, err := (&cnfg.ENV{Pfx: "UN_WEBSERVER"}).UnmarshalMap(pairs, dst)
	require.NoError(t, err)
	assert.True(ok)
	assert.Equal(src.Roles, dst.Roles)
}

func TestUnmarshalMapTimeKeyWithUnderscore(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type cfg struct {
		When map[string]time.Time `xml:"when"`
	}

	stamp := time.Date(2019, 12, 18, 0, 35, 49, 0, time.FixedZone("", 8*3600))
	config := &cfg{}
	ok, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"WHEN_shift_start": "2019-12-18T00:35:49+08:00",
	}, config)

	require.NoError(t, err)
	assert.True(ok)
	require.Contains(t, config.When, "shift_start")
	assert.True(stamp.Equal(config.When["shift_start"]))
}

func TestUnmarshalMapNestedMapKeepsFirstToken(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type cfg struct {
		Nested map[string]map[string]string `xml:"nested"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"NESTED_outer_inner_key": "val",
	}, config)

	require.NoError(t, err)
	assert.True(ok)
	require.Contains(t, config.Nested, "outer")
	assert.Equal("val", config.Nested["outer"]["inner_key"])
	assert.NotContains(config.Nested, "outer_inner")
}

func TestUnmarshalMapDoesNotInventNestedStructKeys(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type dog struct {
		Name string `xml:"name"`
	}

	type shelter struct {
		Name string `xml:"name"`
		Dogs []dog  `xml:"dogs"`
	}

	type cfg struct {
		S map[string]shelter `xml:"s"`
	}

	config := &cfg{}
	worked, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"S_a_NAME":        "happy",
		"S_a_DOGS_0_NAME": "rex",
	}, config)

	require.NoError(t, err)
	assert.True(worked)
	require.Contains(t, config.S, "a")
	assert.Equal("happy", config.S["a"].Name)
	assert.Len(config.S["a"].Dogs, 1)
	assert.Equal("rex", config.S["a"].Dogs[0].Name)
	assert.NotContains(config.S, "a_DOGS_0")
}

func TestUnmarshalMapStructFieldTagWithUnderscore(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type role struct {
		ReadOnly bool `xml:"read_only"`
	}

	type cfg struct {
		Roles map[string]role `xml:"roles"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"ROLES_admin_READ_ONLY": "true",
	}, config)

	require.NoError(t, err)
	assert.True(ok)
	require.Contains(t, config.Roles, "admin")
	assert.True(config.Roles["admin"].ReadOnly)
	assert.NotContains(config.Roles, "admin_READ_ONLY")
}

func TestUnmarshalMapNestedSliceKeyWithUnderscore(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type cfg struct {
		M map[string][][]int `xml:"m"`
	}

	config := &cfg{}
	worked, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"M_server_99_0_0": "1",
		"M_server_99_0_1": "2",
	}, config)

	require.NoError(t, err)
	assert.True(worked)
	assert.Equal([][]int{{1, 2}}, config.M["server_99"])
	assert.NotContains(config.M, "server")
}

func TestUnmarshalENVEmptySliceMapValueClears(t *testing.T) {
	assert := assert.New(t)

	t.Setenv("APP_VALUES_server_99", "")

	type cfg struct {
		Values map[string][]int `xml:"values"`
	}

	config := &cfg{Values: map[string][]int{"server_99": {1, 2}}}
	ok, err := cnfg.UnmarshalENV(config, "APP")

	require.NoError(t, err)
	assert.True(ok)
	assert.Empty(config.Values["server_99"])
}

func TestUnmarshalMapNamedByteSliceUsesIndexes(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type bytes []byte

	type cfg struct {
		Files map[string]bytes  `xml:"files"`
		Blob  map[string][]byte `xml:"blob"`
	}

	config := &cfg{}
	worked, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"FILES_key_0": "A",
		"BLOB_key":    "AB",
	}, config)

	require.NoError(t, err)
	assert.True(worked)
	require.Contains(t, config.Files, "key")
	assert.Equal(bytes{'A'}, config.Files["key"])
	assert.NotContains(config.Files, "key_0")
	assert.Equal([]byte("AB"), config.Blob["key"])
}

func TestUnmarshalMapAnonymousTaggedEmbedUsesTag(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type Inner struct {
		X string `xml:"x"`
	}

	type outer struct {
		Inner `xml:"Inner"`

		Y string `xml:"y"`
	}

	type cfg struct {
		M map[string]outer `xml:"m"`
	}

	config := &cfg{}
	worked, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"M_a_INNER_X": "hello",
		"M_a_Y":       "y",
	}, config)

	require.NoError(t, err)
	assert.True(worked)
	require.Contains(t, config.M, "a")
	assert.Equal("hello", config.M["a"].X)
	assert.Equal("y", config.M["a"].Y)
}

func TestUnmarshalENVMapTimeXCompanionVar(t *testing.T) {
	assert := assert.New(t)

	t.Setenv("APP_IN_job", "5m")
	t.Setenv("APP_IN_job_X", "10")

	type cfg struct {
		In map[string]TimeX `xml:"in"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalENV(config, "APP")

	require.NoError(t, err)
	assert.True(ok)
	require.Contains(t, config.In, "job")
	assert.Equal(50*time.Minute, config.In["job"].Duration)
	assert.NotContains(config.In, "job_X")
}

func TestUnmarshalENVMapTimeXDelenvAfterParse(t *testing.T) {
	assert := assert.New(t)

	t.Setenv("APP_IN_job", "5m")
	t.Setenv("APP_IN_job_X", "10")

	type cfg struct {
		In map[string]TimeX `xml:"in,delenv"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalENV(config, "APP")

	require.NoError(t, err)
	assert.True(ok)
	require.Contains(t, config.In, "job")
	assert.Equal(50*time.Minute, config.In["job"].Duration)
	assert.Empty(os.Getenv("APP_IN_job"))
	assert.Empty(os.Getenv("APP_IN_job_X"))
}

func TestUnmarshalMapASCIISliceIndexNotUnicodeDigit(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type cfg struct {
		M map[string][]int `xml:"m"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"M_server_١_0": "128",
	}, config)

	require.NoError(t, err)
	assert.True(ok)
	assert.Equal([]int{128}, config.M["server_١"])
	assert.NotContains(config.M, "server")
}

func TestUnmarshalMapUnexportedEmbedDoesNotRecurse(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type node struct {
		*node

		Value string `xml:"value"`
	}

	type cfg struct {
		N map[string]node `xml:"n"`
	}

	config := &cfg{}
	ok, err := cnfg.UnmarshalMap(cnfg.Pairs{
		"N_a_VALUE": "ok",
	}, config)

	require.NoError(t, err)
	assert.True(ok)
	require.Contains(t, config.N, "a")
	assert.Nil(config.N["a"].node)
	assert.Equal("ok", config.N["a"].Value)
}

func BenchmarkUnmarshalMapManyRoles(b *testing.B) {
	type role struct {
		Permissions []string `xml:"permissions"`
	}

	type cfg struct {
		Roles map[string]role `xml:"roles"`
	}

	pairs := make(cnfg.Pairs, 900)

	for idx := range 300 {
		name := "role" + strconv.Itoa(idx)
		pairs["ROLES_"+name+"_PERMISSIONS_0"] = testPermAPI
		pairs["ROLES_"+name+"_PERMISSIONS_1"] = testPermGUI
		pairs["ROLES_"+name+"_PERMISSIONS_2"] = testStats
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		config := &cfg{}
		if _, err := cnfg.UnmarshalMap(pairs, config); err != nil {
			b.Fatal(err)
		}
	}
}
