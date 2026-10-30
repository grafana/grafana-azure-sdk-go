package aztokenprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
				AssertionURL: "https://tokensvc",
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
		retriever, err := newWorkloadIdentityTokenRetriever(inMemorySettings("https://tokensvc"), credentials)
		require.NoError(t, err)
		require.IsType(t, &workloadIdentityInMemoryTokenRetriever{}, retriever)
		assert.Contains(t, retriever.GetCacheKey("mt"), "azure|wi-mem|"+testTenantID+"|"+testClientID+"|https://tokensvc|")
		assert.True(t, strings.HasSuffix(retriever.GetCacheKey("mt"), "|mt"))
	})

	t.Run("in-memory cache key changes when the assertion URL or caller token is rotated", func(t *testing.T) {
		build := func(url, token string) string {
			settings := inMemorySettings(url)
			settings.WorkloadIdentitySettings.AssertionCallerToken = token
			r, err := newWorkloadIdentityTokenRetriever(settings, credentials)
			require.NoError(t, err)
			return r.GetCacheKey("mt")
		}
		base := build("https://tokensvc", "token-1")
		assert.Equal(t, base, build("https://tokensvc", "token-1"))
		assert.NotEqual(t, base, build("https://other", "token-1"))
		assert.NotEqual(t, base, build("https://tokensvc", "token-2"))
		assert.NotContains(t, base, "token-1")
	})

	t.Run("in-memory cache keys never collide with token file cache keys", func(t *testing.T) {
		mem, err := newWorkloadIdentityTokenRetriever(inMemorySettings("https://tokensvc"), credentials)
		require.NoError(t, err)
		file, err := newWorkloadIdentityTokenRetriever(&azsettings.AzureSettings{WorkloadIdentitySettings: &azsettings.WorkloadIdentitySettings{}}, credentials)
		require.NoError(t, err)
		assert.NotEqual(t, file.GetCacheKey("mt"), mem.GetCacheKey("mt"))
	})

	t.Run("in-memory requires an https assertion URL outside localhost", func(t *testing.T) {
		for _, u := range []string{"http://tokensvc", "ftp://tokensvc", "tokensvc", "https://"} {
			_, err := newWorkloadIdentityTokenRetriever(inMemorySettings(u), credentials)
			require.Error(t, err, u)
		}
		for _, u := range []string{"https://tokensvc", "http://localhost:8080", "http://127.0.0.1:8080", "http://[::1]:8080", "http://host.docker.internal:8081"} {
			_, err := newWorkloadIdentityTokenRetriever(inMemorySettings(u), credentials)
			require.NoError(t, err, u)
		}
	})

	t.Run("in-memory requires the assertion URL", func(t *testing.T) {
		_, err := newWorkloadIdentityTokenRetriever(inMemorySettings(""), credentials)
		require.ErrorContains(t, err, "assertion URL")
	})

	t.Run("in-memory requires tenant and client IDs", func(t *testing.T) {
		_, err := newWorkloadIdentityTokenRetriever(inMemorySettings("https://tokensvc"), &azcredentials.AzureWorkloadIdentityCredentials{})
		require.ErrorContains(t, err, "tenant ID and a client ID")
	})

	t.Run("in-memory falls back to server-level tenant and client IDs", func(t *testing.T) {
		settings := inMemorySettings("https://tokensvc")
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
		settings := inMemorySettings("https://tokensvc")
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
	mu         sync.Mutex
	assertions []string
	clientIDs  []string

	// perAssertion makes the access token depend on the assertion, so a test can
	// tell whose token it received.
	perAssertion bool
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
		f.mu.Lock()
		f.assertions = append(f.assertions, form["client_assertion"])
		f.clientIDs = append(f.clientIDs, form["client_id"])
		f.mu.Unlock()
		accessToken := "access-token-from-entra"
		if f.perAssertion {
			accessToken = "access-for-" + form["client_assertion"]
		}
		return respond(200, `{"token_type":"Bearer","expires_in":3600,"ext_expires_in":3600,"access_token":"`+accessToken+`"}`)
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

	token, err := r.GetAccessToken(contextWithNamespace("stacks-42"), []string{"https://management.azure.com/.default"})
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

func TestWorkloadIdentityInMemory_BlocksWithoutAStack(t *testing.T) {
	calls := 0
	tokensvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "assertion"})
	}))
	defer tokensvc.Close()

	settings := inMemorySettings(tokensvc.URL)
	retriever, err := newWorkloadIdentityTokenRetriever(settings, &azcredentials.AzureWorkloadIdentityCredentials{TenantId: testTenantID, ClientId: testClientID})
	require.NoError(t, err)
	require.NoError(t, retriever.Init())

	for _, ctx := range []context.Context{context.Background(), contextWithNamespace("default")} {
		_, err = retriever.GetAccessToken(ctx, []string{"https://management.azure.com/.default"})
		require.ErrorContains(t, err, "requires a Grafana Cloud stack")
	}
	assert.Equal(t, 0, calls, "no assertion should be requested without a stack")
}

