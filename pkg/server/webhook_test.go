package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"open-parcels/pkg/carriers"
	"open-parcels/pkg/models"
)

type mockResolver struct {
	records map[string][]net.IP
	err     error
}

func (m *mockResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if m.err != nil {
		return nil, m.err
	}
	if ips, ok := m.records[host]; ok {
		return ips, nil
	}
	return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
}

func TestSSRFProtection_BlockedIPsAndHosts(t *testing.T) {
	t.Setenv(EnvAllowPrivateWebhooks, "")

	blockedURLs := []string{
		"http://127.0.0.1/webhook",
		"http://127.0.0.1:8080/callback",
		"http://127.1.2.3/hook",
		"http://10.0.0.1/hook",
		"http://10.255.255.1:9000/hook",
		"http://172.16.0.1/hook",
		"http://172.31.255.254/hook",
		"http://192.168.0.1/hook",
		"http://192.168.1.100:3000/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://0.0.0.0/hook",
		"http://100.64.0.1/hook",
		"http://224.0.0.1/hook",
		"http://[::1]:8080/hook",
		"http://[::]/hook",
		"http://[fc00::1]/hook",
		"http://[fd12:3456:789a::1]/hook",
		"http://[fe80::1]/hook",
		"http://[ff02::1]/hook",
		"http://localhost:8080/hook",
		"http://sub.localhost/hook",
		"http://internal.local/hook",
	}

	for _, target := range blockedURLs {
		err := ValidateWebhookURL(target)
		if err == nil {
			t.Errorf("expected SSRF block for %q, got nil", target)
			continue
		}
		if !errors.Is(err, ErrSSRFBlocked) {
			t.Errorf("expected ErrSSRFBlocked for %q, got %v", target, err)
		}
	}

	// Test invalid schemes and malformed URLs
	invalidURLs := []string{
		"",
		"ftp://8.8.8.8/hook",
		"file:///etc/passwd",
		"gopher://127.0.0.1:70/_",
		"not-a-valid-url",
	}
	for _, target := range invalidURLs {
		err := ValidateWebhookURL(target)
		if err == nil {
			t.Errorf("expected error for invalid URL %q, got nil", target)
		}
	}
}

func TestSSRFProtection_DNSResolution(t *testing.T) {
	t.Setenv(EnvAllowPrivateWebhooks, "")

	resolver := &mockResolver{
		records: map[string][]net.IP{
			"rebind-loopback.example.com": {net.ParseIP("127.0.0.1")},
			"rebind-private.example.com":  {net.ParseIP("10.0.0.1")},
			"rebind-mixed.example.com":    {net.ParseIP("93.184.216.34"), net.ParseIP("192.168.1.50")},
			"rebind-ipv6.example.com":     {net.ParseIP("fc00::1")},
			"public.example.com":          {net.ParseIP("93.184.216.34")},
		},
	}

	blockedHosts := []string{
		"https://rebind-loopback.example.com/hook",
		"https://rebind-private.example.com/hook",
		"https://rebind-mixed.example.com/hook",
		"https://rebind-ipv6.example.com/hook",
	}
	for _, target := range blockedHosts {
		err := ValidateWebhookURLWithResolver(context.Background(), target, resolver)
		if !errors.Is(err, ErrSSRFBlocked) {
			t.Errorf("expected ErrSSRFBlocked for resolved host %q, got %v", target, err)
		}
	}

	if err := ValidateWebhookURLWithResolver(context.Background(), "https://public.example.com/hook", resolver); err != nil {
		t.Errorf("expected public host to be allowed, got %v", err)
	}
}

