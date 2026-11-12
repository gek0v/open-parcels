package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"open-parcels/pkg/carriers"
	"open-parcels/pkg/models"
	"open-parcels/pkg/storage"
	"path/filepath"
)

func setupTestServer() *Server {
	dispatcher := NewWebhookDispatcher(
		WithInitialBackoff(5*time.Millisecond),
		WithMaxRetries(3),
	)
	return New(Config{Port: 8089, EnableMCP: true, WebhookDispatcher: dispatcher})
}

func TestHealthEndpoint(t *testing.T) {
	srv := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if res["status"] != "ok" {
		t.Errorf("expected status ok, got %v", res["status"])
	}
}

func TestCarriersEndpoint(t *testing.T) {
	srv := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/carriers?query=cainiao", nil)
	rec := httptest.NewRecorder()

	srv.handleCarriers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	total := int(res["total"].(float64))
	if total != 1 {
		t.Errorf("expected 1 cainiao carrier, got %d", total)
	}
}

func TestDetectEndpoint(t *testing.T) {
	srv := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/detect?number=LP00561234567890", nil)
	rec := httptest.NewRecorder()

	srv.handleDetect(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var res struct {
		TrackingNumber string                `json:"tracking_number"`
		Candidates     []models.DetectResult `json:"candidates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(res.Candidates) == 0 {
		t.Fatalf("expected detection candidate for Cainiao LP number")
	}
	if res.Candidates[0].Carrier.Slug != "cainiao" {
		t.Errorf("expected cainiao carrier, got %s", res.Candidates[0].Carrier.Slug)
	}
}

func TestTrackV1Endpoint(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	srv := setupTestServer()

	// GET test
	req := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=LP00561234567890&lang=es&mock=true", nil)
	rec := httptest.NewRecorder()

	srv.handleTrackV1(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var shp models.Shipment
	if err := json.Unmarshal(rec.Body.Bytes(), &shp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if shp.TrackingNumber != "LP00561234567890" {
		t.Errorf("expected tracking number LP00561234567890, got %s", shp.TrackingNumber)
	}
	if len(shp.Events) == 0 {
		t.Errorf("expected events in shipment response")
	}

	// POST test
	body := []byte(`{"tracking_number":"9400111206213785678901","carrier_hint":"usps","allow_simulated":true}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/track", bytes.NewReader(body))
	postRec := httptest.NewRecorder()

	srv.handleTrackV1(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for POST track, got %d: %s", postRec.Code, postRec.Body.String())
	}
}

func TestParcelsAppV4Compatibility(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	srv := setupTestServer()

	reqPayload := models.ParcelsAppTrackingRequest{
		Language: "en",
		Shipments: []models.ParcelsAppShipmentIn{
			{
				TrackingNumber:     "LP00561234567890",
				DestinationCountry: "ES",
				AllowSimulated:     true,
			},
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleParcelsAppTrackings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /trackings, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.ParcelsAppResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response json: %v", err)
	}

	if resp.RequestID == "" {
		t.Errorf("expected non-empty request_id")
	}
	if !resp.Done || resp.Status != "completed" {
		t.Errorf("expected done=true, status=completed, got %v / %s", resp.Done, resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	resItem := resp.Results[0]
	if resItem.Shipment == nil || resItem.Shipment.TrackingID != "LP00561234567890" {
		t.Errorf("expected matching shipment result")
	}

	// Test GET /trackings/{request_id}
	getReq := httptest.NewRequest(http.MethodGet, "/trackings/"+resp.RequestID, nil)
	getRec := httptest.NewRecorder()
	srv.handleParcelsAppGetTracking(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /trackings/%s, got %d", resp.RequestID, getRec.Code)
	}
}

func TestServerLifecycle(t *testing.T) {
	srv := New(Config{Port: 9999, EnableMCP: true})
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected error on shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("server shutdown timed out")
	}
}

func TestNegativeErrorPaths(t *testing.T) {
	srv := setupTestServer()

	// 1. Detect without ?number= parameter should return 400 Bad Request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/detect", nil)
	rec := httptest.NewRecorder()
	srv.handleDetect(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing number in detect, got %d", rec.Code)
	}

	// 2. Track GET without ?number= parameter should return 400 Bad Request
	reqTrack := httptest.NewRequest(http.MethodGet, "/api/v1/track", nil)
	recTrack := httptest.NewRecorder()
	srv.handleTrackV1(recTrack, reqTrack)
	if recTrack.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing number in track GET, got %d", recTrack.Code)
	}

	// 3. Track POST with invalid JSON should return 400 Bad Request
	reqInvalidJSON := httptest.NewRequest(http.MethodPost, "/api/v1/track", bytes.NewReader([]byte("{invalid-json")))
	recInvalidJSON := httptest.NewRecorder()
	srv.handleTrackV1(recInvalidJSON, reqInvalidJSON)
	if recInvalidJSON.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid json in track POST, got %d", recInvalidJSON.Code)
	}

	// 4. ParcelsApp trackings with empty shipments should return 400 Bad Request
	emptyPayload := []byte(`{"shipments":[]}`)
	reqEmpty := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(emptyPayload))
	recEmpty := httptest.NewRecorder()
	srv.handleParcelsAppTrackings(recEmpty, reqEmpty)
	if recEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty shipments array, got %d", recEmpty.Code)
	}

	// 5. GET /trackings with non-existent request ID should return 404 Not Found
	getReq := httptest.NewRequest(http.MethodGet, "/trackings/req_non_existent", nil)
	getRec := httptest.NewRecorder()
	srv.handleParcelsAppGetTracking(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent tracking request ID, got %d", getRec.Code)
	}

	// 6. Track with unknown shipment should return 404 Not Found
	reqNotFound := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=FAKE000000000", nil)
	recNotFound := httptest.NewRecorder()
	srv.handleTrackV1(recNotFound, reqNotFound)
	if recNotFound.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent shipment, got %d", recNotFound.Code)
	}

	// 7. Track requiring postal code without one should return 422 Unprocessable Entity
	reqPostalCode := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=7561674&carrier=inpost", nil)
	recPostalCode := httptest.NewRecorder()
	srv.handleTrackV1(recPostalCode, reqPostalCode)
	if recPostalCode.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for postal code required, got %d", recPostalCode.Code)
	}

	// 8. Track for catalog carrier without live driver should return 501 Not Implemented
	reqNotImpl := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=TIP123456789&carrier=tipsa", nil)
	recNotImpl := httptest.NewRecorder()
	srv.handleTrackV1(recNotImpl, reqNotImpl)
	if recNotImpl.Code != http.StatusNotImplemented {
		t.Errorf("expected 501 for unimplemented carrier, got %d", recNotImpl.Code)
	}
}

