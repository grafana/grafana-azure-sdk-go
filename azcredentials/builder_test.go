package azcredentials

import (
	"testing"

	"github.com/grafana/grafana-azure-sdk-go/azsettings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromDatasourceData(t *testing.T) {
	t.Run("should return nil when no credentials configured", func(t *testing.T) {
		var data = map[string]interface{}{}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		assert.Nil(t, result)
	})

	t.Run("should return current user credentials when current user auth configured", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType": "currentuser",
			},
		}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AadCurrentUserCredentials{}, result)
	})

	t.Run("should return managed identity credentials when managed identity auth configured", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType": "msi",
			},
		}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AzureManagedIdentityCredentials{}, result)
		credential := (result).(*AzureManagedIdentityCredentials)

		// ClientId currently not parsed
		assert.Equal(t, credential.ClientId, "")
	})

	t.Run("should return client secret credentials when client secret auth configured", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":   "clientsecret",
				"azureCloud": "AzureChinaCloud",
				"tenantId":   "TENANT-ID",
				"clientId":   "CLIENT-TD",
			},
		}
		var secureData = map[string]string{
			"azureClientSecret": "FAKE-SECRET",
		}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AzureClientSecretCredentials{}, result)
		credential := (result).(*AzureClientSecretCredentials)

		assert.Equal(t, credential.AzureCloud, azsettings.AzureChina)
		assert.Equal(t, credential.TenantId, "TENANT-ID")
		assert.Equal(t, credential.ClientId, "CLIENT-TD")
		assert.Equal(t, credential.ClientSecret, "FAKE-SECRET")
	})

	t.Run("should return on-behalf-of credentials when on-behalf-of auth configured", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":   "clientsecret-obo",
				"azureCloud": "AzureChinaCloud",
				"tenantId":   "TENANT-ID",
				"clientId":   "CLIENT-TD",
			},
		}
		var secureData = map[string]string{
			"azureClientSecret": "FAKE-SECRET",
		}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AzureClientSecretOboCredentials{}, result)
		credential := (result).(*AzureClientSecretOboCredentials)

		require.NotNil(t, credential.ClientSecretCredentials)
		assert.Equal(t, credential.ClientSecretCredentials.AzureCloud, azsettings.AzureChina)
		assert.Equal(t, credential.ClientSecretCredentials.TenantId, "TENANT-ID")
		assert.Equal(t, credential.ClientSecretCredentials.ClientId, "CLIENT-TD")
		assert.Equal(t, credential.ClientSecretCredentials.ClientSecret, "FAKE-SECRET")
	})

	t.Run("should return error when credentials not supported", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":   "invalid",
				"azureCloud": "AzureChinaCloud",
				"tenantId":   "TENANT-ID",
				"clientId":   "CLIENT-TD",
			},
		}
		var secureData = map[string]string{
			"azureClientSecret": "FAKE-SECRET",
		}

		_, err := FromDatasourceData(data, secureData)
		assert.Error(t, err)
	})
}
