package aztokenprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// AssertionProvider returns a signed client assertion that Entra ID accepts for
// the federated credential configured on the given managed identity. The
// assertion is a short-lived bearer credential: callers must keep it in memory
// and must not log or persist it.
type AssertionProvider interface {
	GetAssertion(ctx context.Context, tenantID, clientID string) (string, error)
}

const maxAssertionResponseSize = 1 << 20

// httpAssertionProvider asks an assertion service (the token service) to mint
// an assertion. The service decides who the caller is from the bearer token
// and derives subject and audience itself; this client only names the identity
// it wants to authenticate as.
type httpAssertionProvider struct {
	baseURL string
	// callerToken returns the credential that identifies this process to the service. It is a
	// function because the token may be replaced while the process runs.
	callerToken func() (string, error)
	stackID     string
	client      *http.Client
}

// stackHeader carries the stack the request is for. The assertion service
// ignores it unless the caller's credential is allowed to name its stack.
//
// TODO(experimental): temporary. The service trusts this header only from a platform workload on
// its list. When it is hosted inside Grafana it is expected to take the stack from the ID token.
const stackHeader = "X-Grafana-Stack"

// NewHTTPAssertionProvider returns an AssertionProvider backed by the token
// service at baseURL. callerToken identifies this process to the service.
//
// The stack sent with each request is the one Grafana reports in the request's plugin
// context (see stackIDFromContext). stackID (optional) is only the fallback for when
// Grafana reports none, for example a local or single-tenant Grafana.
func NewHTTPAssertionProvider(baseURL, callerToken, stackID string, client *http.Client) (AssertionProvider, error) {
	return newHTTPAssertionProvider(baseURL, func() (string, error) { return callerToken, nil }, stackID, client)
}

// maxCallerTokenFileSize bounds what is read from a caller token file.
const maxCallerTokenFileSize = 64 << 10

// NewHTTPAssertionProviderWithTokenFile is like NewHTTPAssertionProvider, but the caller token
// is read from tokenFile each time an assertion is requested. Use it for a token that is
// replaced while the process runs, such as a Kubernetes projected service account token.
//
// TODO(experimental): temporary, until the token service is moved off Azure Container Apps and
// hosted inside Grafana. There it is expected to authenticate callers with the ID token Grafana
// forwards with each request, so the caller token, this file and the stack header go away.
func NewHTTPAssertionProviderWithTokenFile(baseURL, tokenFile, stackID string, client *http.Client) (AssertionProvider, error) {
	if tokenFile == "" {
		return nil, fmt.Errorf("workload identity caller token file is not configured")
	}
	return newHTTPAssertionProvider(baseURL, func() (string, error) { return readCallerTokenFile(tokenFile) }, stackID, client)
}

// readCallerTokenFile reads the token without ever including it in an error.
func readCallerTokenFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read workload identity caller token file: %w", err)
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, maxCallerTokenFileSize+1))
	if err != nil {
		return "", fmt.Errorf("read workload identity caller token file: %w", err)
	}
	if len(b) > maxCallerTokenFileSize {
		return "", fmt.Errorf("workload identity caller token file is too large")
	}
	token := strings.TrimSpace(string(b))
	if token == "" {
		return "", fmt.Errorf("workload identity caller token file is empty")
	}
	return token, nil
}

func newHTTPAssertionProvider(baseURL string, callerToken func() (string, error), stackID string, client *http.Client) (AssertionProvider, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("workload identity assertion URL is not configured")
	}
	if err := validateAssertionURL(baseURL); err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &httpAssertionProvider{
		baseURL:     strings.TrimRight(baseURL, "/"),
		callerToken: callerToken,
		stackID:     stackID,
		client:      client,
	}, nil
}

// validateAssertionURL requires https, because the caller token and the returned
// assertion are credentials. Plain http is accepted only for local development:
// a loopback host, or host.docker.internal, which is how a container reaches the
// machine it runs on. The assertion service is hosted over https today; if that
// ever changes, this is the place to relax the rule.
func validateAssertionURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid workload identity assertion URL: %w", err)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("invalid workload identity assertion URL: missing host")
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if host := u.Hostname(); host == "localhost" || host == "host.docker.internal" || net.ParseIP(host).IsLoopback() {
			return nil
		}
	}
	return fmt.Errorf("workload identity assertion URL must use https (http is only allowed for localhost and host.docker.internal)")
}

func (p *httpAssertionProvider) GetAssertion(ctx context.Context, tenantID, clientID string) (string, error) {
	body, err := json.Marshal(map[string]string{"tenant_id": tenantID, "client_id": clientID})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/mint", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	callerToken, err := p.callerToken()
	if err != nil {
		return "", err
	}
	if callerToken != "" {
		req.Header.Set("Authorization", "Bearer "+callerToken)
	}
	stack := stackIDFromContext(ctx)
	if stack == "" {
		stack = p.stackID
	}
	if stack != "" {
		req.Header.Set(stackHeader, stack)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request workload identity assertion: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxAssertionResponseSize))
	if err != nil {
		return "", fmt.Errorf("read workload identity assertion response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &e) == nil && e.Code != "" {
			return "", fmt.Errorf("assertion service returned %s: %s", e.Code, e.Message)
		}
		return "", fmt.Errorf("assertion service returned %s", resp.Status)
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode workload identity assertion response: %w", err)
	}
	if out.Token == "" {
		return "", fmt.Errorf("assertion service returned an empty token")
	}
	return out.Token, nil
}