func TestParcelsAppAsyncPolling(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	srv := setupTestServer()

	reqPayload := models.ParcelsAppTrackingRequest{
		Language: "es",
		Shipments: []models.ParcelsAppShipmentIn{
			{
				TrackingNumber: "LP00561234567890",
				AllowSimulated: true,
			},
		},
	}
	body, _ := json.Marshal(reqPayload)

	// 1. Submit asynchronous tracking request (?async=true)
	req := httptest.NewRequest(http.MethodPost, "/trackings?async=true", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.handleParcelsAppTrackings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for async request, got %d", rec.Code)
	}

	var asyncResp models.ParcelsAppResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &asyncResp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if asyncResp.Status != "processing" || asyncResp.Done != false {
		t.Errorf("expected status 'processing' and done=false, got %s / %t", asyncResp.Status, asyncResp.Done)
	}
	if asyncResp.RequestID == "" {
		t.Fatalf("expected non-empty request_id")
	}

	// 2. Poll until completed (goroutine executes concurrently)
	var finalResp *models.ParcelsAppResponse
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		resp, found := srv.getRequest(asyncResp.RequestID)
		if found && resp.Done && resp.Status == "completed" {
			finalResp = resp
			break
		}
	}

	if finalResp == nil {
		t.Fatalf("async job did not complete within timeout")
	}
	if len(finalResp.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(finalResp.Results))
	}
}

