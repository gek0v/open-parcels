package models

import (
	"errors"
	"fmt"
	"testing"
)

func TestPostalCodeErrorCanonicalEquivalence(t *testing.T) {
	if !errors.Is(ErrMissingPostalCode, ErrPostalCodeRequired) {
		t.Fatalf("expected ErrMissingPostalCode to match ErrPostalCodeRequired via errors.Is")
	}
	if !errors.Is(ErrPostalCodeRequired, ErrMissingPostalCode) {
		t.Fatalf("expected ErrPostalCodeRequired to match ErrMissingPostalCode via errors.Is")
	}

	wrappedRequired := fmt.Errorf("%w: carrier requires destination postal code", ErrPostalCodeRequired)
	if !errors.Is(wrappedRequired, ErrPostalCodeRequired) {
		t.Errorf("expected wrapped ErrPostalCodeRequired to match ErrPostalCodeRequired")
	}
	if !errors.Is(wrappedRequired, ErrMissingPostalCode) {
		t.Errorf("expected wrapped ErrPostalCodeRequired to match deprecated alias ErrMissingPostalCode")
	}

	wrappedMissing := fmt.Errorf("%w: missing CP", ErrMissingPostalCode)
	if !errors.Is(wrappedMissing, ErrPostalCodeRequired) {
		t.Errorf("expected wrapped ErrMissingPostalCode to match canonical ErrPostalCodeRequired")
	}
	if !errors.Is(wrappedMissing, ErrMissingPostalCode) {
		t.Errorf("expected wrapped ErrMissingPostalCode to match ErrMissingPostalCode")
	}
}
