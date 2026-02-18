package aztokenprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
)

type clientCertificateTokenRetriever struct {
	cloudConf          cloud.Configuration
	tenantId           string
	clientId           string
	clientCertificate  string
	privateKey         string
	privateKeyPassword string
	credential         azcore.TokenCredential
}

func getClientCertificateTokenRetriever(settings *azsettings.AzureSettings, credentials *azcredentials.AzureClientCertificateCredentials) (TokenRetriever, error) {
	var authorityHost string

	if credentials.Authority != "" {
		// Use AAD authority endpoint configured in credentials
		authorityHost = credentials.Authority
	} else {
		// Resolve cloud settings for the given cloud name
		cloudSettings, err := settings.GetCloud(credentials.AzureCloud)
		if err != nil {
			return nil, err
		}
		authorityHost = cloudSettings.AadAuthority
	}

	return &clientCertificateTokenRetriever{
		cloudConf: cloud.Configuration{
			ActiveDirectoryAuthorityHost: authorityHost,
			Services:                     map[cloud.ServiceName]cloud.ServiceConfiguration{},
		},
		tenantId:           credentials.TenantId,
		clientId:           credentials.ClientId,
		clientCertificate:  credentials.ClientCertificate,
		privateKey:         credentials.PrivateKey,
		privateKeyPassword: credentials.PrivateKeyPassword,
	}, nil
}

func (c *clientCertificateTokenRetriever) GetCacheKey(grafanaMultiTenantId string) string {
	return fmt.Sprintf("azure|clientcertificate|%s|%s|%s|%s|%s", c.cloudConf.ActiveDirectoryAuthorityHost, c.tenantId, c.clientId, hashSecret(c.clientCertificate), grafanaMultiTenantId)
}

func (c *clientCertificateTokenRetriever) Init() error {
	// Join private key and certificate into a single string as they should be parsed together
	joinedKeyCert := []byte(c.privateKey + "\n" + c.clientCertificate)
	certs, key, err := azidentity.ParseCertificates([]byte(joinedKeyCert), []byte(c.privateKeyPassword))
	if err != nil {
		return err
	}

	options := azidentity.ClientCertificateCredentialOptions{}
	options.Cloud = c.cloudConf
	if credential, err := azidentity.NewClientCertificateCredential(c.tenantId, c.clientId, certs, key, &options); err != nil {
		return err
	} else {
		c.credential = credential
		return nil
	}
}

func (c *clientCertificateTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: accessToken.Token, ExpiresOn: accessToken.ExpiresOn}, nil
}

// Empty implementation
func (c *clientCertificateTokenRetriever) GetExpiry() *time.Time {
	return nil
}