func TestParcelsAppWebhookCallback(t *testing.T) {
	t.Setenv("OPEN_PARCELS_ALLOW_PRIVATE_WEBHOOKS", "true")
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	webhookChan := make(chan models.ParcelsAppResponse, 1)
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload models.ParcelsAppResponse
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			webhookChan <- payload
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	srv := setupTestServer()

	reqPayload := models.ParcelsAppTrackingRequest{
		Language:   "en",
		WebhookURL: webhookServer.URL,
		Shipments: []models.ParcelsAppShipmentIn{
			{
				TrackingNumber: "LP00561234567890",
				AllowSimulated: true,
			},
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.handleParcelsAppTrackings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for webhook request, got %d", rec.Code)
	}

	select {
	case received := <-webhookChan:
		if received.Status != "completed" || !received.Done {
			t.Errorf("expected completed webhook payload, got %s / %t", received.Status, received.Done)
		}
		if len(received.Results) == 0 {
			t.Errorf("expected results in webhook payload")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for webhook callback")
	}
}

func TestParcelsAppTTLExpiration(t *testing.T) {
	srv := setupTestServer()

	reqID := "req_expired_test"
	expiredItem := &models.ParcelsAppResponse{
		RequestID: reqID,
		Status:    "completed",
		Done:      true,
	}

	// Insert with negative TTL (already expired)
	srv.setRequest(reqID, expiredItem, -10*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/trackings/"+reqID, nil)
	rec := httptest.NewRecorder()
	srv.handleParcelsAppGetTracking(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for expired record, got %d", rec.Code)
	}
}

func TestTrackV1WAFBlockedAndScraperSchemaChanged(t *testing.T) {
	srv := setupTestServer()

	// 1. WAF 403 -> 503 CARRIER_BLOCKED
	wafFixture, err := carriers.LoadFixture("inpost_waf_blocked.html")
	if err != nil {
		t.Fatalf("failed to load waf fixture: %v", err)
	}
	teardownWAF := carriers.SetInPostHTTPClientForTest(carriers.NewFixtureClient(http.StatusForbidden, wafFixture, "text/html"))

	reqWAF := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=75616714&carrier=inpost&postal_code=47140", nil)
	recWAF := httptest.NewRecorder()
	srv.handleTrackV1(recWAF, reqWAF)
	teardownWAF()

	if recWAF.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable for WAF block, got %d: %s", recWAF.Code, recWAF.Body.String())
	}
	var errRespWAF map[string]any
	if err := json.Unmarshal(recWAF.Body.Bytes(), &errRespWAF); err != nil {
		t.Fatalf("invalid json error response: %v", err)
	}
	if errObj, ok := errRespWAF["error"].(map[string]any); !ok || errObj["code"] != "CARRIER_BLOCKED" {
		t.Errorf("expected code CARRIER_BLOCKED, got %v", errRespWAF)
	}

	// 2. Broken HTML schema -> 502 SCRAPER_ERROR
	brokenFixture, err := carriers.LoadFixture("inpost_broken_schema.html")
	if err != nil {
		t.Fatalf("failed to load broken schema fixture: %v", err)
	}
	teardownBroken := carriers.SetInPostHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, brokenFixture, "text/html"))

	reqBroken := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=75616714&carrier=inpost&postal_code=47140", nil)
	recBroken := httptest.NewRecorder()
	srv.handleTrackV1(recBroken, reqBroken)
	teardownBroken()

	if recBroken.Code != http.StatusBadGateway {
		t.Errorf("expected 502 Bad Gateway for scraper schema change, got %d: %s", recBroken.Code, recBroken.Body.String())
	}
	var errRespBroken map[string]any
	if err := json.Unmarshal(recBroken.Body.Bytes(), &errRespBroken); err != nil {
		t.Fatalf("invalid json error response: %v", err)
	}
	if errObj, ok := errRespBroken["error"].(map[string]any); !ok || errObj["code"] != "SCRAPER_ERROR" {
		t.Errorf("expected code SCRAPER_ERROR, got %v", errRespBroken)
	}
}

