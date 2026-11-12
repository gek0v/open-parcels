package carriers

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// DefaultRequestTimeout is the standard timeout for carrier tracking requests.
	DefaultRequestTimeout = 10 * time.Second

	// DefaultCarrierUserAgent is the standard browser User-Agent applied to scraper and REST requests.
	DefaultCarrierUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
)

// TransportConfig holds HTTP client and proxy parameters.
type TransportConfig struct {
	Timeout   time.Duration
	UserAgent string
	ProxyURL  string
}

// CarrierCredentials encapsulates API keys and tokens for authenticated carriers.
type CarrierCredentials struct {
	APIKey    string `json:"api_key,omitempty"`
	APISecret string `json:"api_secret,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	AccountID string `json:"account_id,omitempty"`
}

// HasAuth returns true if any credential field is populated.
func (c CarrierCredentials) HasAuth() bool {
	return c.APIKey != "" || c.APISecret != "" || c.ClientID != "" || c.AccountID != ""
}

// GetCarrierCredentials fetches credentials from environment variables for a given carrier slug.
// Precedence:
// 1. OPEN_PARCELS_<CARRIER>_KEY / SECRET / CLIENT_ID / ACCOUNT_ID
// 2. <CARRIER>_API_KEY / <CARRIER>_CLIENT_ID, etc.
func GetCarrierCredentials(carrierSlug string) CarrierCredentials {
	normalized := strings.ToUpper(strings.ReplaceAll(carrierSlug, "-", "_"))

	apiKey := os.Getenv("OPEN_PARCELS_" + normalized + "_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPEN_PARCELS_" + normalized + "_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv(normalized + "_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv(normalized + "_KEY")
	}

	apiSecret := os.Getenv("OPEN_PARCELS_" + normalized + "_SECRET")
	if apiSecret == "" {
		apiSecret = os.Getenv("OPEN_PARCELS_" + normalized + "_CLIENT_SECRET")
	}
	if apiSecret == "" {
		apiSecret = os.Getenv(normalized + "_CLIENT_SECRET")
	}
	if apiSecret == "" {
		apiSecret = os.Getenv(normalized + "_SECRET")
	}

	clientID := os.Getenv("OPEN_PARCELS_" + normalized + "_CLIENT_ID")
	if clientID == "" {
		clientID = os.Getenv(normalized + "_CLIENT_ID")
	}

	accountID := os.Getenv("OPEN_PARCELS_" + normalized + "_ACCOUNT_ID")
	if accountID == "" {
		accountID = os.Getenv(normalized + "_ACCOUNT_ID")
	}

	return CarrierCredentials{
		APIKey:    apiKey,
		APISecret: apiSecret,
		ClientID:  clientID,
		AccountID: accountID,
	}
}

// ResolveUserAgent returns the configured or default User-Agent.
func ResolveUserAgent() string {
	if custom := os.Getenv("OPEN_PARCELS_USER_AGENT"); custom != "" {
		return custom
	}
	return DefaultCarrierUserAgent
}

// ResolveProxyURL returns the proxy URL configured in OPEN_PARCELS_PROXY, OPEN_PARCELS_PROXY_URL, HTTPS_PROXY, or HTTP_PROXY.
func ResolveProxyURL() string {
	for _, key := range []string{
		"OPEN_PARCELS_PROXY",
		"OPEN_PARCELS_PROXY_URL",
		"HTTPS_PROXY",
		"https_proxy",
		"HTTP_PROXY",
		"http_proxy",
	} {
		if proxy := strings.TrimSpace(os.Getenv(key)); proxy != "" {
			return proxy
		}
	}
	return ""
}

// ResolveProxy resolves the proxy URL for an outgoing HTTP request.
// Supports http://, https://, socks5://, and socks5h:// URLs configured via
// OPEN_PARCELS_PROXY / OPEN_PARCELS_PROXY_URL, falling back to HTTP(S)_PROXY or http.ProxyFromEnvironment.
func ResolveProxy(req *http.Request) (*url.URL, error) {
	for _, key := range []string{"OPEN_PARCELS_PROXY", "OPEN_PARCELS_PROXY_URL"} {
		if proxyStr := strings.TrimSpace(os.Getenv(key)); proxyStr != "" {
			if !strings.Contains(proxyStr, "://") {
				proxyStr = "http://" + proxyStr
			}
			return url.Parse(proxyStr)
		}
	}

	if req != nil && req.URL != nil && strings.EqualFold(req.URL.Scheme, "https") {
		for _, k := range []string{"HTTPS_PROXY", "https_proxy"} {
			if p := strings.TrimSpace(os.Getenv(k)); p != "" {
				if !strings.Contains(p, "://") {
					p = "http://" + p
				}
				return url.Parse(p)
			}
		}
	}
	for _, k := range []string{"HTTP_PROXY", "http_proxy"} {
		if p := strings.TrimSpace(os.Getenv(k)); p != "" {
			if !strings.Contains(p, "://") {
				p = "http://" + p
			}
			return url.Parse(p)
		}
	}

	return http.ProxyFromEnvironment(req)
}

// NewCarrierHTTPClient builds a properly configured http.Client with HTTP/HTTPS/SOCKS5 proxy support, TLS settings, and timeout.
func NewCarrierHTTPClient(timeout time.Duration) *http.Client {
	return NewCarrierHTTPClientWithConfig(TransportConfig{
		Timeout: timeout,
	})
}

// NewCarrierHTTPClientWithConfig builds a configured http.Client from TransportConfig.
func NewCarrierHTTPClientWithConfig(cfg TransportConfig) *http.Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultRequestTimeout
	}

	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     60 * time.Second,
		DisableCompression: false,
		Proxy:               ResolveProxy,
		ForceAttemptHTTP2:   true,
	}

	if proxyStr := strings.TrimSpace(cfg.ProxyURL); proxyStr != "" {
		if !strings.Contains(proxyStr, "://") {
			proxyStr = "http://" + proxyStr
		}
		if parsedURL, err := url.Parse(proxyStr); err == nil {
			transport.Proxy = http.ProxyURL(parsedURL)
		}
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// NewCarrierRequest creates an *http.Request pre-populated with standard headers (User-Agent, Accept, Accept-Language).
func NewCarrierRequest(method, targetURL string) (*http.Request, error) {
	req, err := http.NewRequest(method, targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", ResolveUserAgent())
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,es;q=0.8")
	return req, nil
}
