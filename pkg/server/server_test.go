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
	return New(Config{Port: 8089, EnableMCP: true})
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