func TestParcelsAppWAFBlockedAndScraperError(t *testing.T) {
	srv := setupTestServer()

	// 1. Blocked via WAF 403
	teardownCorreosWAF := carriers.SetCorreosHTTPClientForTest(carriers.NewFixtureClient(http.StatusForbidden, []byte("WAF Blocked"), "text/html"))
	payload := models.ParcelsAppTrackingRequest{
		Language: "es",
		Shipments: []models.ParcelsAppShipmentIn{
			{TrackingNumber: "PQ4F6P07000000000000000", CarrierHint: "correos-spain"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.handleParcelsAppTrackings(rec, req)
	teardownCorreosWAF()

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when all shipments rejected, got %d: %s", rec.Code, rec.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	errMap := res["error"].(map[string]any)
	rejected := errMap["rejected"].([]any)
	if len(rejected) != 1 || rejected[0].(map[string]any)["code"] != "CARRIER_BLOCKED" {
		t.Errorf("expected rejected item with CARRIER_BLOCKED, got %v", rejected)
	}

	// 2. Broken schema
	brokenFixture, err := carriers.LoadFixture("correos_broken_schema.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardownCorreosSchema := carriers.SetCorreosHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, brokenFixture, "application/json"))
	req2 := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(body))
	rec2 := httptest.NewRecorder()
	srv.handleParcelsAppTrackings(rec2, req2)
	teardownCorreosSchema()

	var res2 map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &res2); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	errMap2 := res2["error"].(map[string]any)
	rejected2 := errMap2["rejected"].([]any)
	if len(rejected2) != 1 || rejected2[0].(map[string]any)["code"] != "SCRAPER_ERROR" {
		t.Errorf("expected rejected item with SCRAPER_ERROR, got %v", rejected2)
	}
}

func TestParcelsAppPersistenceAcrossServerRestart(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	dbPath := filepath.Join(t.TempDir(), "server_restart.db")

	// Instance 1: Create server backed by SQLiteStore
	st1, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite store 1: %v", err)
	}
	srv1 := New(Config{Port: 0, EnableMCP: false, Store: st1})

	reqPayload := models.ParcelsAppTrackingRequest{
		Language: "es",
		Shipments: []models.ParcelsAppShipmentIn{
			{
				TrackingNumber: "LP00561234567890",
				AllowSimulated: true,
			},
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/trackings?async=true", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv1.handleParcelsAppTrackings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var asyncResp models.ParcelsAppResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &asyncResp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	// Wait for async job completion on srv1
	var completed bool
	for i := 0; i < 40; i++ {
		time.Sleep(50 * time.Millisecond)
		resp, found := srv1.getRequest(asyncResp.RequestID)
		if found && resp.Done && resp.Status == "completed" {
			completed = true
			break
		}
	}
	if !completed {
		t.Fatalf("async job did not complete on srv1")
	}

	// Simulate srv1 shutdown & store close
	srv1.wg.Wait()
	if err := st1.Close(); err != nil {
		t.Fatalf("failed to close st1: %v", err)
	}

	// Instance 2: Reopen SQLiteStore and start fresh Server instance (empty memory)
	st2, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen sqlite store 2: %v", err)
	}
	defer st2.Close()

	srv2 := New(Config{Port: 0, EnableMCP: false, Store: st2})

	getReq := httptest.NewRequest(http.MethodGet, "/trackings/"+asyncResp.RequestID, nil)
	getRec := httptest.NewRecorder()
	srv2.handleParcelsAppGetTracking(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from restarted server srv2, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var restoredResp models.ParcelsAppResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &restoredResp); err != nil {
		t.Fatalf("invalid json from srv2: %v", err)
	}
	if !restoredResp.Done || restoredResp.Status != "completed" {
		t.Errorf("expected completed status after restart, got done=%v status=%s", restoredResp.Done, restoredResp.Status)
	}
	if len(restoredResp.Results) != 1 || restoredResp.Results[0].Shipment.TrackingID != "LP00561234567890" {
		t.Errorf("expected preserved tracking result after restart, got %+v", restoredResp.Results)
	}
}

func TestServerV1TrackCacheAndForceRefresh(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	st := storage.NewMemoryStore()
	defer st.Close()
	srv := New(Config{Port: 0, EnableMCP: false, Store: st})

	// 1. First request -> from_cache: false
	r1 := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=LP00561234567890&allow_simulated=true", nil)
	w1 := httptest.NewRecorder()
	srv.handleTrackV1(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}
	var s1 models.Shipment
	_ = json.Unmarshal(w1.Body.Bytes(), &s1)
	if s1.FromCache {
		t.Errorf("expected first request FromCache=false")
	}

	// 2. Second request -> from_cache: true
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=LP00561234567890&allow_simulated=true", nil)
	w2 := httptest.NewRecorder()
	srv.handleTrackV1(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var s2 models.Shipment
	_ = json.Unmarshal(w2.Body.Bytes(), &s2)
	if !s2.FromCache {
		t.Errorf("expected second request FromCache=true")
	}

	// 3. Third request with force_refresh=true -> from_cache: false
	r3 := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=LP00561234567890&allow_simulated=true&force_refresh=true", nil)
	w3 := httptest.NewRecorder()
	srv.handleTrackV1(w3, r3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w3.Code)
	}
	var s3 models.Shipment
	_ = json.Unmarshal(w3.Body.Bytes(), &s3)
	if s3.FromCache {
		t.Errorf("expected force_refresh request FromCache=false")
	}
}

func TestServerOpenModeWithoutAuth(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	// Ensure env vars are empty
	t.Setenv("OPEN_PARCELS_API_KEY", "")
	t.Setenv("OPEN_PARCELS_API_KEYS", "")
	t.Setenv("OPEN_PARCELS_RATE_LIMIT", "")

	srv := New(Config{Port: 0, EnableMCP: true})
	defer srv.Close()
	handler := srv.Handler()

	// 1. /health -> 200
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Errorf("open mode /health: expected 200, got %d", recHealth.Code)
	}

	// 2. /api/v1/track -> 200 without auth header
	reqTrack := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=LP00561234567890&mock=true", nil)
	recTrack := httptest.NewRecorder()
	handler.ServeHTTP(recTrack, reqTrack)
	if recTrack.Code != http.StatusOK {
		t.Errorf("open mode /api/v1/track: expected 200, got %d: %s", recTrack.Code, recTrack.Body.String())
	}

	// 3. /trackings -> 200 without auth header
	trackingsBody := []byte(`{"shipments":[{"tracking_number":"LP00561234567890","allow_simulated":true}]}`)
	reqTrackings := httptest.NewRequest(http.MethodPost, "/trackings", bytes.NewReader(trackingsBody))
	recTrackings := httptest.NewRecorder()
	handler.ServeHTTP(recTrackings, reqTrackings)
	if recTrackings.Code != http.StatusOK {
		t.Errorf("open mode /trackings: expected 200, got %d: %s", recTrackings.Code, recTrackings.Body.String())
	}

	// 4. /mcp -> 200 without auth header
	mcpBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	reqMCP := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(mcpBody))
	recMCP := httptest.NewRecorder()
	handler.ServeHTTP(recMCP, reqMCP)
	if recMCP.Code != http.StatusOK {
		t.Errorf("open mode /mcp: expected 200, got %d: %s", recMCP.Code, recMCP.Body.String())
	}
}

