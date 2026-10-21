package aztokenprovider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenantID = "00000000-1111-2222-3333-444444444444"
	testClientID = "55555555-6666-7777-8888-999999999999"
)

func inMemorySettings(url string) *azsettings.AzureSettings {
	return &azsettings.AzureSettings{
		WorkloadIdentityEnabled: true,
		WorkloadIdentitySettings: &azsettings.WorkloadIdentitySettings{
			InMemory:             true,
			AssertionURL:         url,
			AssertionCallerToken: "caller-token",
		},
	}
}

func TestNewWorkloadIdentityTokenRetriever_Selection(t *testing.T) {
	credentials := &azcredentials.AzureWorkloadIdentityCredentials{TenantId: testTenantID, ClientId: testClientID}

	t.Run("without the toggle it uses the token file retriever, exactly as before", func(t *testing.T) {
		settings := &azsettings.AzureSettings{
			WorkloadIdentityEnabled: true,
			WorkloadIdentitySettings: &azsettings.WorkloadIdentitySettings{
				TokenFile: "/tokens/azure-identity-token",
				// Ignored unless InMemory is set.
				AssertionURL: "http://tokensvc",
			},
		}
		retriever, err := newWorkloadIdentityTokenRetriever(settings, credentials)
		require.NoError(t, err)
		require.IsType(t, &workloadIdentityTokenRetriever{}, retriever)
		assert.Equal(t, "/tokens/azure-identity-token", retriever.(*workloadIdentityTokenRetriever).tokenFile)
	})

	t.Run("without any workload identity settings it still uses the token file retriever", func(t *testing.T) {
		retriever, err := newWorkloadIdentityTokenRetriever(&azsettings.AzureSettings{WorkloadIdentityEnabled: true}, nil)
		require.NoError(t, err)
		require.IsType(t, &workloadIdentityTokenRetriever{}, retriever)
	})

	t.Run("with the toggle it uses the in-memory retriever", func(t *testing.T) {
		retriever, err := newWorkloadIdentityTokenRetriever(inMemorySettings("http://tokensvc"), credentials)
		require.NoError(t, err)
		require.IsType(t, &workloadIdentityInMemoryTokenRetriever{}, retriever)
		assert.Equal(t, "azure|wi-mem|"+testTenantID+"|"+testClientID+"|mt", retriever.GetCacheKey("mt"))
	})

	t.Run("in-memory cache keys never collide with token file cache keys", func(t *testing.T) {
		mem, err := newWorkloadIdentityTokenRetriever(inMemorySettings("http://tokensvc"), credentials)
		require.NoError(t, err)
		file, err := newWorkloadIdentityTokenRetriever(&azsettings.AzureSettings{WorkloadIdentitySettings: &azsettings.WorkloadIdentitySettings{}}, credentials)
		require.NoError(t, err)
		assert.NotEqual(t, file.GetCacheKey("mt"), mem.GetCacheKey("mt"))
	})

	t.Run("in-memory requires the assertion URL", func(t *testing.T) {
		_, err := newWorkloadIdentityTokenRetriever(inMemorySettings(""), credentials)
		require.ErrorContains(t, err, "assertion URL")
	})

	t.Run("in-memory requires tenant and client IDs", func(t *testing.T) {
		_, err := newWorkloadIdentityTokenRetriever(inMemorySettings("http://tokensvc"), &azcredentials.AzureWorkloadIdentityCredentials{})
		require.ErrorContains(t, err, "tenant ID and a client ID")
	})

	t.Run("in-memory falls back to server-level tenant and client IDs", func(t *testing.T) {
		settings := inMemorySettings("http://tokensvc")
		settings.WorkloadIdentitySettings.TenantId = testTenantID
		settings.WorkloadIdentitySettings.ClientId = testClientID
		retriever, err := newWorkloadIdentityTokenRetriever(settings, &azcredentials.AzureWorkloadIdentityCredentials{})
		require.NoError(t, err)
		assert.Contains(t, retriever.GetCacheKey(""), testTenantID)
	})
}

