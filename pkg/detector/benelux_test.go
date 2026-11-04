// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package detector

import (
	"testing"
)

func TestBeneluxTrackingDetection(t *testing.T) {
	d := New()

	tests := []struct {
		name         string
		number       string
		destCountry  string
		expectedSlug string
	}{
		// PostNL (Netherlands)
		{"PostNL UPU S10 valid", "CD123456785NL", "NL", "postnl"},
		{"PostNL UPU S10 without hint", "CD123456785NL", "", "postnl"},
		{"PostNL domestic 3S 14 chars", "3SABCD12345678", "NL", "postnl"},
		{"PostNL domestic 3S 17 chars", "3S123456789012345", "NL", "postnl"},
		{"PostNL domestic KG 16 digits", "KG1234567890123456", "NL", "postnl"},

		// bpost (Belgium)
		{"bpost UPU S10 valid", "CD123456785BE", "BE", "bpost"},
		{"bpost UPU S10 without hint", "CD123456785BE", "", "bpost"},
		{"bpost domestic 3232 prefix", "323212345678901234", "BE", "bpost"},
		{"bpost domestic 2182 prefix", "218212345678901234", "BE", "bpost"},
		{"bpost domestic 24 digits", "123456789012345678901234", "BE", "bpost"},

		// POST Luxembourg
		{"POST Luxembourg UPU S10 valid", "CD123456785LU", "LU", "luxembourg-post"},
		{"POST Luxembourg UPU S10 without hint", "CD123456785LU", "", "luxembourg-post"},
		{"POST Luxembourg domestic LU prefix", "LU123456789012", "LU", "luxembourg-post"},
		{"POST Luxembourg domestic LU 10 digits", "LU1234567890", "LU", "luxembourg-post"},
		{"POST Luxembourg domestic LU 16 digits", "LU1234567890123456", "LU", "luxembourg-post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := d.BestMatch(tt.number, tt.destCountry)
			if res == nil {
				t.Fatalf("expected match for %s, got nil", tt.number)
			}
			if res.Carrier.Slug != tt.expectedSlug {
				t.Errorf("expected slug %q, got %q (confidence: %f, reason: %s)", tt.expectedSlug, res.Carrier.Slug, res.Confidence, res.Reason)
			}
		})
	}
}

func TestBeneluxUPUChecksumMod11Confidence(t *testing.T) {
	d := New()

	cases := []struct {
		name          string
		validNumber   string
		invalidNumber string
		expectedSlug  string
	}{
		{"PostNL Mod11", "CD123456785NL", "CD123456789NL", "postnl"},
		{"PostNL Mod11 check=0", "CD000000120NL", "CD000000129NL", "postnl"},
		{"PostNL Mod11 check=5", "CD000000005NL", "CD000000009NL", "postnl"},
		{"bpost Mod11", "CD123456785BE", "CD123456789BE", "bpost"},
		{"POST Luxembourg Mod11", "CD123456785LU", "CD123456789LU", "luxembourg-post"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validRes := d.Detect(tc.validNumber, "")
			invalidRes := d.Detect(tc.invalidNumber, "")

			var validConf float64 = -1
			for _, r := range validRes {
				if r.Carrier.Slug == tc.expectedSlug {
					validConf = r.Confidence
					break
				}
			}

			var invalidConf float64 = -1
			for _, r := range invalidRes {
				if r.Carrier.Slug == tc.expectedSlug {
					invalidConf = r.Confidence
					break
				}
			}

			if validConf == -1 {
				t.Fatalf("carrier %q not found in candidates for valid tracking number %s", tc.expectedSlug, tc.validNumber)
			}
			if invalidConf == -1 {
				t.Fatalf("carrier %q not found in candidates for invalid tracking number %s", tc.expectedSlug, tc.invalidNumber)
			}

			if validConf <= invalidConf {
				t.Errorf("expected valid checksum confidence (%f) to be strictly greater than invalid (%f) for %s", validConf, invalidConf, tc.expectedSlug)
			}
		})
	}
}