func TestServerAuthMiddleware(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	srv := New(Config{
		Port:      0,
		EnableMCP: true,
		APIKeys:   []string{"secret-token-alpha", "secret-token-beta"},
	})
	defer srv.Close()
	handler := srv.Handler()

	// 1. /health must remain accessible without any token
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected /health to return 200 without auth, got %d", recHealth.Code)
	}

	trackingsBody := []byte(`{"shipments":[{"tracking_number":"LP00561234567890","allow_simulated":true}]}`)
	mcpBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)

	protectedEndpoints := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{"api_v1_track", http.MethodGet, "/api/v1/track?number=LP00561234567890&mock=true", nil},
		{"api_v1_carriers", http.MethodGet, "/api/v1/carriers?query=cainiao", nil},
		{"api_v1_detect", http.MethodGet, "/api/v1/detect?number=LP00561234567890", nil},
		{"trackings", http.MethodPost, "/trackings", trackingsBody},
		{"mcp", http.MethodPost, "/mcp", mcpBody},
	}

	for _, ep := range protectedEndpoints {
		// Missing token -> 401
		reqNoAuth := httptest.NewRequest(ep.method, ep.path, bytes.NewReader(ep.body))
		recNoAuth := httptest.NewRecorder()
		handler.ServeHTTP(recNoAuth, reqNoAuth)
		if recNoAuth.Code != http.StatusUnauthorized {
			t.Errorf("%s without token: expected 401 Unauthorized, got %d", ep.name, recNoAuth.Code)
		}

		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(recNoAuth.Body.Bytes(), &errResp); err != nil {
			t.Errorf("%s without token: expected JSON error body, got %v", ep.name, err)
		} else if errResp.Error.Code != "UNAUTHORIZED" {
			t.Errorf("%s without token: expected error code UNAUTHORIZED, got %s", ep.name, errResp.Error.Code)
		}

		// Invalid token -> 401
		reqBad := httptest.NewRequest(ep.method, ep.path, bytes.NewReader(ep.body))
		reqBad.Header.Set("Authorization", "Bearer invalid-token")
		recBad := httptest.NewRecorder()
		handler.ServeHTTP(recBad, reqBad)
		if recBad.Code != http.StatusUnauthorized {
			t.Errorf("%s with wrong Bearer token: expected 401 Unauthorized, got %d", ep.name, recBad.Code)
		}

		// Valid Bearer token -> 200 OK
		reqBearer := httptest.NewRequest(ep.method, ep.path, bytes.NewReader(ep.body))
		reqBearer.Header.Set("Authorization", "Bearer secret-token-alpha")
		recBearer := httptest.NewRecorder()
		handler.ServeHTTP(recBearer, reqBearer)
		if recBearer.Code != http.StatusOK {
			t.Errorf("%s with valid Bearer token: expected 200 OK, got %d: %s", ep.name, recBearer.Code, recBearer.Body.String())
		}

		// Valid X-API-Key header -> 200 OK
		reqAPIKey := httptest.NewRequest(ep.method, ep.path, bytes.NewReader(ep.body))
		reqAPIKey.Header.Set("X-API-Key", "secret-token-beta")
		recAPIKey := httptest.NewRecorder()
		handler.ServeHTTP(recAPIKey, reqAPIKey)
		if recAPIKey.Code != http.StatusOK {
			t.Errorf("%s with valid X-API-Key: expected 200 OK, got %d: %s", ep.name, recAPIKey.Code, recAPIKey.Body.String())
		}
	}
}

