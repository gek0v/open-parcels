package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	srv := setupTestServer()

	// GET test
	req := httptest.NewRequest(http.MethodGet, "/api/v1/track?number=LP00561234567890&lang=es", nil)
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
	body := []byte(`{"tracking_number":"9400111206213785678901","carrier_hint":"usps"}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/track", bytes.NewReader(body))
	postRec := httptest.NewRecorder()

	srv.handleTrackV1(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for POST track, got %d: %s", postRec.Code, postRec.Body.String())
	}
}

func TestParcelsAppV4Compatibility(t *testing.T) {
	srv := setupTestServer()

	reqPayload := models.ParcelsAppTrackingRequest{
		Language: "en",
		Shipments: []models.ParcelsAppShipmentIn{
			{
				TrackingNumber:     "LP00561234567890",
				DestinationCountry: "ES",
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
