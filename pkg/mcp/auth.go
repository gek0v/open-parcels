package mcp

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// ParseAPIKeys normalizes and deduplicates a slice of API keys and comma-separated key strings.
func ParseAPIKeys(keys []string, csvStrings ...string) []string {
	var result []string
	seen := make(map[string]bool)

	add := func(k string) {
		k = strings.TrimSpace(k)
		if k != "" && !seen[k] {
			seen[k] = true
			result = append(result, k)
		}
	}

	for _, k := range keys {
		add(k)
	}
	for _, csv := range csvStrings {
		if csv == "" {
			continue
		}
		for _, part := range strings.Split(csv, ",") {
			add(part)
		}
	}

	return result
}

// LoadAPIKeysFromEnv reads API keys from OPEN_PARCELS_API_KEY and OPEN_PARCELS_API_KEYS.
func LoadAPIKeysFromEnv() []string {
	return ParseAPIKeys(nil, os.Getenv("OPEN_PARCELS_API_KEY"), os.Getenv("OPEN_PARCELS_API_KEYS"))
}

// ExtractToken retrieves the Bearer token from Authorization or X-API-Key header.
func ExtractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "Bearer") {
			token := strings.TrimSpace(parts[1])
			if token != "" {
				return token
			}
		}
	}

	apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if apiKey != "" {
		return apiKey
	}

	return ""
}

// ValidateAPIKey verifies token against validKeys using constant-time comparison.
func ValidateAPIKey(token string, validKeys []string) bool {
	if token == "" || len(validKeys) == 0 {
		return false
	}

	tokenBytes := []byte(token)
	matched := false
	for _, key := range validKeys {
		keyBytes := []byte(key)
		if len(tokenBytes) == len(keyBytes) {
			if subtle.ConstantTimeCompare(tokenBytes, keyBytes) == 1 {
				matched = true
			}
		}
	}
	return matched
}

// WriteUnauthorized sends a standard JSON 401 Unauthorized response.
func WriteUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("WWW-Authenticate", `Bearer realm="open-parcels"`)
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    "UNAUTHORIZED",
			"message": "Invalid or missing API key.",
		},
	})
}

// AuthMiddleware returns an http.Handler middleware enforcing Bearer or X-API-Key authentication.
// If validKeys is empty, requests pass through without authentication.
func AuthMiddleware(validKeys []string, next http.Handler) http.Handler {
	normalized := ParseAPIKeys(validKeys)
	if len(normalized) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ExtractToken(r)
		if !ValidateAPIKey(token, normalized) {
			WriteUnauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}
