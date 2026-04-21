package aztokenprovider

import (
	"testing"

	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureTokenProvider_getFederatedIdentityTokenRetriever(t *testing.T) {
	var settings = &azsettings.AzureSettings{
		ManagedIdentityEnabled:  true,
		ManagedIdentityClientId: "default-msi-client-id",
	}

	defaultCredentials := func() *azcredentials.AzureFederatedIdentityCredentials {
		return &azcredentials.AzureFederatedIdentityCredentials{
			SourceClientId:              "source-client-id",
			TargetTenantId:              "d33b45df-af84-4d65-acde-1bd47e9d2ad9",
			TargetClientId:              "ff326f8b-ff33-4844-9311-58cea4dfa073",
			FederatedCredentialAudience: "api://AzureADTokenExchange",
		}
	}

	t.Run("should return retriever", func(t *testing.T) {
		credentials := defaultCredentials()

		result, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.NoError(t, err)

		assert.IsType(t, &federatedIdentityTokenRetriever{}, result)
		retriever := result.(*federatedIdentityTokenRetriever)

		assert.Equal(t, "source-client-id", retriever.sourceClientId)
		assert.Equal(t, "d33b45df-af84-4d65-acde-1bd47e9d2ad9", retriever.targetTenantId)
		assert.Equal(t, "ff326f8b-ff33-4844-9311-58cea4dfa073", retriever.targetClientId)
		assert.Equal(t, "api://AzureADTokenExchange", retriever.federatedCredentialAudience)
	})

	t.Run("should use default MSI client ID from settings when not specified", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.SourceClientId = ""

		result, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.NoError(t, err)

		retriever := result.(*federatedIdentityTokenRetriever)
		assert.Equal(t, "default-msi-client-id", retriever.sourceClientId)
	})

	t.Run("should fail when target tenant ID is missing", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.TargetTenantId = ""

		_, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target tenant ID is required")
	})

	t.Run("should fail when target client ID is missing", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.TargetClientId = ""

		_, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target client ID is required")
	})

	t.Run("should fail when federated credential audience is missing", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.FederatedCredentialAudience = ""

		_, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "federated credential audience is required")
	})

	t.Run("should fail when federated credential audience is invalid", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.FederatedCredentialAudience = "api://InvalidAudience"

		_, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("should produce correct cache key", func(t *testing.T) {
		credentials := defaultCredentials()

		result, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.NoError(t, err)

		cacheKey := result.GetCacheKey("tenant-123")
		assert.Equal(t, "azure|fic|source-client-id|d33b45df-af84-4d65-acde-1bd47e9d2ad9|ff326f8b-ff33-4844-9311-58cea4dfa073|api://AzureADTokenExchange|tenant-123", cacheKey)
	})

	t.Run("should use 'system' in cache key when source client ID is empty and no default", func(t *testing.T) {
		settingsNoDefault := &azsettings.AzureSettings{
			ManagedIdentityEnabled: true,
		}
		credentials := defaultCredentials()
		credentials.SourceClientId = ""

		result, err := getFederatedIdentityTokenRetriever(settingsNoDefault, credentials)
		require.NoError(t, err)

		cacheKey := result.GetCacheKey("")
		assert.Contains(t, cacheKey, "|system|")
	})

	t.Run("should return nil from GetExpiry", func(t *testing.T) {
		credentials := defaultCredentials()

		result, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.NoError(t, err)

		assert.Nil(t, result.GetExpiry())
	})

	t.Run("should accept US Gov audience", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.FederatedCredentialAudience = "api://AzureADTokenExchangeUSGov"

		_, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.NoError(t, err)
	})

	t.Run("should accept China audience", func(t *testing.T) {
		credentials := defaultCredentials()
		credentials.FederatedCredentialAudience = "api://AzureADTokenExchangeChina"

		_, err := getFederatedIdentityTokenRetriever(settings, credentials)
		require.NoError(t, err)
	})
}
