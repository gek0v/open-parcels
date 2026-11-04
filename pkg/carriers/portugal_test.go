// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

func TestPortugueseCarriersInterface(t *testing.T) {
	var _ Carrier = (*PortugalPostCarrier)(nil)
	var _ Carrier = (*DPDPortugalCarrier)(nil)
	var _ Carrier = (*VaspExpressoCarrier)(nil)
	var _ Carrier = (*DelnextCarrier)(nil)
}

func TestPortugueseCarriersMetadata(t *testing.T) {
	tests := []struct {
		name         string
		carrier      Carrier
		expectedSlug string
		expectedName string
		expectedType models.CarrierType
		expectedURL  string
	}{
		{
			name:         "Portugal Post / CTT Portugal",
			carrier:      NewPortugalPostCarrier(),
			expectedSlug: "portugal-post",
			expectedName: "CTT Portugal",
			expectedType: models.CarrierTypePostal,
			expectedURL:  "https://appserver.ctt.pt/CustomerArea/PublicArea/ObjectSearchResults?objects=DA123456789PT",
		},
		{
			name:         "DPD Portugal",
			carrier:      NewDPDPortugalCarrier(),
			expectedSlug: "dpd-portugal",
			expectedName: "DPD Portugal",
			expectedType: models.CarrierTypeExpress,
			expectedURL:  "https://tracking.dpd.pt/status/pt-PT/09441234567890",
		},
		{
			name:         "Vasp Expresso",
			carrier:      NewVaspExpressoCarrier(),
			expectedSlug: "vasp-expresso",
			expectedName: "VASP Expresso",
			expectedType: models.CarrierTypeExpress,
			expectedURL:  "https://www.vaspexpresso.pt/tracking/VP12345678PT",
		},
		{
			name:         "Delnext",
			carrier:      NewDelnextCarrier(),
			expectedSlug: "delnext",
			expectedName: "Delnext",
			expectedType: models.CarrierTypeLogistics,
			expectedURL:  "https://www.delnext.com/tracking/?code=DLN12345678PT",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.carrier.Slug() != tc.expectedSlug {
				t.Errorf("expected slug %q, got %q", tc.expectedSlug, tc.carrier.Slug())
			}
			if tc.carrier.Country() != "PT" {
				t.Errorf("expected country 'PT', got %q", tc.carrier.Country())
			}
			if tc.carrier.CountryName() != "Portugal" {
				t.Errorf("expected country name 'Portugal', got %q", tc.carrier.CountryName())
			}
			if tc.carrier.Type() != tc.expectedType {
				t.Errorf("expected type %q, got %q", tc.expectedType, tc.carrier.Type())
			}
			if tc.carrier.Website() == "" {
				t.Errorf("expected non-empty website")
			}

			info := tc.carrier.Info()
			if info.Slug != tc.expectedSlug || info.Country != "PT" {
				t.Errorf("mismatched Info() metadata: %+v", info)
			}

			trackingNumber := "DA123456789PT"
			if tc.expectedSlug == "dpd-portugal" {
				trackingNumber = "09441234567890"
			} else if tc.expectedSlug == "vasp-expresso" {
				trackingNumber = "VP12345678PT"
			} else if tc.expectedSlug == "delnext" {
				trackingNumber = "DLN12345678PT"
			}

			trackingURL := tc.carrier.TrackingURL(trackingNumber)
			if trackingURL != tc.expectedURL {
				t.Errorf("expected tracking URL %q, got %q", tc.expectedURL, trackingURL)
			}
		})
	}
}

func TestPortugueseCarriersRegistry(t *testing.T) {
	reg := GetRegistry()

	expectedSlugs := []string{
		"portugal-post",
		"ctt-portugal",
		"dpd-portugal",
		"vasp-expresso",
		"delnext",
	}

	for _, slug := range expectedSlugs {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("expected carrier %q to be registered in default registry", slug)
		}
	}

	list := reg.List("Portugal")
	if len(list) < 4 {
		t.Errorf("expected at least 4 Portuguese carriers in registry search, got %d", len(list))
	}

	all := reg.AllCarriers()
	if len(all) < len(list) {
		t.Errorf("expected AllCarriers to return carriers map, got %d", len(all))
	}

	// Test base carrier TrackingURL with empty template
	emptyCarrier := NewBaseCarrier("test", "Test", "PT", "Portugal", models.CarrierTypePostal, "https://test.com", "")
	if url := emptyCarrier.TrackingURL("123"); url != "" {
		t.Errorf("expected empty tracking URL, got %q", url)
	}
}

