package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"open-parcels/pkg/models"
)

func TestNormalizeArgsWithPositional(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedFlags      []string
		expectedPositional string
	}{
		{
			name:               "tracking number positional first",
			args:               []string{"LP00561234567890", "-c", "cainiao", "-l", "es"},
			expectedFlags:      []string{"-c", "cainiao", "-l", "es"},
			expectedPositional: "LP00561234567890",
		},
		{
			name:               "flag with value first then positional",
			args:               []string{"--carrier", "correosexpress", "6900123456789012"},
			expectedFlags:      []string{"--carrier", "correosexpress"},
			expectedPositional: "6900123456789012",
		},
		{
			name:               "flag with equals notation",
			args:               []string{"--query=cainiao"},
			expectedFlags:      []string{"--query=cainiao"},
			expectedPositional: "",
		},
		{
			name:               "boolean flag followed by positional",
			args:               []string{"--json", "LP00561234567890"},
			expectedFlags:      []string{"--json"},
			expectedPositional: "LP00561234567890",
		},
		{
			name:               "postal code shorthand -z followed by positional",
			args:               []string{"-z", "28001", "7561674"},
			expectedFlags:      []string{"-z", "28001"},
			expectedPositional: "7561674",
		},
		{
			name:               "mock boolean flag with postal-code and positional",
			args:               []string{"--mock", "--postal-code", "28001", "7561674"},
			expectedFlags:      []string{"--mock", "--postal-code", "28001"},
			expectedPositional: "7561674",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, pos := normalizeArgsWithPositional(tt.args)
			if pos != tt.expectedPositional {
				t.Errorf("expected positional '%s', got '%s'", tt.expectedPositional, pos)
			}
			if len(flags) != len(tt.expectedFlags) {
				t.Fatalf("expected flags %v, got %v", tt.expectedFlags, flags)
			}
			if len(flags) > 0 && !reflect.DeepEqual(flags, tt.expectedFlags) {
				t.Errorf("flags mismatch: expected %v, got %v", tt.expectedFlags, flags)
			}
		})
	}
}

func TestClassifyTrackError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{"carrier blocked", fmt.Errorf("wrapped: %w", models.ErrCarrierBlocked), "CARRIER_BLOCKED"},
		{"scraper schema changed", fmt.Errorf("wrapped: %w", models.ErrScraperSchemaChanged), "SCRAPER_ERROR"},
		{"rate limited", models.ErrCarrierRateLimited, "CARRIER_RATE_LIMITED"},
		{"missing postal code", models.ErrPostalCodeRequired, "MISSING_POSTAL_CODE"},
		{"not implemented", models.ErrCarrierNotImplemented, "NOT_IMPLEMENTED"},
		{"not found", models.ErrShipmentNotFound, "NOT_FOUND"},
		{"unknown fallback", errors.New("unexpected failure"), "NOT_FOUND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, suggestion := classifyTrackError(tt.err)
			if code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, code)
			}
			if suggestion == "" {
				t.Errorf("expected non-empty suggestion")
			}
		})
	}
}

