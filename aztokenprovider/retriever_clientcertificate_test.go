package aztokenprovider

import (
	"testing"

	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureTokenProvider_getClientCertificateCredential(t *testing.T) {
	var settings = &azsettings.AzureSettings{
		Cloud: azsettings.AzurePublic,
	}

	defaultCredentials := func() *azcredentials.AzureClientCertificateCredentials {
		return &azcredentials.AzureClientCertificateCredentials{
			AzureCloud:          azsettings.AzurePublic,
			Authority:           "",
			TenantId:            "7dcf1d1a-4ec0-41f2-ac29-c1538a698bc4",
			ClientId:            "1af7c188-e5b6-4f96-81b8-911761bdd459",
			CertificateFormat:   "pem",
			ClientCertificate:   "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----",
			PrivateKey:          "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----",
			EncryptedPrivateKey: "",
			PrivateKeyPassword:  "fake-private-key-password",
		}
	}

	t.Run("should return clientCertificateTokenRetriever with values", func(t *testing.T) {
		credentials := defaultCredentials()

		result, err := getClientCertificateTokenRetriever(settings, credentials)
		require.NoError(t, err)

		assert.IsType(t, &clientCertificateTokenRetriever{}, result)
		credential := (result).(*clientCertificateTokenRetriever)

		assert.Equal(t, "https://login.microsoftonline.com/", credential.cloudConf.ActiveDirectoryAuthorityHost)
		assert.Equal(t, "7dcf1d1a-4ec0-41f2-ac29-c1538a698bc4", credential.tenantId)
		assert.Equal(t, "1af7c188-e5b6-4f96-81b8-911761bdd459", credential.clientId)
		assert.Equal(t, "pem", credential.certificateFormat)
		assert.Equal(t, "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----", credential.clientCertificate)
		assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----", credential.privateKey)
		assert.Equal(t, "", credential.encryptedPrivateKey)
		assert.Equal(t, "fake-private-key-password", credential.privateKeyPassword)
	})

	t.Run("should map pem fields", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.CertificateFormat = "pem"
		credentials.ClientCertificate = "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----"
		credentials.PrivateKey = "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----"

		result, err := getClientCertificateTokenRetriever(settings, credentials)
		require.NoError(t, err)
		require.IsType(t, &clientCertificateTokenRetriever{}, result)

		credential := result.(*clientCertificateTokenRetriever)
		assert.Equal(t, "pem", credential.certificateFormat)
		assert.Equal(t, "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----", credential.clientCertificate)
		assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----", credential.privateKey)
	})

	t.Run("should map pfx fields", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.CertificateFormat = "pfx"
		credentials.EncryptedPrivateKey = "BASE64_PFX_BLOB"
		credentials.ClientCertificate = ""
		credentials.PrivateKey = ""

		result, err := getClientCertificateTokenRetriever(settings, credentials)
		require.NoError(t, err)
		require.IsType(t, &clientCertificateTokenRetriever{}, result)

		credential := result.(*clientCertificateTokenRetriever)
		assert.Equal(t, "pfx", credential.certificateFormat)
		assert.Equal(t, "BASE64_PFX_BLOB", credential.encryptedPrivateKey)
		assert.Equal(t, "fake-private-key-password", credential.privateKeyPassword)
	})

	t.Run("authority should be selected based on cloud", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.AzureCloud = azsettings.AzureChina

		result, err := getClientCertificateTokenRetriever(settings, credentials)
		require.NoError(t, err)

		assert.IsType(t, &clientCertificateTokenRetriever{}, result)
		credential := (result).(*clientCertificateTokenRetriever)

		assert.Equal(t, "https://login.chinacloudapi.cn/", credential.cloudConf.ActiveDirectoryAuthorityHost)
	})

	t.Run("explicitly set authority should have priority over cloud", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.AzureCloud = azsettings.AzureChina
		credentials.Authority = "https://another.com/"

		result, err := getClientCertificateTokenRetriever(settings, credentials)
		require.NoError(t, err)

		assert.IsType(t, &clientCertificateTokenRetriever{}, result)
		credential := (result).(*clientCertificateTokenRetriever)

		assert.Equal(t, "https://another.com/", credential.cloudConf.ActiveDirectoryAuthorityHost)
	})

	t.Run("should fail with error if cloud is not supported", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.AzureCloud = "InvalidCloud"

		_, err := getClientCertificateTokenRetriever(settings, credentials)
		require.Error(t, err)
	})
}
