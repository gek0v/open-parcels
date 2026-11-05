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

	// Verify APAC carriers from Issue #3
	for _, slug := range []string{"australia-post", "japan-post", "korea-post", "india-post", "singapore-post", "new-zealand-post", "ninja-van", "delhivery", "bluedart", "yamato-japan", "sagawa-jp"} {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("expected APAC carrier %s to be registered", slug)
		}
	}

	// Verify global carriers from Issue #2
	for _, slug := range []string{"speedpak", "pitney-bowes", "aramex", "landmark-global", "wanbexpress", "winit", "ceva-logistics", "dsv"} {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("expected global carrier %s to be registered", slug)
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

func TestPortugalCarriers(t *testing.T) {
	reg := GetRegistry()

	carriers := []struct {
		slug            string
		name            string
		country         string
		expectedType    models.CarrierType
		testTrackNumber string
	}{
		{
			slug:            "delnext",
			name:            "Delnext",
			country:         "PT",
			expectedType:    models.CarrierTypeExpress,
			testTrackNumber: "DLN123456789PT",
		},
		{
			slug:            "dpd-portugal",
			name:            "DPD Portugal",
			country:         "PT",
			expectedType:    models.CarrierTypeExpress,
			testTrackNumber: "09801234567890",
		},
		{
			slug:            "portugal-post",
			name:            "Portugal Post - CTT",
			country:         "PT",
			expectedType:    models.CarrierTypePostal,
			testTrackNumber: "DA123456785PT",
		},
		{
			slug:            "wepickup-pt",
			name:            "WePickUp",
			country:         "PT",
			expectedType:    models.CarrierTypeLogistics,
			testTrackNumber: "WPU123456789",
		},
	}

	for _, tc := range carriers {
		t.Run(tc.slug, func(t *testing.T) {
			carrier, ok := reg.Get(tc.slug)
			if !ok || carrier == nil {
				t.Fatalf("carrier %s not found in registry", tc.slug)
			}

			if carrier.Slug() != tc.slug {
				t.Errorf("expected slug %s, got %s", tc.slug, carrier.Slug())
			}
			if carrier.Name() != tc.name {
				t.Errorf("expected name %s, got %s", tc.name, carrier.Name())
			}
			if carrier.Country() != tc.country {
				t.Errorf("expected country %s, got %s", tc.country, carrier.Country())
			}
			if carrier.CountryName() != "Portugal" {
				t.Errorf("expected country name Portugal, got %s", carrier.CountryName())
			}
			if carrier.Type() != tc.expectedType {
				t.Errorf("expected type %s, got %s", tc.expectedType, carrier.Type())
			}
			if carrier.Website() == "" {
				t.Errorf("expected non-empty website for %s", tc.slug)
			}
			trackingURL := carrier.TrackingURL(tc.testTrackNumber)
			if trackingURL == "" {
				t.Errorf("expected non-empty tracking URL for %s", tc.slug)
			}

			info := carrier.Info()
			if info.Slug != tc.slug {
				t.Errorf("expected info slug %s, got %s", tc.slug, info.Slug)
			}

			ctx := context.Background()
			shipment, err := carrier.Track(ctx, tc.testTrackNumber, models.TrackOptions{Language: "pt"})
			if err != nil {
				t.Fatalf("unexpected error tracking %s: %v", tc.slug, err)
			}

			if shipment.TrackingNumber != tc.testTrackNumber {
				t.Errorf("expected tracking number %s, got %s", tc.testTrackNumber, shipment.TrackingNumber)
			}
			if shipment.Origin.Code != "PT" {
				t.Errorf("expected origin code PT, got %s", shipment.Origin.Code)
			}
			if shipment.Destination.Code != "PT" {
				t.Errorf("expected destination code PT, got %s", shipment.Destination.Code)
			}
			if shipment.Carrier.Slug != tc.slug {
				t.Errorf("expected shipment carrier slug %s, got %s", tc.slug, shipment.Carrier.Slug)
			}
			if len(shipment.Events) == 0 {
				t.Errorf("expected events for %s", tc.slug)
			}

			// Validate events chronological ordering (oldest to newest)
			for i := 1; i < len(shipment.Events); i++ {
				if shipment.Events[i].Date.Before(shipment.Events[i-1].Date) {
					t.Errorf("events not in chronological order for %s at index %d", tc.slug, i)
				}
			}

			if shipment.Status == "" {
				t.Errorf("expected non-empty status for %s", tc.slug)
			}
			if shipment.Progress <= 0 {
				t.Errorf("expected positive progress for %s, got %d", tc.slug, shipment.Progress)
			}
		})
	}
}
