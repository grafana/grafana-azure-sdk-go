package azsettings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClouds(t *testing.T) {
	settings := &AzureSettings{}

	clouds := settings.Clouds()

	assert.Len(t, clouds, 3)
	assert.Equal(t, clouds[0].Name, "AzureCloud")
	assert.Equal(t, clouds[1].Name, "AzureChinaCloud")
	assert.Equal(t, clouds[2].Name, "AzureUSGovernment")
}

func TestGetCloud(t *testing.T) {
	settings := &AzureSettings{}

	t.Run("should return cloud settings", func(t *testing.T) {
		cloud, err := settings.GetCloud(AzurePublic)
		require.NoError(t, err)

		assert.Equal(t, AzurePublic, cloud.Name)
		assert.Equal(t, "https://management.azure.com", cloud.Properties["resourceManager"])
	})

	t.Run("should return error if cloud not found", func(t *testing.T) {
		_, err := settings.GetCloud("InvalidCloud")
		assert.Error(t, err)
	})
}
