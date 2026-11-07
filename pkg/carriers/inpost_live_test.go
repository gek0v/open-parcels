//go:build integration

package carriers

import (
	"context"
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

	if shipment.Status != models.StatusInfoReceived {
		t.Errorf("expected status %s, got %s", models.StatusInfoReceived, shipment.Status)
	}

	if shipment.StatusText != "En preparación" {
		t.Errorf("expected status text 'En preparación', got %q", shipment.StatusText)
	}

	if len(shipment.Events) == 0 {
		t.Fatalf("expected at least 1 tracking event, got 0")
	}

	foundEvent := false
	for _, ev := range shipment.Events {
		if ev.Date.Format("2006-01-02") == "2026-11-06" &&
			ev.Description == "El envío ha sido registrado informáticamente y estamos a la espera de recibir el paquete." {
			foundEvent = true
			break
		}
	}

	if !foundEvent {
		t.Errorf("did not find expected 2026-11-06 event in events: %+v", shipment.Events)
	}

	if shipment.Origin.Name != "Unknown" {
		t.Errorf("expected Origin.Name 'Unknown', got %q", shipment.Origin.Name)
	}
	if shipment.Origin.Code != "" {
		t.Errorf("expected Origin.Code '', got %q", shipment.Origin.Code)
	}
	if shipment.Destination.Name != "Spain" || shipment.Destination.Code != "ES" {
		t.Errorf("expected Destination 'Spain' (ES), got %q (%q)", shipment.Destination.Name, shipment.Destination.Code)
	}
}
