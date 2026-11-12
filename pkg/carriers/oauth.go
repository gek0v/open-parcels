package carriers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"open-parcels/pkg/models"
)

// OAuthTokenStore defines an optional persistent backend for OAuth2 access tokens.
type OAuthTokenStore interface {
	GetOAuthToken(ctx context.Context, provider string) (token string, expiresAt time.Time, found bool, err error)
	SetOAuthToken(ctx context.Context, provider, token string, expiresAt time.Time) error
}

// OAuthTokenCache manages in-memory and optional persistent caching of OAuth2 client credentials tokens.
type OAuthTokenCache struct {
	mu     sync.RWMutex
	tokens map[string]cachedToken
	store  OAuthTokenStore
}

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

var globalOAuthCache = NewOAuthTokenCache(nil)

// NewOAuthTokenCache creates a new OAuthTokenCache optionally backed by a persistent store.
func NewOAuthTokenCache(store OAuthTokenStore) *OAuthTokenCache {
	return &OAuthTokenCache{
		tokens: make(map[string]cachedToken),
		store:  store,
	}
}

// SetStore attaches a persistent OAuthTokenStore to the cache.
func (c *OAuthTokenCache) SetStore(store OAuthTokenStore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = store
}

// Clear removes all in-memory cached tokens.
func (c *OAuthTokenCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens = make(map[string]cachedToken)
}

// SetGlobalOAuthStore configures the persistent token store for the global OAuth cache.
func SetGlobalOAuthStore(store OAuthTokenStore) {
	globalOAuthCache.SetStore(store)
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // in seconds
}

// GetToken returns a valid OAuth2 access token, renewing it if expired or absent.
func (c *OAuthTokenCache) GetToken(
	ctx context.Context,
	cacheKey string,
	tokenURL string,
	clientID string,
	clientSecret string,
	httpClient *http.Client,
	useBasicAuth bool,
) (string, error) {
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("%w: missing client_id or client_secret for OAuth2", models.ErrCarrierNotImplemented)
	}

	c.mu.RLock()
	cached, found := c.tokens[cacheKey]
	c.mu.RUnlock()

	// Return cached token if valid with at least 60s safety buffer
	if found && time.Now().Add(60*time.Second).Before(cached.expiresAt) {
		return cached.accessToken, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check under write lock
	if cached, found := c.tokens[cacheKey]; found && time.Now().Add(60*time.Second).Before(cached.expiresAt) {
		return cached.accessToken, nil
	}

	// Check persistent store if configured
	if c.store != nil {
		if token, expiresAt, ok, err := c.store.GetOAuthToken(ctx, cacheKey); err == nil && ok {
			if time.Now().Add(60*time.Second).Before(expiresAt) {
				c.tokens[cacheKey] = cachedToken{
					accessToken: token,
					expiresAt:   expiresAt,
				}
				return token, nil
			}
		}
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	var req *http.Request
	var err error

	if useBasicAuth {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			return "", err
		}
		authHeader := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, clientSecret)))
		req.Header.Set("Authorization", "Basic "+authHeader)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		data.Set("client_id", clientID)
		data.Set("client_secret", clientSecret)
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	req.Header.Set("User-Agent", ResolveUserAgent())
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: OAuth2 token request failed: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("%w: invalid OAuth2 credentials for %s", models.ErrCarrierNotImplemented, cacheKey)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: OAuth2 token request returned status %d", models.ErrCarrierTemporarilyUnavailable, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", err
	}

	var tokenResp oauthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("%w: malformed OAuth2 token response: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("%w: empty access_token received from %s", models.ErrCarrierTemporarilyUnavailable, cacheKey)
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}

	expiresAt := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
	c.tokens[cacheKey] = cachedToken{
		accessToken: tokenResp.AccessToken,
		expiresAt:   expiresAt,
	}

	if c.store != nil {
		_ = c.store.SetOAuthToken(ctx, cacheKey, tokenResp.AccessToken, expiresAt)
	}

	return tokenResp.AccessToken, nil
}