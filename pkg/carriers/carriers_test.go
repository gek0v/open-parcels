// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
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

	shipment, err := cainiao.Track(context.Background(), "LP00561234567890", models.TrackOptions{Language: "en", DestinationCountry: "ES"})
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

func TestSpanishCarriersRegistryAndAliases(t *testing.T) {
	reg := GetRegistry()

	expectedAliases := map[string]string{
		"mrw-spain":      "mrw",
		"correosexpress": "correos-express",
		"tourline":       "ctt-express",
		"nacex-spain":    "nacex",
		"sending-es":     "sending",
		"redur-spain":    "redur",
		"inpost-es":      "inpost",
		"asm-gls-spain":  "gls-spain",
	}

	for alias, expectedCanonical := range expectedAliases {
		// Test lowercase
		c, ok := reg.Get(alias)
		if !ok || c == nil {
			t.Fatalf("expected alias %q to resolve to a carrier, got ok=%v", alias, ok)
		}
		if strings.ToLower(c.Slug()) != expectedCanonical {
			t.Errorf("expected alias %q to resolve to canonical slug %q, got %q", alias, expectedCanonical, c.Slug())
		}

		// Test uppercase
		cUpper, okUpper := reg.Get(strings.ToUpper(alias))
		if !okUpper || cUpper == nil {
			t.Fatalf("expected uppercase alias %q to resolve to a carrier", strings.ToUpper(alias))
		}
		if strings.ToLower(cUpper.Slug()) != expectedCanonical {
			t.Errorf("expected uppercase alias %q to resolve to canonical slug %q, got %q", strings.ToUpper(alias), expectedCanonical, cUpper.Slug())
		}

		// Test mixed case
		cMixed, okMixed := reg.Get(strings.Title(alias))
		if !okMixed || cMixed == nil {
			t.Fatalf("expected title-case alias %q to resolve to a carrier", strings.Title(alias))
		}

		// Test canonical slug lowercase and uppercase
		cCanon, okCanon := reg.Get(expectedCanonical)
		if !okCanon || cCanon == nil {
			t.Fatalf("expected canonical slug %q to resolve", expectedCanonical)
		}
		cCanonUpper, okCanonUpper := reg.Get(strings.ToUpper(expectedCanonical))
		if !okCanonUpper || cCanonUpper == nil {
			t.Fatalf("expected uppercase canonical slug %q to resolve", strings.ToUpper(expectedCanonical))
		}
	}

	// Test Registry.List does not contain duplicate entries caused by aliases
	list := reg.List("")
	seenSlugs := make(map[string]int)
	for _, info := range list {
		slug := strings.ToLower(info.Slug)
		seenSlugs[slug]++
		if seenSlugs[slug] > 1 {
			t.Errorf("duplicate carrier slug in Registry.List: %q", slug)
		}
	}

	// Verify that aliases are not listed as duplicate carrier entries
	for alias := range expectedAliases {
		if seenSlugs[alias] > 0 && alias != expectedAliases[alias] {
			t.Errorf("alias %q appeared as a standalone entry in Registry.List", alias)
		}
	}

	// Test custom alias registration
	reg.RegisterAlias("custom-alias-mrw", "mrw")
	cCustom, okCustom := reg.Get("custom-alias-mrw")
	if !okCustom || cCustom == nil {
		t.Fatalf("expected custom alias to resolve")
	}
	if strings.ToLower(cCustom.Slug()) != "mrw" {
		t.Errorf("expected custom alias to resolve to mrw, got %q", cCustom.Slug())
	}
	// Case insensitivity of custom alias
	cCustomUpper, okCustomUpper := reg.Get("CUSTOM-ALIAS-MRW")
	if !okCustomUpper || cCustomUpper == nil {
		t.Fatalf("expected custom uppercase alias to resolve")
	}
	if strings.ToLower(cCustomUpper.Slug()) != "mrw" {
		t.Errorf("expected custom uppercase alias to resolve to mrw, got %q", cCustomUpper.Slug())
	}

	// Test edge cases: empty strings, self alias, non-existent slug
	reg.RegisterAlias("", "mrw")
	reg.RegisterAlias("mrw", "")
	reg.RegisterAlias("mrw", "mrw")
	reg.RegisterAlias("custom-alias-mrw", "mrw") // duplicate reverse alias branch

	if _, ok := reg.Get(""); ok {
		t.Errorf("expected empty string to return false")
	}
	if _, ok := reg.Get("   "); ok {
		t.Errorf("expected whitespace to return false")
	}
	if _, ok := reg.Get("nonexistent-carrier-12345"); ok {
		t.Errorf("expected nonexistent carrier to return false")
	}

	// Test chained aliases (A -> B -> canonical)
	reg.RegisterAlias("chain-1", "custom-alias-mrw")
	if cChain, ok := reg.Get("chain-1"); !ok || cChain.Slug() != "mrw" {
		t.Errorf("expected chained alias to resolve to mrw")
	}

	// Test reverse alias resolution when carrier slug matches alias
	customCarrier := NewBaseCarrier("temp-carrier", "Temp Carrier", "ES", "Spain", models.CarrierTypeExpress, "https://temp.es", "https://temp.es/%s")
	dummy := &customDummyCarrier{BaseCarrier: customCarrier}
	reg.Register(dummy)
	reg.RegisterAlias("temp-carrier", "canonical-temp")
	cRev, okRev := reg.Get("canonical-temp")
	if !okRev || cRev == nil || cRev.Slug() != "temp-carrier" {
		t.Errorf("expected reverse alias lookup to resolve: ok=%v, cRev=%v", okRev, cRev)
	}
}

