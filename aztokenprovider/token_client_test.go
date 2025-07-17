package aztokenprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-azure-sdk-go/v2/azusercontext"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenClient_RequestUrlForm_CookieForwarding(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func() context.Context
		expectedCookie string
		description    string
	}{
		{
			name: "forwards cookies when user context exists",
			setupContext: func() context.Context {
				ctx := context.Background()
				currentUser := azusercontext.CurrentUserContext{
					User: &backend.User{
						Login: "test-user",
					},
					Cookies: "session=abc123; theme=dark",
				}
				return azusercontext.WithCurrentUser(ctx, currentUser)
			},
			expectedCookie: "session=abc123; theme=dark",
			description:    "Should forward cookies from user context",
		},
		{
			name: "handles empty cookies in user context",
			setupContext: func() context.Context {
				ctx := context.Background()
				currentUser := azusercontext.CurrentUserContext{
					User: &backend.User{
						Login: "test-user",
					},
					Cookies: "",
				}
				return azusercontext.WithCurrentUser(ctx, currentUser)
			},
			expectedCookie: "",
			description:    "Should not add Cookie header when cookies are empty",
		},
		{
			name: "handles missing user context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedCookie: "",
			description:    "Should not add Cookie header when user context is missing",
		},
		{
			name: "sanitizes cookies with newlines",
			setupContext: func() context.Context {
				ctx := context.Background()
				currentUser := azusercontext.CurrentUserContext{
					User: &backend.User{
						Login: "test-user",
					},
					Cookies: "session=abc123\n; theme=dark\r\n; lang=en",
				}
				return azusercontext.WithCurrentUser(ctx, currentUser)
			},
			expectedCookie: "session=abc123; theme=dark; lang=en",
			description:    "Should remove newline characters from cookies",
		},
		{
			name: "handles cookies that become empty after sanitization",
			setupContext: func() context.Context {
				ctx := context.Background()
				currentUser := azusercontext.CurrentUserContext{
					User: &backend.User{
						Login: "test-user",
					},
					Cookies: "\n\r\n",
				}
				return azusercontext.WithCurrentUser(ctx, currentUser)
			},
			expectedCookie: "",
			description:    "Should not add Cookie header when cookies are only newlines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that captures the request
			var capturedRequest *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r

				// Return a valid token response
				response := tokenResponse{
					AccessToken: "test-token",
					ExpiresIn:   3600,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create token client
			client := &tokenClientImpl{
				httpClient:           http.DefaultClient,
				endpointUrl:          server.URL,
				clientAuthentication: ClientSecret,
				clientId:             "test-client-id",
				clientSecret:         "test-client-secret",
			}

			// Execute request with the test context
			ctx := tt.setupContext()
			_, err := client.FromClientSecret(ctx, []string{"https://graph.microsoft.com/.default"})
			require.NoError(t, err)

			// Verify cookie header
			if tt.expectedCookie != "" {
				assert.Equal(t, tt.expectedCookie, capturedRequest.Header.Get("Cookie"), tt.description)
			} else {
				assert.Empty(t, capturedRequest.Header.Get("Cookie"), tt.description)
			}
		})
	}
}

