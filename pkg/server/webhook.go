package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// HeaderWebhookSignature is the HTTP header carrying the HMAC-SHA256 digest.
	HeaderWebhookSignature = "X-OpenParcels-Signature"

	// EnvWebhookSecret is the global environment variable for signing webhooks.
	EnvWebhookSecret = "OPEN_PARCELS_WEBHOOK_SECRET"

	// EnvAllowPrivateWebhooks allows dispatching webhooks to private/loopback IPs when set to "true".
	EnvAllowPrivateWebhooks = "OPEN_PARCELS_ALLOW_PRIVATE_WEBHOOKS"
)

var (
	// ErrSSRFBlocked indicates a webhook target resolved to a restricted or private IP range.
	ErrSSRFBlocked = errors.New("webhook destination blocked by SSRF protection")

	// ErrInvalidWebhookURL indicates the webhook URL is malformed or uses an unsupported scheme.
	ErrInvalidWebhookURL = errors.New("invalid webhook URL")

	// ErrMaxRetriesExceeded indicates all retry attempts for webhook delivery failed.
	ErrMaxRetriesExceeded = errors.New("webhook delivery failed after maximum retries")
)

var blockedCIDRStrings = []string{
	// IPv4 non-routable / private / special ranges
	"0.0.0.0/8",          // Current network
	"10.0.0.0/8",         // Private-Use (RFC 1918)
	"100.64.0.0/10",      // Shared Address Space / CGNAT (RFC 6598)
	"127.0.0.0/8",        // Loopback (RFC 1122)
	"169.254.0.0/16",     // Link-Local (RFC 3927)
	"172.16.0.0/12",      // Private-Use (RFC 1918)
	"192.0.0.0/24",       // IETF Protocol Assignments (RFC 6890)
	"192.0.2.0/24",       // TEST-NET-1 (RFC 5737)
	"192.168.0.0/16",     // Private-Use (RFC 1918)
	"198.18.0.0/15",      // Benchmarking (RFC 2544)
	"198.51.100.0/24",    // TEST-NET-2 (RFC 5737)
	"203.0.113.0/24",     // TEST-NET-3 (RFC 5737)
	"224.0.0.0/4",        // Multicast (RFC 5771)
	"240.0.0.0/4",        // Reserved (RFC 1112)
	"255.255.255.255/32", // Broadcast

	// IPv6 non-routable / private / special ranges
	"::/128",        // Unspecified
	"::1/128",       // Loopback
	"100::/64",      // Discard-Only
	"2001::/23",     // IETF Protocol Assignments
	"2001:db8::/32", // Documentation
	"fc00::/7",      // Unique Local Address (RFC 4193)
	"fe80::/10",     // Link-Local (RFC 4291)
	"ff00::/8",      // Multicast (RFC 4291)
}

var blockedCIDRs []*net.IPNet

func init() {
	for _, cidr := range blockedCIDRStrings {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil && ipNet != nil {
			blockedCIDRs = append(blockedCIDRs, ipNet)
		}
	}
}

func isPrivateWebhookAllowed() bool {
	val := strings.TrimSpace(os.Getenv(EnvAllowPrivateWebhooks))
	return strings.EqualFold(val, "true") || val == "1"
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4.IsLoopback() || ip4.IsPrivate() || ip4.IsLinkLocalUnicast() ||
			ip4.IsLinkLocalMulticast() || ip4.IsMulticast() || ip4.IsUnspecified() {
			return true
		}
		for _, cidr := range blockedCIDRs {
			if cidr.Contains(ip4) {
				return true
			}
		}
		return false
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// HostResolver resolves hostnames to IP addresses.
type HostResolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

// ValidateWebhookURL validates that the webhook URL uses http/https and does not resolve to restricted IPs.
func ValidateWebhookURL(targetURL string) error {
	return ValidateWebhookURLWithResolver(context.Background(), targetURL, net.DefaultResolver)
}

// ValidateWebhookURLWithResolver validates a webhook URL using the provided HostResolver.
func ValidateWebhookURLWithResolver(ctx context.Context, targetURL string, resolver HostResolver) error {
	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidWebhookURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: unsupported scheme %q", ErrInvalidWebhookURL, parsed.Scheme)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidWebhookURL)
	}

	if isPrivateWebhookAllowed() {
		return nil
	}

	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: %s is a restricted IP address", ErrSSRFBlocked, ip.String())
		}
		return nil
	}

	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") || strings.HasSuffix(lowerHost, ".local") {
		return fmt.Errorf("%w: %s is a restricted local hostname", ErrSSRFBlocked, host)
	}

	if resolver == nil {
		resolver = net.DefaultResolver
	}
	ips, err := resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("failed to resolve webhook host %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("%w: no IP addresses resolved for %s", ErrInvalidWebhookURL, host)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: host %s resolved to restricted IP %s", ErrSSRFBlocked, host, ip.String())
		}
	}
	return nil
}

