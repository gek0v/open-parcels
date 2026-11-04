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

		// Portugal carriers detection
		{"Portugal Post UPU S10 valid check digit", "RR123456785PT", "PT", "portugal-post"},
		{"Portugal Post UPU S10 without dest hint", "DA123456785PT", "", "portugal-post"},
		{"CTT Expresso domestic barcode 13 digits", "0123456789012", "PT", "portugal-post"},
		{"CTT Expresso domestic barcode 14 digits", "01234567890123", "PT", "portugal-post"},
		{"DPD Portugal 14 digits 058 prefix", "05812345678901", "PT", "dpd-portugal"},
		{"DPD Portugal 14 digits 099 prefix", "09912345678901", "", "dpd-portugal"},
		{"DPD Portugal 14 digits 140 prefix", "14012345678901", "PT", "dpd-portugal"},
		{"DPD Portugal 14 digits 082 prefix", "08212345678901", "PT", "dpd-portugal"},
		{"VASP Expresso VASP prefix", "VASP12345678", "PT", "vasp-expresso"},
		{"VASP Expresso VP prefix", "VP1234567890", "", "vasp-expresso"},
		{"Delnext DLN prefix", "DLN12345678", "PT", "delnext"},
		{"Delnext DN prefix", "DN123456789012", "", "delnext"},
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

	// Test PT destination country hint bonus and candidate ranking
	t.Run("PT Destination Hint Bonus", func(t *testing.T) {
		// DPD Portugal number: without hint vs with PT hint
		withoutHint := d.Detect("05812345678901", "")
		withPTHint := d.Detect("05812345678901", "PT")

		if len(withoutHint) == 0 || len(withPTHint) == 0 {
			t.Fatalf("expected detections for 05812345678901")
		}

		// Top candidate should be dpd-portugal in both
		if withoutHint[0].Carrier.Slug != "dpd-portugal" {
			t.Errorf("expected dpd-portugal top match without hint, got %s", withoutHint[0].Carrier.Slug)
		}
		if withPTHint[0].Carrier.Slug != "dpd-portugal" {
			t.Errorf("expected dpd-portugal top match with PT hint, got %s", withPTHint[0].Carrier.Slug)
		}

		// Confidence with PT hint should be >= confidence without hint
		if withPTHint[0].Confidence < withoutHint[0].Confidence {
			t.Errorf("expected PT hint to boost confidence, got with=%f without=%f",
				withPTHint[0].Confidence, withoutHint[0].Confidence)
		}

		// Verify CTT Expresso domestic barcode candidates for 14-digit number
		foundCTT := false
		for _, c := range withPTHint {
			if c.Carrier.Slug == "portugal-post" {
				foundCTT = true
				if c.Carrier.Country != "PT" {
					t.Errorf("expected country PT for portugal-post, got %s", c.Carrier.Country)
				}
				break
			}
		}
		if !foundCTT {
			t.Errorf("expected portugal-post candidate for 14 digits starting with 0")
		}
	})

	// Test Modulo 11 check digit validation on UPU S10 PT numbers
	t.Run("UPU S10 Modulo 11 Check Digit Validation", func(t *testing.T) {
		// Valid check digit: RR123456785PT (checksum = 5)
		validRes := d.BestMatch("RR123456785PT", "PT")
		if validRes == nil || validRes.Carrier.Slug != "portugal-post" {
			t.Fatalf("expected valid UPU S10 to match portugal-post, got %v", validRes)
		}

		// Invalid check digit: RR123456780PT (expected 5, got 0)
		invalidRes := d.BestMatch("RR123456780PT", "PT")
		if invalidRes == nil {
			t.Fatalf("expected result for invalid checksum number")
		}

		// Valid checksum should have strictly higher confidence than invalid checksum
		if validRes.Confidence <= invalidRes.Confidence {
			t.Errorf("expected valid check digit confidence (%f) > invalid (%f)",
				validRes.Confidence, invalidRes.Confidence)
		}
	})

	// Test non-ambiguous patterns for Vasp Expresso and Delnext
	t.Run("Vasp and Delnext Candidates", func(t *testing.T) {
		vaspCandidates := d.Detect("VASP12345678", "PT")
		if len(vaspCandidates) == 0 || vaspCandidates[0].Carrier.Slug != "vasp-expresso" {
			t.Errorf("expected vasp-expresso as top candidate, got %v", vaspCandidates)
		}

		delnextCandidates := d.Detect("DLN12345678", "PT")
		if len(delnextCandidates) == 0 || delnextCandidates[0].Carrier.Slug != "delnext" {
			t.Errorf("expected delnext as top candidate, got %v", delnextCandidates)
		}
	})
}

