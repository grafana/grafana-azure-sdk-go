package aztokenprovider

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/azsettings"
)

type workloadIdentityTokenRetriever struct {
	tenantId   string
	clientId   string
	credential azcore.TokenCredential
}

func getWorkloadIdentityTokenRetriever(settings *azsettings.AzureSettings, credentials *azcredentials.AzureWorkloadIdentityCredentials) TokenRetriever {
	var tenantId, clientId string
	// TODO: See https://azure.github.io/azure-workload-identity/docs/faq.html#how-to-federate-multiple-identities-with-a-kubernetes-service-account
	// if credentials.ClientId != "" {
	// 	clientId = credentials.ClientId
	// } else {
	// 	clientId = settings.ManagedIdentityClientId
	// }
	tenantId = ""
	clientId = ""

	return &workloadIdentityTokenRetriever{
		tenantId: tenantId,
		clientId: clientId,
	}
}

func (c *workloadIdentityTokenRetriever) GetCacheKey() string {
	// TODO: Review the caching key
	clientId := c.clientId
	if clientId == "" {
		clientId = "system"
	}
	return fmt.Sprintf("azure|wi|%s", clientId)
}

func (c *workloadIdentityTokenRetriever) Init() error {
	options := &azidentity.WorkloadIdentityCredentialOptions{}
	// TODO: See https://azure.github.io/azure-workload-identity/docs/faq.html#how-to-federate-multiple-identities-with-a-kubernetes-service-account
	if c.tenantId != "" {
		options.TenantID = c.tenantId
	}
	if c.clientId != "" {
		options.ClientID = c.clientId
	}

	credential, err := azidentity.NewWorkloadIdentityCredential(options)
	if err != nil {
		return err
	} else {
		c.credential = credential
		return nil
	}
}

func (c *workloadIdentityTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: accessToken.Token, ExpiresOn: accessToken.ExpiresOn}, nil
}
