// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestOceanCarrierImplementations(t *testing.T) {
	// Verify that all ocean carrier constructors return Carrier implementations
	carriersToTest := []struct {
		carrier Carrier
		slug    string
		name    string
		num     string
	}{
		{carrier: NewMaerskLineCarrier(), slug: "maersk-line", name: "Maersk Line", num: "MAEU123456789"},
		{carrier: NewMSCCarrier(), slug: "msc", name: "MSC", num: "MEDUZA123456"},
		{carrier: NewCMACGMCarrier(), slug: "cma-cgm", name: "CMA CGM", num: "CMAU1234567890"},
		{carrier: NewCOSCOCarrier(), slug: "cosco", name: "COSCO Shipping", num: "COSU6321234560"},
		{carrier: NewHapagLloydCarrier(), slug: "hapag-lloyd", name: "Hapag Lloyd", num: "HLCUHAM123456"},
		{carrier: NewEvergreenCarrier(), slug: "evergreen", name: "Evergreen", num: "EGLV147000380361"},
		{carrier: NewONELineCarrier(), slug: "one-line", name: "ONE Ocean Network Express", num: "ONEU1234567890"},
		{carrier: NewYangMingCarrier(), slug: "yang-ming", name: "Yang Ming", num: "YMLU1234567890"},
		{carrier: NewHMMCarrier(), slug: "hmm", name: "HMM", num: "HDMU1234567890"},
		{carrier: NewZIMCarrier(), slug: "zim", name: "ZIM", num: "ZIMU1234567890"},
		{carrier: NewDoleOceanCargoCarrier(), slug: "dole-ocean-cargo", name: "Dole Ocean Cargo", num: "DOLU1234567890"},
		{carrier: NewOceanContainerCarrier(), slug: "ocean-container", name: "ISO Ocean Container", num: "MSKU9070323"},
	}

	for _, tc := range carriersToTest {
		t.Run(tc.slug, func(t *testing.T) {
			c := tc.carrier
			if c.Slug() != tc.slug {
				t.Errorf("expected slug %q, got %q", tc.slug, c.Slug())
			}
			if c.Name() != tc.name {
				t.Errorf("expected name %q, got %q", tc.name, c.Name())
			}
			if c.Country() != "GLOBAL" {
				t.Errorf("expected country GLOBAL, got %q", c.Country())
			}
			if c.CountryName() != "International" {
				t.Errorf("expected country name International, got %q", c.CountryName())
			}
			if c.Type() != models.CarrierTypeOcean && c.Type() != models.CarrierTypeFreight {
				t.Errorf("unexpected carrier type: %s", c.Type())
			}
			if c.Website() == "" {
				t.Errorf("expected website for %s", tc.slug)
			}
			if tc.slug != "ocean-container" && c.TrackingURL(tc.num) == "" {
				t.Errorf("expected tracking url for %s", tc.slug)
			}

			// Track
			shipment, err := c.Track(context.Background(), tc.num, models.TrackOptions{Language: "en"})
			if err != nil {
				t.Fatalf("unexpected error tracking %s: %v", tc.slug, err)
			}
			if shipment == nil {
				t.Fatalf("expected non-nil shipment for %s", tc.slug)
			}
			if shipment.TrackingNumber != tc.num {
				t.Errorf("expected tracking number %s, got %s", tc.num, shipment.TrackingNumber)
			}
			if shipment.Carrier.Slug != tc.slug {
				t.Errorf("expected carrier slug %s, got %s", tc.slug, shipment.Carrier.Slug)
			}

			// Verify SeaLegs detailing POL, POD, vessel, voyage
			if len(shipment.SeaLegs) == 0 {
				t.Fatalf("expected at least 1 sea leg for %s", tc.slug)
			}
			leg := shipment.SeaLegs[0]
			if leg.OriginUNLOCODE == "" {
				t.Errorf("expected POL OriginUNLOCODE to be set for %s", tc.slug)
			}
			if leg.DestinationUNLOCODE == "" {
				t.Errorf("expected POD DestinationUNLOCODE to be set for %s", tc.slug)
			}
			if leg.Vessel == nil || leg.Vessel.Name == "" || leg.Vessel.IMO == "" {
				t.Errorf("expected vessel details (name, IMO) for %s, got %+v", tc.slug, leg.Vessel)
			}
			if leg.VoyageNumber == "" {
				t.Errorf("expected voyage number for %s", tc.slug)
			}

			// Verify container
			if shipment.Container == nil {
				t.Fatalf("expected container details for %s", tc.slug)
			}
			if shipment.Container.Number == "" {
				t.Errorf("expected container number for %s", tc.slug)
			}
			if shipment.Container.Type == "" {
				t.Errorf("expected container type for %s", tc.slug)
			}

			// Verify Port codes
			if shipment.FromPort == "" || shipment.ToPort == "" {
				t.Errorf("expected FromPort and ToPort for %s, got from=%s, to=%s", tc.slug, shipment.FromPort, shipment.ToPort)
			}

			// Verify HasOceanData helper
			if !shipment.HasOceanData() {
				t.Errorf("expected HasOceanData to return true for %s", tc.slug)
			}

			// Verify events
			if len(shipment.Events) == 0 {
				t.Errorf("expected tracking events for %s", tc.slug)
			}
		})
	}
}

func TestRegistryOceanCarriersRegistered(t *testing.T) {
	reg := GetRegistry()
	oceanSlugs := []string{
		"maersk-line",
		"msc",
		"cma-cgm",
		"cosco",
		"hapag-lloyd",
		"evergreen",
		"one-line",
		"yang-ming",
		"hmm",
		"zim",
		"dole-ocean-cargo",
		"ocean-container",
	}

	for _, slug := range oceanSlugs {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("expected ocean carrier %q to be registered in default registry", slug)
		}
	}
}
