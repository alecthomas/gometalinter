package config

import (
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{``, "Format", Template{DefaultIssueFormat}}, // Test that defaults do not get overwritten.
		{`exclude = ["foo"]`, "Exclude", []Regexp{{regexp.MustCompile("foo")}}},
		{`exclude = ["*"]`, "Exclude", errors.New("InvalidRegex")},
		{`fast = true`, "Fast", true},
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
