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
		{"Correios Brazil UPU S10", "RR123456785BR", "", "correios-brazil"},
		{"Mexico Post UPU S10", "RR123456785MX", "", "mexico-post"},
		{"Correo Argentino UPU S10", "RR123456785AR", "", "correo-argentino"},
		{"Peru Post UPU S10", "RR123456785PE", "", "peru-post"},
		{"Uruguay Post UPU S10", "RR123456785UY", "", "uruguay-post"},
		{"Estafeta Mexico", "2212345678901234567890", "", "estafeta"},
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
