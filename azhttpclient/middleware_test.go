package azhttpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
	"github.com/grafana/grafana-azure-sdk-go/v2/aztokenprovider"
	"github.com/grafana/grafana-azure-sdk-go/v2/azusercontext"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureMiddleware(t *testing.T) {
	azureSettings := &azsettings.AzureSettings{
		Cloud: azsettings.AzurePublic,
	}

	clientOpts := httpclient.Options{}
	next := &testRoundTripper{}

	t.Run("should use custom provider if registered for given custom credentials", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})
		testTokenProvider := &customTokenProvider{}
		authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
			return testTokenProvider, nil
		})

		credentials := &customCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
		require.NoError(t, err)

		resp, err := middleware.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.True(t, testTokenProvider.Called)
	})

	t.Run("should return error if custom provider not registered for given custom credentials", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})

		credentials := &customCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
		require.NoError(t, err)

		_, err = middleware.RoundTrip(req)
		assert.Error(t, err)
	})

	t.Run("should use custom provider if registered for built-in credentials", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})
		testTokenProvider := &customTokenProvider{}
		authOpts.AddTokenProvider(azcredentials.AzureAuthManagedIdentity, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
			return testTokenProvider, nil
		})

		credentials := &azcredentials.AzureManagedIdentityCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
		require.NoError(t, err)

		_, err = middleware.RoundTrip(req)
		require.NoError(t, err)
		assert.True(t, testTokenProvider.Called)
	})

	t.Run("should not use custom provider if registered for different credentials", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})
		testTokenProvider := &customTokenProvider{}
		authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
			return testTokenProvider, nil
		})

		credentials := &azcredentials.AzureManagedIdentityCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
		require.NoError(t, err)

		_, err = middleware.RoundTrip(req)
		assert.EqualError(t, err, "invalid Azure configuration: managed identity authentication is not enabled in Grafana config")
		assert.False(t, testTokenProvider.Called)
	})

	t.Run("given allowed endpoints configured", func(t *testing.T) {
		authOpts := NewAuthOptions(azureSettings)
		authOpts.Scopes([]string{"https://datasource.example.org/.default"})
		testTokenProvider := &customTokenProvider{}
		authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
			return testTokenProvider, nil
		})

		err := authOpts.AllowedEndpoints([]string{
			"https://*.example.com",
		})
		require.NoError(t, err)

		credentials := &customCredentials{}
		middleware := AzureMiddleware(authOpts, credentials).CreateMiddleware(clientOpts, next)

		t.Run("should allow endpoint in the allowlist", func(t *testing.T) {
			req, err := http.NewRequest("GET", "https://test.example.com", nil)
			require.NoError(t, err)

			resp, err := middleware.RoundTrip(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
			assert.True(t, testTokenProvider.Called)
		})

		t.Run("should not allow http when https allowed", func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://test.example.com", nil)
			require.NoError(t, err)

			_, err = middleware.RoundTrip(req)
			assert.Error(t, err)
		})

		t.Run("sould not allow endpoint not in the allowlist", func(t *testing.T) {
			req, err := http.NewRequest("GET", "https://another.com", nil)
			require.NoError(t, err)

			_, err = middleware.RoundTrip(req)
			assert.Error(t, err)
		})
	})

	t.Run("given rate-limit session enabled", func(t *testing.T) {
		newMiddleware := func(capture *testRoundTripper) http.RoundTripper {
			authOpts := NewAuthOptions(azureSettings)
			authOpts.Scopes([]string{"https://datasource.example.org/.default"})
			authOpts.AddRateLimitSession(true)
			authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
				return &customTokenProvider{}, nil
			})
			return AzureMiddleware(authOpts, &customCredentials{}).CreateMiddleware(clientOpts, capture)
		}

		t.Run("should set the rate-limit header when a user is in context", func(t *testing.T) {
			capture := &testRoundTripper{}
			middleware := newMiddleware(capture)

			req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
			require.NoError(t, err)
			usrctx := azusercontext.WithCurrentUser(req.Context(), azusercontext.CurrentUserContext{
				User: &backend.User{Login: "user1@example.org"},
			})

			resp, err := middleware.RoundTrip(req.WithContext(usrctx))
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
			assert.NotEmpty(t, capture.lastReq.Header.Get("x-ms-ratelimit-id"))
		})

		t.Run("should succeed without the rate-limit header when no user is in context", func(t *testing.T) {
			capture := &testRoundTripper{}
			middleware := newMiddleware(capture)

			// Simulates service-context calls such as multi-tenant health checks where
			// there is no acting Grafana user. The request must not fail.
			req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
			require.NoError(t, err)

			resp, err := middleware.RoundTrip(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
			assert.Empty(t, capture.lastReq.Header.Get("x-ms-ratelimit-id"))
		})
	})

	t.Run("given scope resolver configured", func(t *testing.T) {
		t.Run("should use dynamic scopes when resolver returns scopes", func(t *testing.T) {
			captureProvider := &capturingTokenProvider{}
			authOpts := NewAuthOptions(azureSettings)
			authOpts.Scopes([]string{"https://default.example.org/.default"})
			authOpts.SetScopeResolver(func(_ context.Context, _ *http.Request) ([]string, error) {
				return []string{"https://dynamic.example.org/.default"}, nil
			})
			authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
				return captureProvider, nil
			})

			capture := &testRoundTripper{}
			middleware := AzureMiddleware(authOpts, &customCredentials{}).CreateMiddleware(clientOpts, capture)
			req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
			require.NoError(t, err)

			resp, err := middleware.RoundTrip(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, []string{"https://dynamic.example.org/.default"}, captureProvider.LastScopes)
		})

		t.Run("should fall back to default scopes when resolver returns error", func(t *testing.T) {
			captureProvider := &capturingTokenProvider{}
			authOpts := NewAuthOptions(azureSettings)
			authOpts.Scopes([]string{"https://default.example.org/.default"})
			authOpts.SetScopeResolver(func(_ context.Context, _ *http.Request) ([]string, error) {
				return nil, errors.New("resolver failed")
			})
			authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
				return captureProvider, nil
			})

			capture := &testRoundTripper{}
			middleware := AzureMiddleware(authOpts, &customCredentials{}).CreateMiddleware(clientOpts, capture)
			req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
			require.NoError(t, err)

			resp, err := middleware.RoundTrip(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, []string{"https://default.example.org/.default"}, captureProvider.LastScopes)
		})

		t.Run("should return configuration error when resolver returns no scopes and defaults are empty", func(t *testing.T) {
			captureProvider := &capturingTokenProvider{}
			authOpts := NewAuthOptions(azureSettings)
			authOpts.SetScopeResolver(func(_ context.Context, _ *http.Request) ([]string, error) {
				return nil, nil
			})
			authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {
				return captureProvider, nil
			})

			capture := &testRoundTripper{}
			middleware := AzureMiddleware(authOpts, &customCredentials{}).CreateMiddleware(clientOpts, capture)
			req, err := http.NewRequest("GET", "https://testendpoint.microsoft.com", nil)
			require.NoError(t, err)

			_, err = middleware.RoundTrip(req)
			require.Error(t, err)
			assert.EqualError(t, err, "invalid Azure configuration: scopes not configured")
		})

		t.Run("should resolve scopes per-request under concurrency", func(t *testing.T) {
			// Resolver derives scopes from the request host so each cluster gets
			// its own scope set.
			authOpts := NewAuthOptions(azureSettings)
			authOpts.Scopes([]string{"https://default.example.org/.default"})
			authOpts.SetScopeResolver(func(_ context.Context, req *http.Request) ([]string, error) {
				return []string{fmt.Sprintf("https://%s/.default", req.URL.Host)}, nil
			})
			authOpts.AddTokenProvider(azureAuthCustom, func(_ *azsettings.AzureSettings, _ azcredentials.AzureCredentials) (aztokenprovider.AzureTokenProvider, error) {

				return &scopeEchoTokenProvider{}, nil
			})

			middleware := AzureMiddleware(authOpts, &customCredentials{}).CreateMiddleware(clientOpts, &authEchoRoundTripper{})

			hosts := []string{"clustera.kusto.windows.net", "clusterb.kusto.windows.net"}
			const workers = 50

			var wg sync.WaitGroup
			errs := make(chan error, workers)
			for i := 0; i < workers; i++ {
				host := hosts[i%len(hosts)]
				wg.Add(1)
				go func(host string) {
					defer wg.Done()
					req, err := http.NewRequest("GET", "https://"+host, nil)
					if err != nil {
						errs <- err
						return
					}
					resp, err := middleware.RoundTrip(req)
					if err != nil {
						errs <- err
						return
					}
					want := fmt.Sprintf("Bearer https://%s/.default", host)
					if got := resp.Header.Get("X-Echo-Authorization"); got != want {
						errs <- fmt.Errorf("host %s: got %q, want %q", host, got, want)
					}
				}(host)
			}
			wg.Wait()
			close(errs)
			for err := range errs {
				require.NoError(t, err)
			}
		})
	})
}

