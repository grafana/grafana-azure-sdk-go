package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var data = map[string]interface{}{
	"number_field":  42,
	"boolean_field": true,
	"string_field":  "string_value",
	"object_field":  map[string]interface{}{},
}

func TestGetMap(t *testing.T) {
	t.Run("should return error if given field not found", func(t *testing.T) {
		_, err := GetMap(data, "not_exist")
		assert.Error(t, err)
	})

	t.Run("should return error if value not a map", func(t *testing.T) {
		_, err := GetMap(data, "string_field")
		assert.Error(t, err)
	})

	t.Run("should return map value of the given field", func(t *testing.T) {
		value, err := GetMap(data, "object_field")
		require.NoError(t, err)

		assert.NotNil(t, value)
		assert.IsType(t, map[string]interface{}{}, value)
	})
}

func TestGetMapOptional(t *testing.T) {
	t.Run("should return nil if given field not found", func(t *testing.T) {
		value, err := GetMapOptional(data, "not_exist")
		require.NoError(t, err)

		assert.Nil(t, value)
	})

	t.Run("should return error if value not a map", func(t *testing.T) {
		_, err := GetMapOptional(data, "string_field")
		assert.Error(t, err)
	})

	t.Run("should return map value of the given field", func(t *testing.T) {
		value, err := GetMapOptional(data, "object_field")
		require.NoError(t, err)

		assert.NotNil(t, value)
		assert.IsType(t, map[string]interface{}{}, value)
	})
}

func TestGetBool(t *testing.T) {
	t.Run("should return error if given field not found", func(t *testing.T) {
		_, err := GetBool(data, "not_exist")
		assert.Error(t, err)
	})

	t.Run("should return error if value not a bool", func(t *testing.T) {
		_, err := GetBool(data, "string_field")
		assert.Error(t, err)
	})

	t.Run("should return bool value of the given field", func(t *testing.T) {
		value, err := GetBool(data, "boolean_field")
		require.NoError(t, err)

		assert.Equal(t, true, value)
	})
}

func TestGetBoolOptional(t *testing.T) {
	t.Run("should return false if given field not found", func(t *testing.T) {
		value, err := GetBoolOptional(data, "not_exist")
		require.NoError(t, err)

		assert.Equal(t, false, value)
	})

	t.Run("should return error if value not a bool", func(t *testing.T) {
		_, err := GetBoolOptional(data, "string_field")
		assert.Error(t, err)
	})

	t.Run("should return bool value of the given field", func(t *testing.T) {
		value, err := GetBoolOptional(data, "boolean_field")
		require.NoError(t, err)

		assert.Equal(t, true, value)
	})
}

func TestGetString(t *testing.T) {
	t.Run("should return error if given field not found", func(t *testing.T) {
		_, err := GetString(data, "not_exist")
		assert.Error(t, err)
	})

	t.Run("should return error if value not a string", func(t *testing.T) {
		_, err := GetString(data, "number_field")
		assert.Error(t, err)
	})

	t.Run("should return string value of the given field", func(t *testing.T) {
		value, err := GetString(data, "string_field")
		require.NoError(t, err)

		assert.Equal(t, "string_value", value)
	})
}

func TestGetStringOptional(t *testing.T) {
	t.Run("should return empty string if given field not found", func(t *testing.T) {
		value, err := GetStringOptional(data, "not_exist")
		require.NoError(t, err)

		assert.Equal(t, "", value)
	})

	t.Run("should return error if value not a string", func(t *testing.T) {
		_, err := GetStringOptional(data, "number_field")
		assert.Error(t, err)
	})

	t.Run("should return string value of the given field", func(t *testing.T) {
		value, err := GetStringOptional(data, "string_field")
		require.NoError(t, err)

		assert.Equal(t, "string_value", value)
	})
}