func TestTokenClient_RequestUrlForm_Headers(t *testing.T) {
	// Create a test server that captures the request
	var capturedRequest *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r

		// Return a valid token response
		response := tokenResponse{
			AccessToken: "test-token",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create token client
	client := &tokenClientImpl{
		httpClient:           http.DefaultClient,
		endpointUrl:          server.URL,
		clientAuthentication: ClientSecret,
		clientId:             "test-client-id",
		clientSecret:         "test-client-secret",
	}

	// Execute request
	ctx := context.Background()
	_, err := client.FromClientSecret(ctx, []string{"https://graph.microsoft.com/.default"})
	require.NoError(t, err)

	// Verify standard headers are set
	assert.Equal(t, "application/x-www-form-urlencoded; charset=utf-8", capturedRequest.Header.Get("Content-Type"))
	assert.Equal(t, "application/json", capturedRequest.Header.Get("Accept"))
	assert.Equal(t, "github.com/grafana/grafana-azure-sdk-go/v2", capturedRequest.Header.Get("X-Client-SKU"))
	assert.Equal(t, "2.0", capturedRequest.Header.Get("X-Client-Ver"))
}

func TestTokenClient_RequestUrlForm_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name: "handles non-200 status code",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid_client",
				})
			},
			expectedError: "request failed with status 401 Unauthorized",
		},
		{
			name: "handles invalid content type",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("not json"))
			},
			expectedError: "invalid response content-type 'text/plain'",
		},
		{
			name: "handles missing access token in response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"some_field": "some_value",
				})
			},
			expectedError: "token response doesn't contain 'access_token' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &tokenClientImpl{
				httpClient:           http.DefaultClient,
				endpointUrl:          server.URL,
				clientAuthentication: ClientSecret,
				clientId:             "test-client-id",
				clientSecret:         "test-client-secret",
			}

			ctx := context.Background()
			_, err := client.FromClientSecret(ctx, []string{"https://graph.microsoft.com/.default"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestTokenClient_MultipleCookieHeaders(t *testing.T) {
	// Test that multiple Cookie headers are properly handled
	var capturedCookies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture all Cookie headers
		capturedCookies = r.Header["Cookie"]

		response := tokenResponse{
			AccessToken: "test-token",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &tokenClientImpl{
		httpClient:           http.DefaultClient,
		endpointUrl:          server.URL,
		clientAuthentication: ClientSecret,
		clientId:             "test-client-id",
		clientSecret:         "test-client-secret",
	}

	// Create context with cookies
	ctx := context.Background()
	currentUser := azusercontext.CurrentUserContext{
		User: &backend.User{
			Login: "test-user",
		},
		Cookies: "session=abc123; preference=dark",
	}
	ctx = azusercontext.WithCurrentUser(ctx, currentUser)

	_, err := client.FromClientSecret(ctx, []string{"https://graph.microsoft.com/.default"})
	require.NoError(t, err)

	// Verify only one Cookie header was added
	assert.Len(t, capturedCookies, 1)
	assert.Equal(t, "session=abc123; preference=dark", capturedCookies[0])
}

func TestTokenClient_AllMethods_CookieForwarding(t *testing.T) {
	// Test that cookies are forwarded for all token request methods
	testCases := []struct {
		name       string
		method     string
		callMethod func(client *tokenClientImpl, ctx context.Context) (*AccessToken, error)
	}{
		{
			name:   "FromClientSecret",
			method: "client_credentials",
			callMethod: func(client *tokenClientImpl, ctx context.Context) (*AccessToken, error) {
				return client.FromClientSecret(ctx, []string{"https://graph.microsoft.com/.default"})
			},
		},
		{
			name:   "FromRefreshToken",
			method: "refresh_token",
			callMethod: func(client *tokenClientImpl, ctx context.Context) (*AccessToken, error) {
				return client.FromRefreshToken(ctx, "test-refresh-token", []string{"https://graph.microsoft.com/.default"})
			},
		},
		{
			name:   "OnBehalfOf",
			method: "urn:ietf:params:oauth:grant-type:jwt-bearer",
			callMethod: func(client *tokenClientImpl, ctx context.Context) (*AccessToken, error) {
				return client.OnBehalfOf(ctx, "test-id-token", []string{"https://graph.microsoft.com/.default"})
			},
		},
		{
			name:   "FromUsername",
			method: "username",
			callMethod: func(client *tokenClientImpl, ctx context.Context) (*AccessToken, error) {
				return client.FromUsername(ctx, "test-user", []string{"https://graph.microsoft.com/.default"})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedRequest *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r

				// Verify grant type
				r.ParseForm()
				assert.Equal(t, tc.method, r.FormValue("grant_type"))

				response := tokenResponse{
					AccessToken: "test-token",
					ExpiresIn:   3600,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := &tokenClientImpl{
				httpClient:           http.DefaultClient,
				endpointUrl:          server.URL,
				clientAuthentication: ClientSecret,
				clientId:             "test-client-id",
				clientSecret:         "test-client-secret",
			}

			// Create context with cookies
			ctx := context.Background()
			currentUser := azusercontext.CurrentUserContext{
				User: &backend.User{
					Login: "test-user",
				},
				Cookies: "auth=xyz789; session=test123",
			}
			ctx = azusercontext.WithCurrentUser(ctx, currentUser)

			// Call the specific method
			_, err := tc.callMethod(client, ctx)
			require.NoError(t, err)

			// Verify cookies were forwarded
			assert.Equal(t, "auth=xyz789; session=test123", capturedRequest.Header.Get("Cookie"))
		})
	}
}