// ComputeHMACSignature computes "sha256=<hex_digest>" over payload using secret.
func ComputeHMACSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMACSignature verifies a signature header against payload and secret in constant time.
func VerifyHMACSignature(secret string, payload []byte, signatureHeader string) bool {
	if secret == "" || signatureHeader == "" {
		return false
	}
	expected := ComputeHMACSignature(secret, payload)
	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}

// DeliveryReport captures the outcome of a webhook delivery attempt sequence.
type DeliveryReport struct {
	RequestID   string        `json:"request_id"`
	WebhookURL  string        `json:"webhook_url"`
	Success     bool          `json:"success"`
	StatusCode  int           `json:"status_code,omitempty"`
	Attempts    int           `json:"attempts"`
	LastError   string        `json:"last_error,omitempty"`
	DeadLetter  bool          `json:"dead_letter"`
	DeliveredAt time.Time     `json:"delivered_at"`
	Duration    time.Duration `json:"duration"`
}

// WebhookOption configures a WebhookDispatcher.
type WebhookOption func(*WebhookDispatcher)

// WithMaxRetries sets the maximum number of retry attempts after the initial attempt.
func WithMaxRetries(retries int) WebhookOption {
	return func(d *WebhookDispatcher) {
		if retries >= 0 {
			d.maxRetries = retries
		}
	}
}

// WithInitialBackoff sets the initial delay before the first retry.
func WithInitialBackoff(backoff time.Duration) WebhookOption {
	return func(d *WebhookDispatcher) {
		if backoff > 0 {
			d.initialBackoff = backoff
		}
	}
}

// WithBackoffFactor sets the exponential multiplier between retries.
func WithBackoffFactor(factor float64) WebhookOption {
	return func(d *WebhookDispatcher) {
		if factor >= 1.0 {
			d.backoffFactor = factor
		}
	}
}

// WithWebhookHTTPClient sets a custom HTTP client for webhook dispatch.
func WithWebhookHTTPClient(client *http.Client) WebhookOption {
	return func(d *WebhookDispatcher) {
		if client != nil {
			d.client = client
		}
	}
}

// WithHostResolver sets a custom host resolver for SSRF checks.
func WithHostResolver(resolver HostResolver) WebhookOption {
	return func(d *WebhookDispatcher) {
		if resolver != nil {
			d.resolver = resolver
		}
	}
}

// WebhookDispatcher handles SSRF-safe, HMAC-signed webhook delivery with exponential backoff retries.
type WebhookDispatcher struct {
	client         *http.Client
	resolver       HostResolver
	maxRetries     int
	initialBackoff time.Duration
	backoffFactor  float64

	mu          sync.RWMutex
	reports     []DeliveryReport
	deadLetters []DeliveryReport
}

// NewWebhookDispatcher creates a WebhookDispatcher with default production settings
// (3 retries with 1s, 2s, 4s exponential backoff and SSRF-safe transport).
func NewWebhookDispatcher(opts ...WebhookOption) *WebhookDispatcher {
	d := &WebhookDispatcher{
		resolver:       net.DefaultResolver,
		maxRetries:     3,
		initialBackoff: 1 * time.Second,
		backoffFactor:  2.0,
	}

	for _, opt := range opts {
		opt(d)
	}

	if d.client == nil {
		d.client = newSSRFSafeHTTPClient(d.resolver)
	}

	return d
}

