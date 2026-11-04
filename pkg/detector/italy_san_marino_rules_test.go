// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package detector

import (
	"testing"

	"open-parcels/pkg/carriers"
)

func TestUPUS10ItalianAndSanMarino(t *testing.T) {
	d := New()

	// Modulo 11 check calculation for "12345678":
	// 1*8 + 2*6 + 3*4 + 4*2 + 5*3 + 6*5 + 7*9 + 8*7 = 204
	// 204 % 11 = 6; 11 - 6 = 5. Check digit = 5.
	// Valid Poste Italiane UPU S10: "RA123456785IT"
	// Valid Poste San Marino UPU S10: "RR123456785SM"

	t.Run("Poste Italiane UPU S10 valid", func(t *testing.T) {
		res := d.BestMatch("RA123456785IT", "")
		if res == nil {
			t.Fatalf("expected match for RA123456785IT, got nil")
		}
		if res.Carrier.Slug != "poste-italiane" {
			t.Errorf("expected slug poste-italiane, got %s", res.Carrier.Slug)
		}
		if res.Confidence < 0.95 {
			t.Errorf("expected high confidence for valid UPU S10, got %f", res.Confidence)
		}
	})

	t.Run("Poste San Marino UPU S10 valid", func(t *testing.T) {
		res := d.BestMatch("RR123456785SM", "")
		if res == nil {
			t.Fatalf("expected match for RR123456785SM, got nil")
		}
		if res.Carrier.Slug != "poste-san-marino" {
			t.Errorf("expected slug poste-san-marino, got %s", res.Carrier.Slug)
		}
		if res.Confidence < 0.95 {
			t.Errorf("expected high confidence for valid UPU S10, got %f", res.Confidence)
		}
	})

	t.Run("Poste Italiane UPU S10 invalid checksum penalized", func(t *testing.T) {
		// "RA123456789IT" has check digit 9 instead of 5
		candidates := d.Detect("RA123456789IT", "")
		if len(candidates) == 0 {
			t.Fatalf("expected candidates for RA123456789IT")
		}
		var itMatch *float64
		for _, c := range candidates {
			if c.Carrier.Slug == "poste-italiane" {
				conf := c.Confidence
				itMatch = &conf
				break
			}
		}
		if itMatch == nil {
			t.Fatalf("expected poste-italiane in candidates")
		}
		// Invalid checksum incurs a penalty
		if *itMatch >= 0.95 {
			t.Errorf("expected lower confidence for invalid checksum, got %f", *itMatch)
		}
	})

	t.Run("Poste San Marino UPU S10 invalid checksum penalized", func(t *testing.T) {
		candidates := d.Detect("RR123456789SM", "")
		if len(candidates) == 0 {
			t.Fatalf("expected candidates for RR123456789SM")
		}
		var smMatch *float64
		for _, c := range candidates {
			if c.Carrier.Slug == "poste-san-marino" {
				conf := c.Confidence
				smMatch = &conf
				break
			}
		}
		if smMatch == nil {
			t.Fatalf("expected poste-san-marino in candidates")
		}
		if *smMatch >= 0.95 {
			t.Errorf("expected lower confidence for invalid checksum, got %f", *smMatch)
		}
	})
}

func TestPosteItalianeDomesticAndCrono(t *testing.T) {
	d := New()

	t.Run("Crono parcel code", func(t *testing.T) {
		res := d.BestMatch("CRONO123456789", "")
		if res == nil {
			t.Fatalf("expected match for CRONO123456789, got nil")
		}
		if res.Carrier.Slug != "poste-italiane" {
			t.Errorf("expected slug poste-italiane, got %s", res.Carrier.Slug)
		}
	})

	t.Run("Poste Italiane domestic raccomandata with IT hint", func(t *testing.T) {
		candidates := d.Detect("386123456789", "IT")
		foundPoste := false
		for _, c := range candidates {
			if c.Carrier.Slug == "poste-italiane" {
				foundPoste = true
				break
			}
		}
		if !foundPoste {
			t.Errorf("expected poste-italiane in candidates for 386123456789 with IT hint")
		}
	})
}

func TestBRTBartolini(t *testing.T) {
	d := New()

	t.Run("Spedizione 12 digits", func(t *testing.T) {
		candidates := d.Detect("012345678901", "IT")
		foundBRT := false
		for _, c := range candidates {
			if c.Carrier.Slug == "brt-bartolini" {
				foundBRT = true
				break
			}
		}
		if !foundBRT {
			t.Errorf("expected brt-bartolini in candidates for 12-digit number with IT hint")
		}
	})

	t.Run("ID Collo 14 digits", func(t *testing.T) {
		res := d.BestMatch("05512345678901", "IT")
		if res == nil {
			t.Fatalf("expected match for 14-digit ID Collo, got nil")
		}
		if res.Carrier.Slug != "brt-italy-parcelid" && res.Carrier.Slug != "brt-bartolini" {
			t.Errorf("expected BRT slug, got %s", res.Carrier.Slug)
		}
	})

	t.Run("ID Collo 18 digits", func(t *testing.T) {
		res := d.BestMatch("055123456789012345", "IT")
		if res == nil {
			t.Fatalf("expected match for 18-digit ID Collo, got nil")
		}
		if res.Carrier.Slug != "brt-italy-parcelid" && res.Carrier.Slug != "brt-bartolini" {
			t.Errorf("expected BRT slug, got %s", res.Carrier.Slug)
		}
	})
}

