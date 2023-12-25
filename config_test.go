package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	main "github.com/lesomnus/vcpkg-cache-http"
	"github.com/mattn/go-isatty"
	"github.com/stretchr/testify/require"
)

func TestStoreConfigString(t *testing.T) {
	require := require.New(t)

	opts_kv := map[string]string{"opt1": "val1", "opt2": "val2"}
	opts_k_only := map[string]string{"opt1": "", "opt2": ""}

	tcs := []*main.StoreConfig{
		{Kind: "kind", Path: "path", Opts: map[string]string{}},
		{Kind: "kind", Path: "", Opts: opts_kv},
		{Kind: "kind", Path: "path", Opts: opts_kv},
		{Kind: "kind", Path: "", Opts: opts_k_only},
		{Kind: "kind", Path: "path", Opts: opts_k_only},
	}
	for _, tc := range tcs {
		conf, err := main.ParseStoreConfig(tc.String())
		require.NoError(err)
		require.Equal(tc, conf)
	}
}

func TestParseBackendConfig(t *testing.T) {
	require := require.New(t)

	case_2opts := main.StoreConfig{
		Kind: "kind",
		Path: "path",
		Opts: map[string]string{
			"opt1": "val1",
			"opt2": "val2",
		},
	}

	case_key_only := main.StoreConfig{
		Kind: "kind",
		Path: "path",
		Opts: map[string]string{
			"opt1": "",
			"opt2": "",
		},
	}

	test_cases := []struct {
		given    string
		expected main.StoreConfig
	}{
		{
			given: "kind",
			expected: main.StoreConfig{
				Kind: "kind",
				Opts: map[string]string{},
			},
		},
		{
			given: "kind:",
			expected: main.StoreConfig{
				Kind: "kind",
				Opts: map[string]string{},
			},
		},
		{
			given: "kind:,opt1,opt2=val2",
			expected: main.StoreConfig{
				Kind: "kind",
				Opts: map[string]string{
					"opt1": "",
					"opt2": "val2",
				},
			},
		},
		{
			given: "kind:path",
			expected: main.StoreConfig{
				Kind: "kind",
				Path: "path",
				Opts: map[string]string{},
			},
		},
		{
			given:    "kind:path,opt1=val1,opt2=val2",
			expected: case_2opts,
		},
		{
			given:    "kind:path,opt1=val1,,opt2=val2",
			expected: case_2opts,
		},
		{
			given:    "kind:path,opt1,opt2",
			expected: case_key_only,
		},
		{
			given:    "kind:path,opt1,,opt2",
			expected: case_key_only,
		},
		{
			given:    "kind:path,opt1,=,opt2",
			expected: case_key_only,
		},
		{
			given:    "kind:path,opt1=,opt2",
			expected: case_key_only,
		},
		{
			given:    "kind:path,opt1=,=val3,opt2",
			expected: case_key_only,
		},
	}
	for _, test_case := range test_cases {
		actual, err := main.ParseStoreConfig(test_case.given)
		require.NoError(err)
		require.NotNil(actual)
		require.Equal(*actual, test_case.expected)
	}
}

func TestParseBackendConfigFail(t *testing.T) {
	t.Run("kind must be specified", func(t *testing.T) {
		require := require.New(t)

		test_cases := []struct {
			given string
		}{
			{given: ""},
			{given: ":"},
			{given: ":path"},
			{given: ":path,opt1=val1"},
			{given: ":,opt1=val1"},
			{given: ",opt1=val1"},
			{given: ",opt1"},
			{given: ","},
			{given: "opt1=val1"},
			{given: "opt1="},
		}
		for _, test_case := range test_cases {
			_, err := main.ParseStoreConfig(test_case.given)
			require.ErrorContains(err, "kind")
		}
	})
}

func writeConfig(t *testing.T, conf *main.AppConfig) string {
	require := require.New(t)

	data, err := json.Marshal(conf)
	require.NoError(err)

	conf_path := filepath.Join(t.TempDir(), "conf.json")
	f, err := os.OpenFile(conf_path, os.O_WRONLY|os.O_CREATE, 0644)
	require.NoError(err)
	defer f.Close()

	_, err = f.Write(data)
	require.NoError(err)

	return conf_path
}

