package aztokenprovider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grafana/grafana-azure-sdk-go/v2/azusercontext"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testClientId = "00000000-0000-0000-0000-000000000001"

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// testJWT signs with RS256, as Entra does.
func testJWT(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(testRSAKey(t))
	require.NoError(t, err)
	return token
}

func TestChooseAssertion(t *testing.T) {
	provider := &userTokenProvider{clientId: testClientId}
	idToken := testJWT(t, jwt.MapClaims{"aud": testClientId})

	tests := []struct {
		name        string
		accessToken string
		wantKind    assertionKind
	}{
		{name: "access token for this app (GUID audience)", accessToken: testJWT(t, jwt.MapClaims{"aud": testClientId}), wantKind: assertionAccessToken},
		{name: "access token for this app (api:// audience)", accessToken: testJWT(t, jwt.MapClaims{"aud": "api://" + testClientId}), wantKind: assertionAccessToken},
		{name: "access token for Graph falls back to ID token", accessToken: testJWT(t, jwt.MapClaims{"aud": "00000003-0000-0000-c000-000000000000"}), wantKind: assertionIdToken},
		{name: "access token for a custom App ID URI falls back to ID token", accessToken: testJWT(t, jwt.MapClaims{"aud": "api://custom-grafana"}), wantKind: assertionIdToken},
		{name: "unparseable access token falls back to ID token", accessToken: "not-a-jwt", wantKind: assertionIdToken},
		{name: "no access token falls back to ID token", accessToken: "", wantKind: assertionIdToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := azusercontext.CurrentUserContext{IdToken: idToken, AccessToken: tt.accessToken}

			assertion, kind, err := provider.chooseAssertion(context.Background(), user)
			require.NoError(t, err)
			assert.Equal(t, tt.wantKind, kind)
			if tt.wantKind == assertionAccessToken {
				assert.Equal(t, tt.accessToken, assertion)
			} else {
				assert.Equal(t, idToken, assertion)
			}
		})
	}

	t.Run("matches nothing when the client ID isn't configured", func(t *testing.T) {
		user := azusercontext.CurrentUserContext{IdToken: idToken, AccessToken: testJWT(t, jwt.MapClaims{"aud": ""})}
		_, kind, err := (&userTokenProvider{}).chooseAssertion(context.Background(), user)
		require.NoError(t, err)
		assert.Equal(t, assertionIdToken, kind)
	})

	t.Run("errors with the tokens checked when neither can be used", func(t *testing.T) {
		user := azusercontext.CurrentUserContext{AccessToken: testJWT(t, jwt.MapClaims{"aud": "00000003-0000-0000-c000-000000000000"})}
		_, _, err := provider.chooseAssertion(context.Background(), user)
		require.ErrorContains(t, err, "no ID token associated with the Grafana user")
		require.ErrorContains(t, err, "access token audience is not this app, audience '00000003-0000-0000-c000-000000000000'")
	})

	t.Run("errors when there are no tokens at all", func(t *testing.T) {
		_, _, err := provider.chooseAssertion(context.Background(), azusercontext.CurrentUserContext{})
		require.ErrorContains(t, err, "no ID token associated with the Grafana user")
		require.ErrorContains(t, err, "no access token")
	})
}

type erroringTokenCache struct{ err error }

func (c *erroringTokenCache) GetAccessToken(_ context.Context, _ TokenRetriever, _ []string) (string, error) {
	return "", c.err
}

func TestGetAccessToken_UserIdentity_AssertionHint(t *testing.T) {
	scopes := []string{"https://management.azure.com/.default"}
	entraErr := errors.New("AADSTS240002: Input id_token cannot be used as 'urn:ietf:params:oauth:grant-type:jwt-bearer' grant")

	userCtx := func(accessToken string) context.Context {
		return azusercontext.WithCurrentUser(context.Background(), azusercontext.CurrentUserContext{
			User:        &backend.User{Login: "user1@example.org"},
			IdToken:     testJWT(t, jwt.MapClaims{"aud": testClientId}),
			AccessToken: accessToken,
		})
	}

	t.Run("adds a hint when Entra rejects the ID token", func(t *testing.T) {
		provider := &userTokenProvider{clientId: testClientId, tokenCache: &erroringTokenCache{err: entraErr}}
		_, err := provider.GetAccessToken(userCtx(""), scopes)
		require.ErrorIs(t, err, entraErr)
		require.ErrorContains(t, err, "add the '<client-id>/.default' scope to the Grafana login scopes")
	})

	t.Run("doesn't add the hint for other errors", func(t *testing.T) {
		otherErr := errors.New("AADSTS65001: consent required")
		provider := &userTokenProvider{clientId: testClientId, tokenCache: &erroringTokenCache{err: otherErr}}
		_, err := provider.GetAccessToken(userCtx(""), scopes)
		require.ErrorIs(t, err, otherErr)
		require.NotContains(t, err.Error(), "login scopes")
	})

	t.Run("doesn't add the hint for username assertions", func(t *testing.T) {
		provider := &userTokenProvider{clientId: testClientId, usernameAssertion: true, tokenCache: &erroringTokenCache{err: entraErr}}
		_, err := provider.GetAccessToken(userCtx(""), scopes)
		require.ErrorIs(t, err, entraErr)
		require.NotContains(t, err.Error(), "login scopes")
	})
}

func TestOnBehalfOfCacheKeyDiffersByAssertionKind(t *testing.T) {
	idRetriever := &onBehalfOfTokenRetriever{userId: "user", assertionKind: assertionIdToken}
	accessRetriever := &onBehalfOfTokenRetriever{userId: "user", assertionKind: assertionAccessToken}
	assert.NotEqual(t, idRetriever.GetCacheKey("tenant"), accessRetriever.GetCacheKey("tenant"))
}

func TestOnBehalfOfExpiryFromAssertion(t *testing.T) {
	for _, kind := range []assertionKind{assertionIdToken, assertionAccessToken} {
		t.Run(string(kind), func(t *testing.T) {
			exp := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
			retriever := &onBehalfOfTokenRetriever{assertion: testJWT(t, jwt.MapClaims{"exp": exp.Unix()}), assertionKind: kind}
			assert.Equal(t, exp, retriever.GetExpiry().UTC())
		})
	}
}