func TestSDAExpressCourier(t *testing.T) {
	d := New()

	t.Run("13 alphanumeric characters", func(t *testing.T) {
		res := d.BestMatch("1W12345678901", "IT")
		if res == nil {
			t.Fatalf("expected match for SDA 13-char code, got nil")
		}
		if res.Carrier.Slug != "sda" {
			t.Errorf("expected slug sda, got %s", res.Carrier.Slug)
		}
	})

	t.Run("13 digits code", func(t *testing.T) {
		res := d.BestMatch("1234567890123", "IT")
		if res == nil {
			t.Fatalf("expected match for SDA 13-digit code, got nil")
		}
		if res.Carrier.Slug != "sda" {
			t.Errorf("expected slug sda, got %s", res.Carrier.Slug)
		}
	})
}

func TestGLSItaly(t *testing.T) {
	d := New()

	t.Run("GLSIT prefix", func(t *testing.T) {
		res := d.BestMatch("GLSIT123456789", "")
		if res == nil {
			t.Fatalf("expected match for GLSIT prefix, got nil")
		}
		if res.Carrier.Slug != "gls-italy" {
			t.Errorf("expected slug gls-italy, got %s", res.Carrier.Slug)
		}
	})

	t.Run("9 digits", func(t *testing.T) {
		res := d.BestMatch("123456789", "IT")
		if res == nil {
			t.Fatalf("expected match for 9-digit code, got nil")
		}
		if res.Carrier.Slug != "gls-italy" {
			t.Errorf("expected slug gls-italy, got %s", res.Carrier.Slug)
		}
	})

	t.Run("Branch code alphanumeric format L9", func(t *testing.T) {
		res := d.BestMatch("L912345678", "IT")
		if res == nil {
			t.Fatalf("expected match for branch code L912345678, got nil")
		}
		if res.Carrier.Slug != "gls-italy" {
			t.Errorf("expected slug gls-italy, got %s", res.Carrier.Slug)
		}
	})
}

func TestPosteSanMarinoDomestic(t *testing.T) {
	d := New()

	t.Run("SM prefix domestic", func(t *testing.T) {
		res := d.BestMatch("SM123456789", "")
		if res == nil {
			t.Fatalf("expected match for SM prefix, got nil")
		}
		if res.Carrier.Slug != "poste-san-marino" {
			t.Errorf("expected slug poste-san-marino, got %s", res.Carrier.Slug)
		}
	})

	t.Run("SM hint prioritizes Poste San Marino", func(t *testing.T) {
		candidates := d.Detect("123456789012", "SM")
		if len(candidates) == 0 {
			t.Fatalf("expected candidates for 12-digit number with SM hint")
		}
		// First candidate should be Poste San Marino due to destCountryHint="SM" boost
		if candidates[0].Carrier.Slug != "poste-san-marino" {
			t.Errorf("expected top candidate to be poste-san-marino, got %s", candidates[0].Carrier.Slug)
		}
	})
}

func TestCountryHintPrioritization(t *testing.T) {
	d := New()

	t.Run("IT hint boosts Italian carriers over global/foreign", func(t *testing.T) {
		candidates := d.Detect("123456789012", "IT")
		if len(candidates) < 2 {
			t.Fatalf("expected multiple candidates for ambiguous 12-digit number, got %d", len(candidates))
		}

		// The top candidate should have country IT
		if candidates[0].Carrier.Country != "IT" {
			t.Errorf("expected top candidate to have country IT, got %s (%s)", candidates[0].Carrier.Country, candidates[0].Carrier.Slug)
		}
	})
}

func TestAllResultingSlugsExistInRegistry(t *testing.T) {
	d := New()
	reg := carriers.GetRegistry()

	testNumbers := []struct {
		number string
		hint   string
	}{
		{"RA123456785IT", ""},
		{"CRONO123456789", ""},
		{"386123456789", "IT"},
		{"012345678901", "IT"},
		{"05512345678901", "IT"},
		{"055123456789012345", "IT"},
		{"1W12345678901", "IT"},
		{"1234567890123", "IT"},
		{"GLSIT123456789", ""},
		{"123456789", "IT"},
		{"L912345678", "IT"},
		{"RR123456785SM", ""},
		{"SM123456789", ""},
		{"123456789012", "SM"},
	}

	for _, tc := range testNumbers {
		candidates := d.Detect(tc.number, tc.hint)
		for _, c := range candidates {
			_, exists := reg.Get(c.Carrier.Slug)
			// Carrier should exist in registry if it is an IT or SM carrier
			if c.Carrier.Country == "IT" || c.Carrier.Country == "SM" {
				if !exists {
					t.Errorf("detected slug %q for number %s (country %s) not found in registry", c.Carrier.Slug, tc.number, c.Carrier.Country)
				}
			}
		}
	}
}
