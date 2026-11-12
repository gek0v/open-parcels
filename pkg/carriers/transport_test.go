package carriers

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestCarrierAdapterInterface(t *testing.T) {
	cainiao := NewCainiaoCarrier()
	var adapter CarrierAdapter = cainiao

	if adapter.ID() != "cainiao" {
		t.Errorf("expected ID cainiao, got %s", adapter.ID())
	}
	caps := adapter.Capabilities()
	if !caps.SupportsLiveAPI {
		t.Errorf("expected cainiao to support live API")
	}

	inpost := NewInPostCarrier()
	var inpostAdapter CarrierAdapter = inpost
	if !inpostAdapter.Capabilities().RequiresPostalCode {
		t.Errorf("expected inpost to require postal code")
	}
	if !inpostAdapter.Capabilities().SupportsWebScraping {
		t.Errorf("expected inpost to support web scraping")
	}

	alias := &AliasCarrier{Carrier: inpost, slug: "inpost-alias"}
	if alias.ID() != "inpost-alias" {
		t.Errorf("expected alias ID inpost-alias, got %s", alias.ID())
	}
	if !alias.Capabilities().RequiresPostalCode {
		t.Errorf("expected alias to delegate capabilities")
	}
}

func TestTransportConfigAndCredentials(t *testing.T) {
	// 1. Credentials resolution with OPEN_PARCELS prefix
	os.Setenv("OPEN_PARCELS_TEST_CARRIER_KEY", "secret-key-123")
	os.Setenv("OPEN_PARCELS_TEST_CARRIER_SECRET", "secret-val-456")
	defer func() {
		os.Unsetenv("OPEN_PARCELS_TEST_CARRIER_KEY")
		os.Unsetenv("OPEN_PARCELS_TEST_CARRIER_SECRET")
	}()

	creds := GetCarrierCredentials("test-carrier")
	if creds.APIKey != "secret-key-123" {
		t.Errorf("expected APIKey secret-key-123, got %s", creds.APIKey)
	}
	if creds.APISecret != "secret-val-456" {
		t.Errorf("expected APISecret secret-val-456, got %s", creds.APISecret)
	}
	if !creds.HasAuth() {
		t.Errorf("expected HasAuth=true")
	}

	// 2. Fallback to standard CARRIER_KEY format
	os.Setenv("DHL_API_KEY", "dhl-direct-key")
	defer os.Unsetenv("DHL_API_KEY")

	dhlCreds := GetCarrierCredentials("dhl")
	if dhlCreds.APIKey != "dhl-direct-key" {
		t.Errorf("expected fallback APIKey dhl-direct-key, got %s", dhlCreds.APIKey)
	}

	// 3. User-Agent and Proxy resolution
	ua := ResolveUserAgent()
	if ua == "" {
		t.Errorf("expected non-empty default user agent")
	}

	os.Setenv("OPEN_PARCELS_USER_AGENT", "CustomAgent/1.0")
	if ResolveUserAgent() != "CustomAgent/1.0" {
		t.Errorf("expected custom user agent")
	}
	os.Unsetenv("OPEN_PARCELS_USER_AGENT")

	// 4. Client creation
	client := NewCarrierHTTPClient(5 * time.Second)
	if client.Timeout != 5*time.Second {
		t.Errorf("expected client timeout 5s, got %v", client.Timeout)
	}

	req, err := NewCarrierRequest("GET", "https://example.com/track")
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}
	if req.Header.Get("User-Agent") == "" {
		t.Errorf("expected User-Agent header populated")
	}
}

func TestProxySupportAndResolution(t *testing.T) {
	// 1. OPEN_PARCELS_PROXY takes precedence
	os.Setenv("OPEN_PARCELS_PROXY", "socks5://127.0.0.1:1080")
	os.Setenv("HTTP_PROXY", "http://fallback.proxy:8080")
	defer func() {
		os.Unsetenv("OPEN_PARCELS_PROXY")
		os.Unsetenv("HTTP_PROXY")
	}()

	resolved := ResolveProxyURL()
	if resolved != "socks5://127.0.0.1:1080" {
		t.Errorf("expected socks5 proxy, got %s", resolved)
	}

	req, err := http.NewRequest("GET", "https://api.example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	proxyURL, err := ResolveProxy(req)
	if err != nil {
		t.Fatalf("unexpected error resolving proxy: %v", err)
	}
	if proxyURL == nil || proxyURL.Scheme != "socks5" || proxyURL.Host != "127.0.0.1:1080" {
		t.Errorf("expected resolved socks5 proxy URL, got %v", proxyURL)
	}

	// 2. Test fallback to HTTP_PROXY when OPEN_PARCELS_PROXY is unset
	os.Unsetenv("OPEN_PARCELS_PROXY")
	fallbackReq, _ := http.NewRequest("GET", "http://api.example.com", nil)
	fallbackProxy, err := ResolveProxy(fallbackReq)
	if err != nil {
		t.Fatalf("unexpected error on fallback proxy: %v", err)
	}
	if fallbackProxy == nil || fallbackProxy.Host != "fallback.proxy:8080" {
		t.Errorf("expected fallback proxy host fallback.proxy:8080, got %v", fallbackProxy)
	}
}

