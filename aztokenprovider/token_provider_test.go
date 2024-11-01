package aztokenprovider

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/naizerjohn-ms/grafana-azure-sdk-go/azcredentials"
	"github.com/naizerjohn-ms/grafana-azure-sdk-go/azsettings"
	"github.com/naizerjohn-ms/grafana-azure-sdk-go/azusercontext"
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

var mockClientSecretCredentials = &azcredentials.AzureClientSecretCredentials{
	AzureCloud:   azsettings.AzurePublic,
	TenantId:     "TEST-TENANT",
	ClientId:     "TEST-CLIENT-ID",
	ClientSecret: "TEST-CLIENT-SECRET",
}

var mockMsiCredentials = &azcredentials.AzureManagedIdentityCredentials{}
var mockWorkloadIdentityCredentials = &azcredentials.AzureWorkloadIdentityCredentials{}

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
	settingsFallbackEnabled := &azsettings.AzureSettings{
		UserIdentityEnabled: true,
		UserIdentityTokenEndpoint: &azsettings.TokenEndpointSettings{
			TokenUrl:     "FAKE_TOKEN_URL",
			ClientId:     "FAKE_CLIENT_ID",
			ClientSecret: "FAKE_CLIENT_SECRET",
		},
		UserIdentityFallbackCredentialsEnabled: true,
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

		provider, err := NewAzureAccessTokenProvider(settings, &azcredentials.AadCurrentUserCredentials{
			ServiceCredentialsEnabled: true,
			ServiceCredentials:        mockClientSecretCredentials,
		}, true)
		require.NoError(t, err)
		require.IsType(t, &userTokenProvider{}, provider)
	})

	t.Run("should error if fallback credentials set to user credentials", func(t *testing.T) {

		_, err := NewAzureAccessTokenProvider(settingsFallbackEnabled, &azcredentials.AadCurrentUserCredentials{
			ServiceCredentialsEnabled: true,
			ServiceCredentials:        &azcredentials.AadCurrentUserCredentials{},
		}, true)
		require.Error(t, err)
		require.ErrorContains(t, err, "user identity authentication not valid for fallback credentials")
	})

	t.Run("should error if fallback credentials set to OBO credentials", func(t *testing.T) {

		_, err := NewAzureAccessTokenProvider(settingsFallbackEnabled, &azcredentials.AadCurrentUserCredentials{
			ServiceCredentialsEnabled: true,
			ServiceCredentials:        &azcredentials.AzureClientSecretOboCredentials{},
		}, true)
		require.Error(t, err)
		require.ErrorContains(t, err, "user identity authentication not valid for fallback credentials")
	})
}