func TestNewAzureAccessTokenProvider_WorkloadIdentityInMemory(t *testing.T) {
	credentials := &azcredentials.AzureWorkloadIdentityCredentials{TenantId: testTenantID, ClientId: testClientID}

	t.Run("fails fast when the in-memory configuration is incomplete", func(t *testing.T) {
		_, err := NewAzureAccessTokenProvider(inMemorySettings(""), credentials, false)
		require.Error(t, err)
	})

	t.Run("still requires workload identity to be enabled", func(t *testing.T) {
		settings := inMemorySettings("http://tokensvc")
		settings.WorkloadIdentityEnabled = false
		_, err := NewAzureAccessTokenProvider(settings, credentials, false)
		require.ErrorContains(t, err, "not enabled")
	})
}

func TestHTTPAssertionProvider(t *testing.T) {
	t.Run("sends the identity and caller token, returns the token", func(t *testing.T) {
		var gotAuth, gotPath, gotStack string
		var gotBody map[string]string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth, gotPath, gotStack = r.Header.Get("Authorization"), r.URL.Path, r.Header.Get(stackHeader)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "signed.jwt.value", "expires_at": 1})
		}))
		defer srv.Close()

		provider, err := NewHTTPAssertionProvider(srv.URL+"/", "caller-token", "my-stack", nil)
		require.NoError(t, err)
		token, err := provider.GetAssertion(context.Background(), testTenantID, testClientID)
		require.NoError(t, err)

		assert.Equal(t, "signed.jwt.value", token)
		assert.Equal(t, "Bearer caller-token", gotAuth)
		assert.Equal(t, "my-stack", gotStack)
		assert.Equal(t, "/v1/mint", gotPath)
		assert.Equal(t, map[string]string{"tenant_id": testTenantID, "client_id": testClientID}, gotBody)
	})

	t.Run("surfaces the service error without the caller token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":"PERMISSION_DENIED","message":"connection is not owned by the calling stack"}`))
		}))
		defer srv.Close()

		provider, err := NewHTTPAssertionProvider(srv.URL, "caller-token", "", nil)
		require.NoError(t, err)
		_, err = provider.GetAssertion(context.Background(), testTenantID, testClientID)
		require.ErrorContains(t, err, "PERMISSION_DENIED")
		assert.NotContains(t, err.Error(), "caller-token")
	})

	t.Run("rejects an empty token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"token":""}`))
		}))
		defer srv.Close()

		provider, err := NewHTTPAssertionProvider(srv.URL, "", "", nil)
		require.NoError(t, err)
		_, err = provider.GetAssertion(context.Background(), testTenantID, testClientID)
		require.ErrorContains(t, err, "empty token")
	})

	t.Run("requires a URL", func(t *testing.T) {
		_, err := NewHTTPAssertionProvider("", "", "", nil)
		require.Error(t, err)
	})
}

// fakeEntra answers MSAL's tenant discovery and token requests, and records the
// client_assertion it receives so the test can prove where it came from.
type fakeEntra struct {
	assertions []string
	clientIDs  []string
}

func (f *fakeEntra) Do(req *http.Request) (*http.Response, error) {
	respond := func(status int, body string) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	}
	switch {
	case strings.HasSuffix(req.URL.Path, "/oauth2/v2.0/token"):
		raw, _ := io.ReadAll(req.Body)
		form := mustParseForm(string(raw))
		f.assertions = append(f.assertions, form["client_assertion"])
		f.clientIDs = append(f.clientIDs, form["client_id"])
		return respond(200, `{"token_type":"Bearer","expires_in":3600,"ext_expires_in":3600,"access_token":"access-token-from-entra"}`)
	case strings.Contains(req.URL.Path, "openid-configuration"):
		base := req.URL.Scheme + "://" + req.URL.Host + "/" + testTenantID
		return respond(200, `{"token_endpoint":"`+base+`/oauth2/v2.0/token","authorization_endpoint":"`+base+`/oauth2/v2.0/authorize","issuer":"`+base+`/v2.0"}`)
	default:
		return respond(404, `{}`)
	}
}

func mustParseForm(s string) map[string]string {
	out := map[string]string{}
	for _, kv := range strings.Split(s, "&") {
		k, v, _ := strings.Cut(kv, "=")
		out[unescape(k)] = unescape(v)
	}
	return out
}

func unescape(s string) string {
	r := strings.NewReplacer("%3A", ":", "%2D", "-", "+", " ", "%2F", "/", "%3D", "=", "%2E", ".")
	return r.Replace(s)
}

