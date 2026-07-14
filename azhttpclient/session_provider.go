package azhttpclient

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/grafana/grafana-azure-sdk-go/v2/azusercontext"
)

var (
	once          sync.Once
	processSeed   []byte
	processSeedOk bool
)

// ErrUserContextNotConfigured is returned by GetSessionId when there is no Grafana
// user in the request context (e.g. service-context calls such as multi-tenant health
// checks). Callers should treat the rate-limit session id as optional in that case.
var ErrUserContextNotConfigured = errors.New("user context not configured")

type userSessionProvider struct {
	seed []byte
}

func newSessionProvider() (*userSessionProvider, error) {
	// Session anonymized with an in-memory seed generated for the process instance
	seed, err := perProcessSeed()
	if err != nil {
		return nil, errors.New("failed to initialize the user session provider")
	}

	return &userSessionProvider{
		seed,
	}, nil
}

func perProcessSeed() ([]byte, error) {
	once.Do(func() {
		seed := make([]byte, 32)
		_, err := rand.Read(seed)
		if err == nil {
			processSeed = seed
			processSeedOk = true
		}
	})

	if !processSeedOk {
		return nil, errors.New("failed to generate seed")
	}
	return processSeed, nil
}

func (p *userSessionProvider) GetSessionId(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("parameter 'ctx' cannot be nil")
	}

	currentUser, ok := azusercontext.GetCurrentUser(ctx)
	if !ok || currentUser.User == nil {
		return "", ErrUserContextNotConfigured
	}

	hash := sha256.New()
	_, err := hash.Write(p.seed)
	if err != nil {
		return "", err
	}
	_, err = hash.Write([]byte(currentUser.User.Login))
	if err != nil {
		return "", err
	}
	sessionId := base64.URLEncoding.EncodeToString(hash.Sum(nil))

	return sessionId, nil
}
