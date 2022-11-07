package azsettings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var UserDefinedAzureCustomized = "AzureCustomizedCloud"

func TestNormalizeAzureCloud(t *testing.T) {
	t.Run("should return unknown clouds as is", func(t *testing.T) {
		cloud := UserDefinedAzureCustomized
		normalized := NormalizeAzureCloud(cloud)
		assert.Equal(t, UserDefinedAzureCustomized, normalized)
	})
}
