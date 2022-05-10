package azsettings

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromEnv(t *testing.T) {
	t.Run("should set cloud if variable is set", func(t *testing.T) {
		err := os.Setenv("GFAZPL_AZURE_CLOUD", "TestCloud")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, "TestCloud", azureSettings.Cloud)
	})

	t.Run("should set cloud if fallback variable is set", func(t *testing.T) {
		err := os.Setenv("GFAZPL_AZURE_CLOUD", "")
		require.NoError(t, err)
		err = os.Setenv("AZURE_CLOUD", "FallbackCloud")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, "FallbackCloud", azureSettings.Cloud)
	})

	t.Run("should set cloud to public cloud if variable is not set", func(t *testing.T) {
		err := os.Setenv("GFAZPL_AZURE_CLOUD", "")
		require.NoError(t, err)
		err = os.Setenv("AZURE_CLOUD", "")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, AzurePublic, azureSettings.Cloud)
	})

	t.Run("should enable managed identity if variable is set", func(t *testing.T) {
		err := os.Setenv("GFAZPL_MANAGED_IDENTITY_ENABLED", "true")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.True(t, azureSettings.ManagedIdentityEnabled)
	})

	t.Run("should enable managed identity if fallback variable is set", func(t *testing.T) {
		err := os.Setenv("GFAZPL_MANAGED_IDENTITY_ENABLED", "")
		require.NoError(t, err)
		err = os.Setenv("AZURE_MANAGED_IDENTITY_ENABLED", "true")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.True(t, azureSettings.ManagedIdentityEnabled)
	})

	t.Run("should disable managed identity if variable is not set", func(t *testing.T) {
		err := os.Setenv("GFAZPL_AZURE_MANAGED_IDENTITY_ENABLED", "")
		require.NoError(t, err)
		err = os.Setenv("AZURE_MANAGED_IDENTITY_ENABLED", "")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.False(t, azureSettings.ManagedIdentityEnabled)
	})

	t.Run("should set client ID if variable is set", func(t *testing.T) {
		err := os.Setenv("GFAZPL_MANAGED_IDENTITY_ENABLED", "true")
		require.NoError(t, err)
		err = os.Setenv("GFAZPL_MANAGED_IDENTITY_CLIENT_ID", "TestClientId")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, "TestClientId", azureSettings.ManagedIdentityClientId)
	})

	t.Run("should set client ID if fallback variable is set", func(t *testing.T) {
		err := os.Setenv("GFAZPL_MANAGED_IDENTITY_ENABLED", "true")
		require.NoError(t, err)
		err = os.Setenv("GFAZPL_MANAGED_IDENTITY_CLIENT_ID", "")
		require.NoError(t, err)
		err = os.Setenv("AZURE_MANAGED_IDENTITY_CLIENT_ID", "FallbackClientId")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, "FallbackClientId", azureSettings.ManagedIdentityClientId)
	})

	t.Run("should not set client ID if managed identity is not enabled", func(t *testing.T) {
		err := os.Setenv("GFAZPL_MANAGED_IDENTITY_ENABLED", "false")
		require.NoError(t, err)
		err = os.Setenv("GFAZPL_MANAGED_IDENTITY_CLIENT_ID", "TestClientId")
		require.NoError(t, err)

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, "", azureSettings.ManagedIdentityClientId)
	})
}

func TestWriteToEnvStr(t *testing.T) {
	t.Run("should return empty list if AzureSettings not set", func(t *testing.T) {
		envs := WriteToEnvStr(nil)

		assert.Len(t, envs, 0)
	})

	t.Run("should return cloud if set", func(t *testing.T) {
		azureSettings := &AzureSettings{
			Cloud: "AzureCloud",
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 1)
		assert.Equal(t, "GFAZPL_AZURE_CLOUD=AzureCloud", envs[0])
	})

	t.Run("should return managed identity set if enabled", func(t *testing.T) {
		azureSettings := &AzureSettings{
			ManagedIdentityEnabled: true,
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 1)
		assert.Equal(t, "GFAZPL_MANAGED_IDENTITY_ENABLED=true", envs[0])
	})

	t.Run("should return managed identity client ID if provided", func(t *testing.T) {
		azureSettings := &AzureSettings{
			ManagedIdentityEnabled:  true,
			ManagedIdentityClientId: "c2e68b2e",
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 2)
		assert.Equal(t, "GFAZPL_MANAGED_IDENTITY_ENABLED=true", envs[0])
		assert.Equal(t, "GFAZPL_MANAGED_IDENTITY_CLIENT_ID=c2e68b2e", envs[1])
	})

	t.Run("should not return managed identity client ID if not enabled", func(t *testing.T) {
		azureSettings := &AzureSettings{
			ManagedIdentityClientId: "c2e68b2e",
		}

		envs := WriteToEnvStr(azureSettings)

		assert.Len(t, envs, 0)
	})
}
