package azsettings

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromEnv(t *testing.T) {
	t.Run("should set cloud if variable is set", func(t *testing.T) {
		unset, err := setEnvVar("GFAZPL_AZURE_CLOUD", "TestCloud")
		require.NoError(t, err)
		defer unset()

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, "TestCloud", azureSettings.Cloud)
	})

	t.Run("should set azureAuthEnabled if variable is set", func(t *testing.T) {
		unset, err := setEnvVar("GFAZPL_AZURE_AUTH_ENABLED", "true")
		require.NoError(t, err)
		defer unset()

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, true, azureSettings.AzureAuthEnabled)
	})

	t.Run("should set cloud if fallback variable is set", func(t *testing.T) {
		unset1, err := setEnvVar("GFAZPL_AZURE_CLOUD", "")
		require.NoError(t, err)
		defer unset1()
		unset2, err := setEnvVar("AZURE_CLOUD", "FallbackCloud")
		require.NoError(t, err)
		defer unset2()

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, "FallbackCloud", azureSettings.Cloud)
	})

	t.Run("should set cloud to public cloud if variable is not set", func(t *testing.T) {
		unset1, err := setEnvVar("GFAZPL_AZURE_CLOUD", "")
		require.NoError(t, err)
		defer unset1()
		unset2, err := setEnvVar("AZURE_CLOUD", "")
		require.NoError(t, err)
		defer unset2()

		azureSettings, err := ReadFromEnv()
		require.NoError(t, err)

		assert.Equal(t, AzurePublic, azureSettings.Cloud)
	})

	t.Run("managed identity", func(t *testing.T) {
		t.Run("should enable managed identity if variable is set", func(t *testing.T) {
			unset, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_ENABLED", "true")
			require.NoError(t, err)
			defer unset()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.True(t, azureSettings.ManagedIdentityEnabled)
		})

		t.Run("should enable managed identity if fallback variable is set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_ENABLED", "")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("AZURE_MANAGED_IDENTITY_ENABLED", "true")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.True(t, azureSettings.ManagedIdentityEnabled)
		})

		t.Run("should disable managed identity if variable is not set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_ENABLED", "")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("AZURE_MANAGED_IDENTITY_ENABLED", "")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.False(t, azureSettings.ManagedIdentityEnabled)
		})

		t.Run("should set client ID if variable is set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_ENABLED", "true")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_CLIENT_ID", "TestClientId")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.Equal(t, "TestClientId", azureSettings.ManagedIdentityClientId)
		})

		t.Run("should set client ID if fallback variable is set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_ENABLED", "true")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_CLIENT_ID", "")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("AZURE_MANAGED_IDENTITY_CLIENT_ID", "FallbackClientId")
			require.NoError(t, err)
			defer unset3()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.Equal(t, "FallbackClientId", azureSettings.ManagedIdentityClientId)
		})

		t.Run("should not set client ID if managed identity is not enabled", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_ENABLED", "false")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_MANAGED_IDENTITY_CLIENT_ID", "TestClientId")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.Equal(t, "", azureSettings.ManagedIdentityClientId)
		})
	})

	t.Run("workload identity", func(t *testing.T) {
		t.Run("should enable workload identity if variable is set", func(t *testing.T) {
			unset, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_ENABLED", "true")
			require.NoError(t, err)
			defer unset()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.True(t, azureSettings.WorkloadIdentityEnabled)
		})

		t.Run("should disable workload identity if variable is not set", func(t *testing.T) {
			unset, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_ENABLED", "")
			require.NoError(t, err)
			defer unset()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.False(t, azureSettings.WorkloadIdentityEnabled)
		})

		t.Run("should set tenant ID if variable is set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_ENABLED", "true")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_TENANT_ID", "ba556b7e")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.NotNil(t, azureSettings.WorkloadIdentitySettings)
			assert.Equal(t, "ba556b7e", azureSettings.WorkloadIdentitySettings.TenantId)
		})

		t.Run("should not set tenant ID if workload identity is not enabled", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_ENABLED", "false")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_TENANT_ID", "ba556b7e")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			if azureSettings.WorkloadIdentitySettings != nil {
				assert.Equal(t, "", azureSettings.WorkloadIdentitySettings.TenantId)
			}
		})

		t.Run("should set client ID if variable is set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_ENABLED", "true")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_CLIENT_ID", "547121e7")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.NotNil(t, azureSettings.WorkloadIdentitySettings)
			assert.Equal(t, "547121e7", azureSettings.WorkloadIdentitySettings.ClientId)
		})

		t.Run("should not set client ID if workload identity is not enabled", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_ENABLED", "false")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_CLIENT_ID", "547121e7")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			if azureSettings.WorkloadIdentitySettings != nil {
				assert.Equal(t, "", azureSettings.WorkloadIdentitySettings.ClientId)
			}
		})

		t.Run("should set token file if variable is set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_ENABLED", "true")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_TOKEN_FILE", "/var/test-token")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.NotNil(t, azureSettings.WorkloadIdentitySettings)
			assert.Equal(t, "/var/test-token", azureSettings.WorkloadIdentitySettings.TokenFile)
		})

		t.Run("should not set token file if workload identity is not enabled", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_ENABLED", "false")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_WORKLOAD_IDENTITY_TOKEN_FILE", "/var/test-token")
			require.NoError(t, err)
			defer unset2()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			if azureSettings.WorkloadIdentitySettings != nil {
				assert.Equal(t, "", azureSettings.WorkloadIdentitySettings.TokenFile)
			}
		})
	})

	t.Run("when user identity enabled", func(t *testing.T) {
		unset, err := setEnvVar("GFAZPL_USER_IDENTITY_ENABLED", "true")
		require.NoError(t, err)
		defer unset()

		t.Run("should fail if user token URL isn't set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "client_secret_post")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "f85aa887-490d-4fac-9306-9b99ad0aa31d")
			require.NoError(t, err)
			defer unset3()

			_, err = ReadFromEnv()
			assert.Error(t, err)
		})

		t.Run("should fail if client authentication isn't set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "f85aa887-490d-4fac-9306-9b99ad0aa31d")
			require.NoError(t, err)
			defer unset3()

			_, err = ReadFromEnv()
			assert.Error(t, err)
		})

		t.Run("should fail if client ID isn't set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "client_secret_post")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "")
			require.NoError(t, err)
			defer unset3()

			_, err = ReadFromEnv()
			assert.Error(t, err)
		})

		t.Run("should be enabled and endpoint settings initialized with token URL, client authentication, and client ID", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "client_secret_post")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "f85aa887-490d-4fac-9306-9b99ad0aa31d")
			require.NoError(t, err)
			defer unset3()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.True(t, azureSettings.UserIdentityEnabled)

			require.NotNil(t, azureSettings.UserIdentityTokenEndpoint)
			assert.Equal(t, "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token", azureSettings.UserIdentityTokenEndpoint.TokenUrl)
			assert.Equal(t, "client_secret_post", azureSettings.UserIdentityTokenEndpoint.ClientAuthentication)
			assert.Equal(t, "f85aa887-490d-4fac-9306-9b99ad0aa31d", azureSettings.UserIdentityTokenEndpoint.ClientId)
		})

		t.Run("should initialize endpoint settings with client secret if client secret is set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "client_secret_post")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "f85aa887-490d-4fac-9306-9b99ad0aa31d")
			require.NoError(t, err)
			defer unset3()
			unset4, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_SECRET", "87808761-ff7b-492e-bb0d-5de2437ffa55")
			require.NoError(t, err)
			defer unset4()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			require.NotNil(t, azureSettings.UserIdentityTokenEndpoint)
			assert.Equal(t, "87808761-ff7b-492e-bb0d-5de2437ffa55", azureSettings.UserIdentityTokenEndpoint.ClientSecret)
		})

		t.Run("should initialize endpoint settings with managed identity client ID and federated credential audience if managed identity client ID and federated credential audience are set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "managed_identity")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "f85aa887-490d-4fac-9306-9b99ad0aa31d")
			require.NoError(t, err)
			defer unset3()
			unset4, err := setEnvVar("GFAZPL_USER_IDENTITY_MANAGED_IDENTITY_CLIENT_ID", "87808761-ff7b-492e-bb0d-5de2437ffa55")
			require.NoError(t, err)
			defer unset4()
			unset5, err := setEnvVar("GFAZPL_USER_IDENTITY_FEDERATED_CREDENTIAL_AUDIENCE", "api://AzureADTokenExchange")
			require.NoError(t, err)
			defer unset5()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			require.NotNil(t, azureSettings.UserIdentityTokenEndpoint)
			assert.Equal(t, "87808761-ff7b-492e-bb0d-5de2437ffa55", azureSettings.UserIdentityTokenEndpoint.ManagedIdentityClientId)
			assert.Equal(t, "api://AzureADTokenExchange", azureSettings.UserIdentityTokenEndpoint.FederatedCredentialAudience)
		})

		t.Run("should be enabled and default to enabling service credentials", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "client_secret_post")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "f85aa887-490d-4fac-9306-9b99ad0aa31d")
			require.NoError(t, err)
			defer unset3()
			unset4, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_SECRET", "87808761-ff7b-492e-bb0d-5de2437ffa55")
			require.NoError(t, err)
			defer unset4()
			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.True(t, azureSettings.UserIdentityEnabled)
			assert.True(t, azureSettings.UserIdentityFallbackCredentialsEnabled)

		})

		t.Run("should be enabled and disable service credentials", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "client_secret_post")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "f85aa887-490d-4fac-9306-9b99ad0aa31d")
			require.NoError(t, err)
			defer unset3()
			unset4, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_SECRET", "87808761-ff7b-492e-bb0d-5de2437ffa55")
			require.NoError(t, err)
			defer unset4()
			unset5, err := setEnvVar("GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED", "false")
			require.NoError(t, err)
			defer unset5()
			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.True(t, azureSettings.UserIdentityEnabled)
			assert.False(t, azureSettings.UserIdentityFallbackCredentialsEnabled)

		})
	})

	t.Run("when user identity disabled", func(t *testing.T) {
		unset, err := setEnvVar("GFAZPL_USER_IDENTITY_ENABLED", "false")
		require.NoError(t, err)
		defer unset()

		t.Run("should be disabled and endpoint settings should be nil even when token URL, client authentication, and client ID are set", func(t *testing.T) {
			unset1, err := setEnvVar("GFAZPL_USER_IDENTITY_TOKEN_URL", "https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token")
			require.NoError(t, err)
			defer unset1()
			unset2, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION", "client_secret_post")
			require.NoError(t, err)
			defer unset2()
			unset3, err := setEnvVar("GFAZPL_USER_IDENTITY_CLIENT_ID", "f85aa887-490d-4fac-9306-9b99ad0aa31d")
			require.NoError(t, err)
			defer unset3()

			azureSettings, err := ReadFromEnv()
			require.NoError(t, err)

			assert.False(t, azureSettings.UserIdentityEnabled)

			require.Nil(t, azureSettings.UserIdentityTokenEndpoint)
		})
	})
}

