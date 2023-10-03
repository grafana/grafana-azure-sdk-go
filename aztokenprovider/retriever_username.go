package aztokenprovider

import (
	"context"
	"fmt"
)

type usernameTokenRetriever struct {
	client   TokenClient
	username string
}

func (r *usernameTokenRetriever) GetCacheKey(multiTenantID string) string {
	return fmt.Sprintf("currentuser|username|%s|%s", r.username, multiTenantID)
}

func (r *usernameTokenRetriever) Init() error {
	// Nothing to initialize
	return nil
}

func (r *usernameTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := r.client.FromUsername(ctx, r.username, scopes)
	if err != nil {
		return nil, err
	}

	return accessToken, nil
}
