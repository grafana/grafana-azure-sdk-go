package aztokenprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
)

type federatedIdentityTokenRetriever struct {
	sourceClientId              string
	targetTenantId              string
	targetClientId              string
	federatedCredentialAudience string

	credential azcore.TokenCredential
}

func getFederatedIdentityTokenRetriever(settings *azsettings.AzureSettings, credentials *azcredentials.AzureFederatedIdentityCredentials) (TokenRetriever, error) {
	if credentials.TargetTenantId == "" {
		return nil, fmt.Errorf("target tenant ID is required for federated identity authentication")
	}
	if credentials.TargetClientId == "" {
		return nil, fmt.Errorf("target client ID is required for federated identity authentication")
	}
	if credentials.FederatedCredentialAudience == "" {
		return nil, fmt.Errorf("federated credential audience is required for federated identity authentication")
	}
	if err := validateFederatedCredentialAudience(credentials.FederatedCredentialAudience); err != nil {
		return nil, err
	}

	sourceClientId := credentials.SourceClientId
	if sourceClientId == "" {
		sourceClientId = settings.ManagedIdentityClientId
	}

	return &federatedIdentityTokenRetriever{
		sourceClientId:              sourceClientId,
		targetTenantId:              credentials.TargetTenantId,
		targetClientId:              credentials.TargetClientId,
		federatedCredentialAudience: credentials.FederatedCredentialAudience,
	}, nil
}

func (c *federatedIdentityTokenRetriever) GetCacheKey(grafanaMultiTenantId string) string {
	sourceClientId := c.sourceClientId
	if sourceClientId == "" {
		sourceClientId = "system"
	}
	return fmt.Sprintf("azure|fic|%s|%s|%s|%s|%s",
		sourceClientId, c.targetTenantId, c.targetClientId,
		c.federatedCredentialAudience, grafanaMultiTenantId)
}

func (c *federatedIdentityTokenRetriever) Init() error {
	options := &azidentity.ManagedIdentityCredentialOptions{}
	if c.sourceClientId != "" {
		options.ID = azidentity.ClientID(c.sourceClientId)
	}
	sourceCredential, err := azidentity.NewManagedIdentityCredential(options)
	if err != nil {
		return fmt.Errorf("failed to create managed identity credential for federated identity: %w", err)
	}

	audience := c.federatedCredentialAudience
	getAssertion := func(ctx context.Context) (string, error) {
		tk, err := sourceCredential.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{fmt.Sprintf("%s/.default", audience)},
		})
		if err != nil {
			return "", fmt.Errorf("failed to get source identity token for federated identity exchange: %w", err)
		}
		return tk.Token, nil
	}

	credential, err := azidentity.NewClientAssertionCredential(c.targetTenantId, c.targetClientId, getAssertion, nil)
	if err != nil {
		return fmt.Errorf("failed to create client assertion credential for federated identity: %w", err)
	}

	c.credential = credential
	return nil
}

func (c *federatedIdentityTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: accessToken.Token, ExpiresOn: accessToken.ExpiresOn}, nil
}

func (c *federatedIdentityTokenRetriever) GetExpiry() *time.Time {
	return nil
}
