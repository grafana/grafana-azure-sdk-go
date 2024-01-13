package azsettings

import (
	"fmt"
	"sort"
)

type AzureCloudInfo struct {
	Name        string
	DisplayName string
}

type AzureCloudSettings struct {
	Name         string
	DisplayName  string
	AadAuthority string
	Properties   map[string]string
}

var predefinedClouds = map[string]*AzureCloudSettings{
	AzurePublic: {
		Name:         "AzureCloud",
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
	AzureChina: {
		Name:         "AzureChinaCloud",
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
	AzureUSGovernment: {
		Name:         "AzureUSGovernment",
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

func (*AzureSettings) Clouds() []AzureCloudInfo {
	clouds := make([]AzureCloudInfo, 0, len(predefinedClouds))
	for _, cloud := range predefinedClouds {
		clouds = append(clouds, AzureCloudInfo{
			Name:        cloud.Name,
			DisplayName: cloud.DisplayName,
		})
	}

	// Sort by name
	sort.Slice(clouds, func(i, j int) bool {
		istr := clouds[i].DisplayName
		if istr == "" {
			istr = clouds[i].Name
		}
		jstr := clouds[j].DisplayName
		if jstr == "" {
			jstr = clouds[j].Name
		}
		return istr < jstr
	})

	return clouds
}

func (*AzureSettings) GetCloud(cloudName string) (*AzureCloudSettings, error) {
	if cloudSettings, ok := predefinedClouds[cloudName]; !ok {
		return nil, fmt.Errorf("the Azure cloud '%s' not supported", cloudName)
	} else {
		return cloudSettings, nil
	}
}
