package azusercontext

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestWithUserFromQueryReq(t *testing.T) {
	t.Run("should extract cookies from query request", func(t *testing.T) {
		user := &backend.User{Login: "test-user"}
		req := &backend.QueryDataRequest{
			PluginContext: backend.PluginContext{
				User: user,
			},
			Headers: map[string]string{
				"X-ID-Token":           "test-id-token",
				"Authorization":        "Bearer test-access-token",
				"http_X-Grafana-Id":    "test-grafana-id",
				backend.CookiesHeaderName: "session=abc123; theme=dark",
			},
		}

		ctx := WithUserFromQueryReq(context.Background(), req)

		currentUser, ok := GetCurrentUser(ctx)
		require.True(t, ok)
		require.Equal(t, user, currentUser.User)
		require.Equal(t, "test-id-token", currentUser.IdToken)
		require.Equal(t, "test-access-token", currentUser.AccessToken)
		require.Equal(t, "test-grafana-id", currentUser.GrafanaIdToken)
		require.Equal(t, "session=abc123; theme=dark", currentUser.Cookies)
	})

	t.Run("should handle nil request", func(t *testing.T) {
		ctx := WithUserFromQueryReq(context.Background(), nil)

		_, ok := GetCurrentUser(ctx)
		require.False(t, ok)
	})

	t.Run("should handle empty cookies", func(t *testing.T) {
		user := &backend.User{Login: "test-user"}
		req := &backend.QueryDataRequest{
			PluginContext: backend.PluginContext{
				User: user,
			},
			Headers: map[string]string{
				"X-ID-Token": "test-id-token",
			},
		}

		ctx := WithUserFromQueryReq(context.Background(), req)

		currentUser, ok := GetCurrentUser(ctx)
		require.True(t, ok)
		require.Equal(t, user, currentUser.User)
		require.Equal(t, "test-id-token", currentUser.IdToken)
		require.Equal(t, "", currentUser.Cookies)
	})
}

func TestWithUserFromResourceReq(t *testing.T) {
	t.Run("should extract cookies from resource request", func(t *testing.T) {
		user := &backend.User{Login: "test-user"}
		req := &backend.CallResourceRequest{
			PluginContext: backend.PluginContext{
				User: user,
			},
			Headers: map[string][]string{
				"X-ID-Token":           {"test-id-token"},
				"Authorization":        {"Bearer test-access-token"},
				"X-Grafana-Id":         {"test-grafana-id"},
				backend.CookiesHeaderName: {"session=abc123; theme=dark"},
			},
		}

		ctx := WithUserFromResourceReq(context.Background(), req)

		currentUser, ok := GetCurrentUser(ctx)
		require.True(t, ok)
		require.Equal(t, user, currentUser.User)
		require.Equal(t, "test-id-token", currentUser.IdToken)
		require.Equal(t, "test-access-token", currentUser.AccessToken)
		require.Equal(t, "test-grafana-id", currentUser.GrafanaIdToken)
		require.Equal(t, "session=abc123; theme=dark", currentUser.Cookies)
	})

	t.Run("should handle nil request", func(t *testing.T) {
		ctx := WithUserFromResourceReq(context.Background(), nil)

		_, ok := GetCurrentUser(ctx)
		require.False(t, ok)
	})

	t.Run("should handle empty cookies", func(t *testing.T) {
		user := &backend.User{Login: "test-user"}
		req := &backend.CallResourceRequest{
			PluginContext: backend.PluginContext{
				User: user,
			},
			Headers: map[string][]string{
				"X-ID-Token": {"test-id-token"},
			},
		}

		ctx := WithUserFromResourceReq(context.Background(), req)

		currentUser, ok := GetCurrentUser(ctx)
		require.True(t, ok)
		require.Equal(t, user, currentUser.User)
		require.Equal(t, "test-id-token", currentUser.IdToken)
		require.Equal(t, "", currentUser.Cookies)
	})
}

func TestWithUserFromHealthCheckReq(t *testing.T) {
	t.Run("should extract cookies from health check request", func(t *testing.T) {
		user := &backend.User{Login: "test-user"}
		req := &backend.CheckHealthRequest{
			PluginContext: backend.PluginContext{
				User: user,
			},
			Headers: map[string]string{
				"X-ID-Token":           "test-id-token",
				"Authorization":        "Bearer test-access-token",
				"http_X-Grafana-Id":    "test-grafana-id",
				backend.CookiesHeaderName: "session=abc123; theme=dark",
			},
		}

		ctx := WithUserFromHealthCheckReq(context.Background(), req)

		currentUser, ok := GetCurrentUser(ctx)
		require.True(t, ok)
		require.Equal(t, user, currentUser.User)
		require.Equal(t, "test-id-token", currentUser.IdToken)
		require.Equal(t, "test-access-token", currentUser.AccessToken)
		require.Equal(t, "test-grafana-id", currentUser.GrafanaIdToken)
		require.Equal(t, "session=abc123; theme=dark", currentUser.Cookies)
	})

	t.Run("should handle nil request", func(t *testing.T) {
		ctx := WithUserFromHealthCheckReq(context.Background(), nil)

		_, ok := GetCurrentUser(ctx)
		require.False(t, ok)
	})

	t.Run("should handle empty cookies", func(t *testing.T) {
		user := &backend.User{Login: "test-user"}
		req := &backend.CheckHealthRequest{
			PluginContext: backend.PluginContext{
				User: user,
			},
			Headers: map[string]string{
				"X-ID-Token": "test-id-token",
			},
		}

		ctx := WithUserFromHealthCheckReq(context.Background(), req)

		currentUser, ok := GetCurrentUser(ctx)
		require.True(t, ok)
		require.Equal(t, user, currentUser.User)
		require.Equal(t, "test-id-token", currentUser.IdToken)
		require.Equal(t, "", currentUser.Cookies)
	})
}