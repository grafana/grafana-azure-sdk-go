package aztokenprovider

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/grafana/grafana-azure-sdk-go/v2/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/v2/azsettings"
	"software.sslmate.com/src/go-pkcs12"
)

type clientCertificateTokenRetriever struct {
	cloudConf          cloud.Configuration
	tenantId           string
	clientId           string
	certificateFormat  string
	clientCertificate  string
	privateKey         string
	certificatePassword string
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
		certificateFormat:  credentials.CertificateFormat,
		clientCertificate:  credentials.ClientCertificate,
		privateKey:         credentials.PrivateKey,
		certificatePassword: credentials.CertificatePassword,
	}, nil
}

func (c *clientCertificateTokenRetriever) GetCacheKey(grafanaMultiTenantId string) string {
	return fmt.Sprintf("azure|clientcertificate|%s|%s|%s|%s|%s", c.cloudConf.ActiveDirectoryAuthorityHost, c.tenantId, c.clientId, hashSecret(c.clientCertificate), grafanaMultiTenantId)
}

func (c *clientCertificateTokenRetriever) Init() error {
	var joinedKeyCert []byte
	var certs []*x509.Certificate
	var key crypto.PrivateKey
	var err error

	switch c.certificateFormat {
	case "pem":
		// Join private key and certificate into a single string as they should be parsed together
		joinedKeyCert = []byte(c.privateKey + "\n" + c.clientCertificate)
		certs, key, err = azidentity.ParseCertificates([]byte(joinedKeyCert), []byte(c.certificatePassword))
	case "pfx":
		// If we have a password, we need to decode the private key and use the pkcs12 library to parse the certificate and private key
		// We only accept pfx files that are base64 encoded
		clientCertificateDecoded, err := base64.StdEncoding.DecodeString(c.clientCertificate)
		if err != nil {
			return err
		}
		privateKey, cert, caCerts, err := pkcs12.DecodeChain(clientCertificateDecoded, c.certificatePassword)
		if err != nil {
			return err
		}
		certs = append(certs, cert)
		certs = append(certs, caCerts...)
		key = privateKey
	}
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
