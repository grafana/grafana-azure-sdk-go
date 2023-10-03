package aztokenprovider

import (
	"context"
	"fmt"
)

type onBehalfOfTokenRetriever struct {
	client  TokenClient
	userId  string
	idToken string
}

func (r *onBehalfOfTokenRetriever) GetCacheKey(grafanaMultiTenantId string) string {
	return fmt.Sprintf("currentuser|idtoken|%s|%s", r.userId, grafanaMultiTenantId)
}

func (r *onBehalfOfTokenRetriever) Init() error {
	// Nothing to initialize
	return nil
}

func (r *onBehalfOfTokenRetriever) GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error) {
	accessToken, err := r.client.OnBehalfOf(ctx, r.idToken, scopes)
	if err != nil {
		return nil, err
	}

	return accessToken, nil
}
