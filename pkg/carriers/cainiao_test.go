package carriers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"open-parcels/pkg/models"
)

func TestCainiaoParsingModuleAndData(t *testing.T) {
	// 1. Test parsing "module" structure (live Cainiao API format)
	moduleJSON := `{
		"success": true,
		"module": [
			{
				"mailNo": "32300035210642701471403",
				"originCountry": "España",
				"destCountry": "España",
				"status": "DELIVERED",
				"statusDesc": "Entregado",
				"detailList": [
					{
						"time": 1700050000000,
						"timeStr": "2024-11-15 17:45:05",
						"desc": "Package delivered",
						"standerdDesc": "Entregado",
						"actionCode": "GTMS_SIGNED"
					},
					{
						"time": 1700020000000,
						"timeStr": "2024-11-15 10:27:25",
						"desc": "EN REPARTO",
						"standerdDesc": "En proceso de entrega",
						"actionCode": "GTMS_DELIVERING"
					}
				]
			}
		]
	}`

	var cRespModule cainiaoResponse
	if err := json.Unmarshal([]byte(moduleJSON), &cRespModule); err != nil {
		t.Fatalf("failed unmarshaling module json: %v", err)
	}

	if len(cRespModule.Module) == 0 {
		t.Fatalf("expected non-empty module slice")
	}

	item := cRespModule.Module[0]
	if item.Status != "DELIVERED" {
		t.Errorf("expected item status DELIVERED, got %s", item.Status)
	}

	var events []models.TrackingEvent
	for _, det := range item.DetailList {
		evTime, rawDate := parseCainiaoTime(det)
		status, stage := normalizeCainiaoEvent(det)
		events = append(events, models.TrackingEvent{
			Date:        evTime,
			RawDate:     rawDate,
			Status:      status,
			Stage:       stage,
			Description: det.Desc,
		})
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Status != models.StatusDelivered {
		t.Errorf("expected first event status %s, got %s", models.StatusDelivered, events[0].Status)
	}
	if events[0].Stage != models.StageDelivery {
		t.Errorf("expected first event stage %s, got %s", models.StageDelivery, events[0].Stage)
	}
	if events[1].Status != models.StatusOutForDelivery {
		t.Errorf("expected second event status %s, got %s", models.StatusOutForDelivery, events[1].Status)
	}

	overallStatus, _ := normalizeCainiaoShipmentStatus(item.Status, events)
	if overallStatus != models.StatusDelivered {
		t.Errorf("expected overallStatus %s, got %s", models.StatusDelivered, overallStatus)
	}

	// 2. Test backward-compatibility parsing "data" structure
	dataJSON := `{
		"success": true,
		"data": [
			{
				"mailNo": "LP00561234567890",
				"originCountry": "China",
				"destCountry": "ES",
				"destMailNo": "6900123456789012",
				"destCp": "Correos Express",
				"sectionList": [
					{
						"sectionName": "Transit",
						"statusDesc": "In transit",
						"detailList": [
							{
								"desc": "Package arrived at airport",
								"time": "2024-11-14 10:00:00",
								"date": "2024-11-14"
							}
						]
					}
				]
			}
		]
	}`

	var cRespData cainiaoResponse
	if err := json.Unmarshal([]byte(dataJSON), &cRespData); err != nil {
		t.Fatalf("failed unmarshaling data json: %v", err)
	}

	if len(cRespData.Data) == 0 {
		t.Fatalf("expected non-empty data slice")
	}

	dataItem := cRespData.Data[0]
	if len(dataItem.SectionList) == 0 || len(dataItem.SectionList[0].DetailList) == 0 {
		t.Fatalf("expected sectionList with detailList in data item")
	}
}

func TestCainiaoTrackDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCainiaoHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCainiaoCarrier()
	shipment, err := c.Track(context.Background(), "32300035210642701471403", models.TrackOptions{
		Language: "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking delivered fixture: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
	if shipment.Progress != 100 {
		t.Errorf("expected Progress 100, got %d", shipment.Progress)
	}
	if shipment.LastMileCarrier == nil {
		t.Fatalf("expected LastMileCarrier to be set")
	}
	if shipment.LastMileCarrier.Slug != "correosexpress" {
		t.Errorf("expected LastMileCarrier slug correosexpress, got %s", shipment.LastMileCarrier.Slug)
	}
	if shipment.LastMileTrackingNumber != "6900123456789012" {
		t.Errorf("expected LastMileTrackingNumber 6900123456789012, got %s", shipment.LastMileTrackingNumber)
	}
	if len(shipment.Events) < 3 {
		t.Errorf("expected at least 3 events, got %d", len(shipment.Events))
	}
}

func TestCainiaoTrackInTransitFixture(t *testing.T) {
	fixtureData, err := LoadFixture("cainiao_intransit.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCainiaoHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCainiaoCarrier()
	shipment, err := c.Track(context.Background(), "LP00561234567890", models.TrackOptions{
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking in transit fixture: %v", err)
	}

	if shipment.Status != models.StatusInTransit {
		t.Errorf("expected StatusInTransit, got %s", shipment.Status)
	}
	if shipment.LastMileCarrier == nil {
		t.Fatalf("expected LastMileCarrier to be set for handover to Correos")
	}
	if shipment.LastMileCarrier.Slug != "correos-spain" {
		t.Errorf("expected LastMileCarrier slug correos-spain, got %s", shipment.LastMileCarrier.Slug)
	}
	if shipment.LastMileTrackingNumber != "PQ4100123456789" {
		t.Errorf("expected LastMileTrackingNumber PQ4100123456789, got %s", shipment.LastMileTrackingNumber)
	}
}

func TestCainiaoNotFoundFixture(t *testing.T) {
	fixtureData, err := LoadFixture("cainiao_not_found.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCainiaoHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewCainiaoCarrier()
	_, err = c.Track(context.Background(), "LP99999999999999", models.TrackOptions{})
	if err == nil {
		t.Fatalf("expected ErrShipmentNotFound, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got %v", err)
	}
}

func TestCainiaoAntiBotChallenge(t *testing.T) {
	c := NewCainiaoCarrier()

	// 1. AntiBot JSON body with RGV587_FLAG
	fixtureData, err := LoadFixture("cainiao_antibot.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetCainiaoHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	_, err = c.Track(context.Background(), "32300035210642701471403", models.TrackOptions{})
	if err == nil {
		t.Fatalf("expected ErrCarrierTemporarilyUnavailable, got nil")
	}
	if !errors.Is(err, models.ErrCarrierTemporarilyUnavailable) {
		t.Errorf("expected ErrCarrierTemporarilyUnavailable, got: %v", err)
	}

	// 2. HTTP 403 Forbidden
	cainiaoHTTPClient = NewFixtureClient(http.StatusForbidden, []byte("Forbidden WAF"), "text/plain")
	_, err = c.Track(context.Background(), "32300035210642701471403", models.TrackOptions{})
	if !errors.Is(err, models.ErrCarrierTemporarilyUnavailable) {
		t.Errorf("expected ErrCarrierTemporarilyUnavailable for HTTP 403, got: %v", err)
	}

	// 3. HTTP 429 Rate Limit
	cainiaoHTTPClient = NewFixtureClient(http.StatusTooManyRequests, []byte("Rate limit exceeded"), "text/plain")
	_, err = c.Track(context.Background(), "32300035210642701471403", models.TrackOptions{})
	if !errors.Is(err, models.ErrCarrierRateLimited) {
		t.Errorf("expected ErrCarrierRateLimited for HTTP 429, got: %v", err)
	}
}
