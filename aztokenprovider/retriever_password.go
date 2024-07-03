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

type passwordTokenRetriever struct {
	password   string
	userId     string
	clientId   string
	tenantID   string
	credential azcore.TokenCredential
}

func getPasswordTokenRetriever(settings *azsettings.AzureSettings, credentials *azcredentials.AzureClientPasswordCredentials) TokenRetriever {
	clientId := credentials.ClientId
	if credentials.ClientId == "" {
		clientId = settings.ManagedIdentityClientId
	}
	return &passwordTokenRetriever{
		password: credentials.Password,
		userId:   credentials.UserId,
		clientId: clientId,
		tenantID: credentials.TenantId,
	}
}

func (p *passwordTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := p.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: accessToken.Token, ExpiresOn: accessToken.ExpiresOn}, nil
}

func (p *passwordTokenRetriever) GetCacheKey(grafanaMultiTenantId string) string {
	return fmt.Sprintf("azure|password|%s|%s|%s|%s", p.userId, p.clientId, hashSecret(p.password), grafanaMultiTenantId)
}

func (p *passwordTokenRetriever) GetExpiry() *time.Time {
	return nil
}

func (p *passwordTokenRetriever) Init() error {
	options := &azidentity.UsernamePasswordCredentialOptions{}
	var err error
	p.credential, err = azidentity.NewUsernamePasswordCredential(p.tenantID, p.clientId, p.userId, p.password, options)
	if err != nil {
		return err
	}
	return nil

}
