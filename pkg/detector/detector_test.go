// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package detector

import (
	"testing"
)

func TestDetector(t *testing.T) {
	d := New()

	tests := []struct {
		name         string
		number       string
		destCountry  string
		expectedSlug string
	}{
		{"UPS standard 1Z", "1Z9999999999999999", "US", "ups"},
		{"YunExpress YT", "YT2401234567890123", "", "yunexpress"},
		{"Cainiao LP", "LP00561234567890", "ES", "cainiao"},
		{"Cainiao CNESP", "CNESP0012345678", "ES", "cainiao"},
		{"Correos Express 16 digits", "6900123456789012", "ES", "correos-express"},
		{"CTT Express", "0001234567890123456789", "ES", "ctt-express"},
		{"SEUR 14 digits", "36150127630406", "ES", "seur"},
		{"USPS 22 digits", "9400111206213785678901", "US", "usps"},
		{"FedEx 12 digits", "123456789012", "US", "fedex"},
		{"DHL Express 10 digits", "1234567890", "", "dhl-express"},
		{"DHL Parcel GM", "GM5551234567890123", "DE", "dhl-parcel"},
		{"Air Cargo AWB", "157-52471764", "", "air-cargo"},
		{"Ocean Container", "MRKU4656243", "", "ocean-container"},
		{"Paack", "PCK98765432101", "ES", "paack"},
		{"Correos Spain UPU S10", "PQ4F6P07000000000000000", "ES", "correos-spain"},
		{"Royal Mail UPU", "RN123456789GB", "GB", "royal-mail"},
		{"Colissimo", "6A12345678901", "FR", "colissimo"},

		// Ocean Bill of Lading (B/L) detection tests
		{"Evergreen B/L canonical", "EGLV147000380361", "", "evergreen"},
		{"Maersk Line B/L MAEU", "MAEU123456789", "", "maersk-line"},
		{"Maersk Line B/L MSK", "MSK9876543210", "", "maersk-line"},
		{"MSC B/L alphanumeric", "MEDUZA123456", "", "msc"},
		{"MSC B/L digits", "MSCU1234567890", "", "msc"},
		{"Hapag-Lloyd B/L Hamburg", "HLCUHAM123456", "", "hapag-lloyd"},
		{"Hapag-Lloyd B/L digits", "HLCU1234567890", "", "hapag-lloyd"},
		{"COSCO B/L canonical", "COSU6321234560", "", "cosco"},
		{"CMA CGM B/L", "CMAU1234567890", "", "cma-cgm"},
		{"ONE Line B/L", "ONEU1234567890", "", "one-line"},
		{"Yang Ming B/L", "YMLU1234567890", "", "yang-ming"},
		{"HMM B/L", "HDMU1234567890", "", "hmm"},
		{"ZIM B/L", "ZIMU1234567890", "", "zim"},
		{"Dole Ocean Cargo B/L", "DOLU1234567890", "", "dole-ocean-cargo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := d.BestMatch(tt.number, tt.destCountry)
			if res == nil {
				t.Fatalf("expected match for %s, got nil", tt.number)
			}
			if res.Carrier.Slug != tt.expectedSlug {
				t.Errorf("expected slug %s, got %s (reason: %s)", tt.expectedSlug, res.Carrier.Slug, res.Reason)
			}
		})
	}
}

func TestDetectorCandidates(t *testing.T) {
	d := New()
	// A 12-digit number could be FedEx, MRW, GLS
	candidates := d.Detect("123456789012", "ES")
	if len(candidates) < 2 {
		t.Fatalf("expected multiple candidates for ambiguous 12-digit number, got %d", len(candidates))
	}
	// For ES hint, MRW or Spanish carriers should be prioritized or present
	foundMRW := false
	for _, c := range candidates {
		if c.Carrier.Slug == "mrw" {
			foundMRW = true
			break
		}
	}
	if !foundMRW {
		t.Errorf("expected mrw in candidates for 12 digits with ES hint")
	}

	// Country hint penalty branch test: carrier country doesn't match hint and is not GLOBAL/CN
	resPen := d.Detect("36150127630406", "US") // SEUR (ES carrier) with US destination hint
	if len(resPen) == 0 {
		t.Errorf("expected detection candidates even with country penalty")
	}

	// Empty tracking number
	emptyRes := d.Detect("   ", "")
	if emptyRes != nil {
		t.Errorf("expected nil for empty tracking number")
	}

	// BestMatch returns nil when no rules match
	noMatch := d.BestMatch("???INVALID???", "")
	if noMatch != nil {
		t.Errorf("expected nil best match for invalid number")
	}
}

func TestDetectorOceanContainerCandidates(t *testing.T) {
	d := New()
	// An ISO container with MSKU prefix should match ocean-container as best match,
	// and also propose maersk-line among candidates via BIC prefix mapping.
	candidates := d.Detect("MSKU9070323", "")
	if len(candidates) == 0 {
		t.Fatalf("expected candidates for MSKU9070323")
	}
	if candidates[0].Carrier.Slug != "ocean-container" {
		t.Errorf("expected top candidate ocean-container for backward compatibility, got %s", candidates[0].Carrier.Slug)
	}

	foundCarrier := false
	for _, c := range candidates {
		if c.Carrier.Slug == "maersk-line" {
			foundCarrier = true
			break
		}
	}
	if !foundCarrier {
		t.Errorf("expected maersk-line in candidates for MSKU container prefix")
	}
}