func TestGetAccessToken_UserIdentity(t *testing.T) {
	ctx := context.Background()

	scopes := []string{
		"https://management.azure.com/.default",
	}

	var err error

	t.Run("frontend requests (user in scope)", func(t *testing.T) {
		t.Run("should fail if user context not configured", func(t *testing.T) {
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache: &tokenCacheFake{},
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &onBehalfOfTokenRetriever{}, retriever)
			}

			_, err = provider.GetAccessToken(ctx, scopes)
			assert.Error(t, err)
			assert.ErrorContains(t, err, "user context not configured")
		})

		t.Run("will error if user is not authenticated with Azure AD and ID forwarding enabled", func(t *testing.T) {
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache: &tokenCacheFake{},
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &onBehalfOfTokenRetriever{}, retriever)
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"authenticatedBy": "not_azuread",
			})
			jwtToken, _ := token.SignedString([]byte("test-key")) // 👈

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: &backend.User{
					Login: "user1@example.org",
				},
				IdToken:        "FAKE_ID_TOKEN",
				GrafanaIdToken: jwtToken,
			})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GF_INSTANCE_FEATURE_TOGGLES_ENABLE": "idForwarding",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.Error(t, err)
			require.ErrorContains(t, err, "user is not authenticated with Azure AD")
		})

		t.Run("will assume request is frontend if user != nil and ID forwarding disabled", func(t *testing.T) {
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
			assert.ErrorContains(t, err, "user identity authentication not possible because there's no ID token associated with the Grafana user")
		})

		t.Run("should fail if no username in user context", func(t *testing.T) {
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache: &tokenCacheFake{},
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &onBehalfOfTokenRetriever{}, retriever)
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: &backend.User{
					Login: "",
				},
			})

			_, err = provider.GetAccessToken(usrctx, scopes)
			assert.Error(t, err)
			assert.ErrorContains(t, err, "user identity authentication only possible in context of a Grafana user: request not associated with a Grafana user")
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
	})

	t.Run("backend requests", func(t *testing.T) {
		t.Run("will be treated as a backend request if ID forwarding enabled and ID token is empty", func(t *testing.T) {
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache:     &tokenCacheFake{},
				tokenRetriever: &clientSecretTokenRetriever{},
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &clientSecretTokenRetriever{}, retriever)
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: &backend.User{},
			})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GF_INSTANCE_FEATURE_TOGGLES_ENABLE":                        "idForwarding",
				"GFAZPL_USER_IDENTITY_ENABLED":                              "true",
				"GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED": "true",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.NoError(t, err)
		})

		t.Run("will be treated as a backend request if current user context is empty", func(t *testing.T) {
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache:     &tokenCacheFake{},
				tokenRetriever: &clientSecretTokenRetriever{},
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &clientSecretTokenRetriever{}, retriever)
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GF_INSTANCE_FEATURE_TOGGLES_ENABLE":                        "idForwarding",
				"GFAZPL_USER_IDENTITY_ENABLED":                              "true",
				"GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED": "true",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.NoError(t, err)
		})

		t.Run("will be treated as a backend request if current user is nil", func(t *testing.T) {
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache:     &tokenCacheFake{},
				tokenRetriever: &clientSecretTokenRetriever{},
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &clientSecretTokenRetriever{}, retriever)
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: nil,
			})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GF_INSTANCE_FEATURE_TOGGLES_ENABLE":                        "idForwarding",
				"GFAZPL_USER_IDENTITY_ENABLED":                              "true",
				"GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED": "true",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.NoError(t, err)
		})

		t.Run("will not use fallback credentials if username assertion enabled and fallback credentials enabled", func(t *testing.T) {
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache:        &tokenCacheFake{},
				tokenRetriever:    &clientSecretTokenRetriever{},
				usernameAssertion: true,
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: nil,
			})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GF_INSTANCE_FEATURE_TOGGLES_ENABLE":                        "idForwarding",
				"GFAZPL_USER_IDENTITY_ENABLED":                              "true",
				"GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED": "true",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.Error(t, err)
			require.ErrorContains(t, err, "fallback credentials not enabled")
		})

		t.Run("will not use fallback credentials if disabled", func(t *testing.T) {
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache:     &tokenCacheFake{},
				tokenRetriever: &clientSecretTokenRetriever{},
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: nil,
			})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GF_INSTANCE_FEATURE_TOGGLES_ENABLE":                        "idForwarding",
				"GFAZPL_USER_IDENTITY_ENABLED":                              "true",
				"GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED": "false",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.Error(t, err)
			require.ErrorContains(t, err, "fallback credentials not enabled")
		})

		t.Run("should use clientSecretTokenRetriever when service principal credentials are enabled", func(t *testing.T) {
			tokenRetriever, _ := getClientSecretTokenRetriever(&azsettings.AzureSettings{UserIdentityFallbackCredentialsEnabled: true}, mockClientSecretCredentials)
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache:     &tokenCacheFake{},
				tokenRetriever: tokenRetriever,
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &clientSecretTokenRetriever{}, retriever)
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: nil,
			})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GFAZPL_USER_IDENTITY_ENABLED":                              "true",
				"GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED": "true",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.NoError(t, err)
		})

		t.Run("should use msiTokenRetriever when service principal credentials are enabled", func(t *testing.T) {
			tokenRetriever := getManagedIdentityTokenRetriever(&azsettings.AzureSettings{UserIdentityFallbackCredentialsEnabled: true, ManagedIdentityEnabled: true, ManagedIdentityClientId: "test-msi"}, mockMsiCredentials)
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache:     &tokenCacheFake{},
				tokenRetriever: tokenRetriever,
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &managedIdentityTokenRetriever{}, retriever)
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: nil,
			})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GFAZPL_USER_IDENTITY_ENABLED":                              "true",
				"GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED": "true",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.NoError(t, err)
		})

		t.Run("should use workloadIdentityTokenRetriever when service principal credentials are enabled", func(t *testing.T) {
			tokenRetriever := getWorkloadIdentityTokenRetriever(&azsettings.AzureSettings{UserIdentityFallbackCredentialsEnabled: true, WorkloadIdentityEnabled: true, WorkloadIdentitySettings: &azsettings.WorkloadIdentitySettings{
				TenantId:  "test-tenant-id",
				ClientId:  "test-client-id",
				TokenFile: "test-token-file",
			}}, mockWorkloadIdentityCredentials)
			var provider AzureTokenProvider = &userTokenProvider{
				tokenCache:     &tokenCacheFake{},
				tokenRetriever: tokenRetriever,
			}

			getAccessTokenFunc = func(retriever TokenRetriever, scopes []string) {
				assert.IsType(t, &workloadIdentityTokenRetriever{}, retriever)
			}

			usrctx := azusercontext.WithCurrentUser(ctx, azusercontext.CurrentUserContext{
				User: nil,
			})
			settingsctx := backend.WithGrafanaConfig(usrctx, backend.NewGrafanaCfg(map[string]string{
				"GFAZPL_USER_IDENTITY_ENABLED":                              "true",
				"GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED": "true",
			}))

			_, err = provider.GetAccessToken(settingsctx, scopes)
			require.NoError(t, err)
		})

	})

}
