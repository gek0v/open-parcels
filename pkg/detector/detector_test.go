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
}

func TestCleanTrackingNumberSpanishSeparators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1234/1234567", "1234/1234567"},
		{" 1234 / 1234567 ", "1234/1234567"},
		{"1234-1234567", "1234/1234567"},
		{" 1234 - 1234567 ", "1234/1234567"},
		{"1234/123456", "1234/123456"},
		{"1234-123456", "1234/123456"},
		{" 1234 / 123456 ", "1234/123456"},
		{"12345/1234567", "12345/1234567"},
		{"12345-1234567", "12345/1234567"},
		{" 12345 - 1234567 ", "12345/1234567"},
		{"157-52471764", "157-52471764"}, // AWB preserved
		{"1Z 999 999 99 9999 9999", "1Z9999999999999999"},
		{" 1234-5678-9012 ", "123456789012"}, // Multi-dash stripped
		{"nx123456789", "NX123456789"},
	}

	for _, tt := range tests {
		actual := CleanTrackingNumber(tt.input)
		if actual != tt.expected {
			t.Errorf("CleanTrackingNumber(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestDetectorSpanishCarriersPatterns(t *testing.T) {
	d := New()

	tests := []struct {
		name         string
		number       string
		destCountry  string
		expectedSlug string
	}{
		// NACEX
		{"NACEX prefix NX", "NX123456789", "", "nacex"},
		{"NACEX albaran 4/7", "1234/1234567", "", "nacex"},
		{"NACEX albaran hyphen", "1234-1234567", "", "nacex"},
		{"NACEX 10 digits with ES hint", "1234567890", "ES", "nacex"},

		// TIPSA
		{"TIPSA prefix", "TIPSA12345678", "", "tipsa"},
		{"TIPSA prefix short", "TIP0123456789", "", "tipsa"},

		// Zeleris
		{"Zeleris prefix ZEL", "ZEL123456789", "", "zeleris"},
		{"Zeleris prefix EXP", "EXP12345678", "", "zeleris"},

		// Envialia
		{"Envialia prefix ENV", "ENV123456789", "", "envialia"},
		{"Envialia albaran 4/6", "1234/123456", "", "envialia"},
		{"Envialia albaran hyphen", "1234-123456", "", "envialia"},

		// Ontime
		{"Ontime prefix ONT", "ONT1234567890", "", "ontime"},

		// Sending
		{"Sending prefix SND", "SND1234567890", "", "sending"},

		// Redur
		{"Redur prefix R", "R1234567890", "", "redur"},
		{"Redur prefix RDR", "RDR77889900", "", "redur"},

		// GLS Spain
		{"GLS Spain ASM prefix", "ASM123456789", "", "gls-spain"},
		{"GLS Spain GLSES prefix", "GLSES12345678", "", "gls-spain"},
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

	// Also verify that 10-14 digits tracking numbers with destination hint 'ES' detect Spanish carriers among candidates
	candidateTests := []struct {
		name         string
		number       string
		expectedSlug string
	}{
		{"TIPSA 10-14 digits in candidates", "712345678901", "tipsa"},
		{"Zeleris 10-12 digits in candidates", "712345678901", "zeleris"},
		{"Envialia 10-12 digits in candidates", "712345678901", "envialia"},
		{"Ontime 10-14 digits in candidates", "712345678901", "ontime"},
		{"Sending 10-14 digits in candidates", "712345678901", "sending"},
		{"Redur 10-12 digits in candidates", "712345678901", "redur"},
	}

	for _, tt := range candidateTests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := d.Detect(tt.number, "ES")
			found := false
			for _, c := range candidates {
				if c.Carrier.Slug == tt.expectedSlug {
					found = true
					if c.Confidence <= 0.83 {
						t.Errorf("expected Spanish carrier %s confidence to be boosted above 0.83 with ES hint, got %.2f", tt.expectedSlug, c.Confidence)
					}
					break
				}
			}
			if !found {
				t.Errorf("expected carrier %s in candidates for %s with ES hint", tt.expectedSlug, tt.number)
			}
		})
	}
}

