//go:build integration

package carriers

import (
	"context"
	"strings"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestInPostLiveTracking(t *testing.T) {
	c := NewInPostCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	shipment, err := c.Track(ctx, "75616714", models.TrackOptions{
		PostalCode:     "47140",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("live tracking failed for 75616714: %v", err)
	}

	if shipment.Status != models.StatusInTransit {
		t.Errorf("expected status %s, got %s", models.StatusInTransit, shipment.Status)
	}

	if shipment.StatusText != "En camino" {
		t.Errorf("expected status text 'En camino', got %q", shipment.StatusText)
	}

	if shipment.DeliveredAt != nil {
		t.Errorf("expected DeliveredAt nil for in-transit parcel, got %v", shipment.DeliveredAt)
	}

	if len(shipment.Events) < 4 {
		t.Fatalf("expected at least 4 tracking events, got %d", len(shipment.Events))
	}

	foundOriginEvent := false
	foundLogisticsCenterEvent := false
	for _, ev := range shipment.Events {
		if ev.Date.Format("2006-01-02") == "2026-11-06" &&
			ev.Description == "El envío ha sido registrado informáticamente y estamos a la espera de recibir el paquete." {
			foundOriginEvent = true
		}
		if ev.EventCode == "MMD.1001" && ev.Status == models.StatusInTransit {
			foundLogisticsCenterEvent = true
		}
	}

	if !foundOriginEvent {
		t.Errorf("did not find expected 2026-11-06 event in events: %+v", shipment.Events)
	}
	if !foundLogisticsCenterEvent {
		t.Errorf("did not find expected MMD.1001 in-transit event in events: %+v", shipment.Events)
	}

	if shipment.Origin.Name == "" {
		t.Errorf("expected non-empty Origin.Name, got empty")
	}
	if shipment.Destination.Code != "ES" || !strings.Contains(shipment.Destination.Name, "Spain") {
		t.Errorf("expected Destination Spain (ES), got %q (%q)", shipment.Destination.Name, shipment.Destination.Code)
	}

	// Verify InPost Spain variant
	cSpain := NewInPostSpainCarrier()
	shipmentSpain, errSpain := cSpain.Track(ctx, "75616714", models.TrackOptions{
		PostalCode:     "47140",
		AllowSimulated: false,
	})
	if errSpain != nil {
		t.Fatalf("live tracking failed for inpost-es: %v", errSpain)
	}
	if shipmentSpain.Status != models.StatusInTransit {
		t.Errorf("expected status %s for inpost-es, got %s", models.StatusInTransit, shipmentSpain.Status)
	}
	if shipmentSpain.Carrier.Slug != "inpost-es" {
		t.Errorf("expected carrier slug inpost-es, got %s", shipmentSpain.Carrier.Slug)
	}

	// Verify InPost Italy variant (inpost-it)
	cItaly := NewInPostItalyCarrier()
	shipmentItaly, errItaly := cItaly.Track(ctx, "75616714", models.TrackOptions{
		AllowSimulated: false,
	})
	if errItaly != nil {
		t.Fatalf("live tracking failed for inpost-it: %v", errItaly)
	}
	if shipmentItaly.Status != models.StatusInTransit {
		t.Errorf("expected status %s for inpost-it, got %s", models.StatusInTransit, shipmentItaly.Status)
	}
	if shipmentItaly.Origin.Code != "IT" || shipmentItaly.Origin.Name == "" {
		t.Errorf("expected InPost Italy Origin (IT), got %q (%q)", shipmentItaly.Origin.Name, shipmentItaly.Origin.Code)
	}
	if shipmentItaly.Destination.Code != "ES" || shipmentItaly.Destination.Name == "" {
		t.Errorf("expected InPost Italy Destination (ES), got %q (%q)", shipmentItaly.Destination.Name, shipmentItaly.Destination.Code)
	}
}
