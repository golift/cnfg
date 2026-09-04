package cnfg_test

import (
	"os"
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
	assert.Empty(os.Getenv("foo_bar"), "delenv must not unset the bare map key")
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