type customDummyCarrier struct {
	BaseCarrier
}

func (c *customDummyCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return &models.Shipment{TrackingNumber: trackingNumber, Carrier: c.Info()}, nil
}

func TestAllCarriersAndMethods(t *testing.T) {
	reg := GetRegistry()
	all := reg.AllCarriers()
	if len(all) == 0 {
		t.Fatalf("expected AllCarriers to return registered carriers")
	}

	ctx := context.Background()
	for slug, carrier := range all {
		if carrier.Slug() == "" {
			t.Errorf("carrier %s has empty slug", slug)
		}
		if carrier.Name() == "" {
			t.Errorf("carrier %s has empty name", slug)
		}
		_ = carrier.Country()
		_ = carrier.CountryName()
		_ = carrier.Type()
		_ = carrier.Website()
		_ = carrier.TrackingURL("TEST123456")

		info := carrier.Info()
		if info.Slug == "" {
			t.Errorf("carrier info %s has empty slug", slug)
		}

		shipment, err := carrier.Track(ctx, "TRACK123456", models.TrackOptions{Language: "es", DestinationCountry: "ES"})
		if err != nil {
			t.Errorf("error tracking %s: %v", slug, err)
		}
		if shipment == nil {
			t.Errorf("shipment is nil for %s", slug)
		}
	}
}

func TestRegistryListFiltering(t *testing.T) {
	reg := GetRegistry()

	// Filter by empty query
	all := reg.List("")
	if len(all) == 0 {
		t.Errorf("expected non-empty list for empty query")
	}

	// Filter by carrier name
	nacexList := reg.List("NACEX")
	if len(nacexList) == 0 {
		t.Errorf("expected at least 1 carrier matching NACEX")
	}

	// Filter by country code
	esList := reg.List("ES")
	if len(esList) == 0 {
		t.Errorf("expected at least 1 carrier matching ES")
	}

	// Filter by country name
	spainList := reg.List("Spain")
	if len(spainList) == 0 {
		t.Errorf("expected at least 1 carrier matching Spain")
	}

	// Filter by alias query
	tourlineList := reg.List("tourline")
	if len(tourlineList) == 0 {
		t.Errorf("expected at least 1 carrier matching tourline alias")
	}

	// Filter with no matches
	noneList := reg.List("this-will-never-match-any-carrier-xyz")
	if len(noneList) != 0 {
		t.Errorf("expected 0 carriers for non-matching query, got %d", len(noneList))
	}

	// Empty url template
	emptyURLCarrier := NewBaseCarrier("empty-url", "Empty", "ES", "Spain", models.CarrierTypeExpress, "https://empty.es", "")
	if url := emptyURLCarrier.TrackingURL("123"); url != "" {
		t.Errorf("expected empty tracking url, got %q", url)
	}
}

