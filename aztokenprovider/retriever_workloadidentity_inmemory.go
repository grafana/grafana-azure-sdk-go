package aztokenprovider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
)

// workloadIdentityInMemoryTokenRetriever is the workloadIdentityInMemory
// variant of the workload identity retriever. It authenticates to Entra ID with
// a client assertion obtained from an AssertionProvider on demand instead of
// reading one from a token file, so no credential is ever written to disk.
//
// azidentity invokes the assertion callback only when it needs a fresh access
// token, so an assertion is minted once per token refresh.
type workloadIdentityInMemoryTokenRetriever struct {
	tenantId string
	clientId string
	provider AssertionProvider

	// assertionURL and callerTokenHash identify the assertion service
	// configuration the retriever was built with. They are part of the cache key
	// so that rotating either yields a new retriever instead of the cached one.
	// Only a hash of the caller token is kept, so the key never holds the secret.
	assertionURL    string
	callerTokenHash string
	stackID         string

	// clientOptions lets tests replace the HTTP transport.
	clientOptions azcore.ClientOptions
	credential    azcore.TokenCredential
}

// newWorkloadIdentityTokenRetriever picks the workload identity implementation.
// Without the workloadIdentityInMemory toggle it returns exactly what
// getWorkloadIdentityTokenRetriever always returned (token file based).
func newWorkloadIdentityTokenRetriever(settings *azsettings.AzureSettings, credentials *azcredentials.AzureWorkloadIdentityCredentials) (TokenRetriever, error) {
	if wiSettings := settings.WorkloadIdentitySettings; wiSettings != nil && wiSettings.InMemory {
		return getWorkloadIdentityInMemoryTokenRetriever(settings, credentials)
	}
	return getWorkloadIdentityTokenRetriever(settings, credentials), nil
}

func getWorkloadIdentityInMemoryTokenRetriever(settings *azsettings.AzureSettings, credentials *azcredentials.AzureWorkloadIdentityCredentials) (TokenRetriever, error) {
	wiSettings := settings.WorkloadIdentitySettings
	if wiSettings == nil {
		return nil, fmt.Errorf("workload identity settings are not available")
	}

	// Unlike the token file path, there is no environment to fall back on
	// (AZURE_TENANT_ID / AZURE_CLIENT_ID): the identity must be named.
	tenantId := wiSettings.TenantId
	clientId := wiSettings.ClientId
	if credentials != nil {
		if credentials.TenantId != "" {
			tenantId = credentials.TenantId
		}
		if credentials.ClientId != "" {
			clientId = credentials.ClientId
		}
	}
	if tenantId == "" || clientId == "" {
		return nil, fmt.Errorf("workload identity in memory requires a tenant ID and a client ID")
	}

	provider, err := NewHTTPAssertionProvider(wiSettings.AssertionURL, wiSettings.AssertionCallerToken, wiSettings.StackID, &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		return nil, err
	}

	return &workloadIdentityInMemoryTokenRetriever{
		tenantId: tenantId,
		clientId: clientId,
		provider: provider,

		assertionURL:    wiSettings.AssertionURL,
		callerTokenHash: hashSecret(wiSettings.AssertionCallerToken),
		stackID:         wiSettings.StackID,
	}, nil
}

func (c *workloadIdentityInMemoryTokenRetriever) GetCacheKey(grafanaMultiTenantId string) string {
	return fmt.Sprintf("azure|wi-mem|%s|%s|%s|%s|%s|%s", c.tenantId, c.clientId, c.assertionURL, c.callerTokenHash, c.stackID, grafanaMultiTenantId)
}

func (c *workloadIdentityInMemoryTokenRetriever) Init() error {
	getAssertion := func(ctx context.Context) (string, error) {
		return c.provider.GetAssertion(ctx, c.tenantId, c.clientId)
	}
	credential, err := azidentity.NewClientAssertionCredential(c.tenantId, c.clientId, getAssertion, &azidentity.ClientAssertionCredentialOptions{
		ClientOptions: c.clientOptions,
	})
	if err != nil {
		return err
	}
	c.credential = credential
	return nil
}

// scopedToStack opts in to keeping cache entries apart per stack (see stackScopedTokenRetriever).
func (c *workloadIdentityInMemoryTokenRetriever) scopedToStack() {}

func (c *workloadIdentityInMemoryTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	// Without a stack the assertion service cannot tell whose identity to mint for,
	// so fail here instead of attempting an exchange.
	if stackIDFromContext(ctx) == "" && c.stackID == "" {
		return nil, fmt.Errorf("workload identity in memory requires a Grafana Cloud stack, but none was found in the request or the configuration")
	}

	accessToken, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: accessToken.Token, ExpiresOn: accessToken.ExpiresOn}, nil
}

// Empty implementation
func (c *workloadIdentityInMemoryTokenRetriever) GetExpiry() *time.Time {
	return nil
}