func TestSSRFProtection_AllowPrivateEnvBypass(t *testing.T) {
	localTargets := []string{
		"http://127.0.0.1:8080/webhook",
		"http://10.0.0.1/webhook",
		"http://192.168.1.10/webhook",
		"http://localhost:3000/webhook",
		"http://[::1]:8080/webhook",
	}

	t.Setenv(EnvAllowPrivateWebhooks, "true")
	for _, target := range localTargets {
		if err := ValidateWebhookURL(target); err != nil {
			t.Errorf("expected %q to be allowed when %s=true, got %v", target, EnvAllowPrivateWebhooks, err)
		}
	}

	t.Setenv(EnvAllowPrivateWebhooks, "false")
	for _, target := range localTargets {
		if err := ValidateWebhookURL(target); !errors.Is(err, ErrSSRFBlocked) {
			t.Errorf("expected %q to be blocked when %s=false, got %v", target, EnvAllowPrivateWebhooks, err)
		}
	}
}

func TestHMACSignatureGenerationAndVerification(t *testing.T) {
	secret := "super-secret-signing-key"
	payload := []byte(`{"request_id":"req_123","status":"completed","done":true}`)

	sig := ComputeHMACSignature(secret, payload)
	if !strings.HasPrefix(sig, "sha256=") {
		t.Fatalf("expected sha256= prefix, got %q", sig)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if sig != expected {
		t.Fatalf("signature mismatch: got %q, want %q", sig, expected)
	}

	if !VerifyHMACSignature(secret, payload, sig) {
		t.Error("expected VerifyHMACSignature to return true for valid signature")
	}
	if VerifyHMACSignature("wrong-secret", payload, sig) {
		t.Error("expected VerifyHMACSignature to return false for wrong secret")
	}
	if VerifyHMACSignature(secret, []byte(`{"tampered":true}`), sig) {
		t.Error("expected VerifyHMACSignature to return false for tampered payload")
	}
	if VerifyHMACSignature(secret, payload, "") {
		t.Error("expected VerifyHMACSignature to return false for empty signature")
	}
}

func TestWebhookDispatcher_HMACHeaderDelivery(t *testing.T) {
	t.Setenv(EnvAllowPrivateWebhooks, "true")

	secret := "my-webhook-secret"
	var receivedSig string
	var receivedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get(HeaderWebhookSignature)
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dispatcher := NewWebhookDispatcher(WithInitialBackoff(5 * time.Millisecond))
	payload := map[string]any{"request_id": "req_sig_1", "done": true}

	if err := dispatcher.Dispatch(context.Background(), srv.URL, secret, "req_sig_1", payload); err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}

	if !VerifyHMACSignature(secret, receivedBody, receivedSig) {
		t.Errorf("received signature %q failed verification against body %s", receivedSig, string(receivedBody))
	}

	// Without secret: header should be absent
	receivedSig = "should-be-cleared"
	t.Setenv(EnvWebhookSecret, "")
	if err := dispatcher.Dispatch(context.Background(), srv.URL, "", "req_sig_2", payload); err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}
	if receivedSig != "" {
		t.Errorf("expected empty %s header when no secret set, got %q", HeaderWebhookSignature, receivedSig)
	}
}

func TestWebhookDispatcher_ExponentialRetriesSuccess(t *testing.T) {
	t.Setenv(EnvAllowPrivateWebhooks, "true")

	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dispatcher := NewWebhookDispatcher(
		WithMaxRetries(3),
		WithInitialBackoff(15*time.Millisecond),
		WithBackoffFactor(2.0),
	)

	start := time.Now()
	err := dispatcher.Dispatch(context.Background(), srv.URL, "", "req_retry_ok", map[string]string{"ok": "1"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts (1 initial + 2 retries), got %d", got)
	}
	// Minimum expected backoff: 15ms + 30ms = 45ms
	if elapsed < 40*time.Millisecond {
		t.Errorf("expected exponential backoff delay >= 40ms, got %s", elapsed)
	}

	reports := dispatcher.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected 1 delivery report, got %d", len(reports))
	}
	if !reports[0].Success || reports[0].DeadLetter || reports[0].Attempts != 3 || reports[0].StatusCode != 200 {
		t.Errorf("unexpected delivery report: %+v", reports[0])
	}
	if len(dispatcher.DeadLetters()) != 0 {
		t.Errorf("expected 0 dead letters, got %d", len(dispatcher.DeadLetters()))
	}
}

