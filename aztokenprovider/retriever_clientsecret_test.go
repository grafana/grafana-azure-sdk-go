package aztokenprovider

import (
	"testing"

	"github.com/naizerjohn-ms/grafana-azure-sdk-go/azcredentials"
	"github.com/naizerjohn-ms/grafana-azure-sdk-go/azsettings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureTokenProvider_getClientSecretCredential(t *testing.T) {
	var settings = &azsettings.AzureSettings{
		Cloud: azsettings.AzurePublic,
	}

	defaultCredentials := func() *azcredentials.AzureClientSecretCredentials {
		return &azcredentials.AzureClientSecretCredentials{
			AzureCloud:   azsettings.AzurePublic,
			Authority:    "",
			TenantId:     "7dcf1d1a-4ec0-41f2-ac29-c1538a698bc4",
			ClientId:     "1af7c188-e5b6-4f96-81b8-911761bdd459",
			ClientSecret: "0416d95e-8af8-472c-aaa3-15c93c46080a",
		}
	}

	t.Run("should return clientSecretTokenRetriever with values", func(t *testing.T) {
		credentials := defaultCredentials()

		result, err := getClientSecretTokenRetriever(settings, credentials)
		require.NoError(t, err)

		assert.IsType(t, &clientSecretTokenRetriever{}, result)
		credential := (result).(*clientSecretTokenRetriever)

		assert.Equal(t, "https://login.microsoftonline.com/", credential.cloudConf.ActiveDirectoryAuthorityHost)
		assert.Equal(t, "7dcf1d1a-4ec0-41f2-ac29-c1538a698bc4", credential.tenantId)
		assert.Equal(t, "1af7c188-e5b6-4f96-81b8-911761bdd459", credential.clientId)
		assert.Equal(t, "0416d95e-8af8-472c-aaa3-15c93c46080a", credential.clientSecret)
	})

	t.Run("authority should selected based on cloud", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.AzureCloud = azsettings.AzureChina

		result, err := getClientSecretTokenRetriever(settings, credentials)
		require.NoError(t, err)

		assert.IsType(t, &clientSecretTokenRetriever{}, result)
		credential := (result).(*clientSecretTokenRetriever)

		assert.Equal(t, "https://login.chinacloudapi.cn/", credential.cloudConf.ActiveDirectoryAuthorityHost)
	})

	t.Run("explicitly set authority should have priority over cloud", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.AzureCloud = azsettings.AzureChina
		credentials.Authority = "https://another.com/"

		result, err := getClientSecretTokenRetriever(settings, credentials)
		require.NoError(t, err)

		assert.IsType(t, &clientSecretTokenRetriever{}, result)
		credential := (result).(*clientSecretTokenRetriever)

		assert.Equal(t, "https://another.com/", credential.cloudConf.ActiveDirectoryAuthorityHost)
	})

	t.Run("should fail with error if cloud is not supported", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.AzureCloud = "InvalidCloud"

		_, err := getClientSecretTokenRetriever(settings, credentials)
		require.Error(t, err)
	})
}
