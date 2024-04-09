package azhttpclient

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/grafana/grafana-azure-sdk-go/azusercontext"
)

var (
	once          sync.Once
	processSeed   []byte
	processSeedOk bool
)

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
		err := fmt.Errorf("parameter 'ctx' cannot be nil")
		return "", err
	}

	currentUser, ok := azusercontext.GetCurrentUser(ctx)
	if !ok {
		err := fmt.Errorf("user context not configured")
		return "", err
	}

	hash := sha256.New()
	_, _ = hash.Write(p.seed)
	_, _ = hash.Write([]byte(currentUser.User.Login))
	sessionId := base64.URLEncoding.EncodeToString(hash.Sum(nil))

	return sessionId, nil
}