func TestSpanishCarriersTrackingOutput(t *testing.T) {
	reg := GetRegistry()
	ctx := context.Background()

	type carrierTestCase struct {
		name               string
		slug               string
		aliases            []string
		constructor        func() Carrier
		testTrackingNumber string
		expectedURLPattern string
	}

	testCases := []carrierTestCase{
		{
			name:               "NACEX",
			slug:               "nacex",
			aliases:            []string{"nacex-spain"},
			constructor:        func() Carrier { return NewNacexCarrier() },
			testTrackingNumber: "NX123456789",
			expectedURLPattern: "https://www.nacex.com/seguimientoDetalle.do?ag_cod_exp=NX123456789",
		},
		{
			name:               "TIPSA",
			slug:               "tipsa",
			aliases:            []string{},
			constructor:        func() Carrier { return NewTipsaCarrier() },
			testTrackingNumber: "TIP0123456789",
			expectedURLPattern: "https://www.tip-sa.com/cliente/seguimiento-envios?albaran=TIP0123456789",
		},
		{
			name:               "Zeleris",
			slug:               "zeleris",
			aliases:            []string{},
			constructor:        func() Carrier { return NewZelerisCarrier() },
			testTrackingNumber: "ZEL987654321",
			expectedURLPattern: "https://www.zeleris.com/seguimiento-envio.action?tracking=ZEL987654321",
		},
		{
			name:               "Envialia",
			slug:               "envialia",
			aliases:            []string{},
			constructor:        func() Carrier { return NewEnvialiaCarrier() },
			testTrackingNumber: "ENV00123456",
			expectedURLPattern: "https://www.envialia.com/seguimiento/?num=ENV00123456",
		},
		{
			name:               "Ontime",
			slug:               "ontime",
			aliases:            []string{},
			constructor:        func() Carrier { return NewOntimeCarrier() },
			testTrackingNumber: "ONT55667788",
			expectedURLPattern: "https://www.ontime.es/seguimiento?tracking=ONT55667788",
		},
		{
			name:               "Sending",
			slug:               "sending",
			aliases:            []string{"sending-es"},
			constructor:        func() Carrier { return NewSendingCarrier() },
			testTrackingNumber: "SND33445566",
			expectedURLPattern: "https://www.sending.es/tracking?id=SND33445566",
		},
		{
			name:               "Redur",
			slug:               "redur",
			aliases:            []string{"redur-spain", "redur-spain-ref", "redur-tracking"},
			constructor:        func() Carrier { return NewRedurCarrier() },
			testTrackingNumber: "RDR77889900",
			expectedURLPattern: "https://redur.es/seguimiento-envios/?albaran=RDR77889900",
		},
		{
			name:               "GLS Spain",
			slug:               "gls-spain",
			aliases:            []string{"asm-gls-spain"},
			constructor:        func() Carrier { return NewGLSSpainCarrier() },
			testTrackingNumber: "GLSES11223344",
			expectedURLPattern: "https://www.gls-spain.es/seguimiento-envios/?match=GLSES11223344",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Verify constructor returns Carrier interface
			carrier := tc.constructor()
			if carrier == nil {
				t.Fatalf("[%s] constructor returned nil", tc.name)
			}

			// Verify Slug matches
			if carrier.Slug() != tc.slug {
				t.Errorf("[%s] expected slug %q, got %q", tc.name, tc.slug, carrier.Slug())
			}

			// Verify Country is ES
			if carrier.Country() != "ES" {
				t.Errorf("[%s] expected country ES, got %q", tc.name, carrier.Country())
			}

			// Verify CountryName is Spain
			if carrier.CountryName() != "Spain" {
				t.Errorf("[%s] expected country name Spain, got %q", tc.name, carrier.CountryName())
			}

			// 2. Verify TrackingURL formatting
			trackingURL := carrier.TrackingURL(tc.testTrackingNumber)
			if trackingURL != tc.expectedURLPattern {
				t.Errorf("[%s] expected tracking URL %q, got %q", tc.name, tc.expectedURLPattern, trackingURL)
			}

			// 3. Verify registry lookup by canonical slug
			regCarrier, found := reg.Get(tc.slug)
			if !found || regCarrier == nil {
				t.Errorf("[%s] carrier not found in registry by slug %q", tc.name, tc.slug)
			} else if regCarrier.Slug() != tc.slug {
				t.Errorf("[%s] registry returned carrier with slug %q, expected %q", tc.name, regCarrier.Slug(), tc.slug)
			}

			// 4. Verify registry lookup by aliases
			for _, alias := range tc.aliases {
				aliasCarrier, aliasFound := reg.Get(alias)
				if !aliasFound || aliasCarrier == nil {
					t.Errorf("[%s] carrier not found in registry by alias %q", tc.name, alias)
				} else if aliasCarrier.Slug() != tc.slug {
					t.Errorf("[%s] alias %q resolved to slug %q, expected %q", tc.name, alias, aliasCarrier.Slug(), tc.slug)
				}
			}

			// 5. Verify tracking output and canonical timeline
			shipment, err := carrier.Track(ctx, tc.testTrackingNumber, models.TrackOptions{
				Language:           "es",
				DestinationCountry: "ES",
			})
			if err != nil {
				t.Fatalf("[%s] Track returned unexpected error: %v", tc.name, err)
			}
			if shipment == nil {
				t.Fatalf("[%s] Track returned nil shipment", tc.name)
			}

			if shipment.TrackingNumber != tc.testTrackingNumber {
				t.Errorf("[%s] shipment tracking number %q, expected %q", tc.name, shipment.TrackingNumber, tc.testTrackingNumber)
			}

			if shipment.Carrier.Slug != tc.slug {
				t.Errorf("[%s] shipment carrier slug %q, expected %q", tc.name, shipment.Carrier.Slug, tc.slug)
			}

			if shipment.Origin.Code != "ES" {
				t.Errorf("[%s] expected origin ES, got %q", tc.name, shipment.Origin.Code)
			}
			if shipment.Destination.Code != "ES" {
				t.Errorf("[%s] expected destination ES, got %q", tc.name, shipment.Destination.Code)
			}

			if len(shipment.Events) < 3 {
				t.Fatalf("[%s] expected at least 3 canonical timeline events, got %d", tc.name, len(shipment.Events))
			}

			// Verify chronological ordering of canonical timeline events
			for i := 0; i < len(shipment.Events); i++ {
				ev := shipment.Events[i]
				if ev.Description == "" {
					t.Errorf("[%s] event %d has empty description", tc.name, i)
				}
				if ev.CarrierSlug != tc.slug {
					t.Errorf("[%s] event %d carrier slug %q, expected %q", tc.name, i, ev.CarrierSlug, tc.slug)
				}
				if ev.Status == "" {
					t.Errorf("[%s] event %d has empty status", tc.name, i)
				}
				if ev.Stage == "" {
					t.Errorf("[%s] event %d has empty stage", tc.name, i)
				}
				if i > 0 {
					if ev.Date.Before(shipment.Events[i-1].Date) {
						t.Errorf("[%s] event %d date (%v) is before event %d date (%v)", tc.name, i, ev.Date, i-1, shipment.Events[i-1].Date)
					}
				}
			}

			if shipment.Progress <= 0 {
				t.Errorf("[%s] shipment progress must be > 0, got %d", tc.name, shipment.Progress)
			}
			if shipment.DaysInTransit < 0 {
				t.Errorf("[%s] shipment days in transit must be >= 0, got %d", tc.name, shipment.DaysInTransit)
			}
			if shipment.UpdatedAt.IsZero() {
				t.Errorf("[%s] shipment updated_at should not be zero", tc.name)
			}
		})
	}
}
