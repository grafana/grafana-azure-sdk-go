package aztokenprovider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grafana/grafana-azure-sdk-go/v2/azusercontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testJWT(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(testRSAKey(t))
	require.NoError(t, err)
	return token
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func TestChooseAssertion(t *testing.T) {
	const clientId = "00000000-0000-0000-0000-000000000001"
	provider := &userTokenProvider{clientId: clientId}
	idToken := testJWT(t, jwt.MapClaims{"aud": clientId})

	tests := []struct {
		name         string
		accessAud    string
		override     string
		wantKind     assertionKind
		wantIdToken  bool
		noAccess     bool
		wantErrorMsg string
	}{
		{name: "access token for this app (v2, GUID)", accessAud: clientId, wantKind: assertionAccessToken},
		{name: "access token for this app (v1, api://)", accessAud: "api://" + clientId, wantKind: assertionAccessToken},
		{name: "access token for Graph falls back to ID token", accessAud: "00000003-0000-0000-c000-000000000000", wantKind: assertionIdToken, wantIdToken: true},
		{name: "custom App ID URI falls back to ID token", accessAud: "api://custom-grafana", wantKind: assertionIdToken, wantIdToken: true},
		{name: "no access token falls back to ID token", noAccess: true, wantKind: assertionIdToken, wantIdToken: true},
		{name: "override forces access token", accessAud: "api://custom-grafana", override: "accesstoken", wantKind: assertionAccessToken},
		{name: "override forces ID token", accessAud: clientId, override: "idtoken", wantKind: assertionIdToken, wantIdToken: true},
		{name: "invalid override", accessAud: clientId, override: "nonsense", wantErrorMsg: "invalid GF_PLUGIN_AZURE_OBO_ASSERTION_OVERRIDE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(assertionOverrideEnv, tt.override)
			accessToken := ""
			if !tt.noAccess {
				accessToken = testJWT(t, jwt.MapClaims{"aud": tt.accessAud})
			}
			user := azusercontext.CurrentUserContext{IdToken: idToken, AccessToken: accessToken}

			assertion, kind, err := provider.chooseAssertion(context.Background(), user)
			if tt.wantErrorMsg != "" {
				require.ErrorContains(t, err, tt.wantErrorMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantKind, kind)
			if tt.wantIdToken {
				assert.Equal(t, idToken, assertion)
			} else {
				assert.Equal(t, accessToken, assertion)
			}
		})
	}

	t.Run("errors when the chosen token is missing", func(t *testing.T) {
		t.Setenv(assertionOverrideEnv, "")
		_, _, err := provider.chooseAssertion(context.Background(), azusercontext.CurrentUserContext{})
		require.ErrorContains(t, err, "no ID token associated with the Grafana user")
	})
}

func TestOnBehalfOfCacheKeyDiffersByAssertionKind(t *testing.T) {
	idRetriever := &onBehalfOfTokenRetriever{userId: "user", assertionKind: assertionIdToken}
	accessRetriever := &onBehalfOfTokenRetriever{userId: "user", assertionKind: assertionAccessToken}
	assert.NotEqual(t, idRetriever.GetCacheKey("tenant"), accessRetriever.GetCacheKey("tenant"))
}

func TestOnBehalfOfExpiryFromRS256Token(t *testing.T) {
	retriever := &onBehalfOfTokenRetriever{assertion: testJWT(t, jwt.MapClaims{"exp": 4102444800})}
	assert.Equal(t, int64(4102444800), retriever.GetExpiry().Unix())
}
