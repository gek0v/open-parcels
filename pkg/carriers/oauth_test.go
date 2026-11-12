package carriers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockOAuthStore struct {
	mu     sync.RWMutex
	tokens map[string]struct {
		token     string
		expiresAt time.Time
	}
}

func newMockOAuthStore() *mockOAuthStore {
	return &mockOAuthStore{
		tokens: make(map[string]struct {
			token     string
			expiresAt time.Time
		}),
	}
}

func (m *mockOAuthStore) GetOAuthToken(ctx context.Context, provider string) (string, time.Time, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.tokens[provider]
	if !ok {
		return "", time.Time{}, false, nil
	}
	return item.token, item.expiresAt, true, nil
}

func (m *mockOAuthStore) SetOAuthToken(ctx context.Context, provider, token string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[provider] = struct {
		token     string
		expiresAt time.Time
	}{token: token, expiresAt: expiresAt}
	return nil
}

func TestOAuthTokenCacheWithPersistentStore(t *testing.T) {
	var httpCalls int32
	mockClient := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			atomic.AddInt32(&httpCalls, 1)
			tokenJSON := `{"access_token":"remote-generated-token","expires_in":3600}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(tokenJSON)),
				Header:     make(http.Header),
				Request:    req,
			}
		}),
	}

	st := newMockOAuthStore()
	cache := NewOAuthTokenCache(st)

	ctx := context.Background()

	// 1. First fetch: calls remote HTTP endpoint and stores in persistent store
	tok1, err := cache.GetToken(ctx, "fedex", "https://apis.fedex.com/oauth/token", "client-id", "client-secret", mockClient, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok1 != "remote-generated-token" {
		t.Errorf("expected remote token, got %s", tok1)
	}
	if calls := atomic.LoadInt32(&httpCalls); calls != 1 {
		t.Fatalf("expected 1 http call, got %d", calls)
	}

	// Verify it was stored in mock store
	savedTok, savedExp, found, _ := st.GetOAuthToken(ctx, "fedex")
	if !found || savedTok != "remote-generated-token" {
		t.Fatalf("token was not persisted in store: found=%v tok=%s", found, savedTok)
	}
	if savedExp.Before(time.Now().Add(3500 * time.Second)) {
		t.Errorf("expected expiresAt around 3600s in future, got %v", savedExp)
	}

	// 2. Clear in-memory cache to simulate server restart
	cache.Clear()

	// 3. Second fetch: loads directly from persistent store without calling HTTP endpoint
	tok2, err := cache.GetToken(ctx, "fedex", "https://apis.fedex.com/oauth/token", "client-id", "client-secret", mockClient, false)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if tok2 != "remote-generated-token" {
		t.Errorf("expected token from persistent store, got %s", tok2)
	}
	if calls := atomic.LoadInt32(&httpCalls); calls != 1 {
		t.Errorf("expected still 1 http call (loaded from store), got %d", calls)
	}

	// 4. Token expired in persistent store: forces refresh via HTTP
	st.SetOAuthToken(ctx, "fedex", "expired-token", time.Now().Add(-10*time.Minute))
	cache.Clear()

	tok3, err := cache.GetToken(ctx, "fedex", "https://apis.fedex.com/oauth/token", "client-id", "client-secret", mockClient, false)
	if err != nil {
		t.Fatalf("unexpected error refreshing expired token: %v", err)
	}
	if tok3 != "remote-generated-token" {
		t.Errorf("expected refreshed token, got %s", tok3)
	}
	if calls := atomic.LoadInt32(&httpCalls); calls != 2 {
		t.Errorf("expected 2 http calls after expiration, got %d", calls)
	}
}