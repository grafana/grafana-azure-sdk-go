package envutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	envTestKey      = "ENV_VAR_TEST"
	fallbackTestKey = "FALLBACK_VAR_TEST"
)

func TestGet(t *testing.T) {
	t.Run("should return variable value if variable is set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "StringValue")
		require.NoError(t, err)
		defer unset()

		value, err := Get(envTestKey)
		require.NoError(t, err)
		assert.Equal(t, "StringValue", value)
	})

	t.Run("should return error if variable not set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()

		_, err = Get(envTestKey)
		assert.Error(t, err)
	})
}

func TestGetOrDefault(t *testing.T) {
	t.Run("should return variable value if variable is set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "StringValue")
		require.NoError(t, err)
		defer unset()

		value := GetOrDefault(envTestKey, "DefaultValue")
		assert.Equal(t, "StringValue", value)
	})

	t.Run("should return default value if variable not set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()

		value := GetOrDefault(envTestKey, "DefaultValue")
		assert.Equal(t, "DefaultValue", value)
	})
}

func TestGetBool(t *testing.T) {
	t.Run("should return variable value if variable is set to a bool value", func(t *testing.T) {
		tests := []struct {
			EnvValue string
			Expected bool
		}{
			{EnvValue: "false", Expected: false},
			{EnvValue: "true", Expected: true},
			{EnvValue: "FALSE", Expected: false},
			{EnvValue: "TRUE", Expected: true},
			{EnvValue: "False", Expected: false},
			{EnvValue: "True", Expected: true},
			{EnvValue: "0", Expected: false},
			{EnvValue: "1", Expected: true},
		}

		for _, tt := range tests {
			unset, err := setEnvVar(envTestKey, tt.EnvValue)
			require.NoError(t, err)
			defer unset()

			value, err := GetBool(envTestKey)
			require.NoError(t, err)
			assert.Equal(t, tt.Expected, value)
			if err != nil {
				return
			}
		}
	})

	t.Run("should return error if variable not set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()

		_, err = GetBool(envTestKey)
		assert.Error(t, err)
	})

	t.Run("should return error if variable is not bool value", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "StringValue")
		require.NoError(t, err)
		defer unset()

		_, err = GetBool(envTestKey)
		assert.Error(t, err)
	})
}

func TestGetBoolOrDefault(t *testing.T) {
	t.Run("should return variable value if variable is set to a bool value", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "false")
		require.NoError(t, err)
		defer unset()

		value, err := GetBoolOrDefault(envTestKey, true)
		require.NoError(t, err)
		assert.Equal(t, false, value)
	})

	t.Run("should return default value if variable not set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()

		value, err := GetBoolOrDefault(envTestKey, true)
		require.NoError(t, err)
		assert.Equal(t, true, value)
	})

	t.Run("should return error if variable is not bool value", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "StringValue")
		require.NoError(t, err)
		defer unset()

		_, err = GetBoolOrDefault(envTestKey, true)
		assert.Error(t, err)
	})
}

func TestGetOrFallback(t *testing.T) {
	t.Run("should return main value if main variable is set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "StringValue")
		require.NoError(t, err)
		defer unset()
		unset, err = setEnvVar(fallbackTestKey, "FallbackStringValue")
		require.NoError(t, err)
		defer unset()

		value := GetOrFallback(envTestKey, fallbackTestKey, "DefaultValue")
		assert.Equal(t, "StringValue", value)
	})

	t.Run("should return fallback value if main variable is not set but fallback variable is set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()
		unset, err = setEnvVar(fallbackTestKey, "FallbackStringValue")
		require.NoError(t, err)
		defer unset()

		value := GetOrFallback(envTestKey, fallbackTestKey, "DefaultValue")
		assert.Equal(t, "FallbackStringValue", value)
	})

	t.Run("should return default value if neither main nor fallback variables are set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()
		unset, err = setEnvVar(fallbackTestKey, "")
		require.NoError(t, err)
		defer unset()

		value := GetOrFallback(envTestKey, fallbackTestKey, "DefaultValue")
		assert.Equal(t, "DefaultValue", value)
	})
}

func TestGetBoolOrFallback(t *testing.T) {
	t.Run("should return main value if main variable is set to a bool value", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "true")
		require.NoError(t, err)
		defer unset()
		unset, err = setEnvVar(fallbackTestKey, "false")
		require.NoError(t, err)
		defer unset()

		value, err := GetBoolOrFallback(envTestKey, fallbackTestKey, true)
		require.NoError(t, err)
		assert.Equal(t, true, value)
	})

	t.Run("should return fallback value if main variable is not set but fallback variable is set to a bool value", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()
		unset, err = setEnvVar(fallbackTestKey, "false")
		require.NoError(t, err)
		defer unset()

		value, err := GetBoolOrFallback(envTestKey, fallbackTestKey, true)
		require.NoError(t, err)
		assert.Equal(t, false, value)
	})

	t.Run("should return default value if neither main nor fallback variables are set", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()
		unset, err = setEnvVar(fallbackTestKey, "")
		require.NoError(t, err)
		defer unset()

		value, err := GetBoolOrFallback(envTestKey, fallbackTestKey, true)
		require.NoError(t, err)
		assert.Equal(t, true, value)
	})

	t.Run("should return error if main variable is not bool value", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "StringValue")
		require.NoError(t, err)
		defer unset()
		unset, err = setEnvVar(fallbackTestKey, "false")
		require.NoError(t, err)
		defer unset()

		_, err = GetBoolOrFallback(envTestKey, fallbackTestKey, true)
		assert.Error(t, err)
	})

	t.Run("should return error if fallback variable is not bool value", func(t *testing.T) {
		unset, err := setEnvVar(envTestKey, "")
		require.NoError(t, err)
		defer unset()
		unset, err = setEnvVar(fallbackTestKey, "FallbackStringValue")
		require.NoError(t, err)
		defer unset()

		_, err = GetBoolOrFallback(envTestKey, fallbackTestKey, true)
		assert.Error(t, err)
	})
}

type unsetFunc = func()

func setEnvVar(key string, value string) (unsetFunc, error) {
	err := os.Setenv(key, value)
	if err != nil {
		return nil, err
	}

	return func() {
		_ = os.Unsetenv(key)
	}, nil
}
