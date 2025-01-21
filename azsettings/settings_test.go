package azsettings

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestSettingsFromContext(t *testing.T) {
	t.Run("ReadFromContext", func(t *testing.T) {
		tcs := []struct {
			name                string
			cfg                 *backend.GrafanaCfg
			expectedHasSettings bool
			expectedAzure       *AzureSettings
		}{
			{
				name:                "nil config",
				cfg:                 nil,
				expectedAzure:       &AzureSettings{},
				expectedHasSettings: false,
			},
			{
				name:                "empty config",
				cfg:                 &backend.GrafanaCfg{},
				expectedAzure:       &AzureSettings{},
				expectedHasSettings: false,
			},
			{
				name:                "nil config map",
				cfg:                 backend.NewGrafanaCfg(nil),
				expectedAzure:       &AzureSettings{},
				expectedHasSettings: false,
			},
			{
				name:                "empty config map",
				cfg:                 backend.NewGrafanaCfg(make(map[string]string)),
				expectedAzure:       &AzureSettings{},
				expectedHasSettings: false,
			},
			{
				name: "azure settings in config",
				cfg: backend.NewGrafanaCfg(map[string]string{
					AzureCloud:                              AzurePublic,
					AzureAuthEnabled:                        "true",
					ManagedIdentityEnabled:                  "true",
					ManagedIdentityClientID:                 "mock_managed_identity_client_id",
					UserIdentityEnabled:                     "true",
					UserIdentityClientAuthentication:        "mock_user_identity_client_authentication",
					UserIdentityClientID:                    "mock_user_identity_client_id",
					UserIdentityClientSecret:                "mock_user_identity_client_secret",
					UserIdentityManagedIdentityClientID:     "mock_user_identity_managed_identity_client_id",
					UserIdentityFederatedCredentialAudience: "mock_user_identity_federated_credential_audience",
					UserIdentityTokenURL:                    "mock_user_identity_token_url",
					UserIdentityAssertion:                   "username",
					UserIdentityFallbackCredentialsEnabled:  "true",
					WorkloadIdentityEnabled:                 "true",
					WorkloadIdentityClientID:                "mock_workload_identity_client_id",
					WorkloadIdentityTenantID:                "mock_workload_identity_tenant_id",
					WorkloadIdentityTokenFile:               "mock_workload_identity_token_file",
				}),
				expectedAzure: &AzureSettings{
					Cloud:                                  AzurePublic,
					AzureAuthEnabled:                       true,
					ManagedIdentityEnabled:                 true,
					ManagedIdentityClientId:                "mock_managed_identity_client_id",
					UserIdentityEnabled:                    true,
					UserIdentityFallbackCredentialsEnabled: true,
					UserIdentityTokenEndpoint: &TokenEndpointSettings{
						ClientAuthentication:        "mock_user_identity_client_authentication",
						ClientId:                    "mock_user_identity_client_id",
						ClientSecret:                "mock_user_identity_client_secret",
						ManagedIdentityClientId:     "mock_user_identity_managed_identity_client_id",
						FederatedCredentialAudience: "mock_user_identity_federated_credential_audience",
						TokenUrl:                    "mock_user_identity_token_url",
						UsernameAssertion:           true,
					},
					WorkloadIdentityEnabled: true,
					WorkloadIdentitySettings: &WorkloadIdentitySettings{
						ClientId:  "mock_workload_identity_client_id",
						TenantId:  "mock_workload_identity_tenant_id",
						TokenFile: "mock_workload_identity_token_file",
					},
				},
				expectedHasSettings: true,
			},
		}

		for _, tc := range tcs {
			ctx := backend.WithGrafanaConfig(context.Background(), tc.cfg)
			settings, hasSettings := ReadFromContext(ctx)

			require.Equal(t, tc.expectedAzure, settings)
			require.Equal(t, tc.expectedHasSettings, hasSettings)
		}
	})
}