func newSSRFSafeHTTPClient(resolver HostResolver) *http.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}
			if !isPrivateWebhookAllowed() {
				if ip := net.ParseIP(host); ip != nil {
					if isBlockedIP(ip) {
						return nil, fmt.Errorf("%w: dial to %s blocked", ErrSSRFBlocked, ip.String())
					}
				} else {
					res := resolver
					if res == nil {
						res = net.DefaultResolver
					}
					ips, err := res.LookupIP(ctx, "ip", host)
					if err != nil {
						return nil, err
					}
					for _, ip := range ips {
						if isBlockedIP(ip) {
							return nil, fmt.Errorf("%w: host %s resolved to blocked IP %s", ErrSSRFBlocked, host, ip.String())
						}
					}
				}
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return ValidateWebhookURLWithResolver(req.Context(), req.URL.String(), resolver)
		},
	}
}

var defaultWebhookDispatcher = NewWebhookDispatcher()

// Dispatch serializes payload to JSON (or sends raw bytes if payload is []byte) and dispatches it to webhookURL.
func (d *WebhookDispatcher) Dispatch(ctx context.Context, webhookURL, secret, requestID string, payload any) error {
	var data []byte
	var err error
	if raw, ok := payload.([]byte); ok {
		data = raw
	} else {
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal webhook payload: %w", err)
		}
	}
	return d.DispatchRaw(ctx, webhookURL, secret, requestID, data)
}

