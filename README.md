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

### azusercontext

Context object `CurrentUserContext` of the currently signed-in Grafana user which can be passed
via context between business layers.

Used by token provider to get information about the current user for user identity authentication. 

Read/write functions:
- `context = azusercontext.WithCurrentUser(context, currentUser)` extends given context with information about the current user.
- `currentUser = azusercontext..GetCurrentUser(context)` extracts current user from the given context

Helper functions for datasource requests:
- `WithUserFromQueryReq` extracts current user from query request and adds to context. 
- `WithUserFromResourceReq` extracts current user from resource call and adds to context.
- `WithUserFromHealthCheckReq` extracts current from health check request and adds to context. 

### aztokenprovider

### util

- `maputil`

## License

[Apache 2.0 License](https://github.com/grafana/azure-sdk-go/blob/master/LICENSE)