func TestParseArgs(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		require := require.New(t)

		expected := &main.AppConfig{
			Host: "0.0.0.0",
			Port: 15151,

			Store: nil,

			NoColor: !isatty.IsTerminal(os.Stdout.Fd()),
			LogJson: false,

			ReadOnly:  false,
			WriteOnly: false,
		}

		conf, err := main.ParseArgs([]string{""})
		require.NoError(err)
		require.NotNil(conf)
		require.Equal(expected, conf)
	})

	t.Run("config on command line", func(t *testing.T) {
		require := require.New(t)

		expected := &main.AppConfig{
			Host: "bar",
			Port: 1234,

			Store: &main.StoreConfig{
				Kind: "files",
				Path: "store-data-here",
				Opts: map[string]string{},
			},

			NoColor: true,
			LogJson: true,

			ReadOnly:  true,
			WriteOnly: false,
		}

		conf, err := main.ParseArgs([]string{
			"",
			"-host", "bar",
			"-port", "1234",
			"-no-color",
			"-log-json",
			"-read-only",
			"files:store-data-here",
		})
		require.NoError(err)
		require.NotNil(conf)
		require.Equal(expected, conf)
	})

	t.Run("read config from file", func(t *testing.T) {
		require := require.New(t)

		expected := &main.AppConfig{
			Host: "foo",
			Port: 42,

			Store: &main.StoreConfig{
				Kind: "bar",
				Path: "baz",
				Opts: map[string]string{
					"opt1": "val1",
					"opt2": "val2",
				},
			},

			NoColor: isatty.IsTerminal(os.Stdout.Fd()),
			LogJson: true,

			ReadOnly:  true,
			WriteOnly: true,
		}
		conf_path := writeConfig(t, expected)

		conf, err := main.ParseArgs([]string{"", "-conf", conf_path})
		require.NoError(err)
		require.NotNil(conf)
		require.Equal(expected, conf)
	})

	t.Run("config from file is overridden by cmd arguments", func(t *testing.T) {
		require := require.New(t)

		expected := &main.AppConfig{
			Host: "bar",
			Port: 1234,

			Store: nil,

			NoColor: true,
			LogJson: true,

			ReadOnly:  true,
			WriteOnly: true,
		}

		conf_path := writeConfig(t, &main.AppConfig{
			Host: "foo",
			Port: 42,

			Store: nil,

			NoColor: false,
			LogJson: false,

			ReadOnly:  false,
			WriteOnly: false,
		})

		conf, err := main.ParseArgs([]string{
			"",
			"-conf", conf_path,
			"-host", "bar",
			"-port", "1234",
			"-no-color",
			"-log-json",
			"-read-only",
			"-write-only",
		})
		require.NoError(err)
		require.NotNil(conf)
		require.Equal(expected, conf)
	})

	t.Run("fail if store config is invalid", func(t *testing.T) {
		require := require.New(t)

		_, err := main.ParseArgs([]string{"", ":,"})
		require.ErrorContains(err, "parse store config")
	})

	t.Run("only 1 positional argument is allowed", func(t *testing.T) {
		require := require.New(t)

		_, err := main.ParseArgs([]string{"", "files:foo", "files:bar"})
		require.ErrorContains(err, "only 1 positional argument")
	})

	t.Run("given config file must be exist", func(t *testing.T) {
		require := require.New(t)

		tmp := t.TempDir()
		_, err := main.ParseArgs([]string{"", "-conf", filepath.Join(tmp, "not-exists")})
		require.ErrorContains(err, "read config")
	})

	t.Run("given config file shouldn't be a directory", func(t *testing.T) {
		require := require.New(t)

		tmp := t.TempDir()
		_, err := main.ParseArgs([]string{"", "-conf", tmp})
		require.ErrorContains(err, "read config")
	})

	t.Run("given config file must be valid JSON", func(t *testing.T) {
		require := require.New(t)

		conf_path := filepath.Join(t.TempDir(), "conf.json")
		err := os.WriteFile(conf_path, []byte("("), 0644)
		require.NoError(err)

		_, err = main.ParseArgs([]string{"", "-conf", conf_path})
		require.ErrorContains(err, "unmarshal config")
	})
}

func TestParseArgsStrict(t *testing.T) {
	t.Run("read-only and write-only cannot be set together", func(t *testing.T) {
		require := require.New(t)

		_, err := main.ParseArgsStrict([]string{"", "-read-only", "-write-only"})
		require.ErrorContains(err, "cannot be set together")
	})
}
