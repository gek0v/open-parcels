// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestDefaultRegistryCarriers(t *testing.T) {
	reg := GetRegistry()

	expectedSlugs := []string{
		"poste-italiane",
		"brt-bartolini",
		"brt-italy-parcelid",
		"sda",
		"gls-italy",
		"poste-san-marino",
	}

	// 1. Acceptance criteria: GetRegistry().List("") includes at least 6 new operators.
	allCarriers := reg.List("")
	slugMap := make(map[string]models.CarrierInfo)
	for _, c := range allCarriers {
		slugMap[c.Slug] = c
	}

	for _, slug := range expectedSlugs {
		if _, ok := slugMap[slug]; !ok {
			t.Errorf("expected carrier %q to be in GetRegistry().List(\"\")", slug)
		}
	}

	// 2. Acceptance criteria: GetRegistry().Get(...) returns valid and operational instances.
	ctx := context.Background()
	for _, slug := range expectedSlugs {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Fatalf("GetRegistry().Get(%q) returned nil or false", slug)
		}

		if c.Slug() != slug {
			t.Errorf("Get(%q) returned carrier with slug %q", slug, c.Slug())
		}
		if c.Name() == "" {
			t.Errorf("carrier %q has empty name", slug)
		}
		if c.Country() != "IT" && c.Country() != "SM" {
			t.Errorf("carrier %q has unexpected country %q", slug, c.Country())
		}

		// Operational test: Track
		shipment, err := c.Track(ctx, "TEST12345678", models.TrackOptions{Language: "it"})
		if err != nil {
			t.Errorf("carrier %q Track() failed: %v", slug, err)
		}
		if shipment == nil {
			t.Errorf("carrier %q Track() returned nil shipment", slug)
		} else {
			if len(shipment.Events) == 0 {
				t.Errorf("carrier %q Track() returned 0 events", slug)
			}
			if shipment.Carrier.Slug != slug {
				t.Errorf("carrier %q Track() returned shipment carrier slug %q", slug, shipment.Carrier.Slug)
			}
		}

		// Operational test: TrackingURL
		url := c.TrackingURL("TEST12345678")
		if url == "" {
			t.Errorf("carrier %q TrackingURL() returned empty string", slug)
		}
	}
}

func TestNewBRTItalyParcelIDCarrierConstructor(t *testing.T) {
	carrier := NewBRTItalyParcelIDCarrier()
	if carrier == nil {
		t.Fatalf("expected NewBRTItalyParcelIDCarrier to return non-nil carrier")
	}
	if carrier.Slug() != "brt-italy-parcelid" {
		t.Errorf("expected slug brt-italy-parcelid, got %s", carrier.Slug())
	}
}
