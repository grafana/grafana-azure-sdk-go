package aztokenprovider

import (
	"context"
	"testing"

	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/azsettings"
	"github.com/grafana/grafana-azure-sdk-go/azusercontext"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var getAccessTokenFunc func(credential TokenRetriever, scopes []string)

type tokenCacheFake struct{}

func (c *tokenCacheFake) GetAccessToken(_ context.Context, credential TokenRetriever, scopes []string) (string, error) {
	getAccessTokenFunc(credential, scopes)
	return "4cb83b87-0ffb-4abd-82f6-48a8c08afc53", nil
}

func TestNewAzureAccessTokenProvider_ServiceIdentity(t *testing.T) {
	ctx := context.Background()

	settings := &azsettings.AzureSettings{}

	scopes := []string{
		"https://management.azure.com/.default",
	}

	original := azureTokenCache
	azureTokenCache = &tokenCacheFake{}
	t.Cleanup(func() { azureTokenCache = original })

	t.Run("when managed identities enabled", func(t *testing.T) {
		settings.ManagedIdentityEnabled = true

		t.Run("should resolve managed identity retriever if auth type is managed identity", func(t *testing.T) {
			credentials := &azcredentials.AzureManagedIdentityCredentials{}

			provider, err := NewAzureAccessTokenProvider(settings, credentials, false)
			require.NoError(t, err)
			require.IsType(t, &serviceTokenProvider{}, provider)

			getAccessTokenFunc = func(credential TokenRetriever, scopes []string) {
				assert.IsType(t, &managedIdentityTokenRetriever{}, credential)
			}

			_, err = provider.GetAccessToken(ctx, scopes)
			require.NoError(t, err)
		})
	})

	t.Run("when managed identities disabled", func(t *testing.T) {
		settings.ManagedIdentityEnabled = false

		t.Run("should return error if auth type is managed identity", func(t *testing.T) {
			credentials := &azcredentials.AzureManagedIdentityCredentials{}

			_, err := NewAzureAccessTokenProvider(settings, credentials, false)
			assert.Error(t, err, "managed identity authentication is not enabled in Grafana config")
		})
	})

	t.Run("when workload identities enabled", func(t *testing.T) {
		settings.WorkloadIdentityEnabled = true

		t.Run("should resolve workload identity retriever if auth type is workload identity", func(t *testing.T) {
			credentials := &azcredentials.AzureWorkloadIdentityCredentials{}

			provider, err := NewAzureAccessTokenProvider(settings, credentials, false)
			require.NoError(t, err)
			require.IsType(t, &serviceTokenProvider{}, provider)

			getAccessTokenFunc = func(credential TokenRetriever, scopes []string) {
				assert.IsType(t, &workloadIdentityTokenRetriever{}, credential)
			}

			_, err = provider.GetAccessToken(ctx, scopes)
			require.NoError(t, err)
		})
	})

	t.Run("when workload identities disabled", func(t *testing.T) {
		settings.WorkloadIdentityEnabled = false

		t.Run("should return error if auth type is workload identity", func(t *testing.T) {
			credentials := &azcredentials.AzureWorkloadIdentityCredentials{}

			_, err := NewAzureAccessTokenProvider(settings, credentials, false)
			assert.Error(t, err, "workload identity authentication is not enabled in Grafana config")
		})
	})

	t.Run("should resolve client secret retriever if auth type is client secret", func(t *testing.T) {
		credentials := &azcredentials.AzureClientSecretCredentials{AzureCloud: azsettings.AzurePublic}

		provider, err := NewAzureAccessTokenProvider(settings, credentials, false)
		require.NoError(t, err)
		require.IsType(t, &serviceTokenProvider{}, provider)

		getAccessTokenFunc = func(credential TokenRetriever, scopes []string) {
			assert.IsType(t, &clientSecretTokenRetriever{}, credential)
		}

		_, err = provider.GetAccessToken(ctx, scopes)
		require.NoError(t, err)
	})
}

var mockUserCredentials = &azcredentials.AadCurrentUserCredentials{
	ServicePrincipal: azcredentials.AzureClientSecretCredentials{
		AzureCloud:   azsettings.AzurePublic,
		TenantId:     "TEST-TENANT",
		ClientId:     "TEST-CLIENT-ID",
		ClientSecret: "TEST-CLIENT-SECRET",
	},
}

func TestNewAzureAccessTokenProvider_UserIdentity(t *testing.T) {
	settingsNotConfigured := &azsettings.AzureSettings{}

	settings := &azsettings.AzureSettings{
		UserIdentityEnabled: true,
		UserIdentityTokenEndpoint: &azsettings.TokenEndpointSettings{
			TokenUrl:     "FAKE_TOKEN_URL",
			ClientId:     "FAKE_CLIENT_ID",
			ClientSecret: "FAKE_CLIENT_SECRET",
		},
	}

	t.Run("should fail when user identity not supported", func(t *testing.T) {
		credentials := &azcredentials.AadCurrentUserCredentials{}

		_, err := NewAzureAccessTokenProvider(settingsNotConfigured, credentials, false)
		assert.Error(t, err)
	})

	t.Run("should fail when user identity not configured", func(t *testing.T) {
		credentials := &azcredentials.AadCurrentUserCredentials{}

		_, err := NewAzureAccessTokenProvider(settingsNotConfigured, credentials, true)
		assert.Error(t, err)
	})

	t.Run("should return user provider when user identity configured", func(t *testing.T) {
		credentials := &azcredentials.AadCurrentUserCredentials{}

		provider, err := NewAzureAccessTokenProvider(settings, credentials, true)
		require.NoError(t, err)
		require.IsType(t, &userTokenProvider{}, provider)
	})

	t.Run("should return user provider with service principal credentials when user identity configured", func(t *testing.T) {

		provider, err := NewAzureAccessTokenProvider(settings, mockUserCredentials, true)
		require.NoError(t, err)
		require.IsType(t, &userTokenProvider{}, provider)
	})
}

func TestGetAccessToken_UserIdentity(t *testing.T) {
	ctx := context.Background()

	scopes := []string{
		"https://management.azure.com/.default",
	}

	var err error

	t.Run("should fail if user context not configured", func(t *testing.T) {
		var provider AzureTokenProvider = &userTokenProvider{
			tokenCache: &tokenCacheFake{},
		}

		getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
			assert.IsType(t, &onBehalfOfTokenRetriever{}, retriever)
		}

		_, err = provider.GetAccessToken(ctx, scopes)
		assert.Error(t, err)
	})

	t.Run("should fail if no user in user context", func(t *testing.T) {
		var provider AzureTokenProvider = &userTokenProvider{
			tokenCache: &tokenCacheFake{},
		}

		getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
			assert.IsType(t, &onBehalfOfTokenRetriever{}, retriever)
		}

		usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{})

		_, err = provider.GetAccessToken(usrctx, scopes)
		assert.Error(t, err)
	})

	t.Run("should fail if no ID token in user context", func(t *testing.T) {
		var provider AzureTokenProvider = &userTokenProvider{
			tokenCache: &tokenCacheFake{},
		}

		getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
			assert.IsType(t, &onBehalfOfTokenRetriever{}, retriever)
		}

		usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
			User: &backend.User{
				Login: "user1@example.org",
			},
		})

		_, err = provider.GetAccessToken(usrctx, scopes)
		assert.Error(t, err)
	})

	t.Run("should use onBehalfOfTokenRetriever by default", func(t *testing.T) {
		var provider AzureTokenProvider = &userTokenProvider{
			tokenCache: &tokenCacheFake{},
		}

		getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
			assert.IsType(t, &onBehalfOfTokenRetriever{}, retriever)
		}

		usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
			User: &backend.User{
				Login: "user1@example.org",
			},
			IdToken: "FAKE_ID_TOKEN",
		})

		_, err = provider.GetAccessToken(usrctx, scopes)
		require.NoError(t, err)
	})

	t.Run("should use usernameTokenRetriever for usernameAssertion", func(t *testing.T) {
		var provider AzureTokenProvider = &userTokenProvider{
			tokenCache:        &tokenCacheFake{},
			usernameAssertion: true,
		}

		getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
			assert.IsType(t, &usernameTokenRetriever{}, retriever)
		}

		usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
			User: &backend.User{
				Login: "user1@example.org",
			},
		})

		_, err = provider.GetAccessToken(usrctx, scopes)
		require.NoError(t, err)
	})

	t.Run("should use clientSecretTokenRetriever when service principal credentials are available without an access token or id token", func(t *testing.T) {

		tokenRetriever, _ := getClientSecretTokenRetriever(&mockUserCredentials.ServicePrincipal)
		var provider AzureTokenProvider = &userTokenProvider{
			tokenCache:     &tokenCacheFake{},
			tokenRetriever: tokenRetriever,
		}

		getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
			assert.IsType(t, &clientSecretTokenRetriever{}, retriever)
		}

		usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
			User: &backend.User{},
		})

		_, err = provider.GetAccessToken(usrctx, scopes)
		require.NoError(t, err)
	})

	t.Run("should use clientSecretTokenRetriever when service principal credentials are available without a user in context", func(t *testing.T) {

		tokenRetriever, _ := getClientSecretTokenRetriever(&mockUserCredentials.ServicePrincipal)
		var provider AzureTokenProvider = &userTokenProvider{
			tokenCache:     &tokenCacheFake{},
			tokenRetriever: tokenRetriever,
		}

		getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
			assert.IsType(t, &clientSecretTokenRetriever{}, retriever)
		}

		_, err = provider.GetAccessToken(ctx, scopes)
		require.NoError(t, err)
	})
}
