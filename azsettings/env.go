package azsettings

import (
	"fmt"

	"github.com/grafana/grafana-azure-sdk-go/azsettings/internal/envutil"
)

const (
	envAzureCloud = "GFAZPL_AZURE_CLOUD"

	envManagedIdentityEnabled  = "GFAZPL_MANAGED_IDENTITY_ENABLED"
	envManagedIdentityClientId = "GFAZPL_MANAGED_IDENTITY_CLIENT_ID"

	envUserIdentityEnabled      = "GFAZPL_USER_IDENTITY_ENABLED"
	envUserIdentityTokenUrl     = "GFAZPL_USER_IDENTITY_TOKEN_URL"
	envUserIdentityClientId     = "GFAZPL_USER_IDENTITY_CLIENT_ID"
	envUserIdentityClientSecret = "GFAZPL_USER_IDENTITY_CLIENT_SECRET"
	envUserIdentityAssertion    = "GFAZPL_USER_IDENTITY_ASSERTION"

	// Pre Grafana 9.x variables
	fallbackAzureCloud              = "AZURE_CLOUD"
	fallbackManagedIdentityEnabled  = "AZURE_MANAGED_IDENTITY_ENABLED"
	fallbackManagedIdentityClientId = "AZURE_MANAGED_IDENTITY_CLIENT_ID"
)

func ReadFromEnv() (*AzureSettings, error) {
	azureSettings := &AzureSettings{}

	azureSettings.Cloud = envutil.GetOrFallback(envAzureCloud, fallbackAzureCloud, AzurePublic)

	// Managed Identity authentication
	if msiEnabled, err := envutil.GetBoolOrFallback(envManagedIdentityEnabled, fallbackManagedIdentityEnabled, false); err != nil {
		err = fmt.Errorf("invalid Azure configuration: %w", err)
		return nil, err
	} else if msiEnabled {
		azureSettings.ManagedIdentityEnabled = true
		azureSettings.ManagedIdentityClientId = envutil.GetOrFallback(envManagedIdentityClientId, fallbackManagedIdentityClientId, "")
	}

	// User Identity authentication
	if userIdentityEnabled, err := envutil.GetBoolOrDefault(envUserIdentityEnabled, false); err != nil {
		err = fmt.Errorf("invalid Azure configuration: %w", err)
		return nil, err
	} else if userIdentityEnabled {
		tokenUrl, err := envutil.Get(envUserIdentityTokenUrl)
		if err != nil {
			err = fmt.Errorf("token URL must be set when user identity authentication enabled: %w", err)
			return nil, err
		}

		clientId, err := envutil.Get(envUserIdentityClientId)
		if err != nil {
			err = fmt.Errorf("client ID must be set when user identity authentication enabled: %w", err)
			return nil, err
		}

		clientSecret := envutil.GetOrDefault(envUserIdentityClientSecret, "")

		assertion := envutil.GetOrDefault(envUserIdentityAssertion, "")
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
			envs = append(envs, fmt.Sprintf("%s=%s", envAzureCloud, azureSettings.Cloud))
		}

		if azureSettings.ManagedIdentityEnabled {
			envs = append(envs, fmt.Sprintf("%s=true", envManagedIdentityEnabled))

			if azureSettings.ManagedIdentityClientId != "" {
				envs = append(envs, fmt.Sprintf("%s=%s", envManagedIdentityClientId, azureSettings.ManagedIdentityClientId))
			}
		}

		if azureSettings.UserIdentityEnabled {
			envs = append(envs, fmt.Sprintf("%s=true", envUserIdentityEnabled))

			if azureSettings.UserIdentityTokenEndpoint != nil {
				if azureSettings.UserIdentityTokenEndpoint.TokenUrl != "" {
					envs = append(envs, fmt.Sprintf("%s=%s", envUserIdentityTokenUrl, azureSettings.UserIdentityTokenEndpoint.TokenUrl))
				}
				if azureSettings.UserIdentityTokenEndpoint.ClientId != "" {
					envs = append(envs, fmt.Sprintf("%s=%s", envUserIdentityClientId, azureSettings.UserIdentityTokenEndpoint.ClientId))
				}
				if azureSettings.UserIdentityTokenEndpoint.ClientSecret != "" {
					envs = append(envs, fmt.Sprintf("%s=%s", envUserIdentityClientSecret, azureSettings.UserIdentityTokenEndpoint.ClientSecret))
				}
				if azureSettings.UserIdentityTokenEndpoint.UsernameAssertion {
					envs = append(envs, fmt.Sprintf("%s=username", envUserIdentityAssertion))
				}
			}
		}
	}

	return envs
}
