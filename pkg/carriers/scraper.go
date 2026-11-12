package carriers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"open-parcels/pkg/models"
)

// isWAFChallenge inspects status codes, headers, and response bodies for WAF, CAPTCHA, or anti-bot blocks.
func isWAFChallenge(statusCode int, body []byte, headers http.Header) bool {
	if statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests {
		return true
	}

	if headers != nil {
		if strings.EqualFold(headers.Get("cf-mitigated"), "challenge") {
			return true
		}
	}

	if len(body) == 0 {
		return false
	}

	sample := body
	if len(sample) > 32768 {
		sample = sample[:32768]
	}
	lower := strings.ToLower(string(sample))

	wafMarkers := []string{
		"cf-browser-verification",
		"cf_chl_opt",
		"__cf_chl_rt_tk",
		"/cdn-cgi/challenge-platform/",
		"attention required! | cloudflare",
		"checking your browser before accessing",
		"enable javascript and cookies to continue",
		"_incapsula_resource",
		"datadome",
		"captcha-delivery",
		"px-captcha",
		"perimeterx",
		"g-recaptcha",
		"h-captcha",
		"cf-turnstile",
		"rgv587_flag",
		"sec_challenge",
		"punish_action",
		"x5sec",
		"bot protection",
	}
	for _, marker := range wafMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}

	if strings.Contains(lower, "just a moment...") && (strings.Contains(lower, "cloudflare") || strings.Contains(lower, "ray id") || strings.Contains(lower, "<html")) {
		return true
	}

	if looksLikeHTML(sample) && strings.Contains(lower, "access denied") && (strings.Contains(lower, "reference #") || strings.Contains(lower, "akamai") || strings.Contains(lower, "edgesuite") || strings.Contains(lower, "firewall") || strings.Contains(lower, "you don't have permission")) {
		return true
	}

	return false
}

// looksLikeHTML returns true when the trimmed response body starts with an HTML tag.
func looksLikeHTML(body []byte) bool {
	trimmed := strings.ToLower(string(bytes.TrimSpace(body)))
	return strings.HasPrefix(trimmed, "<!doctype") ||
		strings.HasPrefix(trimmed, "<html") ||
		strings.HasPrefix(trimmed, "<head") ||
		strings.HasPrefix(trimmed, "<body") ||
		strings.HasPrefix(trimmed, "<script")
}

// classifyScraperHTTPStatus maps non-200 HTTP responses from public scraping endpoints to canonical domain errors.
func classifyScraperHTTPStatus(statusCode int, body []byte, headers http.Header, carrierName string) error {
	switch statusCode {
	case http.StatusNotFound:
		return models.ErrShipmentNotFound
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %w: %s returned HTTP 429", models.ErrCarrierRateLimited, models.ErrCarrierBlocked, carrierName)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s returned HTTP 403", models.ErrCarrierBlocked, carrierName)
	default:
		if isWAFChallenge(statusCode, body, headers) {
			return fmt.Errorf("%w: %w: %s returned WAF challenge (HTTP %d)", models.ErrCarrierBlocked, models.ErrCarrierTemporarilyUnavailable, carrierName, statusCode)
		}
		return fmt.Errorf("%w: HTTP status %d", models.ErrCarrierTemporarilyUnavailable, statusCode)
	}
}

// hasAnyJSONKey reports whether at least one expected schema key is present in a decoded JSON object.
func hasAnyJSONKey(raw map[string]any, keys ...string) bool {
	if raw == nil {
		return false
	}
	for _, k := range keys {
		if _, ok := raw[k]; ok {
			return true
		}
	}
	return false
}
