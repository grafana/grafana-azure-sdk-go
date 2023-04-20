package aztokenprovider

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/azsettings"
)

var (
	azureTokenCache = NewConcurrentTokenCache()
)

type AzureTokenProvider interface {
	GetAccessToken(ctx context.Context, scopes []string) (string, error)
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

	tokenProvider := &serviceTokenProvider{
		tokenCache:     azureTokenCache,
		tokenRetriever: tokenRetriever,
	}

	return tokenProvider, nil
}

type serviceTokenProvider struct {
	tokenCache     ConcurrentTokenCache
	tokenRetriever TokenRetriever
}

func (provider *serviceTokenProvider) GetAccessToken(ctx context.Context, scopes []string) (string, error) {
	if ctx == nil {
		err := fmt.Errorf("parameter 'ctx' cannot be nil")
		return "", err
	}
	if scopes == nil {
		err := fmt.Errorf("parameter 'scopes' cannot be nil")
		return "", err
	}

	accessToken, err := provider.tokenCache.GetAccessToken(ctx, provider.tokenRetriever, scopes)
	if err != nil {
		return "", err
	}
	return accessToken, nil
}
