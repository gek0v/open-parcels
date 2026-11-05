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

	// Verify LATAM carriers from Issue #14
	for _, slug := range []string{"correios-brazil", "mexico-post", "correo-argentino", "peru-post", "uruguay-post", "estafeta", "chilexpress-cl", "andreani", "starken"} {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("expected LATAM carrier %s to be registered", slug)
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
