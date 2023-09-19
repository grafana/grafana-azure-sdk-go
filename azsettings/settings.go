package azsettings

type AzureSettings struct {
	Cloud                   string
	ManagedIdentityEnabled  bool
	ManagedIdentityClientId string

	WorkloadIdentityEnabled  bool
	WorkloadIdentitySettings *WorkloadIdentitySettings

	UserIdentityEnabled       bool
	UserIdentityTokenEndpoint *TokenEndpointSettings
}

type WorkloadIdentitySettings struct {
	TenantId      string
	ClientId      string
	TokenFilePath string
}

type TokenEndpointSettings struct {
	TokenUrl     string
	ClientId     string
	ClientSecret string

	// UsernameAssertion allows to use a custom token request assertion when Grafana is behind auth proxy
	UsernameAssertion bool
}

func (settings *AzureSettings) GetDefaultCloud() string {
	cloudName := settings.Cloud
	if cloudName == "" {
		return AzurePublic
	}
	return cloudName
}
