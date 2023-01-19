# github.com/grafana/grafana-azure-sdk-go

SDK for integration of Grafana datasources with Azure services.

## Packages

### azsettings

Common Azure configuration.

### azcredentials

The built-in `AzureCredentials`:
- `AadCurrentUserCredentials`
- `AzureManagedIdentityCredentials`
- `AzureClientSecretCredentials`
- `AzureClientSecretOboCredentials`

### azhttpclient

Azure authentication middleware for Grafana Plugin SDK `httpclient`.

#### Usage

```go
// Initialize the authentication options
authOpts := azhttpclient.NewAuthOptions(azureSettings)

// Configure instance-level scopes
authOpts.Scopes(new []string {"https://datasource.example.org/.default"})

// Optionally, register custom token providers
authOpts.AddTokenProvider("custom-auth-type, func (...) (aztokenprovider.AzureTokenProvider, error) {
	return NewCustomTokenProvider(...), nil
})

// Configure the client
clientOpts := httpclient.Options{}
azhttpclient.AddAzureAuthentication(&clientOpts, authOpts, credentials)

httpClient, err := httpclient.NewProvider().New(clientOpts)
```

### aztokenprovider

### util

- `maputil`

## License

[Apache 2.0 License](https://github.com/grafana/azure-sdk-go/blob/master/LICENSE)
