package aztokenprovider

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/azsettings"
)

type clientSecretTokenRetriever struct {
	cloudConf    cloud.Configuration
	tenantId     string
	clientId     string
	clientSecret string
	credential   azcore.TokenCredential
}

func getClientSecretTokenRetriever(credentials *azcredentials.AzureClientSecretCredentials) (TokenRetriever, error) {
	var cloudConf cloud.Configuration
	if credentials.Authority != "" {
		cloudConf.ActiveDirectoryAuthorityHost = credentials.Authority
	} else {
		var err error
		cloudConf, err = resolveCloudConfiguration(credentials.AzureCloud)
		if err != nil {
			return nil, err
		}
	}
	return &clientSecretTokenRetriever{
		cloudConf:    cloudConf,
		tenantId:     credentials.TenantId,
		clientId:     credentials.ClientId,
		clientSecret: credentials.ClientSecret,
	}, nil
}

func (c *clientSecretTokenRetriever) GetCacheKey() string {
	return fmt.Sprintf("azure|clientsecret|%s|%s|%s|%s", c.cloudConf.ActiveDirectoryAuthorityHost, c.tenantId, c.clientId, hashSecret(c.clientSecret))
}

func (c *clientSecretTokenRetriever) Init() error {
	options := azidentity.ClientSecretCredentialOptions{}
	options.Cloud = c.cloudConf
	if credential, err := azidentity.NewClientSecretCredential(c.tenantId, c.clientId, c.clientSecret, &options); err != nil {
		return err
	} else {
		c.credential = credential
		return nil
	}
}

func (c *clientSecretTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: scopes})
	if err != nil {
		return nil, err
	}

	return &AccessToken{Token: accessToken.Token, ExpiresOn: accessToken.ExpiresOn}, nil
}

func resolveCloudConfiguration(cloudName string) (cloud.Configuration, error) {
	// Known Azure clouds
	switch cloudName {
	case azsettings.AzurePublic:
		return cloud.AzurePublic, nil
	case azsettings.AzureChina:
		return cloud.AzureChina, nil
	case azsettings.AzureUSGovernment:
		return cloud.AzureGovernment, nil
	default:
		err := fmt.Errorf("the Azure cloud '%s' not supported", cloudName)
		return cloud.Configuration{}, err
	}
}

func hashSecret(secret string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(secret))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
