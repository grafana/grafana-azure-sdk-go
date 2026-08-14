package azhttpclient

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azhttpclient/internal/azendpoint"
	"github.com/grafana/grafana-azure-sdk-go/v2/aztokenprovider"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

const azureMiddlewareName = "AzureAuthentication"

func AzureMiddleware(authOpts *AuthOptions, credentials azcredentials.AzureCredentials) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(azureMiddlewareName, func(clientOpts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		var err error
		var tokenProvider aztokenprovider.AzureTokenProvider
		var sessionProvider *userSessionProvider

		if tokenProviderFactory, ok := authOpts.customProviders[credentials.AzureAuthType()]; ok && tokenProviderFactory != nil {
			tokenProvider, err = tokenProviderFactory(authOpts.settings, credentials)
		} else {
			tokenProvider, err = aztokenprovider.NewAzureAccessTokenProvider(authOpts.settings, credentials, authOpts.userIdentitySupported)
		}
		if err != nil {
			return errorResponse(err)
		}

		if authOpts.rateLimitSession {
			sessionProvider, err = newSessionProvider()
			if err != nil {
				return errorResponse(err)
			}
		}

		return applyAzureAuth(tokenProvider, sessionProvider, authOpts.scopes, authOpts.endpoints, authOpts.scopeResolver, next)
	})
}

func applyAzureAuth(tokenProvider aztokenprovider.AzureTokenProvider, sessionProvider *userSessionProvider,
	scopes []string, endpoints *azendpoint.EndpointAllowlist, scopeResolver ScopeResolver, next http.RoundTripper) http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req == nil {
			return nil, fmt.Errorf("request is nil")
		}
		reqContext := req.Context()
		requestScopes := scopes

		if endpoints != nil {
			endpoint := azendpoint.Endpoint(*req.URL)
			if endpoint == nil {
				return nil, fmt.Errorf("request to invalid endpoint '%s' is not allowed by the datasource", req.URL.String())
			}
			if !endpoints.IsAllowed(endpoint) {
				return nil, fmt.Errorf("request to endpoint '%s' is not allowed by the datasource", endpoint.String())
			}
		}

		if scopeResolver != nil {
			dynamicScopes, err := scopeResolver(reqContext, req)
			switch {
			case err != nil:
				backend.Logger.FromContext(reqContext).Warn("failed to resolve scopes, falling back to default scopes", "error", err)
			case len(dynamicScopes) == 0:
				backend.Logger.FromContext(reqContext).Warn("scope resolver returned no scopes, falling back to default scopes")
			default:
				requestScopes = dynamicScopes
			}
		}

		if len(requestScopes) == 0 {
			err := errors.New("scopes not configured")
			return nil, fmt.Errorf("invalid Azure configuration: %s", err)
		}

		token, err := tokenProvider.GetAccessToken(reqContext, requestScopes)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve Azure access token: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		if sessionProvider != nil {
			sessionId, err := sessionProvider.GetSessionId(reqContext)
			switch {
			case errors.Is(err, ErrUserContextNotConfigured):
				// No user in context (e.g. service-context calls such as multi-tenant
				// health checks). The rate-limit session id is optional metadata, so
				// omit the header instead of failing the request.
			case err != nil:
				return nil, fmt.Errorf("failed to obtain user session: %w", err)
			case sessionId != "":
				req.Header.Set("x-ms-ratelimit-id", sessionId)
			}
		}

		return next.RoundTrip(req)
	})
}

func errorResponse(err error) http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("invalid Azure configuration: %s", err)
	})
}
