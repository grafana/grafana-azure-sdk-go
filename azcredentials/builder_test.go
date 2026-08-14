package azcredentials

import (
	"testing"

	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
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

	t.Run("should return current user credentials with service credentials (client secret)", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":                  "currentuser",
				"serviceCredentialsEnabled": true,
				"serviceCredentials": map[string]interface{}{
					"authType":   "clientsecret",
					"azureCloud": "AzureCloud",
					"tenantId":   "TENANT-ID",
					"clientId":   "CLIENT-ID",
				},
			},
		}
		var secureData = map[string]string{
			"azureClientSecret": "FAKE-SECRET",
		}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AadCurrentUserCredentials{}, result)

		credential := result.(*AadCurrentUserCredentials)
		serviceCredential := credential.ServiceCredentials

		assert.Equal(t, credential.ServiceCredentialsEnabled, true)
		assert.NotNil(t, credential.ServiceCredentials)
		assert.IsType(t, &AzureClientSecretCredentials{}, serviceCredential)
		assert.Equal(t, serviceCredential.(*AzureClientSecretCredentials).ClientId, "CLIENT-ID")
		assert.Equal(t, serviceCredential.(*AzureClientSecretCredentials).TenantId, "TENANT-ID")
		assert.Equal(t, serviceCredential.(*AzureClientSecretCredentials).ClientSecret, "FAKE-SECRET")
		assert.Equal(t, serviceCredential.(*AzureClientSecretCredentials).AzureCloud, "AzureCloud")
	})

	t.Run("should return current user credentials with service credentials (workload identity)", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":                  "currentuser",
				"serviceCredentialsEnabled": true,
				"serviceCredentials": map[string]interface{}{
					"authType": "workloadidentity",
				},
			},
		}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AadCurrentUserCredentials{}, result)

		credential := result.(*AadCurrentUserCredentials)
		serviceCredential := credential.ServiceCredentials

		assert.Equal(t, credential.ServiceCredentialsEnabled, true)
		assert.NotNil(t, credential.ServiceCredentials)
		assert.IsType(t, &AzureWorkloadIdentityCredentials{}, serviceCredential)
	})

	t.Run("should return current user credentials with service credentials (managed identity)", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":                  "currentuser",
				"serviceCredentialsEnabled": true,
				"serviceCredentials": map[string]interface{}{
					"authType": "msi",
				},
			},
		}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AadCurrentUserCredentials{}, result)

		credential := result.(*AadCurrentUserCredentials)
		serviceCredential := credential.ServiceCredentials

		assert.Equal(t, credential.ServiceCredentialsEnabled, true)
		assert.NotNil(t, credential.ServiceCredentials)
		assert.IsType(t, &AzureManagedIdentityCredentials{}, serviceCredential)
	})

	t.Run("should return current user credentials without service credentials if disabled", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":                  "currentuser",
				"serviceCredentialsEnabled": false,
				"serviceCredentials": map[string]interface{}{
					"authType": "msi",
				},
			},
		}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AadCurrentUserCredentials{}, result)

		credential := result.(*AadCurrentUserCredentials)

		assert.Equal(t, credential.ServiceCredentialsEnabled, false)
		assert.Nil(t, credential.ServiceCredentials)
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

	t.Run("should return workload identity credentials when workload identity auth configured", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType": "workloadidentity",
				"tenantId": "TENANT-ID",
				"clientId": "CLIENT-ID",
			},
		}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AzureWorkloadIdentityCredentials{}, result)
		credential := (result).(*AzureWorkloadIdentityCredentials)

		assert.Equal(t, credential.TenantId, "TENANT-ID")
		assert.Equal(t, credential.ClientId, "CLIENT-ID")
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

	t.Run("should return client certificate credentials when certificate auth configured (pem)", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":          "clientcertificate",
				"azureCloud":        "AzureChinaCloud",
				"tenantId":          "TENANT-ID",
				"clientId":          "CLIENT-TD",
				"certificateFormat": "pem",
			},
		}
		var secureData = map[string]string{
			"clientCertificate": "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----",
			"privateKey":        "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----",
		}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AzureClientCertificateCredentials{}, result)
		credential := (result).(*AzureClientCertificateCredentials)

		assert.Equal(t, credential.AzureCloud, azsettings.AzureChina)
		assert.Equal(t, credential.TenantId, "TENANT-ID")
		assert.Equal(t, credential.ClientId, "CLIENT-TD")
		assert.Equal(t, credential.CertificateFormat, "pem")
		assert.Equal(t, credential.ClientCertificate, "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----")
		assert.Equal(t, credential.PrivateKey, "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----")
	})

	t.Run("should return error for certificate auth when certificate missing (pem)", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":          "clientcertificate",
				"azureCloud":        "AzureCloud",
				"tenantId":          "TENANT-ID",
				"clientId":          "CLIENT-TD",
				"certificateFormat": "pem",
			},
		}
		var secureData = map[string]string{
			"privateKey": "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----",
		}

		_, err := FromDatasourceData(data, secureData)
		require.Error(t, err)
		require.ErrorContains(t, err, "no certificate provided")
	})

	t.Run("should return error for certificate auth when private key missing (pem)", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":          "clientcertificate",
				"azureCloud":        "AzureCloud",
				"tenantId":          "TENANT-ID",
				"clientId":          "CLIENT-TD",
				"certificateFormat": "pem",
			},
		}
		var secureData = map[string]string{
			"clientCertificate": "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----",
		}

		_, err := FromDatasourceData(data, secureData)
		require.Error(t, err)
		require.ErrorContains(t, err, "no private key provided")
	})

	t.Run("should return error for certificate auth when certificate format missing", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":   "clientcertificate",
				"azureCloud": "AzureCloud",
				"tenantId":   "TENANT-ID",
				"clientId":   "CLIENT-TD",
			},
		}
		var secureData = map[string]string{
			"clientCertificate": "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----",
			"privateKey":        "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----",
		}

		_, err := FromDatasourceData(data, secureData)
		require.Error(t, err)
		require.ErrorContains(t, err, "no certificate format provided")
	})

	t.Run("should return error for certificate auth when certificate format invalid", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":          "clientcertificate",
				"azureCloud":        "AzureCloud",
				"tenantId":          "TENANT-ID",
				"clientId":          "CLIENT-TD",
				"certificateFormat": "invalid",
			},
		}
		var secureData = map[string]string{
			"clientCertificate": "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----",
			"privateKey":        "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----",
		}

		_, err := FromDatasourceData(data, secureData)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid certificate format provided")
	})

	t.Run("should return client certificate credentials when certificate auth configured (pfx)", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":          "clientcertificate",
				"azureCloud":        "AzureCloud",
				"tenantId":          "TENANT-ID",
				"clientId":          "CLIENT-TD",
				"certificateFormat": "pfx",
			},
		}
		var secureData = map[string]string{
			"clientCertificate":   "BASE64_PFX_BLOB",
			"certificatePassword": "pfx-password",
		}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)
		require.IsType(t, &AzureClientCertificateCredentials{}, result)

		credential := result.(*AzureClientCertificateCredentials)
		assert.Equal(t, "pfx", credential.CertificateFormat)
		assert.Equal(t, "BASE64_PFX_BLOB", credential.ClientCertificate)
	})

	t.Run("should return error for pfx auth when client certificate missing", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":          "clientcertificate",
				"azureCloud":        "AzureCloud",
				"tenantId":          "TENANT-ID",
				"clientId":          "CLIENT-TD",
				"certificateFormat": "pfx",
			},
		}
		var secureData = map[string]string{
			"certificatePassword": "pfx-password",
		}

		_, err := FromDatasourceData(data, secureData)
		require.Error(t, err)
		require.ErrorContains(t, err, "no certificate provided")
	})

	t.Run("should return error for pfx auth when password missing", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":          "clientcertificate",
				"azureCloud":        "AzureCloud",
				"tenantId":          "TENANT-ID",
				"clientId":          "CLIENT-TD",
				"certificateFormat": "pfx",
			},
		}
		var secureData = map[string]string{
			"clientCertificate": "BASE64_PFX_BLOB",
		}

		_, err := FromDatasourceData(data, secureData)
		require.Error(t, err)
		require.ErrorContains(t, err, "no password provided")
	})

	t.Run("should return client secret when legacy client secret saved", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":   "clientsecret",
				"azureCloud": "AzureCloud",
				"tenantId":   "TENANT-ID",
				"clientId":   "CLIENT-TD",
			},
		}
		var secureData = map[string]string{
			"clientSecret": "FAKE-LEGACY-SECRET",
		}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		require.IsType(t, &AzureClientSecretCredentials{}, result)
		credential := (result).(*AzureClientSecretCredentials)

		assert.Equal(t, credential.ClientSecret, "FAKE-LEGACY-SECRET")
	})

	t.Run("should return on-behalf-of client secret when legacy client secret saved", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":   "clientsecret-obo",
				"azureCloud": "AzureCloud",
				"tenantId":   "TENANT-ID",
				"clientId":   "CLIENT-TD",
			},
		}
		var secureData = map[string]string{
			"clientSecret": "FAKE-LEGACY-SECRET",
		}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		require.IsType(t, &AzureClientSecretOboCredentials{}, result)
		credential := (result).(*AzureClientSecretOboCredentials)

		require.NotNil(t, credential.ClientSecretCredentials)
		assert.Equal(t, credential.ClientSecretCredentials.ClientSecret, "FAKE-LEGACY-SECRET")
	})

	t.Run("should ignore legacy client secret if new client secret saved", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":   "clientsecret",
				"azureCloud": "AzureCloud",
				"tenantId":   "TENANT-ID",
				"clientId":   "CLIENT-TD",
			},
		}
		var secureData = map[string]string{
			"azureClientSecret": "FAKE-SECRET",
			"clientSecret":      "FAKE-LEGACY-SECRET",
		}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		require.IsType(t, &AzureClientSecretCredentials{}, result)
		credential := (result).(*AzureClientSecretCredentials)

		assert.Equal(t, credential.ClientSecret, "FAKE-SECRET")
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

	t.Run("should return federated identity credentials when federated identity auth configured", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":                    "federatedidentity",
				"sourceClientId":              "SOURCE-CLIENT-ID",
				"targetTenantId":              "TARGET-TENANT-ID",
				"targetClientId":              "TARGET-CLIENT-ID",
				"federatedCredentialAudience": "api://AzureADTokenExchange",
			},
		}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AzureFederatedIdentityCredentials{}, result)
		credential := result.(*AzureFederatedIdentityCredentials)

		assert.Equal(t, "SOURCE-CLIENT-ID", credential.SourceClientId)
		assert.Equal(t, "TARGET-TENANT-ID", credential.TargetTenantId)
		assert.Equal(t, "TARGET-CLIENT-ID", credential.TargetClientId)
		assert.Equal(t, "api://AzureADTokenExchange", credential.FederatedCredentialAudience)
	})

	t.Run("should return federated identity credentials when source client ID omitted", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":                    "federatedidentity",
				"targetTenantId":              "TARGET-TENANT-ID",
				"targetClientId":              "TARGET-CLIENT-ID",
				"federatedCredentialAudience": "api://AzureADTokenExchange",
			},
		}
		var secureData = map[string]string{}

		result, err := FromDatasourceData(data, secureData)
		require.NoError(t, err)

		require.NotNil(t, result)
		assert.IsType(t, &AzureFederatedIdentityCredentials{}, result)
		credential := result.(*AzureFederatedIdentityCredentials)

		assert.Equal(t, "", credential.SourceClientId)
	})

	t.Run("should return error for federated identity with missing target tenant ID", func(t *testing.T) {
		var data = map[string]interface{}{
			"azureCredentials": map[string]interface{}{
				"authType":                    "federatedidentity",
				"targetClientId":              "TARGET-CLIENT-ID",
				"federatedCredentialAudience": "api://AzureADTokenExchange",
			},
		}
		var secureData = map[string]string{}

		_, err := FromDatasourceData(data, secureData)
		require.Error(t, err)
	})
}