func TestWebhookDispatcher_ExponentialRetriesExhaustion(t *testing.T) {
	t.Setenv(EnvAllowPrivateWebhooks, "true")

	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	dispatcher := NewWebhookDispatcher(
		WithMaxRetries(3),
		WithInitialBackoff(5*time.Millisecond),
		WithBackoffFactor(2.0),
	)

	err := dispatcher.Dispatch(context.Background(), srv.URL, "", "req_retry_fail", map[string]string{"fail": "1"})
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Fatalf("expected ErrMaxRetriesExceeded, got %v", err)
	}
	if got := attempts.Load(); got != 4 {
		t.Fatalf("expected 4 total attempts (1 initial + 3 retries), got %d", got)
	}

	deadLetters := dispatcher.DeadLetters()
	if len(deadLetters) != 1 {
		t.Fatalf("expected 1 dead letter record, got %d", len(deadLetters))
	}
	dl := deadLetters[0]
	if dl.Success || !dl.DeadLetter || dl.Attempts != 4 || dl.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("unexpected dead letter record: %+v", dl)
	}
}

func TestWebhookDispatcher_RetryOn429AndNoRetryOn400(t *testing.T) {
	t.Setenv(EnvAllowPrivateWebhooks, "true")

	// 1. 429 Too Many Requests should retry and succeed
	var attempts429 atomic.Int32
	srv429 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts429.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv429.Close()

	dispatcher := NewWebhookDispatcher(
		WithMaxRetries(3),
		WithInitialBackoff(5*time.Millisecond),
	)

	if err := dispatcher.Dispatch(context.Background(), srv429.URL, "", "req_429", map[string]string{"id": "1"}); err != nil {
		t.Fatalf("expected 429 retry to succeed, got %v", err)
	}
	if attempts429.Load() != 2 {
		t.Errorf("expected 2 attempts for 429, got %d", attempts429.Load())
	}

	// 2. 400 Bad Request should NOT retry
	var attempts400 atomic.Int32
	srv400 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts400.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv400.Close()

	err := dispatcher.Dispatch(context.Background(), srv400.URL, "", "req_400", map[string]string{"id": "2"})
	if err == nil {
		t.Fatal("expected error on 400 response")
	}
	if attempts400.Load() != 1 {
		t.Errorf("expected 1 attempt without retry on 400, got %d", attempts400.Load())
	}
	if len(dispatcher.DeadLetters()) != 1 {
		t.Errorf("expected 1 dead letter from 400 failure, got %d", len(dispatcher.DeadLetters()))
	}
}

