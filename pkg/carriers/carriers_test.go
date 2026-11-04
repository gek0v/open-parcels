// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestRegistry(t *testing.T) {
	reg := GetRegistry()
	list := reg.List("")
	if len(list) < 25 {
		t.Fatalf("expected at least 25 registered carriers, got %d", len(list))
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

func TestAllRegisteredCarriersTrack(t *testing.T) {
	reg := GetRegistry()
	all := reg.AllCarriers()
	if len(all) == 0 {
		t.Fatalf("expected carriers in AllCarriers()")
	}

	for slug, carrier := range all {
		t.Run(slug, func(t *testing.T) {
			if carrier.Slug() == "" {
				t.Errorf("empty slug for carrier")
			}
			if carrier.Name() == "" {
				t.Errorf("empty name for carrier %s", slug)
			}
			info := carrier.Info()
			if info.Slug != carrier.Slug() {
				t.Errorf("info slug mismatch for %s", slug)
			}

			// Call Track
			s, err := carrier.Track(context.Background(), "TRACK1234567890", models.TrackOptions{
				Language:           "en",
				DestinationCountry: "ES",
			})
			if err != nil {
				t.Fatalf("carrier %s track returned error: %v", slug, err)
			}
			if s == nil {
				t.Fatalf("carrier %s track returned nil shipment", slug)
			}
			if s.TrackingNumber != "TRACK1234567890" {
				t.Errorf("carrier %s unexpected tracking number: %s", slug, s.TrackingNumber)
			}
			if len(s.Events) == 0 {
				t.Errorf("carrier %s returned zero events", slug)
			}
		})
	}
}

func TestRegistryListQueries(t *testing.T) {
	reg := GetRegistry()

	// Empty query
	all := reg.List("")
	if len(all) == 0 {
		t.Fatalf("expected non-empty list for empty query")
	}

	// Filter by slug
	maersk := reg.List("maersk")
	if len(maersk) == 0 {
		t.Errorf("expected to find maersk")
	}

	// Filter by name
	royal := reg.List("Royal Mail")
	if len(royal) == 0 {
		t.Errorf("expected to find Royal Mail")
	}

	// Filter by country code
	esCarriers := reg.List("ES")
	if len(esCarriers) == 0 {
		t.Errorf("expected Spanish carriers for 'ES'")
	}

	// Filter by country name
	spainCarriers := reg.List("Spain")
	if len(spainCarriers) == 0 {
		t.Errorf("expected Spanish carriers for 'Spain'")
	}

	// Filter by non-existent
	none := reg.List("non-existent-carrier-query-xyz")
	if len(none) != 0 {
		t.Errorf("expected empty list for non-existent carrier query, got %d", len(none))
	}
}

func TestBaseCarrierMethods(t *testing.T) {
	baseNoURL := NewBaseCarrier("test-carrier", "Test Carrier", "US", "United States", models.CarrierTypeExpress, "https://test.com", "")
	if baseNoURL.TrackingURL("123") != "" {
		t.Errorf("expected empty tracking url when template is empty")
	}
	if baseNoURL.Country() != "US" || baseNoURL.CountryName() != "United States" {
		t.Errorf("unexpected country data")
	}
	if baseNoURL.Website() != "https://test.com" {
		t.Errorf("unexpected website")
	}
	if baseNoURL.Type() != models.CarrierTypeExpress {
		t.Errorf("unexpected type")
	}

	baseWithURL := NewBaseCarrier("test-carrier-2", "Test Carrier 2", "GLOBAL", "International", models.CarrierTypeOcean, "https://ocean.com", "https://ocean.com/track/%s")
	if baseWithURL.TrackingURL("ABC1234567") != "https://ocean.com/track/ABC1234567" {
		t.Errorf("unexpected formatted URL: %s", baseWithURL.TrackingURL("ABC1234567"))
	}
}
