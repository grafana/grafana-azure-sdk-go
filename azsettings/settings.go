package azsettings

type AzureSettings struct {
	Cloud                   string
	ManagedIdentityEnabled  bool
	ManagedIdentityClientId string
}

func (settings *AzureSettings) GetDefaultCloud() string {
	cloudName := settings.Cloud
	if cloudName == "" {
		return AzurePublic
	}
	return cloudName
}