func TestServerEnvAuthAndRateLimitConfig(t *testing.T) {
	t.Setenv("OPEN_PARCELS_API_KEY", "env-primary")
	t.Setenv("OPEN_PARCELS_API_KEYS", "env-secondary, env-tertiary")
	t.Setenv("OPEN_PARCELS_RATE_LIMIT", "120/m")
	t.Setenv("OPEN_PARCELS_RATE_BURST", "3")

	srv := New(Config{Port: 0, EnableMCP: true})
	defer srv.Close()

	keys := srv.APIKeys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys from env, got %d: %v", len(keys), keys)
	}
	if srv.RateLimiter() == nil {
		t.Fatalf("expected rate limiter configured from env")
	}

	handler := srv.Handler()
	for _, k := range []string{"env-primary", "env-secondary", "env-tertiary"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/carriers?query=cainiao", nil)
		req.Header.Set("X-API-Key", k)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for env key %q, got %d", k, rec.Code)
		}
	}
}

func TestServerRateLimiting(t *testing.T) {
	srv := New(Config{
		Port:      0,
		EnableMCP: true,
		RateLimit: 1.0, // 1 req/sec
		RateBurst: 2,   // allow burst of 2
	})
	defer srv.Close()
	handler := srv.Handler()

	// 1st and 2nd requests from IP 10.0.0.1 succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/carriers?query=cainiao", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200 OK, got %d", i+1, rec.Code)
		}
	}

	// 3rd immediate request from same IP exceeds burst -> 429 Too Many Requests + Retry-After
	reqLimited := httptest.NewRequest(http.MethodGet, "/api/v1/carriers?query=cainiao", nil)
	reqLimited.RemoteAddr = "10.0.0.1:12345"
	recLimited := httptest.NewRecorder()
	handler.ServeHTTP(recLimited, reqLimited)
	if recLimited.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request: expected 429 Too Many Requests, got %d", recLimited.Code)
	}
	retryAfter := recLimited.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Errorf("expected Retry-After header on 429 response")
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recLimited.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("expected valid JSON error body, got %v", err)
	}
	if errResp.Error.Code != "TOO_MANY_REQUESTS" {
		t.Errorf("expected TOO_MANY_REQUESTS code, got %s", errResp.Error.Code)
	}

	// Different IP should still succeed
	reqOtherIP := httptest.NewRequest(http.MethodGet, "/api/v1/carriers?query=cainiao", nil)
	reqOtherIP.RemoteAddr = "10.0.0.2:54321"
	recOtherIP := httptest.NewRecorder()
	handler.ServeHTTP(recOtherIP, reqOtherIP)
	if recOtherIP.Code != http.StatusOK {
		t.Errorf("different IP: expected 200 OK, got %d", recOtherIP.Code)
	}

	// /health should not be blocked by rate limit even from limited IP
	for i := 0; i < 5; i++ {
		reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
		reqHealth.RemoteAddr = "10.0.0.1:12345"
		recHealth := httptest.NewRecorder()
		handler.ServeHTTP(recHealth, reqHealth)
		if recHealth.Code != http.StatusOK {
			t.Fatalf("/health attempt %d: expected 200 OK despite IP rate limit, got %d", i+1, recHealth.Code)
		}
	}
}

