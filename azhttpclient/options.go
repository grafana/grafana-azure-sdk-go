package azhttpclient

import (
	"github.com/grafana/grafana-azure-sdk-go/azcredentials"
	"github.com/grafana/grafana-azure-sdk-go/azsettings"
	sdkhttpclient "github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

func AddAzureAuthentication(opts *sdkhttpclient.Options, settings *azsettings.AzureSettings, credentials azcredentials.AzureCredentials, scopes []string) {
	opts.Middlewares = append(opts.Middlewares, AzureMiddleware(settings, credentials, scopes))
}
