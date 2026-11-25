package aztokenprovider

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/metadata"
)

const grafanaTenantId = "tenantID"

// stackScopedTokenRetriever is implemented by retrievers whose cached tokens must
// be kept apart per Grafana Cloud stack when Grafana sends no multi-tenant ID.
type stackScopedTokenRetriever interface {
	TokenRetriever
	scopedToStack()
}

var (
	// timeNow makes it possible to test usage of time
	timeNow = time.Now

	failureHold = 60 * time.Second

	// heldEntraCodes fail again on every request until the credential or the
	// app registration changes: expired secret, invalid secret, unknown app, unknown tenant.
	heldEntraCodes = regexp.MustCompile(`\bAADSTS(7000222|7000215|700016|90002)\b`)

	tokenRequestFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "grafana",
		Subsystem: "azure_sdk",
		Name:      "token_request_failures_total",
		Help:      `Failed Entra token requests, by AADSTS code for the held codes and "other" for the rest.`,
	}, []string{"code"})
)

type AccessToken struct {
	Token     string
	ExpiresOn time.Time
}

type TokenRetriever interface {
	GetCacheKey(grafanaMultiTenantId string) string
	Init() error
	GetAccessToken(ctx context.Context, scopes []string) (*AccessToken, error)
	GetExpiry() *time.Time
}

type ConcurrentTokenCache interface {
	GetAccessToken(ctx context.Context, tokenRetriever TokenRetriever, scopes []string) (string, error)
}

func NewConcurrentTokenCache() ConcurrentTokenCache {
	return &tokenCacheImpl{}
}

type tokenCacheImpl struct {
	cache sync.Map // of *credentialCacheEntry
}
type credentialCacheEntry struct {
	retriever TokenRetriever
	expiry    *time.Time

	credInit  uint32
	credMutex sync.Mutex
	cache     sync.Map // of *scopesCacheEntry
}

type scopesCacheEntry struct {
	retriever TokenRetriever
	scopes    []string

	cond        *sync.Cond
	refreshing  bool
	accessToken *AccessToken

	failure      error
	failureUntil time.Time
}

func (c *tokenCacheImpl) GetAccessToken(ctx context.Context, tokenRetriever TokenRetriever, scopes []string) (string, error) {
	return c.getEntryFor(ctx, tokenRetriever).getAccessToken(ctx, scopes)
}

func (c *tokenCacheImpl) getEntryFor(ctx context.Context, credential TokenRetriever) *credentialCacheEntry {
	var entry interface{}
	var ok bool
	tid := returnGrafanaMultiTenantId(ctx)
	if _, scoped := credential.(stackScopedTokenRetriever); scoped && tid == "" {
		// Grafana does not always send a multi-tenant ID. Without one, the cache key
		// is built only from the Azure tenant and client IDs. Those are not secret
		// and can be shared by several stacks, which would then share one cached
		// token. For retrievers that opt in, use the stack Grafana reports as the ID
		// instead, so each stack gets its own entry.
		//
		// Every credential type has this gap. For now to reduce blast radius, this is opt-in, and
		// only workload identity in memory does, so this change does not alter
		// caching for the other datasources that depend on this SDK.
		tid = stackIDFromContext(ctx)
	}

	key := credential.GetCacheKey(tid)

	entry, ok = c.cache.Load(key)
	// Store new cache value if there isn't an existing one
	if !ok {
		entry, _ = c.cache.LoadOrStore(key, &credentialCacheEntry{
			retriever: credential,
		})
		return entry.(*credentialCacheEntry)
	}

	expiry := entry.(*credentialCacheEntry).retriever.GetExpiry()
	// Store new cache value if the current one has expired (only applies to OBO retriever)
	if expiry != nil && !expiry.After(time.Now()) {
		c.cache.Store(key, &credentialCacheEntry{
			retriever: credential,
		})
		entry, _ = c.cache.Load(key)
	}

	return entry.(*credentialCacheEntry)
}

