// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

func TestItalianAndSanMarinoCarriers(t *testing.T) {
	testCases := []struct {
		name                string
		constructor         func() Carrier
		expectedSlug        string
		expectedName        string
		expectedCountry     string
		expectedCountryName string
		expectedType        models.CarrierType
		expectedWebsite     string
		sampleTrackingNum   string
		expectedURLSubstr   string
	}{
		{
			name:                "Poste Italiane",
			constructor:         func() Carrier { return NewPosteItalianeCarrier() },
			expectedSlug:        "poste-italiane",
			expectedName:        "Poste Italiane",
			expectedCountry:     "IT",
			expectedCountryName: "Italy",
			expectedType:        models.CarrierTypePostal,
			expectedWebsite:     "https://www.poste.it",
			sampleTrackingNum:   "386123456789",
			expectedURLSubstr:   "poste.it",
		},
		{
			name:                "BRT Bartolini",
			constructor:         func() Carrier { return NewBRTBartoliniCarrier() },
			expectedSlug:        "brt-bartolini",
			expectedName:        "BRT Bartolini",
			expectedCountry:     "IT",
			expectedCountryName: "Italy",
			expectedType:        models.CarrierTypeExpress,
			expectedWebsite:     "https://www.brt.it",
			sampleTrackingNum:   "012345678901",
			expectedURLSubstr:   "brt.it",
		},
		{
			name:                "BRT Bartolini Parcel ID",
			constructor:         func() Carrier { return NewBRTParcelIDCarrier() },
			expectedSlug:        "brt-italy-parcelid",
			expectedName:        "BRT Bartolini (Parcel ID)",
			expectedCountry:     "IT",
			expectedCountryName: "Italy",
			expectedType:        models.CarrierTypeExpress,
			expectedWebsite:     "https://www.brt.it",
			sampleTrackingNum:   "05512345678901234",
			expectedURLSubstr:   "brt.it",
		},
		{
			name:                "SDA Express Courier",
			constructor:         func() Carrier { return NewSDACarrier() },
			expectedSlug:        "sda",
			expectedName:        "SDA Express Courier",
			expectedCountry:     "IT",
			expectedCountryName: "Italy",
			expectedType:        models.CarrierTypeExpress,
			expectedWebsite:     "https://www.sda.it",
			sampleTrackingNum:   "1234567890123",
			expectedURLSubstr:   "sda.it",
		},
		{
			name:                "GLS Italy",
			constructor:         func() Carrier { return NewGLSItalyCarrier() },
			expectedSlug:        "gls-italy",
			expectedName:        "GLS Italy",
			expectedCountry:     "IT",
			expectedCountryName: "Italy",
			expectedType:        models.CarrierTypeExpress,
			expectedWebsite:     "https://www.gls-italy.com",
			sampleTrackingNum:   "L912345678",
			expectedURLSubstr:   "gls-italy.com",
		},
		{
			name:                "Poste San Marino",
			constructor:         func() Carrier { return NewPosteSanMarinoCarrier() },
			expectedSlug:        "poste-san-marino",
			expectedName:        "Poste San Marino",
			expectedCountry:     "SM",
			expectedCountryName: "San Marino",
			expectedType:        models.CarrierTypePostal,
			expectedWebsite:     "https://www.poste.sm",
			sampleTrackingNum:   "RR123456789SM",
			expectedURLSubstr:   "poste.sm",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			carrier := tc.constructor()
			if carrier == nil {
				t.Fatalf("constructor returned nil for %s", tc.name)
			}

			// Validate interface methods
			if carrier.Slug() != tc.expectedSlug {
				t.Errorf("expected slug %q, got %q", tc.expectedSlug, carrier.Slug())
			}
			if carrier.Name() != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, carrier.Name())
			}
			if carrier.Country() != tc.expectedCountry {
				t.Errorf("expected country %q, got %q", tc.expectedCountry, carrier.Country())
			}
			if carrier.CountryName() != tc.expectedCountryName {
				t.Errorf("expected country name %q, got %q", tc.expectedCountryName, carrier.CountryName())
			}
			if carrier.Type() != tc.expectedType {
				t.Errorf("expected type %q, got %q", tc.expectedType, carrier.Type())
			}
			if carrier.Website() != tc.expectedWebsite {
				t.Errorf("expected website %q, got %q", tc.expectedWebsite, carrier.Website())
			}

			trackURL := carrier.TrackingURL(tc.sampleTrackingNum)
			if !strings.Contains(trackURL, tc.expectedURLSubstr) {
				t.Errorf("expected tracking URL to contain %q, got %q", tc.expectedURLSubstr, trackURL)
			}
			if !strings.Contains(trackURL, tc.sampleTrackingNum) {
				t.Errorf("expected tracking URL to contain tracking number %q, got %q", tc.sampleTrackingNum, trackURL)
			}

			// Validate CarrierInfo
			info := carrier.Info()
			if info.Slug != tc.expectedSlug || info.Name != tc.expectedName || info.Country != tc.expectedCountry {
				t.Errorf("Info() returned unexpected data: %+v", info)
			}

			// Validate Track method and simulated events
			ctx := context.Background()
			shipment, err := carrier.Track(ctx, tc.sampleTrackingNum, models.TrackOptions{Language: "it"})
			if err != nil {
				t.Fatalf("Track() returned unexpected error: %v", err)
			}
			if shipment == nil {
				t.Fatalf("Track() returned nil shipment")
			}
			if shipment.TrackingNumber != tc.sampleTrackingNum {
				t.Errorf("expected shipment tracking number %q, got %q", tc.sampleTrackingNum, shipment.TrackingNumber)
			}
			if shipment.Carrier.Slug != tc.expectedSlug {
				t.Errorf("expected shipment carrier slug %q, got %q", tc.expectedSlug, shipment.Carrier.Slug)
			}
			if shipment.Carrier.Country != tc.expectedCountry {
				t.Errorf("expected shipment carrier country %q, got %q", tc.expectedCountry, shipment.Carrier.Country)
			}
			if len(shipment.Events) == 0 {
				t.Fatalf("expected non-empty events timeline for %s", tc.name)
			}

			// Verify each event has valid status, stage, description, location, carrier slug
			validStatuses := map[models.Status]bool{
				models.StatusPending:        true,
				models.StatusInfoReceived:   true,
				models.StatusInTransit:      true,
				models.StatusOutForDelivery: true,
				models.StatusDelivered:      true,
				models.StatusFailedAttempt:  true,
				models.StatusException:      true,
				models.StatusExpired:        true,
			}
			validStages := map[models.Stage]bool{
				models.StageCreated:   true,
				models.StagePickedUp:  true,
				models.StageTransit:   true,
				models.StageCustoms:   true,
				models.StageHandover:  true,
				models.StageLastMile:  true,
				models.StageDelivery:  true,
				models.StageException: true,
			}

			for i, ev := range shipment.Events {
				if ev.Description == "" {
					t.Errorf("event %d: empty description", i)
				}
				if !validStatuses[ev.Status] {
					t.Errorf("event %d: invalid status %q", i, ev.Status)
				}
				if !validStages[ev.Stage] {
					t.Errorf("event %d: invalid stage %q", i, ev.Stage)
				}
				if ev.CarrierSlug != tc.expectedSlug {
					t.Errorf("event %d: expected carrier slug %q, got %q", i, tc.expectedSlug, ev.CarrierSlug)
				}
				if ev.CarrierName != tc.expectedName {
					t.Errorf("event %d: expected carrier name %q, got %q", i, tc.expectedName, ev.CarrierName)
				}
				if ev.Date.IsZero() {
					t.Errorf("event %d: zero date", i)
				}
			}
		})
	}
}

func TestRegistryItalianAndSanMarinoCarriers(t *testing.T) {
	reg := GetRegistry()

	expectedSlugs := []string{
		"poste-italiane",
		"brt-bartolini",
		"brt-italy-parcelid",
		"sda",
		"gls-italy",
		"poste-san-marino",
	}

	for _, slug := range expectedSlugs {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("expected carrier with slug %q to be registered in default registry", slug)
			continue
		}
		if strings.ToLower(c.Slug()) != slug {
			t.Errorf("expected slug %q, got %q", slug, c.Slug())
		}
	}
}
