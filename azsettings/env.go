package azsettings

import (
	"fmt"

	"github.com/grafana/grafana-azure-sdk-go/azsettings/internal/envutil"
)

const (
	envAzureCloud              = "GFAZPL_AZURE_CLOUD"
	envManagedIdentityEnabled  = "GFAZPL_MANAGED_IDENTITY_ENABLED"
	envManagedIdentityClientId = "GFAZPL_MANAGED_IDENTITY_CLIENT_ID"
	envClientOnBehalfOfEnabled = "GFAZPL_CLIENT_OBO_ENABLED"

	// Pre Grafana 9.x variables
	fallbackAzureCloud              = "AZURE_CLOUD"
	fallbackManagedIdentityEnabled  = "AZURE_MANAGED_IDENTITY_ENABLED"
	fallbackManagedIdentityClientId = "AZURE_MANAGED_IDENTITY_CLIENT_ID"
)

func ReadFromEnv() (*AzureSettings, error) {
	var err error

	azureSettings := &AzureSettings{}
	azureSettings.Cloud = envutil.GetOrFallback(envAzureCloud, fallbackAzureCloud, AzurePublic)

	// Managed Identity
	if msiEnabled, err := envutil.GetBoolOrFallback(envManagedIdentityEnabled, fallbackManagedIdentityEnabled, false); err != nil {
		err = fmt.Errorf("invalid Azure configuration: %w", err)
		return nil, err
	} else if msiEnabled {
		azureSettings.ManagedIdentityEnabled = true
		azureSettings.ManagedIdentityClientId = envutil.GetOrFallback(envManagedIdentityClientId, fallbackManagedIdentityClientId, "")
	}

	// On-Behalf-Of
	azureSettings.ClientOnBehalfOfEnabled, err = envutil.GetBoolOrDefault(envClientOnBehalfOfEnabled, false)
	if err != nil {
		err = fmt.Errorf("invalid Azure configuration: %w", err)
		return nil, err
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

		if azureSettings.ClientOnBehalfOfEnabled {
			envs = append(envs, fmt.Sprintf("%s=true", envClientOnBehalfOfEnabled))
		}
	}

	return envs
}
