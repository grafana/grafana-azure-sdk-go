package azhttpclient

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/aztokenprovider"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

const azureMiddlewareName = "AzureAuthentication"

func AzureMiddleware(authOpts *AuthOptions, credentials azcredentials.AzureCredentials) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(azureMiddlewareName, func(clientOpts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		var err error
		var tokenProvider aztokenprovider.AzureTokenProvider = nil

		if tokenProviderFactory, ok := authOpts.customProviders[credentials.AzureAuthType()]; ok && tokenProviderFactory != nil {
			tokenProvider, err = tokenProviderFactory(authOpts.settings, credentials)
		} else {
			tokenProvider, err = aztokenprovider.NewAzureAccessTokenProvider(authOpts.settings, credentials)
		}
		if err != nil {
			return errorResponse(err)
		}

		if len(authOpts.scopes) == 0 {
			err = errors.New("scopes not configured")
			return errorResponse(err)
		}

		return ApplyAzureAuth(tokenProvider, authOpts.scopes, next)
	})
}

func ApplyAzureAuth(tokenProvider aztokenprovider.AzureTokenProvider, scopes []string, next http.RoundTripper) http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		token, err := tokenProvider.GetAccessToken(req.Context(), scopes)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve Azure access token: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return next.RoundTrip(req)
	})
}

func errorResponse(err error) http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("invalid Azure configuration: %s", err)
	})
}
