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
