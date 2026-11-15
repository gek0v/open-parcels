package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebHandler_Root(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("expected Content-Type text/html, got %q", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "OPEN PARCELS") {
		t.Errorf("expected dashboard body to contain 'OPEN PARCELS'")
	}
	if !strings.Contains(body, "Tracking Inspector") {
		t.Errorf("expected dashboard body to contain 'Tracking Inspector'")
	}
	if !strings.Contains(body, "Carrier Matrix") {
		t.Errorf("expected dashboard body to contain 'Carrier Matrix'")
	}
}

func TestWebHandler_IndexHTML(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /index.html, got %d", rec.Code)
	}
}

func TestWebHandler_NotFound(t *testing.T) {
	handler := Handler()

	req := httptest.NewRequest(http.MethodGet, "/non-existent-file.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for non-existent file, got %d", rec.Code)
	}
}

func TestIsAsset(t *testing.T) {
	if !IsAsset("/") {
		t.Errorf("expected IsAsset('/') to be true")
	}
	if !IsAsset("/index.html") {
		t.Errorf("expected IsAsset('/index.html') to be true")
	}
	if IsAsset("/api/v1/track") {
		t.Errorf("expected IsAsset('/api/v1/track') to be false")
	}
}
