package azsettings

type AzureSettings struct {
	Cloud                   string
	ManagedIdentityEnabled  bool
	ManagedIdentityClientId string

	UserIdentityEnabled       bool
	UserIdentityTokenEndpoint *TokenEndpointSettings
}

type TokenEndpointSettings struct {
	TokenUrl     string
	ClientId     string
	ClientSecret string
}

func (settings *AzureSettings) GetDefaultCloud() string {
	cloudName := settings.Cloud
	if cloudName == "" {
		return AzurePublic
	}
	return cloudName
}