func TestReadSettings(t *testing.T) {
	expectedAzureContextSettings := &AzureSettings{
		Cloud:                                  AzurePublic,
		ManagedIdentityEnabled:                 true,
		ManagedIdentityClientId:                "mock_managed_identity_client_id",
		UserIdentityEnabled:                    true,
		UserIdentityFallbackCredentialsEnabled: false,
		UserIdentityTokenEndpoint: &TokenEndpointSettings{
			ClientAuthentication:        "mock_user_identity_client_authentication",
			ClientId:                    "mock_user_identity_client_id",
			ClientSecret:                "mock_user_identity_client_secret",
			ManagedIdentityClientId:     "mock_user_identity_managed_identity_client_id",
			FederatedCredentialAudience: "mock_user_identity_federated_credential_audience",
			TokenUrl:                    "mock_user_identity_token_url",
			UsernameAssertion:           true,
		},
		WorkloadIdentityEnabled: true,
		WorkloadIdentitySettings: &WorkloadIdentitySettings{
			ClientId:  "mock_workload_identity_client_id",
			TenantId:  "mock_workload_identity_tenant_id",
			TokenFile: "mock_workload_identity_token_file",
		},
	}

	expectedAzureEnvSettings := &AzureSettings{
		Cloud:                                  "ENV_CLOUD",
		ManagedIdentityEnabled:                 true,
		ManagedIdentityClientId:                "ENV_MI_CLIENT_ID",
		UserIdentityEnabled:                    true,
		UserIdentityFallbackCredentialsEnabled: false,
		UserIdentityTokenEndpoint: &TokenEndpointSettings{
			ClientAuthentication:        "ENV_UI_CLIENT_AUTHENTICATION",
			ClientId:                    "ENV_UI_CLIENT_ID",
			ClientSecret:                "ENV_UI_CLIENT_SECRET",
			ManagedIdentityClientId:     "ENV_UI_MANAGED_IDENTITY_CLIENT_ID",
			FederatedCredentialAudience: "ENV_UI_FEDERATED_CREDENTIAL_AUDIENCE",
			TokenUrl:                    "ENV_UI_TOKEN_URL",
			UsernameAssertion:           true,
		},
		WorkloadIdentityEnabled: true,
		WorkloadIdentitySettings: &WorkloadIdentitySettings{
			ClientId:  "ENV_WI_CLIENT_ID",
			TenantId:  "ENV_WI_TENANT_ID",
			TokenFile: "ENV_WI_TOKEN_FILE",
		},
	}

	unsetCloud, _ := setEnvVar(AzureCloud, "ENV_CLOUD")
	defer unsetCloud()
	unsetMIEnabled, _ := setEnvVar(ManagedIdentityEnabled, "true")
	defer unsetMIEnabled()
	unsetMIClientID, _ := setEnvVar(ManagedIdentityClientID, "ENV_MI_CLIENT_ID")
	defer unsetMIClientID()
	unsetUIEnabled, _ := setEnvVar(UserIdentityEnabled, "true")
	defer unsetUIEnabled()
	unsetUIClientAuthentication, _ := setEnvVar(UserIdentityClientAuthentication, "ENV_UI_CLIENT_AUTHENTICATION")
	defer unsetUIClientAuthentication()
	unsetUIClientID, _ := setEnvVar(UserIdentityClientID, "ENV_UI_CLIENT_ID")
	defer unsetUIClientID()
	unsetUIClientSecret, _ := setEnvVar(UserIdentityClientSecret, "ENV_UI_CLIENT_SECRET")
	defer unsetUIClientSecret()
	unsetUIManagedIdentityClientID, _ := setEnvVar(UserIdentityManagedIdentityClientID, "ENV_UI_MANAGED_IDENTITY_CLIENT_ID")
	defer unsetUIManagedIdentityClientID()
	unsetUIFederatedCredentialAudience, _ := setEnvVar(UserIdentityFederatedCredentialAudience, "ENV_UI_FEDERATED_CREDENTIAL_AUDIENCE")
	defer unsetUIFederatedCredentialAudience()
	unsetUITokenURL, _ := setEnvVar(UserIdentityTokenURL, "ENV_UI_TOKEN_URL")
	defer unsetUITokenURL()
	unsetUIAssertion, _ := setEnvVar(UserIdentityAssertion, "username")
	defer unsetUIAssertion()
	unsetUIServiceCredentials, _ := setEnvVar(UserIdentityFallbackCredentialsEnabled, "false")
	defer unsetUIServiceCredentials()
	unsetWIEnabled, _ := setEnvVar(WorkloadIdentityEnabled, "true")
	defer unsetWIEnabled()
	unsetWIClientID, _ := setEnvVar(WorkloadIdentityClientID, "ENV_WI_CLIENT_ID")
	defer unsetWIClientID()
	unsetWITenantID, _ := setEnvVar(WorkloadIdentityTenantID, "ENV_WI_TENANT_ID")
	defer unsetWITenantID()
	unsetWITokenFile, _ := setEnvVar(WorkloadIdentityTokenFile, "ENV_WI_TOKEN_FILE")
	defer unsetWITokenFile()

	t.Run("ReadSettings", func(t *testing.T) {
		tcs := []struct {
			name          string
			cfg           *backend.GrafanaCfg
			expectedError error
			expectedAzure *AzureSettings
		}{

			{
				name: "read from context",
				cfg: backend.NewGrafanaCfg(map[string]string{
					AzureCloud:                              AzurePublic,
					ManagedIdentityEnabled:                  "true",
					ManagedIdentityClientID:                 "mock_managed_identity_client_id",
					UserIdentityEnabled:                     "true",
					UserIdentityClientAuthentication:        "mock_user_identity_client_authentication",
					UserIdentityClientID:                    "mock_user_identity_client_id",
					UserIdentityClientSecret:                "mock_user_identity_client_secret",
					UserIdentityManagedIdentityClientID:     "mock_user_identity_managed_identity_client_id",
					UserIdentityFederatedCredentialAudience: "mock_user_identity_federated_credential_audience",
					UserIdentityTokenURL:                    "mock_user_identity_token_url",
					UserIdentityAssertion:                   "username",
					UserIdentityFallbackCredentialsEnabled:  "false",
					WorkloadIdentityEnabled:                 "true",
					WorkloadIdentityClientID:                "mock_workload_identity_client_id",
					WorkloadIdentityTenantID:                "mock_workload_identity_tenant_id",
					WorkloadIdentityTokenFile:               "mock_workload_identity_token_file",
				}),
				expectedAzure: expectedAzureContextSettings,
				expectedError: nil,
			},
			{
				name:          "read from env if config is nil",
				cfg:           nil,
				expectedAzure: expectedAzureEnvSettings,
				expectedError: nil,
			},
			{
				name:          "read from env if config is empty",
				cfg:           &backend.GrafanaCfg{},
				expectedAzure: expectedAzureEnvSettings,
				expectedError: nil,
			},
			{
				name:          "read from env if config map is nil",
				cfg:           backend.NewGrafanaCfg(nil),
				expectedAzure: expectedAzureEnvSettings,
				expectedError: nil,
			},
			{
				name:          "read from env if config map is empty",
				cfg:           backend.NewGrafanaCfg(make(map[string]string)),
				expectedAzure: expectedAzureEnvSettings,
				expectedError: nil,
			},
		}

		for _, tc := range tcs {
			ctx := backend.WithGrafanaConfig(context.Background(), tc.cfg)
			settings, err := ReadSettings(ctx)

			require.Equal(t, tc.expectedAzure, settings)
			require.Nil(t, err)
		}
	})

}
