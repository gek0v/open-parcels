// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package detector

import (
	"testing"
)

func TestDetectorEdgeCases(t *testing.T) {
	d := New()

	t.Run("Empty tracking number", func(t *testing.T) {
		res := d.Detect("", "")
		if res != nil {
			t.Errorf("expected nil for empty tracking number, got %v", res)
		}

		best := d.BestMatch("", "")
		if best != nil {
			t.Errorf("expected nil BestMatch for empty string, got %v", best)
		}
	})

	t.Run("Unmatched tracking number", func(t *testing.T) {
		res := d.Detect("ZZZ___NO_MATCH_99999___ZZZ", "")
		if len(res) != 0 {
			t.Errorf("expected no candidates for unknown string, got %d", len(res))
		}

		best := d.BestMatch("ZZZ___NO_MATCH_99999___ZZZ", "")
		if best != nil {
			t.Errorf("expected nil BestMatch for unmatched string, got %v", best)
		}
	})

	t.Run("minFloat and maxFloat", func(t *testing.T) {
		if minFloat(1.5, 2.5) != 1.5 {
			t.Errorf("minFloat failed")
		}
		if minFloat(3.0, 2.0) != 2.0 {
			t.Errorf("minFloat failed")
		}
		if maxFloat(1.5, 2.5) != 2.5 {
			t.Errorf("maxFloat failed")
		}
		if maxFloat(3.0, 2.0) != 3.0 {
			t.Errorf("maxFloat failed")
		}
	})
}

func TestValidateUPS(t *testing.T) {
	// Wrong prefix or length
	if validateUPS("1X1234567890123456") {
		t.Errorf("expected false for non-1Z prefix")
	}
	if validateUPS("1Z123456789") {
		t.Errorf("expected false for short string")
	}
	// Invalid check digit char
	if validateUPS("1Z123456789012345X") {
		t.Errorf("expected false for non-digit check digit")
	}
	// Invalid character in payload (e.g. lowercase or non-ascii)
	if validateUPS("1Z12345678901234!5") {
		t.Errorf("expected false for invalid symbol in payload")
	}
}

func TestValidateAWB(t *testing.T) {
	// Wrong parts
	if validateAWB("123456789") {
		t.Errorf("expected false for no hyphen")
	}
	if validateAWB("12-12345678") {
		t.Errorf("expected false for short prefix")
	}
	if validateAWB("123-1234567") {
		t.Errorf("expected false for short suffix")
	}
	// Non-digit
	if validateAWB("123-ABCDEFGH") {
		t.Errorf("expected false for letters in AWB")
	}
	if validateAWB("123-1234567A") {
		t.Errorf("expected false for letter in check digit")
	}
	// Checksum mismatch
	if validateAWB("157-52471769") {
		t.Errorf("expected false for invalid Mod 7 check")
	}
}

func TestValidateContainer(t *testing.T) {
	// Invalid length
	if validateContainer("ABC") {
		t.Errorf("expected false for short container")
	}
	// Invalid category identifier (must be U, J, or Z)
	if validateContainer("CSQU3054383") != true {
		t.Errorf("expected true for U container")
	}
	if validateContainer("CSQJ3054383") != true {
		t.Errorf("expected true for J container")
	}
	if validateContainer("CSQZ3054383") != true {
		t.Errorf("expected true for Z container")
	}
	if validateContainer("CSQX3054383") {
		t.Errorf("expected false for X container")
	}
}

func TestValidateUPUS10CheckDigitCases(t *testing.T) {
	// Invalid length
	if validateUPUS10("SHORT") {
		t.Errorf("expected false for short S10")
	}

	// Non-digit in serial
	if validateUPUS10("RA12345X785IT") {
		t.Errorf("expected false for non-digit in serial")
	}

	// Test check digit = 0 (when 11 - (sum % 11) == 10, check = 0)
	// Weights: 8, 6, 4, 2, 3, 5, 9, 7
	// Find digits such that sum % 11 == 1:
	// For example: 8*0 + 6*0 + 4*0 + 2*0 + 3*0 + 5*0 + 9*0 + 7*7 = 49. 49 % 11 = 5.
	// We want sum % 11 = 1:
	// 8*0 + 6*0 + 4*0 + 2*0 + 3*0 + 5*0 + 9*0 + 7*5 = 35. 35 % 11 = 2.
	// 7*8 = 56. 56 % 11 = 1!
	// So "RA000000080IT": sum = 8*7 = 56. 56 % 11 = 1. check = 11 - 1 = 10 -> check = 0!
	if !validateUPUS10("RA000000080IT") {
		t.Errorf("expected true for check digit 0 case (RA000000080IT)")
	}

	// Test check digit = 5 (when sum % 11 == 0, check = 11 - 0 = 11 -> check = 5)
	// For "RA000000005IT": sum = 0. 0 % 11 = 0. check = 11 -> 5.
	if !validateUPUS10("RA000000005IT") {
		t.Errorf("expected true for check digit 5 case (RA000000005IT)")
	}
}

func TestValidateSDAEdgeCases(t *testing.T) {
	// UPU S10 format: 2 letters, 9 digits, 2 letters -> returns false
	if validateSDA("RR123456789IT") {
		t.Errorf("expected validateSDA to return false for UPU S10 format")
	}

	// Standard SDA: 13 digits -> returns true
	if !validateSDA("1234567890123") {
		t.Errorf("expected validateSDA to return true for 13 digits")
	}

	// Standard SDA: 12 digits -> returns true
	if !validateSDA("123456789012") {
		t.Errorf("expected validateSDA to return true for 12 digits")
	}
}