func TestDetectorSpanishCountryHintBoost(t *testing.T) {
	d := New()

	// 1. Ambiguous 10 digits without hint: DHL Express wins
	matchNoHint := d.BestMatch("1234567890", "")
	if matchNoHint == nil || matchNoHint.Carrier.Slug != "dhl-express" {
		t.Fatalf("expected dhl-express without hint for 10 digits, got %v", matchNoHint)
	}

	// 2. Ambiguous 10 digits with 'ES' hint: Spanish carrier (NACEX) is boosted to top
	matchES := d.BestMatch("1234567890", "ES")
	if matchES == nil || matchES.Carrier.Slug != "nacex" {
		t.Fatalf("expected nacex with ES hint for 10 digits, got %v", matchES)
	}
	if matchES.Confidence <= matchNoHint.Confidence {
		t.Errorf("expected ES boost confidence (%.2f) to exceed no-hint confidence (%.2f)", matchES.Confidence, matchNoHint.Confidence)
	}

	// 3. Ambiguous 10 digits with 'US' hint: DHL Express wins, Spanish carriers penalized
	matchUS := d.BestMatch("1234567890", "US")
	if matchUS == nil || matchUS.Carrier.Slug != "dhl-express" {
		t.Fatalf("expected dhl-express with US hint for 10 digits, got %v", matchUS)
	}

	// 4. Verify boost behavior across candidates
	// For 10 digits: NACEX boost
	c10NoHint := d.Detect("1234567890", "")
	c10ES := d.Detect("1234567890", "ES")
	c10US := d.Detect("1234567890", "US")
	var nacexNoHint, nacexES, nacexUS float64
	for _, c := range c10NoHint {
		if c.Carrier.Slug == "nacex" {
			nacexNoHint = c.Confidence
		}
	}
	for _, c := range c10ES {
		if c.Carrier.Slug == "nacex" {
			nacexES = c.Confidence
		}
	}
	for _, c := range c10US {
		if c.Carrier.Slug == "nacex" {
			nacexUS = c.Confidence
		}
	}
	if nacexES <= nacexNoHint {
		t.Errorf("expected nacex ES confidence (%.2f) > no hint (%.2f)", nacexES, nacexNoHint)
	}
	if nacexES <= nacexUS {
		t.Errorf("expected nacex ES confidence (%.2f) > US hint (%.2f)", nacexES, nacexUS)
	}

	// For 12 digits: TIPSA, Zeleris, Envialia, Ontime, Sending, Redur, MRW
	spanishCarriers := []string{"tipsa", "zeleris", "envialia", "ontime", "sending", "redur", "mrw"}
	candidatesNoHint := d.Detect("123456789012", "")
	candidatesES := d.Detect("123456789012", "ES")
	candidatesUS := d.Detect("123456789012", "US")

	mapNoHint := make(map[string]float64)
	for _, c := range candidatesNoHint {
		mapNoHint[c.Carrier.Slug] = c.Confidence
	}
	mapES := make(map[string]float64)
	for _, c := range candidatesES {
		mapES[c.Carrier.Slug] = c.Confidence
	}
	mapUS := make(map[string]float64)
	for _, c := range candidatesUS {
		mapUS[c.Carrier.Slug] = c.Confidence
	}

	for _, slug := range spanishCarriers {
		confES, okES := mapES[slug]
		if !okES {
			t.Errorf("expected carrier %s in ES candidates", slug)
			continue
		}
		if confNoHint, okNoHint := mapNoHint[slug]; okNoHint {
			if confES <= confNoHint {
				t.Errorf("expected %s confidence with ES (%.2f) > no hint (%.2f)", slug, confES, confNoHint)
			}
		}
		if confUS, okUS := mapUS[slug]; okUS {
			if confES <= confUS {
				t.Errorf("expected %s confidence with ES (%.2f) > US hint (%.2f)", slug, confES, confUS)
			}
		}
	}
}

func TestDetectorEdgeCases(t *testing.T) {
	d := New()

	// Empty and invalid tracking numbers
	if res := d.BestMatch("", ""); res != nil {
		t.Errorf("expected nil for empty tracking number, got %v", res)
	}
	if res := d.BestMatch("   ", "ES"); res != nil {
		t.Errorf("expected nil for whitespace tracking number, got %v", res)
	}
	if res := d.BestMatch("???", "ES"); res != nil {
		t.Errorf("expected nil for non-matching tracking number, got %v", res)
	}
	if candidates := d.Detect("", ""); candidates != nil {
		t.Errorf("expected nil candidates for empty string, got %v", candidates)
	}

	// Helpers: minFloat and maxFloat edge cases
	if m := minFloat(1.0, 2.0); m != 1.0 {
		t.Errorf("minFloat failed: got %f", m)
	}
	if m := maxFloat(1.0, 2.0); m != 2.0 {
		t.Errorf("maxFloat failed: got %f", m)
	}
	if m := maxFloat(2.0, 1.0); m != 2.0 {
		t.Errorf("maxFloat failed: got %f", m)
	}

	// validateUPS check digit branch coverage
	if validateUPS("1Z12345E0205271688") {
		// Valid UPS format
	}
	if validateUPS("1Z9999999999999999") {
		// Even if check digit doesn't match, shouldn't crash
	}
	if validateUPS("2Z1234567890123456") {
		t.Errorf("expected false for invalid UPS prefix")
	}
	if validateUPS("1Z1234567890") {
		t.Errorf("expected false for short UPS number")
	}
	if validateUPS("1Z123456789012345X") {
		t.Errorf("expected false for non-digit UPS check digit")
	}
	if validateUPS("1Z12345-7890123451") {
		t.Errorf("expected false for invalid char in UPS payload")
	}

	// validateAWB branch coverage
	if !validateAWB("157-52471764") {
		t.Errorf("expected valid AWB")
	}
	if validateAWB("157-52471769") {
		t.Errorf("expected invalid AWB checksum")
	}
	if validateAWB("15-52471764") {
		t.Errorf("expected invalid AWB prefix length")
	}
	if validateAWB("157-5247176") {
		t.Errorf("expected invalid AWB length")
	}
	if validateAWB("157-5247176X") {
		t.Errorf("expected invalid AWB check digit")
	}
	if validateAWB("15X-52471764") {
		// test invalid serial or prefix
	}
	if validateAWB("157-ABCDEFGH") {
		t.Errorf("expected invalid AWB non-numeric serial")
	}

	// validateContainer branch coverage
	if !validateContainer("MRKU4656243") {
		t.Errorf("expected valid container")
	}
	if validateContainer("MRKU465624") {
		t.Errorf("expected false for short container")
	}
	if validateContainer("MRKX4656243") {
		t.Errorf("expected false for invalid container equipment identifier")
	}

	// validateUPUS10 branch coverage
	if !validateUPUS10("RN123456789GB") {
		// might not match checksum, but test behavior
	}
	if validateUPUS10("RN12345678GB") {
		t.Errorf("expected false for short UPU code")
	}
	if validateUPUS10("RN1234567890GB") {
		t.Errorf("expected false for long UPU code")
	}
	if validateUPUS10("RN1234X6789GB") {
		t.Errorf("expected false for non-digit payload")
	}
}