// DispatchRaw sends raw JSON bytes to webhookURL with SSRF validation, optional HMAC signature, and exponential retries.
func (d *WebhookDispatcher) DispatchRaw(ctx context.Context, webhookURL, secret, requestID string, data []byte) error {
	startTime := time.Now()
	webhookURL = strings.TrimSpace(webhookURL)

	if err := ValidateWebhookURLWithResolver(ctx, webhookURL, d.resolver); err != nil {
		report := DeliveryReport{
			RequestID:   requestID,
			WebhookURL:  webhookURL,
			Success:     false,
			Attempts:    0,
			LastError:   err.Error(),
			DeadLetter:  true,
			DeliveredAt: time.Now().UTC(),
			Duration:    time.Since(startTime),
		}
		d.recordReport(report)
		log.Printf("[webhook] SSRF_BLOCKED request_id=%s url=%s error=%q", requestID, webhookURL, err.Error())
		return err
	}

	if secret == "" {
		secret = strings.TrimSpace(os.Getenv(EnvWebhookSecret))
	}

	var signature string
	if secret != "" {
		signature = ComputeHMACSignature(secret, data)
	}

	totalAttempts := d.maxRetries + 1
	var lastErr error
	var lastStatus int
	attemptsMade := 0

	for attempt := 1; attempt <= totalAttempts; attempt++ {
		if attempt > 1 {
			backoff := time.Duration(float64(d.initialBackoff) * math.Pow(d.backoffFactor, float64(attempt-2)))
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				lastErr = ctx.Err()
				report := DeliveryReport{
					RequestID:   requestID,
					WebhookURL:  webhookURL,
					Success:     false,
					StatusCode:  lastStatus,
					Attempts:    attemptsMade,
					LastError:   lastErr.Error(),
					DeadLetter:  true,
					DeliveredAt: time.Now().UTC(),
					Duration:    time.Since(startTime),
				}
				d.recordReport(report)
				log.Printf("[webhook] CANCELED request_id=%s url=%s attempts=%d error=%q", requestID, webhookURL, attemptsMade, lastErr.Error())
				return lastErr
			case <-timer.C:
			}
		}

		attemptsMade = attempt
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(data))
		if err != nil {
			report := DeliveryReport{
				RequestID:   requestID,
				WebhookURL:  webhookURL,
				Success:     false,
				Attempts:    attemptsMade,
				LastError:   err.Error(),
				DeadLetter:  true,
				DeliveredAt: time.Now().UTC(),
				Duration:    time.Since(startTime),
			}
			d.recordReport(report)
			log.Printf("[webhook] DEAD_LETTER request_build_error request_id=%s url=%s error=%q", requestID, webhookURL, err.Error())
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "open-parcels-server/1.0")
		if signature != "" {
			req.Header.Set(HeaderWebhookSignature, signature)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			lastStatus = 0
			log.Printf("[webhook] RETRY_ATTEMPT attempt=%d/%d request_id=%s url=%s error=%q", attempt, totalAttempts, requestID, webhookURL, err.Error())
			if ctx.Err() != nil {
				break
			}
			continue
		}

		lastStatus = resp.StatusCode
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			report := DeliveryReport{
				RequestID:   requestID,
				WebhookURL:  webhookURL,
				Success:     true,
				StatusCode:  resp.StatusCode,
				Attempts:    attemptsMade,
				DeadLetter:  false,
				DeliveredAt: time.Now().UTC(),
				Duration:    time.Since(startTime),
			}
			d.recordReport(report)
			log.Printf("[webhook] DELIVERED request_id=%s url=%s status=%d attempts=%d duration=%s", requestID, webhookURL, resp.StatusCode, attemptsMade, report.Duration)
			return nil
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("webhook endpoint returned status %d", resp.StatusCode)
			log.Printf("[webhook] RETRY_ATTEMPT attempt=%d/%d request_id=%s url=%s status=%d", attempt, totalAttempts, requestID, webhookURL, resp.StatusCode)
			continue
		}

		// Non-retryable 4xx (other than 429) or other non-2xx status
		lastErr = fmt.Errorf("non-retryable webhook status %d", resp.StatusCode)
		report := DeliveryReport{
			RequestID:   requestID,
			WebhookURL:  webhookURL,
			Success:     false,
			StatusCode:  resp.StatusCode,
			Attempts:    attemptsMade,
			LastError:   lastErr.Error(),
			DeadLetter:  true,
			DeliveredAt: time.Now().UTC(),
			Duration:    time.Since(startTime),
		}
		d.recordReport(report)
		log.Printf("[webhook] DEAD_LETTER non_retryable request_id=%s url=%s status=%d attempts=%d", requestID, webhookURL, resp.StatusCode, attemptsMade)
		return lastErr
	}

	finalErr := fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
	report := DeliveryReport{
		RequestID:   requestID,
		WebhookURL:  webhookURL,
		Success:     false,
		StatusCode:  lastStatus,
		Attempts:    attemptsMade,
		LastError:   finalErr.Error(),
		DeadLetter:  true,
		DeliveredAt: time.Now().UTC(),
		Duration:    time.Since(startTime),
	}
	d.recordReport(report)
	log.Printf("[webhook] DEAD_LETTER exhausted request_id=%s url=%s status=%d attempts=%d error=%q", requestID, webhookURL, lastStatus, attemptsMade, finalErr.Error())
	return finalErr
}

func (d *WebhookDispatcher) recordReport(r DeliveryReport) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reports = append(d.reports, r)
	if len(d.reports) > 1000 {
		d.reports = d.reports[len(d.reports)-1000:]
	}
	if r.DeadLetter {
		d.deadLetters = append(d.deadLetters, r)
		if len(d.deadLetters) > 1000 {
			d.deadLetters = d.deadLetters[len(d.deadLetters)-1000:]
		}
	}
}

// Reports returns a copy of recorded webhook delivery reports.
func (d *WebhookDispatcher) Reports() []DeliveryReport {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]DeliveryReport, len(d.reports))
	copy(out, d.reports)
	return out
}

// DeadLetters returns a copy of recorded dead-letter failures.
func (d *WebhookDispatcher) DeadLetters() []DeliveryReport {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]DeliveryReport, len(d.deadLetters))
	copy(out, d.deadLetters)
	return out
}