func TestWorkloadIdentityInMemory_EndToEnd(t *testing.T) {
	var mints int
	tokensvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mints++
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "assertion-from-tokensvc"})
	}))
	defer tokensvc.Close()

	entra := &fakeEntra{}
	retriever, err := getWorkloadIdentityInMemoryTokenRetriever(
		inMemorySettings(tokensvc.URL),
		&azcredentials.AzureWorkloadIdentityCredentials{TenantId: testTenantID, ClientId: testClientID},
	)
	require.NoError(t, err)
	r := retriever.(*workloadIdentityInMemoryTokenRetriever)
	r.clientOptions = azcore.ClientOptions{Transport: entra}
	require.NoError(t, r.Init())

	token, err := r.GetAccessToken(context.Background(), []string{"https://management.azure.com/.default"})
	require.NoError(t, err)

	assert.Equal(t, "access-token-from-entra", token.Token)
	assert.True(t, token.ExpiresOn.After(time.Now()))
	require.Len(t, entra.assertions, 1)
	assert.Equal(t, "assertion-from-tokensvc", entra.assertions[0], "Entra must receive the assertion minted by the token service")
	assert.Equal(t, testClientID, entra.clientIDs[0])
	assert.Equal(t, 1, mints)
}

func TestHTTPAssertionProvider_OmitsStackHeaderWhenUnset(t *testing.T) {
	var present bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, present = r.Header[stackHeader]
		_, _ = w.Write([]byte(`{"token":"t"}`))
	}))
	defer srv.Close()

	provider, err := NewHTTPAssertionProvider(srv.URL, "caller-token", "", nil)
	require.NoError(t, err)
	_, err = provider.GetAssertion(context.Background(), testTenantID, testClientID)
	require.NoError(t, err)
	assert.False(t, present, "an unset stack must not be sent as an empty header")
}

func contextWithNamespace(ns string) context.Context {
	return backend.WithPluginContext(context.Background(), backend.PluginContext{Namespace: ns})
}

func TestStackIDFromContextOnlyAcceptsAStackNamespace(t *testing.T) {
	for ns, want := range map[string]string{
		"stacks-123": "123", "stacks-1": "1",
		"": "", "default": "", "org-1": "", "hosted-grafana": "",
		"stacks-": "", "stacks-12x": "", "stacks--1": "", "stacks-123/../x": "", "xstacks-123": "",
	} {
		assert.Equal(t, want, stackIDFromContext(contextWithNamespace(ns)), "namespace %q", ns)
	}
	assert.Equal(t, "", stackIDFromContext(context.Background()), "no plugin context at all")
}

func TestHTTPAssertionProvider_StackFromContext(t *testing.T) {
	var stacks []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stacks = append(stacks, r.Header.Get(stackHeader))
		_, _ = w.Write([]byte(`{"token":"t"}`))
	}))
	defer srv.Close()
	provider, err := NewHTTPAssertionProvider(srv.URL, "caller-token", "configured", nil)
	require.NoError(t, err)

	for _, ctx := range []context.Context{
		contextWithNamespace("stacks-42"), // Grafana's stack wins over the configured one
		contextWithNamespace("default"),   // not a stack: fall back to the configured one
		context.Background(),              // no plugin context: same
	} {
		_, err := provider.GetAssertion(ctx, testTenantID, testClientID)
		require.NoError(t, err)
	}
	assert.Equal(t, []string{"42", "configured", "configured"}, stacks)
}

// The stack has to survive the trip through azidentity and MSAL to the assertion callback, which is
// the only place it is read.
func TestWorkloadIdentityInMemory_StackFromContextReachesTheTokenService(t *testing.T) {
	var stack string
	tokensvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stack = r.Header.Get(stackHeader)
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "assertion-from-tokensvc"})
	}))
	defer tokensvc.Close()

	settings := inMemorySettings(tokensvc.URL)
	settings.WorkloadIdentitySettings.StackID = "configured"
	retriever, err := getWorkloadIdentityInMemoryTokenRetriever(settings,
		&azcredentials.AzureWorkloadIdentityCredentials{TenantId: testTenantID, ClientId: testClientID})
	require.NoError(t, err)
	r := retriever.(*workloadIdentityInMemoryTokenRetriever)
	r.clientOptions = azcore.ClientOptions{Transport: &fakeEntra{}}
	require.NoError(t, r.Init())

	_, err = r.GetAccessToken(contextWithNamespace("stacks-77"), []string{"https://management.azure.com/.default"})
	require.NoError(t, err)
	assert.Equal(t, "77", stack)
}