func TestPortugueseCarriersTrack(t *testing.T) {
	carriers := []Carrier{
		NewPortugalPostCarrier(),
		NewDPDPortugalCarrier(),
		NewVaspExpressoCarrier(),
		NewDelnextCarrier(),
	}

	ctx := context.Background()

	for _, c := range carriers {
		t.Run(c.Slug()+"_DefaultAndPT", func(t *testing.T) {
			s, err := c.Track(ctx, "TRACK123PT", models.TrackOptions{Language: "pt"})
			if err != nil {
				t.Fatalf("unexpected error tracking %s: %v", c.Slug(), err)
			}
			if s.TrackingNumber != "TRACK123PT" {
				t.Errorf("expected tracking number TRACK123PT, got %s", s.TrackingNumber)
			}
			if s.Origin.Code != "PT" || s.Origin.Name != "Portugal" {
				t.Errorf("expected origin PT / Portugal, got %+v", s.Origin)
			}
			if s.Destination.Code != "PT" || s.Destination.Name != "Portugal" {
				t.Errorf("expected destination PT / Portugal, got %+v", s.Destination)
			}
			if len(s.Events) == 0 {
				t.Fatalf("expected events for %s, got none", c.Slug())
			}

			// Validate Portuguese vocabulary in descriptions
			foundPTWord := false
			ptKeywords := []string{"objeto", "encomenda", "guia", "envio", "trânsito", "distribuição", "recolha", "aceite", "armazém"}
			for _, ev := range s.Events {
				descLower := strings.ToLower(ev.Description)
				for _, kw := range ptKeywords {
					if strings.Contains(descLower, kw) {
						foundPTWord = true
						break
					}
				}
				if ev.CarrierSlug != c.Slug() {
					t.Errorf("event carrier slug %q did not match %q", ev.CarrierSlug, c.Slug())
				}
			}
			if !foundPTWord {
				t.Errorf("expected Portuguese wording in events for %s", c.Slug())
			}
		})

		t.Run(c.Slug()+"_Spanish", func(t *testing.T) {
			s, err := c.Track(ctx, "TRACK123PT", models.TrackOptions{Language: "es"})
			if err != nil {
				t.Fatalf("unexpected error tracking %s: %v", c.Slug(), err)
			}
			foundESWord := false
			esKeywords := []string{"envío", "tránsito", "reparto", "recogida", "recibido", "mercancía"}
			for _, ev := range s.Events {
				descLower := strings.ToLower(ev.Description)
				for _, kw := range esKeywords {
					if strings.Contains(descLower, kw) {
						foundESWord = true
						break
					}
				}
			}
			if !foundESWord {
				t.Errorf("expected Spanish wording in events for %s when Language=es", c.Slug())
			}
		})

		t.Run(c.Slug()+"_English", func(t *testing.T) {
			s, err := c.Track(ctx, "TRACK123PT", models.TrackOptions{Language: "en"})
			if err != nil {
				t.Fatalf("unexpected error tracking %s: %v", c.Slug(), err)
			}
			foundENWord := false
			enKeywords := []string{"parcel", "item", "shipment", "transit", "delivery", "collected", "sorting", "depot"}
			for _, ev := range s.Events {
				descLower := strings.ToLower(ev.Description)
				for _, kw := range enKeywords {
					if strings.Contains(descLower, kw) {
						foundENWord = true
						break
					}
				}
			}
			if !foundENWord {
				t.Errorf("expected English wording in events for %s when Language=en", c.Slug())
			}
		})

		t.Run(c.Slug()+"_DestinationPTExplicit", func(t *testing.T) {
			s, err := c.Track(ctx, "TRACK123PT", models.TrackOptions{
				DestinationCountry: "PT",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Destination.Code != "PT" || s.Destination.Name != "Portugal" {
				t.Errorf("expected destination PT / Portugal, got %+v", s.Destination)
			}
		})

		t.Run(c.Slug()+"_CustomDestination", func(t *testing.T) {
			s, err := c.Track(ctx, "TRACK123PT", models.TrackOptions{
				DestinationCountry: "ES",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Origin.Code != "PT" {
				t.Errorf("expected origin PT, got %s", s.Origin.Code)
			}
			if s.Destination.Code != "ES" {
				t.Errorf("expected destination ES, got %s", s.Destination.Code)
			}
		})

		t.Run(c.Slug()+"_DefaultLanguage", func(t *testing.T) {
			s, err := c.Track(ctx, "TRACK123PT", models.TrackOptions{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.StatusText != "Objeto em distribuição / Out for delivery" {
				t.Errorf("expected default bilingual status text, got %q", s.StatusText)
			}
		})
	}
}

func TestLocalizedStatusHelper(t *testing.T) {
	cases := []struct {
		status   models.Status
		lang     string
		expected string
	}{
		{models.StatusOutForDelivery, "pt", "Objeto em distribuição"},
		{models.StatusOutForDelivery, "es", "Envío en reparto"},
		{models.StatusOutForDelivery, "en", "Out for delivery"},
		{models.StatusOutForDelivery, "", "Objeto em distribuição / Out for delivery"},
		{models.StatusDelivered, "pt", "Objeto entregue"},
		{models.StatusDelivered, "es", "Envío entregado"},
		{models.StatusDelivered, "en", "Delivered"},
		{models.StatusDelivered, "de", "Objeto entregue / Delivered"},
		{models.StatusInTransit, "pt", "Em trânsito"},
		{models.StatusInTransit, "es", "En tránsito"},
		{models.StatusInTransit, "en", "In transit"},
		{models.StatusInTransit, "fr", "Em trânsito / In transit"},
	}

	for _, tc := range cases {
		got := localizedStatus(tc.status, tc.lang)
		if got != tc.expected {
			t.Errorf("localizedStatus(%s, %q) = %q; want %q", tc.status, tc.lang, got, tc.expected)
		}
	}
}
