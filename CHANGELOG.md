# Change Log

## v2.1.1

- Dependency updates

## v2.1.0

- Add Azure Entra Password as a credentials type. This is not a recommended authentication method and is currently only supported for MSSQL. If you require this method the token provider will need to be implemented separately.

## v2.0.0

- **Breaking change:** `mapUtil` is removed [#106](https://github.com/grafana/grafana-azure-sdk-go/pull/106). `mapUtil` functions moved to 
  [grafana-plugin-sdk-go](https://github.com/grafana/grafana-plugin-sdk-go/tree/main/data/utils/maputil)

## v1.6.0

- **Breaking change:** Configurable authentication middleware with `AuthOptions` configuration struct [#28](https://github.com/grafana/grafana-azure-sdk-go/pull/28).
- New context object `CurrentUserContext` to carry currently signed-in user identity [#30](https://github.com/grafana/grafana-azure-sdk-go/pull/30).