const (
	azureAuthCustom = "custom"
)

type customCredentials struct {
}

func (credentials *customCredentials) AzureAuthType() string {
	return azureAuthCustom
}

type customTokenProvider struct {
	Called bool
}

func (provider *customTokenProvider) GetAccessToken(ctx context.Context, scopes []string) (string, error) {
	if ctx == nil {
		err := fmt.Errorf("parameter 'ctx' cannot be nil")
		return "", err
	}
	if scopes == nil {
		err := fmt.Errorf("parameter 'scopes' cannot be nil")
		return "", err
	}

	provider.Called = true

	return "FAKE-ACCESS-TOKEN", nil
}

type capturingTokenProvider struct {
	LastScopes []string
}

func (provider *capturingTokenProvider) GetAccessToken(_ context.Context, scopes []string) (string, error) {
	provider.LastScopes = append([]string{}, scopes...)
	return "FAKE-ACCESS-TOKEN", nil
}

type testRoundTripper struct {
	lastReq *http.Request
}

func (rt *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.lastReq = req
	return &http.Response{Status: "200 OK", StatusCode: 200}, nil
}

// scopeEchoTokenProvider returns the requested scopes as the token, allowing
// tests to assert which scopes a request was authenticated with. Stateless - safe for concurrent use.
type scopeEchoTokenProvider struct{}

func (provider *scopeEchoTokenProvider) GetAccessToken(_ context.Context, scopes []string) (string, error) {
	return strings.Join(scopes, " "), nil
}

// authEchoRoundTripper echoes the received Authorization header back on the
// response so concurrent callers can verify their own request in isolation. Stateless - safe for concurrent use.
type authEchoRoundTripper struct{}

func (rt *authEchoRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	header := http.Header{}
	header.Set("X-Echo-Authorization", req.Header.Get("Authorization"))
	return &http.Response{Status: "200 OK", StatusCode: 200, Header: header}, nil
}
