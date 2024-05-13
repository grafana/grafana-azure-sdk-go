package azsettings

import (
	"encoding/json"
	"fmt"
)

type AzureCloudInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type AzureCloudSettings struct {
	Name         string            `json:"name"`
	DisplayName  string            `json:"displayName"`
	AadAuthority string            `json:"aadAuthority"`
	Properties   map[string]string `json:"properties"`
}

var predefinedClouds = []*AzureCloudSettings{
	{
		Name:         AzurePublic,
		DisplayName:  "Azure",
		AadAuthority: "https://login.microsoftonline.com/",
		Properties: map[string]string{
			"azureDataExplorerSuffix": ".kusto.windows.net",
			"logAnalytics":            "https://api.loganalytics.io",
			"portal":                  "https://portal.azure.com",
			"prometheusResourceId":    "https://prometheus.monitor.azure.com",
			"resourceManager":         "https://management.azure.com",
		},
	},
	{
		Name:         AzureChina,
		DisplayName:  "Azure China",
		AadAuthority: "https://login.chinacloudapi.cn/",
		Properties: map[string]string{
			"azureDataExplorerSuffix": ".kusto.chinacloudapi.cn",
			"logAnalytics":            "https://api.loganalytics.azure.cn",
			"portal":                  "https://portal.azure.cn",
			"prometheusResourceId":    "https://prometheus.monitor.azure.cn",
			"resourceManager":         "https://management.chinacloudapi.cn",
		},
	},
	{
		Name:         AzureUSGovernment,
		DisplayName:  "Azure US Government",
		AadAuthority: "https://login.microsoftonline.us/",
		Properties: map[string]string{
			"azureDataExplorerSuffix": ".kusto.usgovcloudapi.net",
			"logAnalytics":            "https://api.loganalytics.us",
			"portal":                  "https://portal.azure.us",
			"prometheusResourceId":    "https://prometheus.monitor.azure.us",
			"resourceManager":         "https://management.usgovcloudapi.net",
		},
	},
}

func (settings *AzureSettings) GetCloud(cloudName string) (*AzureCloudSettings, error) {
	clouds := settings.getClouds()

	for _, cloud := range clouds {
		if cloud.Name == cloudName {
			return cloud, nil
		}
	}

	return nil, fmt.Errorf("the Azure cloud '%s' is not supported", cloudName)
}

// Returns all clouds configured on the instance, including custom clouds if any
func (settings *AzureSettings) Clouds() []AzureCloudInfo {
	clouds := settings.getClouds()
	return mapCloudInfo(clouds)
}

// Returns only the custom clouds configured on the instance
func (settings *AzureSettings) CustomClouds() []AzureCloudInfo {
	return mapCloudInfo(settings.CustomCloudList)
}

// Parses the JSON list of custom clouds passed in, then stores the list on the instance
func (settings *AzureSettings) SetCustomClouds(customCloudsJSON string) error {
	var customClouds []*AzureCloudSettings
	if err := json.Unmarshal([]byte(customCloudsJSON), &customClouds); err != nil {
		return err
	}

	settings.CustomCloudList = customClouds
	return nil
}

func mapCloudInfo(clouds []*AzureCloudSettings) []AzureCloudInfo {
	results := make([]AzureCloudInfo, 0, len(clouds))
	for _, cloud := range clouds {
		results = append(results, AzureCloudInfo{
			Name:        cloud.Name,
			DisplayName: cloud.DisplayName,
		})
	}

	return results
}

func (settings *AzureSettings) getClouds() []*AzureCloudSettings {
	clouds := settings.CustomCloudList
	if len(settings.CustomCloudList) > 0 {
		allClouds := append(predefinedClouds, clouds...)
		return allClouds
	}

	return predefinedClouds
}
