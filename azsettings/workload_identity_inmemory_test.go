package azsettings

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contextWithConfig(cfg map[string]string) context.Context {
	return backend.WithGrafanaConfig(context.Background(), backend.NewGrafanaCfg(cfg))
}

func TestReadFromContext_WorkloadIdentityInMemory(t *testing.T) {
	base := map[string]string{
		WorkloadIdentityEnabled:   "true",
		WorkloadIdentityTokenFile: "/tokens/azure-identity-token",
	}
	with := func(extra map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range base {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}

	t.Run("is off by default", func(t *testing.T) {
		settings, ok := ReadFromContext(contextWithConfig(base))
		require.True(t, ok)
		assert.False(t, settings.WorkloadIdentitySettings.InMemory)
		assert.Equal(t, "/tokens/azure-identity-token", settings.WorkloadIdentitySettings.TokenFile)
	})

	t.Run("is off when other feature toggles are enabled", func(t *testing.T) {
		settings, _ := ReadFromContext(contextWithConfig(with(map[string]string{featuretoggles.EnabledFeatures: "idForwarding,someOther"})))
		assert.False(t, settings.WorkloadIdentitySettings.InMemory)
	})

	t.Run("is on when the workloadIdentityInMemory feature toggle is enabled", func(t *testing.T) {
		settings, ok := ReadFromContext(contextWithConfig(with(map[string]string{
			featuretoggles.EnabledFeatures:       "idForwarding," + WorkloadIdentityInMemoryFeatureToggle,
			WorkloadIdentityAssertionURL:         "http://tokensvc:8081",
			WorkloadIdentityAssertionCallerToken: "caller-token",
			WorkloadIdentityStackID:              "my-stack",
		})))
		require.True(t, ok)
		wi := settings.WorkloadIdentitySettings
		assert.True(t, wi.InMemory)
		assert.Equal(t, "my-stack", wi.StackID)
		assert.Equal(t, "http://tokensvc:8081", wi.AssertionURL)
		assert.Equal(t, "caller-token", wi.AssertionCallerToken)
	})

	t.Run("does nothing when workload identity itself is disabled", func(t *testing.T) {
		settings, _ := ReadFromContext(contextWithConfig(map[string]string{
			featuretoggles.EnabledFeatures: WorkloadIdentityInMemoryFeatureToggle,
		}))
		assert.False(t, settings.WorkloadIdentityEnabled)
		assert.Nil(t, settings.WorkloadIdentitySettings)
	})
}

func TestReadFromEnv_WorkloadIdentityInMemory(t *testing.T) {
	t.Run("is off by default", func(t *testing.T) {
		t.Setenv(WorkloadIdentityEnabled, "true")
		settings, err := ReadFromEnv()
		require.NoError(t, err)
		assert.False(t, settings.WorkloadIdentitySettings.InMemory)
	})

	t.Run("reads the in-memory settings", func(t *testing.T) {
		t.Setenv(WorkloadIdentityEnabled, "true")
		t.Setenv(WorkloadIdentityInMemory, "true")
		t.Setenv(WorkloadIdentityAssertionURL, "http://tokensvc:8081")
		t.Setenv(WorkloadIdentityAssertionCallerToken, "caller-token")
		t.Setenv(WorkloadIdentityStackID, "my-stack")
		settings, err := ReadFromEnv()
		require.NoError(t, err)
		wi := settings.WorkloadIdentitySettings
		assert.True(t, wi.InMemory)
		assert.Equal(t, "my-stack", wi.StackID)
		assert.Equal(t, "http://tokensvc:8081", wi.AssertionURL)
		assert.Equal(t, "caller-token", wi.AssertionCallerToken)
	})

	t.Run("rejects an invalid value", func(t *testing.T) {
		t.Setenv(WorkloadIdentityEnabled, "true")
		t.Setenv(WorkloadIdentityInMemory, "maybe")
		_, err := ReadFromEnv()
		require.Error(t, err)
	})
}