func TestWriteToEnvStr(t *testing.T) {
	defer func() {
		_ = os.Unsetenv(AzureCloud)
		_ = os.Unsetenv(ManagedIdentityEnabled)
		_ = os.Unsetenv(ManagedIdentityClientID)
	}()

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

	t.Run("should return azureAuthEnabled if set", func(t *testing.T) {
		azureSettings := &AzureSettings{
			AzureAuthEnabled: true,
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 1)
		assert.Equal(t, "GFAZPL_AZURE_AUTH_ENABLED=true", envs[0])
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

	t.Run("should return workload identity set if enabled", func(t *testing.T) {
		azureSettings := &AzureSettings{
			WorkloadIdentityEnabled: true,
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 1)
		assert.Equal(t, "GFAZPL_WORKLOAD_IDENTITY_ENABLED=true", envs[0])
	})

	t.Run("should return workload identity tenant ID if provided", func(t *testing.T) {
		azureSettings := &AzureSettings{
			WorkloadIdentityEnabled: true,
			WorkloadIdentitySettings: &WorkloadIdentitySettings{
				TenantId: "ba556b7e",
			},
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 2)
		assert.Equal(t, "GFAZPL_WORKLOAD_IDENTITY_ENABLED=true", envs[0])
		assert.Equal(t, "GFAZPL_WORKLOAD_IDENTITY_TENANT_ID=ba556b7e", envs[1])
	})

	t.Run("should return workload identity client ID if provided", func(t *testing.T) {
		azureSettings := &AzureSettings{
			WorkloadIdentityEnabled: true,
			WorkloadIdentitySettings: &WorkloadIdentitySettings{
				ClientId: "547121e7",
			},
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 2)
		assert.Equal(t, "GFAZPL_WORKLOAD_IDENTITY_ENABLED=true", envs[0])
		assert.Equal(t, "GFAZPL_WORKLOAD_IDENTITY_CLIENT_ID=547121e7", envs[1])
	})

	t.Run("should return workload identity token file if provided", func(t *testing.T) {
		azureSettings := &AzureSettings{
			WorkloadIdentityEnabled: true,
			WorkloadIdentitySettings: &WorkloadIdentitySettings{
				TokenFile: "/var/test-token",
			},
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 2)
		assert.Equal(t, "GFAZPL_WORKLOAD_IDENTITY_ENABLED=true", envs[0])
		assert.Equal(t, "GFAZPL_WORKLOAD_IDENTITY_TOKEN_FILE=/var/test-token", envs[1])
	})

	t.Run("should return user identity set if enabled", func(t *testing.T) {
		azureSettings := &AzureSettings{
			UserIdentityEnabled: true,
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 2)
		assert.Equal(t, "GFAZPL_USER_IDENTITY_ENABLED=true", envs[0])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED=false", envs[1])
	})

	t.Run("should return user identity set if enabled with service credentials enabled", func(t *testing.T) {
		azureSettings := &AzureSettings{
			UserIdentityEnabled:                    true,
			UserIdentityFallbackCredentialsEnabled: true,
		}

		envs := WriteToEnvStr(azureSettings)

		require.Len(t, envs, 2)
		assert.Equal(t, "GFAZPL_USER_IDENTITY_ENABLED=true", envs[0])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED=true", envs[1])
	})

	t.Run("should return user identity endpoint settings if user identity enabled", func(t *testing.T) {
		azureSettings := &AzureSettings{
			UserIdentityEnabled: true,
			UserIdentityTokenEndpoint: &TokenEndpointSettings{
				TokenUrl:     					"https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token",
				ClientAuthentication: 			"client_secret_post",
				ClientId:     					"f85aa887-490d-4fac-9306-9b99ad0aa31d",
				ClientSecret: 					"87808761-ff7b-492e-bb0d-5de2437ffa55",
				ManagedIdentityClientId: 		"50dbf8ad-5af9-40b8-ac8e-1a451ee30f6d",
				FederatedCredentialAudience: 	"api://AzureADTokenExchange",
			},
		}

		envs := WriteToEnvStr(azureSettings)

		assert.Len(t, envs, 8)
		assert.Equal(t, "GFAZPL_USER_IDENTITY_ENABLED=true", envs[0])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_FALLBACK_SERVICE_CREDENTIALS_ENABLED=false", envs[1])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_TOKEN_URL=https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token", envs[2])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_CLIENT_AUTHENTICATION=client_secret_post", envs[3])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_CLIENT_ID=f85aa887-490d-4fac-9306-9b99ad0aa31d", envs[4])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_CLIENT_SECRET=87808761-ff7b-492e-bb0d-5de2437ffa55", envs[5])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_MANAGED_IDENTITY_CLIENT_ID=50dbf8ad-5af9-40b8-ac8e-1a451ee30f6d", envs[6])
		assert.Equal(t, "GFAZPL_USER_IDENTITY_FEDERATED_CREDENTIAL_AUDIENCE=api://AzureADTokenExchange", envs[7])
	})

	t.Run("should not return user identity endpoint settings if user identity not enabled", func(t *testing.T) {
		azureSettings := &AzureSettings{
			UserIdentityEnabled: false,
			UserIdentityTokenEndpoint: &TokenEndpointSettings{
				TokenUrl:     					"https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token",
				ClientAuthentication: 			"client_secret_post",
				ClientId:     					"f85aa887-490d-4fac-9306-9b99ad0aa31d",
				ClientSecret: 					"87808761-ff7b-492e-bb0d-5de2437ffa55",
				ManagedIdentityClientId: 		"50dbf8ad-5af9-40b8-ac8e-1a451ee30f6d",
				FederatedCredentialAudience: 	"api://AzureADTokenExchange",
			},
		}

		envs := WriteToEnvStr(azureSettings)

		assert.Len(t, envs, 0)
	})

	t.Run("should return assertion if username assertion is set", func(t *testing.T) {
		azureSettings := &AzureSettings{
			UserIdentityEnabled: true,
			UserIdentityTokenEndpoint: &TokenEndpointSettings{
				TokenUrl:     					"https://login.microsoftonline.com/fd719c11-a91c-40fd-8379-1e6cd3c59568/oauth2/v2.0/token",
				ClientAuthentication: 			"client_secret_post",
				ClientId:     					"f85aa887-490d-4fac-9306-9b99ad0aa31d",
				ClientSecret: 					"87808761-ff7b-492e-bb0d-5de2437ffa55",
				ManagedIdentityClientId: 		"50dbf8ad-5af9-40b8-ac8e-1a451ee30f6d",
				FederatedCredentialAudience: 	"api://AzureADTokenExchange",
				UsernameAssertion: true,
			},
		}

		envs := WriteToEnvStr(azureSettings)

		assert.Contains(t, envs, "GFAZPL_USER_IDENTITY_ASSERTION=username")
	})
}

type unsetFunc = func()

func setEnvVar(key string, value string) (unsetFunc, error) {
	err := os.Setenv(key, value)
	if err != nil {
		return nil, err
	}

	return func() {
		_ = os.Unsetenv(key)
	}, nil
}
