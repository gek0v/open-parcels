package carriers

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestRegistry(t *testing.T) {
	reg := GetRegistry()
	list := reg.List("")
	if len(list) < 280 {
		t.Fatalf("expected at least 280 registered carriers (including 271 from China/HK), got %d", len(list))
	}

	for _, slug := range []string{"china-post", "sf-express", "hongkong-post", "yanwen", "4px", "four-px", "139-express", "sytrack", "cne-express"} {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("expected carrier %s to be registered", slug)
			continue
		}
		shp, err := c.Track(context.Background(), "TEST123456", models.TrackOptions{DestinationCountry: "ES"})
		if err != nil {
			t.Errorf("unexpected error tracking %s: %v", slug, err)
		}
		if shp == nil || len(shp.Events) == 0 {
			t.Errorf("expected events for %s", slug)
		}
	}

	cainiao, ok := reg.Get("cainiao")
	if !ok || cainiao == nil {
		t.Fatalf("expected cainiao carrier to be found")
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
