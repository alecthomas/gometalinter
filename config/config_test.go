package config

import (
	"bytes"
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/gometalinter/api"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		config string
		key    string
		value  interface{}
	}{
		{`output = "json"`, "Output", OutputJSON},
		{`output = "invalid"`, "Output", errors.New("InvalidEnum")},
		{`deadline = "5m30s"`, "Deadline", Duration(time.Minute*5 + time.Second*30)},
		{``, "Format", DefaultIssueFormat}, // Test that defaults do not get overwritten.
		{`exclude = ["foo"]`, "Exclude", []Regexp{{regexp.MustCompile("foo")}}},
		{`exclude = ["*"]`, "Exclude", errors.New("InvalidRegex")},
		{`fast = true`, "Fast", true},
		{`enabled = ["gotype"]`, "Enabled", []string{"gotype"}},
	}
	for _, test := range tests {
		name := test.key
		err, errors := test.value.(error)
		if errors {
			name += err.Error()
		}
		t.Run(name, func(t *testing.T) {
			config, err := ReadString(test.config)
			if errors {
				assert.Error(t, err)
				return
			}
			if !assert.NoError(t, err, test.key) {
				return
			}
			v := reflect.ValueOf(config).Elem().FieldByName(test.key)
			assert.Equal(t, test.value, v.Interface(), test.key)
		})
	}
}

func TestFormatTemplate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		config, err := ReadString(`format = "hello {{.world}}"`)
		require.NoError(t, err)
		w := &bytes.Buffer{}
		err = config.Format.Execute(w, map[string]string{"world": "world"})
		require.NoError(t, err)
		require.Equal(t, "hello world", w.String())
	})
	t.Run("Invalid", func(t *testing.T) {
		_, err := ReadString(`format = "hello {{.world"`)
		require.Error(t, err)
	})
}

func TestLinterConfig(t *testing.T) {
	type MisspellConfig struct {
		Locale string
		Ignore []string
	}

	config, err := ReadString(`
[linter.misspell]
locale = "UK"
ignore = ["color", "aluminum"]
`)
	require.NoError(t, err)

	actual := &MisspellConfig{}
	err = config.UnmarshalLinterConfig("misspell", actual)
	require.NoError(t, err)
	require.Equal(t, &MisspellConfig{
		Locale: "UK",
		Ignore: []string{"color", "aluminum"},
	}, actual)

	actual = &MisspellConfig{}
	err = config.UnmarshalLinterConfig("missing", actual)
	require.NoError(t, err)
	require.Equal(t, &MisspellConfig{}, actual)
}

func TestExternalLinterDefinition(t *testing.T) {
	config, err := ReadString(`
[define.misspell]
name = "misspell"
install_from = "github.com/client9/misspell/cmd/misspell"
partition = "files"
is_fast = true
command = "misspell -j 1 --locale \"{{.locale}}\""
pattern = "PATH:LINE:COL:MESSAGE"
severity = "warning"
`)
	require.NoError(t, err)
	misspell := config.Define["misspell"]
	misspell.Command = nil
	assert.Equal(t, predefinedPatterns["PATH:LINE:COL:MESSAGE"], misspell.Pattern.String())
	misspell.Pattern = Regexp{}
	assert.Equal(t, ExternalLinterDefinition{
		Name:              "misspell",
		InstallFrom:       "github.com/client9/misspell/cmd/misspell",
		PartitionStrategy: PartitionByFiles,
		IsFast:            true,
		Severity:          api.Warning,
	}, misspell)
}
