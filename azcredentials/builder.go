package azcredentials

import (
	"fmt"

	"github.com/grafana/grafana-azure-sdk-go/util/maputil"
)

func FromDatasourceData(data map[string]interface{}, secureData map[string]string) (AzureCredentials, error) {
	if credentialsObj, err := maputil.GetMapOptional(data, "azureCredentials"); err != nil {
		return nil, err
	} else if credentialsObj == nil {
		return nil, nil
	} else {
		return getFromCredentialsObject(credentialsObj, secureData)
	}
}

func getFromCredentialsObject(credentialsObj map[string]interface{}, secureData map[string]string) (AzureCredentials, error) {
	authType, err := maputil.GetString(credentialsObj, "authType")
	if err != nil {
		return nil, err
	}

	switch authType {
	case AzureAuthCurrentUserIdentity:
		credentials := &AadCurrentUserCredentials{}
		return credentials, nil

	case AzureAuthManagedIdentity:
		credentials := &AzureManagedIdentityCredentials{}
		return credentials, nil

	case AzureAuthClientSecret:
		cloud, err := maputil.GetString(credentialsObj, "azureCloud")
		if err != nil {
			return nil, err
		}
		tenantId, err := maputil.GetString(credentialsObj, "tenantId")
		if err != nil {
			return nil, err
		}
		clientId, err := maputil.GetString(credentialsObj, "clientId")
		if err != nil {
			return nil, err
		}
		clientSecret := secureData["azureClientSecret"]

		credentials := &AzureClientSecretCredentials{
			AzureCloud:   cloud,
			TenantId:     tenantId,
			ClientId:     clientId,
			ClientSecret: clientSecret,
		}
		return credentials, nil

	case AzureAuthClientSecretObo:
		cloud, err := maputil.GetString(credentialsObj, "azureCloud")
		if err != nil {
			return nil, err
		}
		tenantId, err := maputil.GetString(credentialsObj, "tenantId")
		if err != nil {
			return nil, err
		}
		clientId, err := maputil.GetString(credentialsObj, "clientId")
		if err != nil {
			return nil, err
		}
		clientSecret := secureData["azureClientSecret"]

		credentials := &AzureClientSecretOboCredentials{
			ClientSecretCredentials: AzureClientSecretCredentials{
				AzureCloud:   cloud,
				TenantId:     tenantId,
				ClientId:     clientId,
				ClientSecret: clientSecret,
			},
		}
		return credentials, nil

	default:
		err := fmt.Errorf("the authentication type '%s' not supported", authType)
		return nil, err
	}
}
