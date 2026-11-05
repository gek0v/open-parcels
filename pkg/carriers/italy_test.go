package carriers

import (
	"context"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

func TestItalianAndSanMarinoCarriers(t *testing.T) {
	ctx := context.Background()

	testCarriers := []struct {
		slug            string
		name            string
		country         string
		expectedType    models.CarrierType
		carrierInstance Carrier
		trackingNumber  string
	}{
		{
			slug:            "brt-bartolini",
			name:            "BRT Bartolini",
			country:         "IT",
			expectedType:    models.CarrierTypeExpress,
			carrierInstance: NewBRTBartoliniCarrier(),
			trackingNumber:  "012345678901",
		},
		{
			slug:            "brt-italy-parcelid",
			name:            "BRT Bartolini (Parcel ID)",
			country:         "IT",
			expectedType:    models.CarrierTypeExpress,
			carrierInstance: NewBRTItalyParcelIDCarrier(),
			trackingNumber:  "05512345678901",
		},
		{
			slug:            "mybrt-it",
			name:            "myBRT.it",
			country:         "IT",
			expectedType:    models.CarrierTypeExpress,
			carrierInstance: NewMyBRTCarrier(),
			trackingNumber:  "MYBRT123456789",
		},
		{
			slug:            "poste-italiane",
			name:            "Poste Italiane",
			country:         "IT",
			expectedType:    models.CarrierTypePostal,
			carrierInstance: NewPosteItalianeCarrier(),
			trackingNumber:  "RA123456785IT",
		},
		{
			slug:            "gls-italy",
			name:            "GLS Italy",
			country:         "IT",
			expectedType:    models.CarrierTypeExpress,
			carrierInstance: NewGLSItalyCarrier(),
			trackingNumber:  "GLSIT123456789",
		},
		{
			slug:            "inpost-it",
			name:            "InPost.it",
			country:         "IT",
			expectedType:    models.CarrierTypeLogistics,
			carrierInstance: NewInPostItalyCarrier(),
			trackingNumber:  "623456789012345678901234",
		},
		{
			slug:            "tnt-italy",
			name:            "TNT Italy",
			country:         "IT",
			expectedType:    models.CarrierTypeExpress,
			carrierInstance: NewTNTItalyCarrier(),
			trackingNumber:  "MY123456789",
		},
		{
			slug:            "amazon-it",
			name:            "Amazon Italy",
			country:         "IT",
			expectedType:    models.CarrierTypeLogistics,
			carrierInstance: NewAmazonItalyCarrier(),
			trackingNumber:  "IT1234567890",
		},
		{
			slug:            "sailpost",
			name:            "Sailpost",
			country:         "IT",
			expectedType:    models.CarrierTypePostal,
			carrierInstance: NewSailpostCarrier(),
			trackingNumber:  "SP1234567890",
		},
		{
			slug:            "alfred-it",
			name:            "Alfred 24",
			country:         "IT",
			expectedType:    models.CarrierTypeLogistics,
			carrierInstance: NewAlfredITCarrier(),
			trackingNumber:  "ALF123456789",
		},
		{
			slug:            "arco-spedizioni",
			name:            "Arco Spedizioni",
			country:         "IT",
			expectedType:    models.CarrierTypeFreight,
			carrierInstance: NewArcoSpedizioniCarrier(),
			trackingNumber:  "ARC12345678",
		},
		{
			slug:            "ceva-it",
			name:            "CEVA Logistics Italy",
			country:         "IT",
			expectedType:    models.CarrierTypeFreight,
			carrierInstance: NewCEVAITCarrier(),
			trackingNumber:  "CEVA123456789",
		},
		{
			slug:            "bdm-it",
			name:            "CST Italia",
			country:         "IT",
			expectedType:    models.CarrierTypeLogistics,
			carrierInstance: NewBDMITCarrier(),
			trackingNumber:  "BDM12345678",
		},
		{
			slug:            "deliver-it",
			name:            "Deliver It",
			country:         "IT",
			expectedType:    models.CarrierTypeExpress,
			carrierInstance: NewDeliverITCarrier(),
			trackingNumber:  "DIT123456789",
		},
		{
			slug:            "liccardi-it",
			name:            "Liccardi Trasporti",
			country:         "IT",
			expectedType:    models.CarrierTypeFreight,
			carrierInstance: NewLiccardiITCarrier(),
			trackingNumber:  "LIC12345678",
		},
		{
			slug:            "mbe-it",
			name:            "Mail Boxes Etc. Italy",
			country:         "IT",
			expectedType:    models.CarrierTypeLogistics,
			carrierInstance: NewMBEITCarrier(),
			trackingNumber:  "MBE123456789",
		},
		{
			slug:            "rhenus-it",
			name:            "Rhenus Italia",
			country:         "IT",
			expectedType:    models.CarrierTypeFreight,
			carrierInstance: NewRhenusITCarrier(),
			trackingNumber:  "RH123456789",
		},
		{
			slug:            "spediamo-it",
			name:            "Spediamo",
			country:         "IT",
			expectedType:    models.CarrierTypeLogistics,
			carrierInstance: NewSpediamoITCarrier(),
			trackingNumber:  "SPD12345678",
		},
		{
			slug:            "unieuro-it",
			name:            "Unieuro",
			country:         "IT",
			expectedType:    models.CarrierTypeLogistics,
			carrierInstance: NewUnieuroITCarrier(),
			trackingNumber:  "UNI123456789",
		},
	}

	if len(testCarriers) != 19 {
		t.Fatalf("expected exactly 19 carriers in test suite, got %d", len(testCarriers))
	}

	for _, tc := range testCarriers {
		t.Run(tc.slug, func(t *testing.T) {
			c := tc.carrierInstance
			if c.Slug() != tc.slug {
				t.Errorf("expected slug %s, got %s", tc.slug, c.Slug())
			}
			if c.Name() != tc.name {
				t.Errorf("expected name %s, got %s", tc.name, c.Name())
			}
			if c.Country() != tc.country {
				t.Errorf("expected country %s, got %s", tc.country, c.Country())
			}
			if c.Type() != tc.expectedType {
				t.Errorf("expected type %s, got %s", tc.expectedType, c.Type())
			}
			if c.Website() == "" {
				t.Errorf("expected non-empty website for %s", tc.slug)
			}
			url := c.TrackingURL(tc.trackingNumber)
			if url == "" || !strings.Contains(url, tc.trackingNumber) {
				t.Errorf("expected tracking url containing %s, got %s", tc.trackingNumber, url)
			}

			// Test Track
			shipment, err := c.Track(ctx, tc.trackingNumber, models.TrackOptions{Language: "it"})
			if err != nil {
				t.Fatalf("failed to track %s: %v", tc.slug, err)
			}
			if shipment == nil {
				t.Fatalf("expected non-nil shipment for %s", tc.slug)
			}
			if shipment.TrackingNumber != tc.trackingNumber {
				t.Errorf("expected tracking number %s, got %s", tc.trackingNumber, shipment.TrackingNumber)
			}
			if shipment.Carrier.Slug != tc.slug {
				t.Errorf("expected carrier slug %s, got %s", tc.slug, shipment.Carrier.Slug)
			}
			if len(shipment.Events) == 0 {
				t.Errorf("expected events for %s, got 0", tc.slug)
			}
			if shipment.Status == "" {
				t.Errorf("expected non-empty status for %s", tc.slug)
			}
		})
	}
}

func TestItalianCarriersInRegistry(t *testing.T) {
	reg := GetRegistry()

	expectedSlugs := []string{
		"brt-bartolini",
		"brt-italy-parcelid",
		"mybrt-it",
		"poste-italiane",
		"gls-italy",
		"inpost-it",
		"tnt-italy",
		"amazon-it",
		"sailpost",
		"alfred-it",
		"arco-spedizioni",
		"ceva-it",
		"bdm-it",
		"deliver-it",
		"liccardi-it",
		"mbe-it",
		"rhenus-it",
		"spediamo-it",
		"unieuro-it",
	}

	for _, slug := range expectedSlugs {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("carrier %s not found in registry", slug)
		}
	}
}
