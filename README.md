# github.com/grafana/grafana-azure-sdk-go

SDK for integration of Grafana datasources with Azure services.

## Packages

### azsettings

Common Azure configuration. Can be read from either the environment variables of the Grafana instance (if supplied to the plugin) or from the context supplied to the plugin (if available).

This can be achieved by making use of `ReadSettings` which will determine the settings based on the available context.

**Note:** If the plugin context contains any Azure related variable then it will be used in place of any environment variables present.

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
authOpts.Scopes([]string{"https://datasource.example.org/.default"})

// Optionally, register custom token providers
authOpts.AddTokenProvider("custom-auth-type", func (...) (aztokenprovider.AzureTokenProvider, error) {
	return NewCustomTokenProvider(...), nil
})

// Configure the client
clientOpts := httpclient.Options{}
azhttpclient.AddAzureAuthentication(&clientOpts, authOpts, credentials)

httpClient, err := httpclient.NewProvider().New(clientOpts)
```

### azusercontext

Context object `CurrentUserContext` of the currently signed-in Grafana user which can be passed
via context between business layers.

Used by token provider to get information about the current user for user identity authentication.

Read/write functions:

- `context = azusercontext.WithCurrentUser(context, currentUser)` extends given context with information about the current user.
- `currentUser = azusercontext.GetCurrentUser(context)` extracts current user from the given context

Helper functions for datasource requests:

- `WithUserFromQueryReq` extracts current user from query request and adds to context.
- `WithUserFromResourceReq` extracts current user from resource call and adds to context.
- `WithUserFromHealthCheckReq` extracts current from health check request and adds to context.

### aztokenprovider

### util

- `maputil`

## License

[Apache 2.0 License](https://github.com/grafana/azure-sdk-go/blob/master/LICENSE)
