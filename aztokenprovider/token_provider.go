package aztokenprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/azsettings"
	"github.com/grafana/grafana-azure-sdk-go/azusercontext"
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

	switch c := credentials.(type) {
	case *azcredentials.AzureManagedIdentityCredentials:
		if !settings.ManagedIdentityEnabled {
			err = fmt.Errorf("managed identity authentication is not enabled in Grafana config")
			return nil, err
		}
		tokenRetriever := getManagedIdentityTokenRetriever(settings, c)
		return &serviceTokenProvider{
			tokenCache:     azureTokenCache,
			tokenRetriever: tokenRetriever,
		}, nil
	case *azcredentials.AzureClientSecretCredentials:
		tokenRetriever, err := getClientSecretTokenRetriever(c)
		if err != nil {
			return nil, err
		}
		return &serviceTokenProvider{
			tokenCache:     azureTokenCache,
			tokenRetriever: tokenRetriever,
		}, nil
	case *azcredentials.AadCurrentUserCredentials:
		if !settings.UserIdentityEnabled {
			err = fmt.Errorf("user identity authentication is not enabled in Grafana config")
			return nil, err
		}
		tokenEndpoint := settings.UserIdentityTokenEndpoint
		client, err := NewTokenClient(tokenEndpoint.TokenUrl, tokenEndpoint.ClientId, tokenEndpoint.ClientSecret, http.DefaultClient)
		if err != nil {
			err = fmt.Errorf("failed to initialize user authentication provider: %w", err)
			return nil, err
		}
		return &userTokenProvider{
			tokenCache:        azureTokenCache,
			client:            client,
			usernameAssertion: tokenEndpoint.UsernameAssertion,
		}, nil
	default:
		err = fmt.Errorf("credentials of type '%s' not supported by Azure authentication provider", c.AzureAuthType())
		return nil, err
	}
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

type userTokenProvider struct {
	tokenCache        ConcurrentTokenCache
	client            TokenClient
	usernameAssertion bool
}

func (provider *userTokenProvider) GetAccessToken(ctx context.Context, scopes []string) (string, error) {
	if ctx == nil {
		err := fmt.Errorf("parameter 'ctx' cannot be nil")
		return "", err
	}
	if scopes == nil {
		err := fmt.Errorf("parameter 'scopes' cannot be nil")
		return "", err
	}

	currentUser, ok := azusercontext.GetCurrentUser(ctx)
	if !ok {
		err := fmt.Errorf("user context not configured")
		return "", err
	}

	username, err := extractUsername(currentUser)
	if err != nil {
		err := fmt.Errorf("user identity authentication only possible in context of a Grafana user: %w", err)
		return "", err
	}

	var tokenRetriever TokenRetriever
	if provider.usernameAssertion {
		tokenRetriever = &usernameTokenRetriever{
			client:   provider.client,
			username: username,
		}
	} else {
		idToken := currentUser.IdToken
		if idToken == "" {
			err := fmt.Errorf("user identity authentication not possible because there's no ID token associated with the Grafana user")
			return "", err
		}

		tokenRetriever = &onBehalfOfTokenRetriever{
			client:  provider.client,
			userId:  username,
			idToken: idToken,
		}
	}

	accessToken, err := provider.tokenCache.GetAccessToken(ctx, tokenRetriever, scopes)
	if err != nil {
		err = fmt.Errorf("unable to acquire access token for user '%s': %w", username, err)
		return "", err
	}
	return accessToken, nil
}

func extractUsername(userCtx azusercontext.CurrentUserContext) (string, error) {
	user := userCtx.User
	if user != nil && user.Login != "" {
		return user.Login, nil
	} else {
		return "", errors.New("request not associated with a Grafana user")
	}
}
