package azendpoint

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint(t *testing.T) {
	t.Run("should return nil for url without host", func(t *testing.T) {
		u, err := url.Parse("svc:///test")
		require.NoError(t, err)

		e := Endpoint(*u)

		assert.Nil(t, e)
	})

	t.Run("should return endpoint without path and query string", func(t *testing.T) {
		u, err := url.Parse("https://example.com/api/query?q=foobar&l=1")
		require.NoError(t, err)

		e := Endpoint(*u)

		assert.Equal(t, "https://example.com", e.String())
	})

	t.Run("should not modify original url", func(t *testing.T) {
		u, err := url.Parse("https://example.com/api/query?q=foobar&l=1")
		require.NoError(t, err)

		_ = Endpoint(*u)

		assert.Equal(t, "https://example.com/api/query?q=foobar&l=1", u.String())
	})
}
