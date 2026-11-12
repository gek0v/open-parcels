package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"open-parcels/pkg/mcp"
)

// ParseRateLimit parses a rate limit string into requests per second.
// Supported formats:
//   - "10" or "10.5" -> 10 or 10.5 req/sec
//   - "10/s", "10/sec", "10/second" -> 10 req/sec
//   - "60/m", "60/min", "60/minute" -> 1 req/sec
//   - "3600/h", "3600/hour" -> 1 req/sec
func ParseRateLimit(s string) (float64, error) {
	clean := strings.TrimSpace(strings.ToLower(s))
	if clean == "" {
		return 0, nil
	}

	var divisor float64 = 1.0
	var numStr string

	switch {
	case strings.HasSuffix(clean, "/second"):
		numStr = strings.TrimSuffix(clean, "/second")
		divisor = 1.0
	case strings.HasSuffix(clean, "/sec"):
		numStr = strings.TrimSuffix(clean, "/sec")
		divisor = 1.0
	case strings.HasSuffix(clean, "/s"):
		numStr = strings.TrimSuffix(clean, "/s")
		divisor = 1.0
	case strings.HasSuffix(clean, "/minute"):
		numStr = strings.TrimSuffix(clean, "/minute")
		divisor = 60.0
	case strings.HasSuffix(clean, "/min"):
		numStr = strings.TrimSuffix(clean, "/min")
		divisor = 60.0
	case strings.HasSuffix(clean, "/m"):
		numStr = strings.TrimSuffix(clean, "/m")
		divisor = 60.0
	case strings.HasSuffix(clean, "/hour"):
		numStr = strings.TrimSuffix(clean, "/hour")
		divisor = 3600.0
	case strings.HasSuffix(clean, "/h"):
		numStr = strings.TrimSuffix(clean, "/h")
		divisor = 3600.0
	default:
		numStr = clean
		divisor = 1.0
	}

	val, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
	if err != nil || val <= 0 {
		return 0, fmt.Errorf("invalid rate limit value '%s'", s)
	}

	return val / divisor, nil
}

type clientBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter manages per-client token buckets with automatic background eviction of stale buckets.
type RateLimiter struct {
	mu          sync.Mutex
	rate        float64
	burst       float64
	buckets     map[string]*clientBucket
	inactiveTTL time.Duration
	stopChan    chan struct{}
	stopOnce    sync.Once
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(rate float64, burst int, inactiveTTL time.Duration) *RateLimiter {
	if burst <= 0 {
		burst = int(math.Ceil(rate))
	}
	if burst < 1 {
		burst = 1
	}
	if inactiveTTL <= 0 {
		inactiveTTL = 5 * time.Minute
	}

	rl := &RateLimiter{
		rate:        rate,
		burst:       float64(burst),
		buckets:     make(map[string]*clientBucket),
		inactiveTTL: inactiveTTL,
		stopChan:    make(chan struct{}),
	}

	cleanupInterval := inactiveTTL / 2
	if cleanupInterval < 500*time.Millisecond {
		cleanupInterval = 500 * time.Millisecond
	} else if cleanupInterval > time.Minute {
		cleanupInterval = time.Minute
	}

	go rl.cleanerLoop(cleanupInterval)

	return rl
}

func (rl *RateLimiter) cleanerLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.Cleanup()
		case <-rl.stopChan:
			return
		}
	}
}

// Stop terminates the background cleaner goroutine.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.stopChan)
	})
}

// Cleanup removes entries that have been inactive longer than inactiveTTL.
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, b := range rl.buckets {
		if now.Sub(b.lastRefill) > rl.inactiveTTL {
			delete(rl.buckets, key)
		}
	}
}

// BucketCount returns the current count of active buckets (for testing and metrics).
func (rl *RateLimiter) BucketCount() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.buckets)
}

// Allow evaluates if a request with clientKey can proceed under token-bucket rules.
func (rl *RateLimiter) Allow(clientKey string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[clientKey]
	if !exists {
		rl.buckets[clientKey] = &clientBucket{
			tokens:     rl.burst - 1.0,
			lastRefill: now,
		}
		return true, 0
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true, 0
	}

	needed := 1.0 - b.tokens
	waitSec := needed / rl.rate
	retryAfter := time.Duration(math.Ceil(waitSec)) * time.Second
	if retryAfter < time.Second {
		retryAfter = time.Second
	}

	return false, retryAfter
}

// ResolveClientKey determines client identity (API key or IP).
func ResolveClientKey(r *http.Request) string {
	if token := mcp.ExtractToken(r); token != "" {
		return "key:" + token
	}

	return "ip:" + ClientIP(r)
}

// ClientIP extracts the remote client IP from headers or RemoteAddr.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

// WriteRateLimitExceeded writes a 429 Too Many Requests response with standard JSON and Retry-After header.
func WriteRateLimitExceeded(w http.ResponseWriter, retryAfter time.Duration) {
	retrySec := int(math.Ceil(retryAfter.Seconds()))
	if retrySec < 1 {
		retrySec = 1
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Retry-After", strconv.Itoa(retrySec))
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    "TOO_MANY_REQUESTS",
			"message": "Rate limit exceeded. Try again later.",
		},
	})
}