func TestWebhookDispatcher_ContextCancellation(t *testing.T) {
	t.Setenv(EnvAllowPrivateWebhooks, "true")

	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dispatcher := NewWebhookDispatcher(
		WithMaxRetries(3),
		WithInitialBackoff(5*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	err := dispatcher.Dispatch(ctx, srv.URL, "", "req_cancel", map[string]string{"id": "cancel"})
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
	if attempts.Load() != 1 {
		t.Errorf("expected only 1 attempt before context canceled backoff, got %d", attempts.Load())
	}
}

func TestServerParcelsAppWebhook_HMACAndSSRFIntegration(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	type capturedWebhook struct {
		signature string
		body      []byte
	}
	webhookChan := make(chan capturedWebhook, 1)
	var attemptCount atomic.Int32

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attemptCount.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, _ := io.ReadAll(r.Body)
		webhookChan <- capturedWebhook{
			signature: r.Header.Get(HeaderWebhookSignature),
			body:      body,
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Part 1: Without OPEN_PARCELS_ALLOW_PRIVATE_WEBHOOKS, callback to 127.0.0.1 is blocked by SSRF
	t.Setenv(EnvAllowPrivateWebhooks, "")
	srvBlocked := setupTestServer()
	reqPayload := models.ParcelsAppTrackingRequest{
		Language:      "en",
		WebhookURL:    webhookServer.URL,
		WebhookSecret: "payload-secret-123",
		Shipments: []models.ParcelsAppShipmentIn{
			{TrackingNumber: "LP00561234567890", AllowSimulated: true},
		},
	}
	bodyBytes, _ := json.Marshal(reqPayload)
	req := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()
	srvBlocked.handleParcelsAppTrackings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// Wait briefly for async processing to record dead letter
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(srvBlocked.WebhookDispatcher().DeadLetters()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if attemptCount.Load() != 0 {
		t.Fatalf("expected 0 HTTP attempts when blocked by SSRF, got %d", attemptCount.Load())
	}
	if len(srvBlocked.WebhookDispatcher().DeadLetters()) != 1 {
		t.Fatalf("expected 1 dead letter for SSRF block, got %d", len(srvBlocked.WebhookDispatcher().DeadLetters()))
	}

	// Part 2: With OPEN_PARCELS_ALLOW_PRIVATE_WEBHOOKS=true, retries 500 -> 200 and sends valid HMAC signature
	t.Setenv(EnvAllowPrivateWebhooks, "true")
	srvAllowed := setupTestServer()
	req2 := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(bodyBytes))
	rec2 := httptest.NewRecorder()
	srvAllowed.handleParcelsAppTrackings(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec2.Code)
	}

	select {
	case captured := <-webhookChan:
		if !VerifyHMACSignature("payload-secret-123", captured.body, captured.signature) {
			t.Errorf("HMAC signature verification failed: header=%q", captured.signature)
		}
		var parsed models.ParcelsAppResponse
		if err := json.Unmarshal(captured.body, &parsed); err != nil {
			t.Fatalf("failed to unmarshal webhook body: %v", err)
		}
		if !parsed.Done || parsed.Status != "completed" {
			t.Errorf("expected completed webhook response, got %s / %t", parsed.Status, parsed.Done)
		}
		if attemptCount.Load() != 2 {
			t.Errorf("expected 2 attempts (1 failed 500 + 1 succeeded 200), got %d", attemptCount.Load())
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for retried webhook delivery")
	}
}

func TestServerParcelsAppWebhook_GlobalEnvAndQuerySecret(t *testing.T) {
	t.Setenv(EnvAllowPrivateWebhooks, "true")

	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	type captured struct {
		sig  string
		body []byte
	}
	ch := make(chan captured, 2)
	srvHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ch <- captured{sig: r.Header.Get(HeaderWebhookSignature), body: b}
		w.WriteHeader(http.StatusOK)
	}))
	defer srvHTTP.Close()

	srv := setupTestServer()

	// 1. Query parameter webhook_secret
	payloadNoSecret := models.ParcelsAppTrackingRequest{
		Language:   "en",
		WebhookURL: srvHTTP.URL,
		Shipments: []models.ParcelsAppShipmentIn{
			{TrackingNumber: "LP00561234567890", AllowSimulated: true},
		},
	}
	b, _ := json.Marshal(payloadNoSecret)
	req := httptest.NewRequest(http.MethodPost, "/trackings?webhook_secret=query-secret-xyz", bytes.NewReader(b))
	rec := httptest.NewRecorder()
	srv.handleParcelsAppTrackings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	select {
	case got := <-ch:
		if !VerifyHMACSignature("query-secret-xyz", got.body, got.sig) {
			t.Errorf("query secret HMAC verification failed: %q", got.sig)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for query secret webhook")
	}

	// 2. Global environment variable OPEN_PARCELS_WEBHOOK_SECRET
	t.Setenv(EnvWebhookSecret, "global-env-secret-999")
	reqEnv := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(b))
	recEnv := httptest.NewRecorder()
	srv.handleParcelsAppTrackings(recEnv, reqEnv)
	if recEnv.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recEnv.Code)
	}

	select {
	case got := <-ch:
		if !VerifyHMACSignature("global-env-secret-999", got.body, got.sig) {
			t.Errorf("global env secret HMAC verification failed: %q", got.sig)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for env secret webhook")
	}
}
