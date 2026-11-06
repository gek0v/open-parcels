package carriers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
						"time": 1793785505000,
						"timeStr": "2026-11-04 17:45:05",
						"desc": "Package delivered",
						"standerdDesc": "Entregado",
						"actionCode": "GTMS_SIGNED"
					},
					{
						"time": 1793759245000,
						"timeStr": "2026-11-04 10:27:25",
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
								"time": "2026-11-02 10:00:00",
								"date": "2026-11-02"
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

func TestCainiaoLiveOrMockServer(t *testing.T) {
	// Test with a mock Cainiao server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
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
							"time": 1793785505000,
							"timeStr": "2026-11-04 17:45:05",
							"desc": "Package delivered",
							"standerdDesc": "Entregado",
							"actionCode": "GTMS_SIGNED"
						}
					]
				}
			]
		}`))
	}))
	defer server.Close()

	carrier := NewCainiaoCarrier()
	shipment, err := carrier.Track(context.Background(), "32300035210642701471403", models.TrackOptions{
		Language: "es",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking cainiao: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
	if len(shipment.Events) == 0 {
		t.Fatalf("expected events")
	}
	if shipment.Events[0].Status != models.StatusDelivered {
		t.Errorf("expected event StatusDelivered, got %s", shipment.Events[0].Status)
	}
}