func (c *credentialCacheEntry) getAccessToken(ctx context.Context, scopes []string) (string, error) {
	err := c.ensureInitialized()
	if err != nil {
		return "", err
	}

	return c.getEntryFor(scopes).getAccessToken(ctx)
}

func (c *credentialCacheEntry) ensureInitialized() error {
	if atomic.LoadUint32(&c.credInit) == 0 {
		c.credMutex.Lock()
		defer c.credMutex.Unlock()

		if c.credInit == 0 {
			// Initialize retriever
			err := c.retriever.Init()
			if err != nil {
				return err
			}

			atomic.StoreUint32(&c.credInit, 1)
		}
	}

	return nil
}

func (c *credentialCacheEntry) getEntryFor(scopes []string) *scopesCacheEntry {
	var entry interface{}
	var ok bool

	key := getKeyForScopes(scopes)

	if entry, ok = c.cache.Load(key); !ok {
		entry, _ = c.cache.LoadOrStore(key, &scopesCacheEntry{
			retriever: c.retriever,
			scopes:    scopes,
			cond:      sync.NewCond(&sync.Mutex{}),
		})
	}

	return entry.(*scopesCacheEntry)
}

func (c *scopesCacheEntry) getAccessToken(ctx context.Context) (string, error) {
	var accessToken *AccessToken
	var err error
	shouldRefresh := false

	c.cond.L.Lock()
	for {
		if c.accessToken != nil && c.accessToken.ExpiresOn.After(timeNow().Add(2*time.Minute)) {
			// Use the cached token since it's available and not expired yet
			accessToken = c.accessToken
			break
		}

		if c.failure != nil && timeNow().Before(c.failureUntil) {
			err = c.failure
			break
		}

		if !c.refreshing {
			// Start refreshing the token
			c.refreshing = true
			shouldRefresh = true
			break
		}

		// Wait for the token to be refreshed
		c.cond.Wait()
	}
	c.cond.L.Unlock()

	if err != nil {
		return "", err
	}

	if shouldRefresh {
		accessToken, err = c.refreshAccessToken(ctx)
		if err != nil {
			return "", err
		}
	}

	return accessToken.Token, nil
}

func (c *scopesCacheEntry) refreshAccessToken(ctx context.Context) (*AccessToken, error) {
	var accessToken *AccessToken
	var failure error

	// Safeguarding from panic caused by retriever implementation
	defer func() {
		c.cond.L.Lock()

		c.refreshing = false

		if accessToken != nil {
			c.accessToken = accessToken
		}
		c.failure, c.failureUntil = failure, timeNow().Add(failureHold)

		c.cond.Broadcast()
		c.cond.L.Unlock()
	}()

	token, err := c.retriever.GetAccessToken(ctx, c.scopes)
	if err != nil {
		msg := err.Error()
		code := heldEntraCodes.FindString(msg)
		if code == "" {
			code = "other"
		} else {
			// Callers share the held error, so hold a copy of its text. Error on an
			// azidentity error writes to the response body and is not safe to share.
			failure = errors.New(msg)
		}
		tokenRequestFailures.WithLabelValues(code).Inc()
		return nil, err
	}
	accessToken = token
	return accessToken, nil
}

func getKeyForScopes(scopes []string) string {
	if len(scopes) > 1 {
		arr := make([]string, len(scopes))
		copy(arr, scopes)
		sort.Strings(arr)
		scopes = arr
	}

	return strings.Join(scopes, " ")
}

func returnGrafanaMultiTenantId(ctx context.Context) (grafanaMultiTenantId string) {
	md, exists := metadata.FromIncomingContext(ctx)

	if exists {
		tid := md.Get(grafanaTenantId)
		if len(tid) > 0 && tid[0] != "" {
			grafanaMultiTenantId = tid[0]
		}
	}

	return grafanaMultiTenantId
}