// With the in-memory retriever selected from env settings there is no toggle in the request
// context and Grafana may send no multi-tenant ID. Two stacks sharing the identity and service
// settings must still each get their own assertion, not the other's cached token.
func TestWorkloadIdentityInMemory_StacksDoNotShareACacheEntry(t *testing.T) {
	var stacks []string
	tokensvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stacks = append(stacks, r.Header.Get(stackHeader))
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "assertion-for-" + r.Header.Get(stackHeader)})
	}))
	defer tokensvc.Close()

	retriever, err := newWorkloadIdentityTokenRetriever(
		inMemorySettings(tokensvc.URL),
		&azcredentials.AzureWorkloadIdentityCredentials{TenantId: testTenantID, ClientId: testClientID},
	)
	require.NoError(t, err)
	entra := &fakeEntra{}
	retriever.(*workloadIdentityInMemoryTokenRetriever).clientOptions = azcore.ClientOptions{Transport: entra}

	cache := NewConcurrentTokenCache()
	scopes := []string{"https://management.azure.com/.default"}
	for _, ns := range []string{"stacks-42", "stacks-77", "stacks-42"} {
		_, err := cache.GetAccessToken(contextWithNamespace(ns), retriever, scopes)
		require.NoError(t, err, ns)
	}

	assert.Equal(t, []string{"42", "77"}, stacks, "each stack mints once; the repeat for 42 is served from its own entry")
}

func newStackScopedTestRetriever(t *testing.T, tokensvcURL string, entra *fakeEntra) TokenRetriever {
	t.Helper()
	retriever, err := newWorkloadIdentityTokenRetriever(
		inMemorySettings(tokensvcURL),
		&azcredentials.AzureWorkloadIdentityCredentials{TenantId: testTenantID, ClientId: testClientID},
	)
	require.NoError(t, err)
	retriever.(*workloadIdentityInMemoryTokenRetriever).clientOptions = azcore.ClientOptions{Transport: entra}
	return retriever
}

func stackEchoingTokenService(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "assertion-for-" + r.Header.Get(stackHeader)})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Run with -race. Every stack gets its own cache entry, but they all initialize and use the one
// retriever.
func TestWorkloadIdentityInMemory_ConcurrentStacksAreRaceFree(t *testing.T) {
	tokensvc := stackEchoingTokenService(t)
	retriever := newStackScopedTestRetriever(t, tokensvc.URL, &fakeEntra{})
	cache := NewConcurrentTokenCache()
	scopes := []string{"https://management.azure.com/.default"}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := cache.GetAccessToken(contextWithNamespace(fmt.Sprintf("stacks-%d", i%5)), retriever, scopes)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
}

// After stack 77 has used the retriever, refreshing stack 42's token must still yield 42's token.
func TestWorkloadIdentityInMemory_RefreshStaysWithItsStack(t *testing.T) {
	tokensvc := stackEchoingTokenService(t)
	retriever := newStackScopedTestRetriever(t, tokensvc.URL, &fakeEntra{perAssertion: true})
	cache := NewConcurrentTokenCache()
	scopes := []string{"https://management.azure.com/.default"}

	token42, err := cache.GetAccessToken(contextWithNamespace("stacks-42"), retriever, scopes)
	require.NoError(t, err)
	require.Equal(t, "access-for-assertion-for-42", token42)
	_, err = cache.GetAccessToken(contextWithNamespace("stacks-77"), retriever, scopes)
	require.NoError(t, err)

	// Let the cache consider 42's token due for a refresh, while Entra's token is still valid.
	realNow := timeNow
	timeNow = func() time.Time { return realNow().Add(59 * time.Minute) }
	defer func() { timeNow = realNow }()

	refreshed, err := cache.GetAccessToken(contextWithNamespace("stacks-42"), retriever, scopes)
	require.NoError(t, err)
	assert.Equal(t, "access-for-assertion-for-42", refreshed, "stack 42 must not receive stack 77's token")
}
