package azsettings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCustomClouds = []*AzureCloudSettings{
	{
		Name:         "CustomCloud1",
		DisplayName:  "Custom Cloud 1",
		AadAuthority: "https://login.contoso.com/",
		Properties: map[string]string{
			"azureDataExplorerSuffix": ".kusto.cloud1.contoso.com",
			"logAnalytics":            "https://api.loganalytics.cloud1.contoso.com",
			"portal":                  "https://portal.azure.cloud1.contoso.com",
			"prometheusResourceId":    "https://prometheus.monitor.azure.cloud1.contoso.com",
			"resourceManager":         "https://management.azure.cloud1.contoso.com",
		},
	},
	{
		Name:         "CustomCloud2",
		DisplayName:  "Custom Cloud 2",
		AadAuthority: "https://login.cloud2.contoso.com/",
		Properties: map[string]string{
			"azureDataExplorerSuffix": ".kusto.cloud2.contoso.com",
			"logAnalytics":            "https://api.loganalytics.cloud2.contoso.com",
			"portal":                  "https://portal.azure.cloud2.contoso.com",
			"prometheusResourceId":    "https://prometheus.monitor.cloud2.azure.contoso.com",
			"resourceManager":         "https://management.azure.cloud2.contoso.com",
		},
	},
}

func TestGetCloudsNoCustomClouds(t *testing.T) {
	settings := &AzureSettings{}

	clouds := settings.Clouds()

	assert.Len(t, clouds, 3)
	assert.Equal(t, clouds[0].Name, "AzureCloud")
	assert.Equal(t, clouds[1].Name, "AzureChinaCloud")
	assert.Equal(t, clouds[2].Name, "AzureUSGovernment")
}

func TestGetCloudsWithCustomClouds(t *testing.T) {
	settings := &AzureSettings{}
	settings.customClouds = testCustomClouds

	// should merge predefined and custom clouds into one list
	clouds := settings.Clouds()

	assert.Len(t, clouds, 5)
	assert.Equal(t, clouds[0].Name, "AzureCloud")
	assert.Equal(t, clouds[1].Name, "AzureChinaCloud")
	assert.Equal(t, clouds[2].Name, "AzureUSGovernment")
	assert.Equal(t, clouds[3].Name, "CustomCloud1")
	assert.Equal(t, clouds[4].Name, "CustomCloud2")
}

func TestGetCustomClouds(t *testing.T) {
	settings := &AzureSettings{}
	settings.customClouds = testCustomClouds

	// should return ONLY the custom clouds
	clouds := settings.CustomClouds()

	assert.Len(t, clouds, len(testCustomClouds))
}

func TestGetCloud(t *testing.T) {
	settings := &AzureSettings{}

	t.Run("should return cloud settings", func(t *testing.T) {
		cloud, err := settings.GetCloud(AzurePublic)
		require.NoError(t, err)

		assert.Equal(t, AzurePublic, cloud.Name)
		assert.Equal(t, "https://management.azure.com", cloud.Properties["resourceManager"])
	})

	t.Run("should return error if cloud not found", func(t *testing.T) {
		_, err := settings.GetCloud("InvalidCloud")
		assert.Error(t, err)
	})
}

func TestSetCustomClouds(t *testing.T) {
	settings := &AzureSettings{}

	json := `[
		{
			"name":"CustomCloud1",
			"displayName":"Custom Cloud 1",
			"aadAuthority":"https://login.contoso.com/",
			"properties":{
				"azureDataExplorerSuffix":".kusto.cloud1.contoso.com",
				"logAnalytics":"https://api.loganalytics.cloud1.contoso.com",
				"portal":"https://portal.azure.cloud1.contoso.com",
				"prometheusResourceId":"https://prometheus.monitor.azure.cloud1.contoso.com",
				"resourceManager":"https://management.azure.cloud1.contoso.com"
			}
		}
	]`

	err := settings.SetCustomClouds(json)
	assert.Nil(t, err)

	clouds := settings.getCustomClouds()

	assert.Len(t, clouds, 1)
	cloud := clouds[0]
	assert.Equal(t, cloud.Name, "CustomCloud1")
	assert.Equal(t, cloud.DisplayName, "Custom Cloud 1")
	assert.Equal(t, cloud.AadAuthority, "https://login.contoso.com/")
	assert.Len(t, cloud.Properties, 5)

	cloud2, err := settings.GetCloud("CustomCloud1")
	assert.Nil(t, err)
	assert.Equal(t, cloud2.Name, "CustomCloud1")	
}