func TestValidateISO6346Checksum(t *testing.T) {
	validContainers := []string{
		"MRKU4656243",   // Canonical Maersk container
		"CSQU3054383",   // Canonical Hapag-Lloyd container
		"MSKU9070323",   // Canonical Maersk container
		"CMAU0000005",   // Canonical CMA CGM container
		"HLXU6000000",   // Canonical check digit 0 (modulo remainder 10)
		"mrku4656243",   // Lowercase normalized
		" MSKU9070323 ", // Padded with spaces
	}

	for _, c := range validContainers {
		if !ValidateISO6346Checksum(c) {
			t.Errorf("expected container %q to pass ISO 6346 checksum validation", c)
		}
		if !validateISO6346Checksum(c) {
			t.Errorf("expected internal validateISO6346Checksum for %q to pass", c)
		}
	}

	invalidContainers := []struct {
		number string
		reason string
	}{
		{"MRKU4656249", "bad check digit (should be 3)"},
		{"CSQU3054381", "bad check digit (should be 3)"},
		{"MRKU465624", "too short (10 chars)"},
		{"MRKU46562431", "too long (12 chars)"},
		{"MRKA4656243", "category identifier not U/J/Z"},
		{"12345678901", "digits instead of letter owner code"},
		{"MRKU465624A", "non-digit check digit"},
		{"", "empty string"},
		{"MRKU 4656243", "spaces inside"},
		{"MRKU46562X3", "non-digit in serial number"},
	}

	for _, tt := range invalidContainers {
		if ValidateISO6346Checksum(tt.number) {
			t.Errorf("expected container %q to fail ISO 6346 validation (%s)", tt.number, tt.reason)
		}
	}

	// Direct isoCharValue test for non-letter
	if val := isoCharValue('1'); val != 0 {
		t.Errorf("expected 0 for non-letter, got %d", val)
	}
	if val := isoCharValue('Z'); val != 38 {
		t.Errorf("expected 38 for Z, got %d", val)
	}
}

func TestBICPrefixLookup(t *testing.T) {
	tests := []struct {
		prefix       string
		expectedSlug string
		expectFound  bool
	}{
		{"MSKU", "maersk-line", true},
		{"MAEU", "maersk-line", true},
		{"MRKU", "maersk-line", true},
		{"PONU", "maersk-line", true},
		{"MSCU", "msc", true},
		{"MEDU", "msc", true},
		{"CMAU", "cma-cgm", true},
		{"CGMU", "cma-cgm", true},
		{"ANLU", "cma-cgm", true},
		{"COSU", "cosco", true},
		{"CBHU", "cosco", true},
		{"CCLU", "cosco", true},
		{"HLCU", "hapag-lloyd", true},
		{"HLXU", "hapag-lloyd", true},
		{"CSQU", "hapag-lloyd", true},
		{"EGLV", "evergreen", true},
		{"EMCU", "evergreen", true},
		{"EISU", "evergreen", true},
		{"EGSU", "evergreen", true},
		{"UGXU", "evergreen", true},
		{"ONEU", "one-line", true},
		{"NYKU", "one-line", true},
		{"MOLU", "one-line", true},
		{"KKFU", "one-line", true},
		{"KLXU", "one-line", true},
		{"YMLU", "yang-ming", true},
		{"HDMU", "hmm", true},
		{"ZIMU", "zim", true},
		{"DOLU", "dole-ocean-cargo", true},
		{"msku", "maersk-line", true}, // lowercase
		{"ZZZZ", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		slug, found := LookupBICPrefix(tt.prefix)
		if found != tt.expectFound {
			t.Errorf("LookupBICPrefix(%q): found=%v, want %v", tt.prefix, found, tt.expectFound)
		}
		if slug != tt.expectedSlug {
			t.Errorf("LookupBICPrefix(%q): slug=%q, want %q", tt.prefix, slug, tt.expectedSlug)
		}
	}

	// Test LookupBIC with full container number
	slug, found := LookupBIC("MSKU9070323")
	if !found || slug != "maersk-line" {
		t.Errorf("LookupBIC(MSKU9070323): got slug=%q, found=%v; want maersk-line, true", slug, found)
	}

	// Test LookupBIC with short/invalid input
	_, foundShort := LookupBIC("MSK")
	if foundShort {
		t.Errorf("LookupBIC(MSK): expected false for input shorter than 4 chars")
	}
}

func TestHelperValidators(t *testing.T) {
	// validateUPS branches
	if validateUPS("SHORT") {
		t.Errorf("expected validateUPS to fail on short")
	}
	if validateUPS("1Z999999999999999A") {
		t.Errorf("expected validateUPS to fail on non-digit check")
	}
	if validateUPS("1Z999999-999999991") {
		t.Errorf("expected validateUPS to fail on invalid char in payload")
	}

	// validateAWB branches
	if validateAWB("123-12") {
		t.Errorf("expected validateAWB to fail on short serial")
	}
	if validateAWB("ABC-12345678") {
		t.Errorf("expected validateAWB to fail on non-digit")
	}
	if validateAWB("123-1234567A") {
		t.Errorf("expected validateAWB to fail on non-digit check")
	}

	// validateUPUS10 branches
	if validateUPUS10("SHORT") {
		t.Errorf("expected validateUPUS10 to fail on wrong length")
	}
	if validateUPUS10("RA1234567X9ES") {
		t.Errorf("expected validateUPUS10 to fail on non-digit payload")
	}

	// Math helper branches
	if minFloat(3.0, 5.0) != 3.0 || minFloat(5.0, 3.0) != 3.0 {
		t.Errorf("minFloat failed")
	}
	if maxFloat(3.0, 5.0) != 5.0 || maxFloat(5.0, 3.0) != 5.0 {
		t.Errorf("maxFloat failed")
	}
}
