package azhttpclient

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/naizerjohn-ms/grafana-azure-sdk-go/azusercontext"
	"github.com/stretchr/testify/require"
)

func TestSessionProvider(t *testing.T) {
	t.Run("should return a sessionId", func(t *testing.T) {
		sessionProvider, err := newSessionProvider()
		require.NoError(t, err)
		usrctx := azusercontext.WithCurrentUser(context.Background(), azusercontext.CurrentUserContext{
			User: &backend.User{
				Login: "user1@example.org",
			},
			IdToken: "FAKE_ID_TOKEN",
		})
		sessionId, err := sessionProvider.GetSessionId(usrctx)
		require.NoError(t, err)
		require.NotEmpty(t, sessionId)
	})

	t.Run("should error if no user is in context", func(t *testing.T) {
		sessionProvider, err := newSessionProvider()
		require.NoError(t, err)
		sessionId, err := sessionProvider.GetSessionId(context.Background())
		require.Error(t, err)
		require.Empty(t, sessionId)
	})

	t.Run("should error if no user.User is in context", func(t *testing.T) {
		sessionProvider, err := newSessionProvider()
		require.NoError(t, err)
		usrctx := azusercontext.WithCurrentUser(context.Background(), azusercontext.CurrentUserContext{
			IdToken: "FAKE_ID_TOKEN",
		})
		sessionId, err := sessionProvider.GetSessionId(usrctx)
		require.Error(t, err)
		require.Empty(t, sessionId)
	})
}
