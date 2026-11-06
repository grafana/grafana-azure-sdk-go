package aztokenprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// assertionKind identifies which of the user's tokens is sent as the on-behalf-of assertion.
type assertionKind string

const (
	assertionIdToken     assertionKind = "idtoken"
	assertionAccessToken assertionKind = "accesstoken"
)

type onBehalfOfTokenRetriever struct {
	client        TokenClient
	userId        string
	assertion     string
	assertionKind assertionKind
}

func (r *onBehalfOfTokenRetriever) GetCacheKey(grafanaMultiTenantId string) string {
	return fmt.Sprintf("currentuser|%s|%s|%s", r.assertionKind, r.userId, grafanaMultiTenantId)
}

func (r *onBehalfOfTokenRetriever) Init() error {
	// Nothing to initialize
	return nil
}

func (r *onBehalfOfTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := r.client.OnBehalfOf(ctx, r.assertion, scopes)
	if err != nil {
		return nil, err
	}

	return accessToken, nil
}

// Returns the expiry time from the assertion or the 0 time value (which will always be expired)
func (c *onBehalfOfTokenRetriever) GetExpiry() *time.Time {
	if c != nil && c.assertion != "" {
		claims := jwt.MapClaims{}
		_, _, err := jwt.NewParser(jwt.WithValidMethods([]string{"ES256"})).ParseUnverified(c.assertion, claims)
		if err != nil {
			// Existing token is invalid in some way so store the new one in cache
			return &time.Time{}
		}

		expiry, err := claims.GetExpirationTime()
		if err != nil || expiry == nil {
			// Unable to get expiration from existing token so store the new one in cache
			return &time.Time{}
		}

		return &expiry.Time
	}

	return &time.Time{}
}
