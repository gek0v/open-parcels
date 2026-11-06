package cmd

import (
	"reflect"
	"testing"
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
