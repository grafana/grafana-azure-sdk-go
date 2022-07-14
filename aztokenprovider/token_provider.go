package aztokenprovider

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/azsettings"
)

var (
	azureTokenCache = NewConcurrentTokenCache()
)

type AzureTokenProvider interface {
	GetAccessToken(ctx context.Context, scopes []string) (string, error)
}

type tokenProviderImpl struct {
	tokenRetriever TokenRetriever
}

func NewAzureAccessTokenProvider(settings *azsettings.AzureSettings, credentials azcredentials.AzureCredentials) (AzureTokenProvider, error) {
	var err error

	if settings == nil {
		err = fmt.Errorf("parameter 'settings' cannot be nil")
		return nil, err
	}
	if credentials == nil {
		err = fmt.Errorf("parameter 'credentials' cannot be nil")
		return nil, err
	}

	var tokenRetriever TokenRetriever

	switch c := credentials.(type) {
	case *azcredentials.AzureManagedIdentityCredentials:
		if !settings.ManagedIdentityEnabled {
			err = fmt.Errorf("managed identity authentication is not enabled in Grafana config")
			return nil, err
		} else {
			tokenRetriever = getManagedIdentityTokenRetriever(settings, c)
		}
	case *azcredentials.AzureClientSecretCredentials:
		tokenRetriever, err = getClientSecretTokenRetriever(c)
		if err != nil {
			return nil, err
		}
	default:
		err = fmt.Errorf("credentials of type '%s' not supported by authentication provider", c.AzureAuthType())
		return nil, err
	}

	tokenProvider := &tokenProviderImpl{
		tokenRetriever: tokenRetriever,
	}

	return tokenProvider, nil
}

func (provider *tokenProviderImpl) GetAccessToken(ctx context.Context, scopes []string) (string, error) {
	if ctx == nil {
		err := fmt.Errorf("parameter 'ctx' cannot be nil")
		return "", err
	}
	if scopes == nil {
		err := fmt.Errorf("parameter 'scopes' cannot be nil")
		return "", err
	}

	accessToken, err := azureTokenCache.GetAccessToken(ctx, provider.tokenRetriever, scopes)
	if err != nil {
		return "", err
	}
	return accessToken, nil
}

func getManagedIdentityTokenRetriever(settings *azsettings.AzureSettings, credentials *azcredentials.AzureManagedIdentityCredentials) TokenRetriever {
	var clientId string
	if credentials.ClientId != "" {
		clientId = credentials.ClientId
	} else {
		clientId = settings.ManagedIdentityClientId
	}
	return &managedIdentityTokenRetriever{
		clientId: clientId,
	}
}

func getClientSecretTokenRetriever(credentials *azcredentials.AzureClientSecretCredentials) (TokenRetriever, error) {
	var authority azidentity.AuthorityHost
	if credentials.Authority != "" {
		authority = azidentity.AuthorityHost(credentials.Authority)
	} else {
		var err error
		authority, err = resolveAuthorityForCloud(credentials.AzureCloud)
		if err != nil {
			return nil, err
		}
	}
	return &clientSecretTokenRetriever{
		authority:    authority,
		tenantId:     credentials.TenantId,
		clientId:     credentials.ClientId,
		clientSecret: credentials.ClientSecret,
	}, nil
}

func resolveAuthorityForCloud(cloudName string) (azidentity.AuthorityHost, error) {
	// Known Azure clouds
	switch cloudName {
	case azsettings.AzurePublic:
		return azidentity.AzurePublicCloud, nil
	case azsettings.AzureChina:
		return azidentity.AzureChina, nil
	case azsettings.AzureUSGovernment:
		return azidentity.AzureGovernment, nil
	default:
		err := fmt.Errorf("the Azure cloud '%s' not supported", cloudName)
		return "", err
	}
}

type managedIdentityTokenRetriever struct {
	clientId   string
	credential azcore.TokenCredential
}

func (c *managedIdentityTokenRetriever) GetCacheKey() string {
	clientId := c.clientId
	if clientId == "" {
		clientId = "system"
	}
	return fmt.Sprintf("azure|msi|%s", clientId)
}

func (c *managedIdentityTokenRetriever) Init() error {
	options := &azidentity.ManagedIdentityCredentialOptions{}
	if c.clientId != "" {
		options.ID = azidentity.ClientID(c.clientId)
	}
	credential, err := azidentity.NewManagedIdentityCredential(options)
	if err != nil {
		return err
	} else {
		c.credential = credential
		return nil
	}
}

func (c *managedIdentityTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: accessToken.Token, ExpiresOn: accessToken.ExpiresOn}, nil
}

type clientSecretTokenRetriever struct {
	authority    azidentity.AuthorityHost
	tenantId     string
	clientId     string
	clientSecret string
	credential   azcore.TokenCredential
}

func (c *clientSecretTokenRetriever) GetCacheKey() string {
	return fmt.Sprintf("azure|clientsecret|%s|%s|%s|%s", c.authority, c.tenantId, c.clientId, hashSecret(c.clientSecret))
}

func (c *clientSecretTokenRetriever) Init() error {
	options := &azidentity.ClientSecretCredentialOptions{AuthorityHost: c.authority}
	if credential, err := azidentity.NewClientSecretCredential(c.tenantId, c.clientId, c.clientSecret, options); err != nil {
		return err
	} else {
		c.credential = credential
		return nil
	}
}

func (c *clientSecretTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: accessToken.Token, ExpiresOn: accessToken.ExpiresOn}, nil
}

func hashSecret(secret string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(secret))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