func TestBeneluxDestinationCountryHintBoost(t *testing.T) {
	d := New()

	hints := []struct {
		name         string
		number       string
		targetHint   string
		otherHint    string
		expectedSlug string
	}{
		{"PostNL NL boost", "CD123456785NL", "NL", "ES", "postnl"},
		{"PostNL domestic NL boost", "3SABCD12345678", "NL", "FR", "postnl"},
		{"bpost BE boost", "CD123456785BE", "BE", "US", "bpost"},
		{"bpost domestic BE boost", "323212345678901234", "BE", "ES", "bpost"},
		{"POST Luxembourg LU boost", "CD123456785LU", "LU", "DE", "luxembourg-post"},
		{"POST Luxembourg domestic LU boost", "LU123456789012", "LU", "FR", "luxembourg-post"},
	}

	for _, tc := range hints {
		t.Run(tc.name, func(t *testing.T) {
			withTarget := d.Detect(tc.number, tc.targetHint)
			withOther := d.Detect(tc.number, tc.otherHint)
			withoutHint := d.Detect(tc.number, "")

			var targetConf float64 = -1
			for _, r := range withTarget {
				if r.Carrier.Slug == tc.expectedSlug {
					targetConf = r.Confidence
					break
				}
			}

			var otherConf float64 = -1
			for _, r := range withOther {
				if r.Carrier.Slug == tc.expectedSlug {
					otherConf = r.Confidence
					break
				}
			}

			var noHintConf float64 = -1
			for _, r := range withoutHint {
				if r.Carrier.Slug == tc.expectedSlug {
					noHintConf = r.Confidence
					break
				}
			}

			if targetConf == -1 {
				t.Fatalf("carrier %q not found with target hint %s", tc.expectedSlug, tc.targetHint)
			}

			if targetConf <= noHintConf {
				t.Errorf("expected targetHint confidence (%f) to be > noHint confidence (%f)", targetConf, noHintConf)
			}
			if targetConf <= otherConf {
				t.Errorf("expected targetHint confidence (%f) to be > otherHint confidence (%f)", targetConf, otherConf)
			}
		})
	}
}

func TestDetectorCoverageEdgeCases(t *testing.T) {
	d := New()

	// Empty tracking number
	if res := d.Detect("", ""); res != nil {
		t.Errorf("expected nil result for empty tracking number, got %+v", res)
	}
	if best := d.BestMatch("", ""); best != nil {
		t.Errorf("expected nil BestMatch for empty tracking number, got %+v", best)
	}

	// Unmatched number
	if best := d.BestMatch("XYZ!@#UNKNOWN", ""); best != nil {
		t.Errorf("expected nil BestMatch for unmatched number, got %+v", best)
	}

	// CleanTrackingNumber air cargo preserved
	awb := CleanTrackingNumber("123-12345678")
	if awb != "123-12345678" {
		t.Errorf("expected AWB format preserved, got %q", awb)
	}

	// CleanTrackingNumber alphanumeric uppercased and stripped
	cleaned := CleanTrackingNumber(" 3s-ab cd_123 \n")
	if cleaned != "3SABCD123" {
		t.Errorf("expected cleaned alphanumeric, got %q", cleaned)
	}

	// Test validateUPUS10 edge cases directly
	if validateUPUS10("short") {
		t.Errorf("validateUPUS10 should return false for invalid length")
	}
	if validateUPUS10("AB1234567A8NL") {
		t.Errorf("validateUPUS10 should return false for non-digit character in digits")
	}

	// Test validateUPS edge cases
	if validateUPS("1Z") {
		t.Errorf("validateUPS should return false for short string")
	}
	if validateUPS("2Z1234567890123456") {
		t.Errorf("validateUPS should return false for non-1Z prefix")
	}
	if validateUPS("1Z123456789012345X") {
		t.Errorf("validateUPS should return false for non-numeric check digit")
	}
	if validateUPS("1Z12345!7890123456") {
		t.Errorf("validateUPS should return false for invalid characters")
	}

	// Test validateAWB edge cases
	if validateAWB("invalid-awb") {
		t.Errorf("validateAWB should return false for invalid AWB format")
	}
	if validateAWB("12-12345678") {
		t.Errorf("validateAWB should return false for short prefix")
	}
	if validateAWB("123-1234567X") {
		t.Errorf("validateAWB should return false for non-numeric check digit")
	}
	if validateAWB("12X-12345678") {
		t.Errorf("validateAWB should return false for non-numeric prefix")
	}

	// Test validateContainer edge cases
	if validateContainer("short") {
		t.Errorf("validateContainer should return false for short length")
	}
	if validateContainer("ABCX1234567") {
		t.Errorf("validateContainer should return false when 4th letter is not U, J, or Z")
	}
	if !validateContainer("MSCU1234567") {
		t.Errorf("validateContainer should return true for valid container code")
	}
}