func TestDetectorEdgeCases(t *testing.T) {
	d := New()

	// Empty and clean string handling
	if res := d.Detect("", "PT"); res != nil {
		t.Errorf("expected nil for empty raw number, got %v", res)
	}
	if res := d.BestMatch("???@@@", "PT"); res != nil {
		t.Errorf("expected nil for non-matching raw number, got %v", res)
	}

	// CleanTrackingNumber behavior
	cleaned := CleanTrackingNumber("  rr 12345678 5 pt \n")
	if cleaned != "RR123456785PT" {
		t.Errorf("expected RR123456785PT, got %s", cleaned)
	}
	cleanedAWB := CleanTrackingNumber(" 157-52471764 ")
	if cleanedAWB != "157-52471764" {
		t.Errorf("expected 157-52471764, got %s", cleanedAWB)
	}
}

func TestValidators(t *testing.T) {
	// UPUS10 validation
	if validateUPUS10("SHORT") {
		t.Errorf("expected false for short string")
	}
	if validateUPUS10("RR12345678XPT") {
		t.Errorf("expected false for invalid digits")
	}
	// Check sum % 11 == 1 -> check = 0
	// 0*8 + 0*6 + 0*4 + 0*2 + 0*3 + 0*5 + 9*9 + 0*7 = 81 % 11 = 4 -> 11-4 = 7
	// Let's construct sum % 11 == 1: 00000010: 1*9 = 9 % 11 = 9 -> check = 2
	// 00000003: 3*7 = 21 % 11 = 10 -> check = 1
	// 00000013: 9 + 21 = 30 % 11 = 8 -> check = 3
	// 00000001: 1*7 = 7 % 11 = 7 -> check = 4
	// 00000000: sum = 0 % 11 = 0 -> check = 5 (check == 11 -> 5)
	if !validateUPUS10("RR000000005PT") {
		t.Errorf("expected true for sum=0, check=5")
	}
	// Let's construct sum % 11 = 1 -> check = 10 -> check = 0:
	// 10000000: 1*8 = 8
	// 01000000: 1*6 = 6
	// 8 + 6 = 14 % 11 = 3
	// 10000001: 8 + 7 = 15 % 11 = 4
	// Let's find digits where sum % 11 == 1:
	// 2*8 + 1*6 = 22 % 11 = 0
	// 1*8 + 4*6 = 8 + 24 = 32 % 11 = 10
	// 0*8 + 2*6 = 12 % 11 = 1 -> digits: "02000000" -> sum = 12, rem = 1, check = 10 -> 0!
	if !validateUPUS10("RR020000000PT") {
		t.Errorf("expected true for sum%%11==1 -> check=0")
	}

	// UPS validation
	if validateUPS("SHORT") {
		t.Errorf("expected false for short UPS")
	}
	if validateUPS("2Z1234567890123456") {
		t.Errorf("expected false for non-1Z prefix")
	}
	if validateUPS("1Z123456789012345X") {
		t.Errorf("expected false for non-digit check char")
	}
	if validateUPS("1Z123456789012345?") {
		t.Errorf("expected false for invalid char in payload")
	}

	// AWB validation
	if validateAWB("INVALID") {
		t.Errorf("expected false for invalid AWB format")
	}
	if validateAWB("123-ABCDEFGH") {
		t.Errorf("expected false for non-numeric AWB")
	}
	if validateAWB("157-52471769") {
		t.Errorf("expected false for incorrect Mod 7 check")
	}
	if !validateAWB("157-52471764") {
		t.Errorf("expected true for valid AWB")
	}

	// Container validation
	if validateContainer("SHORT") {
		t.Errorf("expected false for short container")
	}
	if validateContainer("MRKA4656243") {
		t.Errorf("expected false for non-U/J/Z 4th char")
	}
	if !validateContainer("MRKJ4656243") {
		t.Errorf("expected true for container with J")
	}
	if !validateContainer("MRKZ4656243") {
		t.Errorf("expected true for container with Z")
	}

	// maxFloat coverage
	if maxFloat(1.0, 0.5) != 1.0 {
		t.Errorf("expected 1.0")
	}
	if maxFloat(0.3, 0.8) != 0.8 {
		t.Errorf("expected 0.8")
	}
}
