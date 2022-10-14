package azcredentials

const (
	AzureAuthCurrentUserIdentity = "currentuser"
	AzureAuthManagedIdentity     = "msi"
	AzureAuthClientSecret        = "clientsecret"
	AzureAuthClientSecretObo     = "clientsecret-obo"
)

type AzureCredentials interface {
	AzureAuthType() string
}

type AadCurrentUserCredentials struct {
}

type AzureManagedIdentityCredentials struct {
	ClientId string
}

type AzureClientSecretCredentials struct {
	AzureCloud   string
	Authority    string
	TenantId     string
	ClientId     string
	ClientSecret string
}

type AzureClientSecretOboCredentials struct {
	ClientSecretCredentials AzureClientSecretCredentials
}

func (credentials *AadCurrentUserCredentials) AzureAuthType() string {
	return AzureAuthCurrentUserIdentity
}

func (credentials *AzureManagedIdentityCredentials) AzureAuthType() string {
	return AzureAuthManagedIdentity
}

func (credentials *AzureClientSecretCredentials) AzureAuthType() string {
	return AzureAuthClientSecret
}

func (credentials *AzureClientSecretOboCredentials) AzureAuthType() string {
	return AzureAuthClientSecretObo
}
