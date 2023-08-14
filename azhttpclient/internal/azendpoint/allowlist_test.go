package azendpoint

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowlist(t *testing.T) {
	t.Run("given invalid allowlist", func(t *testing.T) {
		t.Run("should not accept empty string", func(t *testing.T) {
			_, err := Allowlist([]string{
				"",
			})
			assert.Error(t, err)
		})

		t.Run("should not accept plain hostname", func(t *testing.T) {
			_, err := Allowlist([]string{
				"example.net",
			})
			assert.Error(t, err)
		})

		t.Run("should not accept relative path", func(t *testing.T) {
			_, err := Allowlist([]string{
				"/foobar",
			})
			assert.Error(t, err)
		})
	})

	t.Run("given exact scheme", func(t *testing.T) {
		a, err := Allowlist([]string{
			"https://example.com",
			"svc://example.net:5001",
		})
		require.NoError(t, err)

		t.Run("should match https scheme", func(t *testing.T) {
			u, err := url.Parse("https://example.com")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should match custom scheme", func(t *testing.T) {
			u, err := url.Parse("svc://example.net:5001")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should not match http instead of https", func(t *testing.T) {
			u, err := url.Parse("http://example.com")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should not match different scheme", func(t *testing.T) {
			u, err := url.Parse("https://example.net:5001")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})
	})

	t.Run("do not support wildcard scheme", func(t *testing.T) {
		_, err := Allowlist([]string{
			"*://example.net",
		})
		assert.Error(t, err)
	})

	t.Run("do not support omitted scheme", func(t *testing.T) {
		_, err := Allowlist([]string{
			"://example.net",
		})
		assert.Error(t, err)
	})

	t.Run("given exact port", func(t *testing.T) {
		a, err := Allowlist([]string{
			"http://example.org:80",
			"https://example.com:443",
			"https://example1.net:3000",
			"svc://example2.net:5001",
		})
		require.NoError(t, err)

		t.Run("should match exact port", func(t *testing.T) {
			u, err := url.Parse("https://example1.net:3000")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should match http without port", func(t *testing.T) {
			u, err := url.Parse("http://example.org")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should match https without port", func(t *testing.T) {
			u, err := url.Parse("https://example.com")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should not match https with custom port to https without port", func(t *testing.T) {
			u, err := url.Parse("https://example1.net")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should not match custom scheme without port", func(t *testing.T) {
			u, err := url.Parse("svc://example2.net")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should not match different port", func(t *testing.T) {
			u, err := url.Parse("svc://example2.net:555")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})
	})

	t.Run("given no port for known scheme", func(t *testing.T) {
		a, err := Allowlist([]string{
			"http://example.org",
			"https://example.com",
		})
		require.NoError(t, err)

		t.Run("should match http without port", func(t *testing.T) {
			u, err := url.Parse("http://example.org")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should match https without port", func(t *testing.T) {
			u, err := url.Parse("https://example.com")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should match http with port", func(t *testing.T) {
			u, err := url.Parse("http://example.org:80")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should match https with port", func(t *testing.T) {
			u, err := url.Parse("https://example.com:443")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should not match different port", func(t *testing.T) {
			u, err := url.Parse("http://example.org:5001")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should not match http port for https", func(t *testing.T) {
			u, err := url.Parse("https://example.com:80")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})
	})

	t.Run("should require port for custom scheme", func(t *testing.T) {
		_, err := Allowlist([]string{
			"tcp://example.net",
		})
		assert.Error(t, err)
	})

	t.Run("do not support wildcard port", func(t *testing.T) {
		_, err := Allowlist([]string{
			"tcp://example.net:*",
		})
		assert.Error(t, err)
	})

	t.Run("given host", func(t *testing.T) {
		a, err := Allowlist([]string{
			"https://example.com",
			"https://*.example.net",
		})
		require.NoError(t, err)

		t.Run("should not allow empty string", func(t *testing.T) {
			ok := a.IsAllowed(nil)
			assert.False(t, ok)
		})

		t.Run("should not allow invalid url", func(t *testing.T) {
			u, err := url.Parse("/test")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should not allow empty host", func(t *testing.T) {
			u, err := url.Parse("svc:///test")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should not allow localhost", func(t *testing.T) {
			u, err := url.Parse("https://localhost/")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should not allow unknown domain", func(t *testing.T) {
			u, err := url.Parse("https://unknown.com/")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should match exact allowed domain", func(t *testing.T) {
			u, err := url.Parse("https://example.com/")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should not match subdomain of exact allowed domain", func(t *testing.T) {
			u, err := url.Parse("https://subdomain.example.com/")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should match allowed suffix", func(t *testing.T) {
			u, err := url.Parse("https://test.example.net/")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should match allowed suffix with nested subdomains", func(t *testing.T) {
			u, err := url.Parse("https://test.subdomain.example.net/")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.True(t, ok)
		})

		t.Run("should not match allowed suffix without subdomain", func(t *testing.T) {
			u, err := url.Parse("https://example.net/")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})

		t.Run("should not match allowed suffix with dot", func(t *testing.T) {
			u, err := url.Parse("https://.example.net/")
			require.NoError(t, err)

			ok := a.IsAllowed(u)
			assert.False(t, ok)
		})
	})
}