func TestRateLimiterCleanupAndAPIKeyIsolation(t *testing.T) {
	rl := NewRateLimiter(1.0, 1, 50*time.Millisecond)
	defer rl.Stop()

	allowed, _ := rl.Allow("key:alpha")
	if !allowed {
		t.Fatalf("expected first request for key:alpha to succeed")
	}
	allowed, _ = rl.Allow("key:beta")
	if !allowed {
		t.Fatalf("expected first request for key:beta to succeed (isolated bucket)")
	}
	allowed, retryAfter := rl.Allow("key:alpha")
	if allowed {
		t.Fatalf("expected second immediate request for key:alpha to be limited")
	}
	if retryAfter < time.Second {
		t.Errorf("expected retryAfter >= 1s, got %v", retryAfter)
	}

	if rl.BucketCount() != 2 {
		t.Fatalf("expected 2 buckets, got %d", rl.BucketCount())
	}

	// Wait for inactive TTL and background cleaner
	time.Sleep(150 * time.Millisecond)
	rl.Cleanup()
	if rl.BucketCount() != 0 {
		t.Errorf("expected inactive buckets to be evicted after TTL, remaining: %d", rl.BucketCount())
	}
}

func TestParseRateLimit(t *testing.T) {
	cases := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"", 0, false},
		{"10", 10, false},
		{"2.5", 2.5, false},
		{"5/s", 5, false},
		{"60/m", 1, false},
		{"3600/h", 1, false},
		{"0", 0, true},
		{"-5", 0, true},
		{"abc/s", 0, true},
	}

	for _, tc := range cases {
		got, err := ParseRateLimit(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseRateLimit(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("ParseRateLimit(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
