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

