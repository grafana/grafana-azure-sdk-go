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
	baseURL     string
	callerToken string
	stackID     string
	client      *http.Client
}

// stackHeader carries the stack the request is for. The assertion service
// ignores it unless the caller's credential is allowed to name its stack.
const stackHeader = "X-Grafana-Stack"

// NewHTTPAssertionProvider returns an AssertionProvider backed by the token
// service at baseURL. callerToken identifies this process to the service.
//
// The stack sent with each request is the one Grafana reports in the request's plugin
// context (see stackIDFromContext). stackID (optional) is only the fallback for when
// Grafana reports none, for example a local or single-tenant Grafana.
func NewHTTPAssertionProvider(baseURL, callerToken, stackID string, client *http.Client) (AssertionProvider, error) {
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
// assertion are credentials. Plain http is accepted only for a loopback host, for
// local development. The assertion service is hosted over https today; if that
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
		if host := u.Hostname(); host == "localhost" || net.ParseIP(host).IsLoopback() {
			return nil
		}
	}
	return fmt.Errorf("workload identity assertion URL must use https (http is only allowed for localhost)")
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
	if p.callerToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.callerToken)
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
