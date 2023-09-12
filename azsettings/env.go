package azsettings

import (
	"fmt"

	"github.com/grafana/grafana-azure-sdk-go/azsettings/internal/envutil"
)

const (
	AzureCloud = "GFAZPL_AZURE_CLOUD"

	ManagedIdentityEnabled  = "GFAZPL_MANAGED_IDENTITY_ENABLED"
	ManagedIdentityClientID = "GFAZPL_MANAGED_IDENTITY_CLIENT_ID"

	WorkloadIdentityEnabled = "GFAZPL_WORKLOAD_IDENTITY_ENABLED"

	UserIdentityEnabled      = "GFAZPL_USER_IDENTITY_ENABLED"
	UserIdentityTokenURL     = "GFAZPL_USER_IDENTITY_TOKEN_URL"
	UserIdentityClientID     = "GFAZPL_USER_IDENTITY_CLIENT_ID"
	UserIdentityClientSecret = "GFAZPL_USER_IDENTITY_CLIENT_SECRET"
	UserIdentityAssertion    = "GFAZPL_USER_IDENTITY_ASSERTION"

	// Pre Grafana 9.x variables
	fallbackAzureCloud              = "AZURE_CLOUD"
	fallbackManagedIdentityEnabled  = "AZURE_MANAGED_IDENTITY_ENABLED"
	fallbackManagedIdentityClientId = "AZURE_MANAGED_IDENTITY_CLIENT_ID"
)

func ReadFromEnv() (*AzureSettings, error) {
	azureSettings := &AzureSettings{}

	azureSettings.Cloud = envutil.GetOrFallback(AzureCloud, fallbackAzureCloud, AzurePublic)

	// Managed Identity authentication
	if msiEnabled, err := envutil.GetBoolOrFallback(ManagedIdentityEnabled, fallbackManagedIdentityEnabled, false); err != nil {
		err = fmt.Errorf("invalid Azure configuration: %w", err)
		return nil, err
	} else if msiEnabled {
		azureSettings.ManagedIdentityEnabled = true
		azureSettings.ManagedIdentityClientId = envutil.GetOrFallback(ManagedIdentityClientID, fallbackManagedIdentityClientId, "")
	}

	// Workload Identity authentication
	if wiEnabled, err := envutil.GetBoolOrDefault(WorkloadIdentityEnabled, false); err != nil {
		err = fmt.Errorf("invalid Azure configuration: %w", err)
		return nil, err
	} else if wiEnabled {
		azureSettings.WorkloadIdentityEnabled = true
	}

	// User Identity authentication
	if userIdentityEnabled, err := envutil.GetBoolOrDefault(UserIdentityEnabled, false); err != nil {
		err = fmt.Errorf("invalid Azure configuration: %w", err)
		return nil, err
	} else if userIdentityEnabled {
		tokenUrl, err := envutil.Get(UserIdentityTokenURL)
		if err != nil {
			err = fmt.Errorf("token URL must be set when user identity authentication enabled: %w", err)
			return nil, err
		}

		clientId, err := envutil.Get(UserIdentityClientID)
		if err != nil {
			err = fmt.Errorf("client ID must be set when user identity authentication enabled: %w", err)
			return nil, err
		}

		clientSecret := envutil.GetOrDefault(UserIdentityClientSecret, "")

		assertion := envutil.GetOrDefault(UserIdentityAssertion, "")
		usernameAssertion := assertion == "username"

		azureSettings.UserIdentityEnabled = true
		azureSettings.UserIdentityTokenEndpoint = &TokenEndpointSettings{
			TokenUrl:          tokenUrl,
			ClientId:          clientId,
			ClientSecret:      clientSecret,
			UsernameAssertion: usernameAssertion,
		}
	}

	return azureSettings, nil
}

func WriteToEnvStr(azureSettings *AzureSettings) []string {
	var envs []string

	if azureSettings != nil {
		if azureSettings.Cloud != "" {
			envs = append(envs, fmt.Sprintf("%s=%s", AzureCloud, azureSettings.Cloud))
		}

		if azureSettings.ManagedIdentityEnabled {
			envs = append(envs, fmt.Sprintf("%s=true", ManagedIdentityEnabled))

			if azureSettings.ManagedIdentityClientId != "" {
				envs = append(envs, fmt.Sprintf("%s=%s", ManagedIdentityClientID, azureSettings.ManagedIdentityClientId))
			}
		}

		if azureSettings.WorkloadIdentityEnabled {
			envs = append(envs, fmt.Sprintf("%s=true", WorkloadIdentityEnabled))
		}

		if azureSettings.UserIdentityEnabled {
			envs = append(envs, fmt.Sprintf("%s=true", UserIdentityEnabled))

			if azureSettings.UserIdentityTokenEndpoint != nil {
				if azureSettings.UserIdentityTokenEndpoint.TokenUrl != "" {
					envs = append(envs, fmt.Sprintf("%s=%s", UserIdentityTokenURL, azureSettings.UserIdentityTokenEndpoint.TokenUrl))
				}
				if azureSettings.UserIdentityTokenEndpoint.ClientId != "" {
					envs = append(envs, fmt.Sprintf("%s=%s", UserIdentityClientID, azureSettings.UserIdentityTokenEndpoint.ClientId))
				}
				if azureSettings.UserIdentityTokenEndpoint.ClientSecret != "" {
					envs = append(envs, fmt.Sprintf("%s=%s", UserIdentityClientSecret, azureSettings.UserIdentityTokenEndpoint.ClientSecret))
				}
				if azureSettings.UserIdentityTokenEndpoint.UsernameAssertion {
					envs = append(envs, fmt.Sprintf("%s=username", UserIdentityAssertion))
				}
			}
		}
	}

	return envs
}
