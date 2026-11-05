package carriers

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestSpainCarriersRegistryAndTracking(t *testing.T) {
	reg := GetRegistry()

	spanishCarriers := []struct {
		slug           string
		trackingNumber string
	}{
		{"activos24", "ACT123456789"},
		{"asendia-es", "AE123456789"},
		{"asigna-es", "ASG123456789"},
		{"celeritas", "CEL123456789"},
		{"correosexpress", "6900123456789012"},
		{"dhl-parcel-es", "JJD0000123456789012"},
		{"envialia", "ENV123456789"},
		{"asm-gls-spain", "ASM123456789"},
		{"inpost-es", "612345678901234567890123"},
		{"instapack-es", "IPK123456789"},
		{"mbe-es", "MBE123456789"},
		{"mrw-spain", "123456789012"},
		{"nacex-spain", "NCX123456789"},
		{"ontime", "ONT123456789"},
		{"paack", "PCK98765432101"},
		{"packlink", "PL123456789"},
		{"redur-spain", "RDR123456789"},
		{"redur-spain-ref", "REF123456789"},
		{"redur-tracking", "RD1234567890"},
		{"sending-es", "SND123456789"},
		{"seur", "36150127630406"},
		{"correos-spain", "PQ4F6P07000000000000000"},
		{"tipsa", "TIP123456789"},
		{"zeleris", "ZEL123456789"},
	}

	if len(spanishCarriers) != 24 {
		t.Fatalf("expected exactly 24 Spanish carriers in test list, got %d", len(spanishCarriers))
	}

	for _, sc := range spanishCarriers {
		t.Run(sc.slug, func(t *testing.T) {
			c, ok := reg.Get(sc.slug)
			if !ok || c == nil {
				t.Fatalf("carrier %s not found in registry", sc.slug)
			}

			if c.Slug() != sc.slug {
				t.Errorf("expected slug %s, got %s", sc.slug, c.Slug())
			}

			if c.Country() != "ES" {
				t.Errorf("carrier %s: expected country ES, got %s", sc.slug, c.Country())
			}

			if c.CountryName() != "Spain" {
				t.Errorf("carrier %s: expected country name Spain, got %s", sc.slug, c.CountryName())
			}

			if c.Name() == "" {
				t.Errorf("carrier %s has empty name", sc.slug)
			}

			if c.Website() == "" {
				t.Errorf("carrier %s has empty website", sc.slug)
			}

			if c.TrackingURL(sc.trackingNumber) == "" {
				t.Errorf("carrier %s returned empty tracking URL", sc.slug)
			}

			shipment, err := c.Track(context.Background(), sc.trackingNumber, models.TrackOptions{Language: "es"})
			if err != nil {
				t.Fatalf("carrier %s track error: %v", sc.slug, err)
			}

			if shipment.TrackingNumber != sc.trackingNumber {
				t.Errorf("carrier %s: expected tracking number %s, got %s", sc.slug, sc.trackingNumber, shipment.TrackingNumber)
			}

			if shipment.Carrier.Slug != sc.slug {
				t.Errorf("carrier %s: expected carrier slug in shipment %s, got %s", sc.slug, sc.slug, shipment.Carrier.Slug)
			}

			if len(shipment.Events) < 3 {
				t.Errorf("carrier %s: expected at least 3 events, got %d", sc.slug, len(shipment.Events))
			}

			// Validate chronological order (Date[i] > Date[i-1])
			for i := 1; i < len(shipment.Events); i++ {
				if !shipment.Events[i].Date.After(shipment.Events[i-1].Date) {
					t.Errorf("carrier %s: events not in chronological order: index %d (%v) is not after index %d (%v)",
						sc.slug, i, shipment.Events[i].Date, i-1, shipment.Events[i-1].Date)
				}
			}

			if shipment.Progress <= 0 {
				t.Errorf("carrier %s: expected progress > 0, got %d", sc.slug, shipment.Progress)
			}

			if shipment.Status == "" {
				t.Errorf("carrier %s: expected status not empty", sc.slug)
			}
		})
	}
}

func TestSpainAliases(t *testing.T) {
	reg := GetRegistry()

	aliases := []struct {
		alias        string
		expectedSlug string
	}{
		{"correos-express", "correosexpress"},
		{"mrw", "mrw-spain"},
		{"gls", "gls"},
		{"inpost", "inpost"},
		{"dhl-parcel", "dhl-parcel"},
	}

	for _, a := range aliases {
		t.Run("alias_"+a.alias, func(t *testing.T) {
			c, ok := reg.Get(a.alias)
			if !ok || c == nil {
				t.Fatalf("expected alias %s to resolve carrier", a.alias)
			}
			if c.Slug() != a.expectedSlug {
				t.Errorf("alias %s: expected underlying carrier slug %s, got %s", a.alias, a.expectedSlug, c.Slug())
			}
		})
	}
}
