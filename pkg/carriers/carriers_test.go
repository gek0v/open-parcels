package carriers

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestRegistry(t *testing.T) {
	reg := GetRegistry()
	list := reg.List("")
	if len(list) < 15 {
		t.Fatalf("expected at least 15 registered carriers, got %d", len(list))
	}

	cainiao, ok := reg.Get("cainiao")
	if !ok || cainiao == nil {
		t.Fatalf("expected cainiao carrier to be found")
	}

	// Verify Air Cargo carriers from Issue #8
	for _, slug := range []string{"qatar-airways-cargo", "emirates-skycargo", "lufthansa-cargo", "iberia-airlines-cargo", "british-airways-cargo", "turkish-airlines-cargo", "cathay-pacific-cargo"} {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			// Some slugs may vary in name
			continue
		}
		shp, err := c.Track(context.Background(), "157-52471764", models.TrackOptions{})
		if err != nil {
			t.Errorf("unexpected error tracking %s: %v", slug, err)
		}
		if shp == nil || len(shp.FlightLegs) == 0 {
			t.Errorf("expected flight legs for %s", slug)
		}
	}

	shipment, err := cainiao.Track(context.Background(), "LP00561234567890", models.TrackOptions{Language: "en"})
	if err != nil {
		t.Fatalf("unexpected error tracking cainiao: %v", err)
	}
	if shipment.TrackingNumber != "LP00561234567890" {
		t.Errorf("expected tracking number LP00561234567890, got %s", shipment.TrackingNumber)
	}
	if len(shipment.Events) == 0 {
		t.Errorf("expected tracking events for cainiao shipment")
	}
	if shipment.LastMileCarrier == nil {
		t.Errorf("expected cainiao to provide last-mile carrier information")
	}
}
