// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

func TestLuxembourgPostCarrier_Interface(t *testing.T) {
	c := NewLuxembourgPostCarrier()
	var _ Carrier = c
	var _ CarrierWithAliases = c

	if c.Slug() != "luxembourg-post" {
		t.Errorf("expected slug 'luxembourg-post', got '%s'", c.Slug())
	}
	if c.Name() != "POST Luxembourg" {
		t.Errorf("expected name 'POST Luxembourg', got '%s'", c.Name())
	}
	if c.Country() != "LU" {
		t.Errorf("expected country 'LU', got '%s'", c.Country())
	}
	if c.CountryName() != "Luxembourg" {
		t.Errorf("expected country name 'Luxembourg', got '%s'", c.CountryName())
	}
	if c.Type() != models.CarrierTypePostal {
		t.Errorf("expected type '%s', got '%s'", models.CarrierTypePostal, c.Type())
	}
	if c.Website() != "https://www.post.lu" {
		t.Errorf("expected website 'https://www.post.lu', got '%s'", c.Website())
	}

	trackingURL := c.TrackingURL("LU123456789LU")
	expectedURL := "https://www.post.lu/en/particuliers/courrier-colis/track-and-trace?barcode=LU123456789LU"
	if trackingURL != expectedURL {
		t.Errorf("expected tracking URL '%s', got '%s'", expectedURL, trackingURL)
	}

	info := c.Info()
	if info.Slug != "luxembourg-post" || info.Country != "LU" || info.CountryName != "Luxembourg" {
		t.Errorf("unexpected carrier info: %+v", info)
	}

	aliases := c.Aliases()
	foundAlias := false
	for _, a := range aliases {
		if a == "post-luxembourg" {
			foundAlias = true
			break
		}
	}
	if !foundAlias {
		t.Errorf("expected 'post-luxembourg' in aliases, got %v", aliases)
	}
}

func TestLuxembourgPostCarrier_Track_Events(t *testing.T) {
	c := NewLuxembourgPostCarrier()
	ctx := context.Background()
	trackingNumber := "LU123456789LU"

	shipment, err := c.Track(ctx, trackingNumber, models.TrackOptions{})
	if err != nil {
		t.Fatalf("unexpected error tracking shipment: %v", err)
	}
	if shipment == nil {
		t.Fatal("expected non-nil shipment")
	}

	if shipment.TrackingNumber != trackingNumber {
		t.Errorf("expected tracking number %s, got %s", trackingNumber, shipment.TrackingNumber)
	}
	if shipment.Origin.Code != "LU" || shipment.Origin.Name != "Luxembourg" {
		t.Errorf("expected origin LU/Luxembourg, got %+v", shipment.Origin)
	}
	if shipment.Carrier.Slug != "luxembourg-post" {
		t.Errorf("expected carrier slug 'luxembourg-post', got '%s'", shipment.Carrier.Slug)
	}
	if len(shipment.Events) == 0 {
		t.Fatal("expected tracking events, got empty slice")
	}

	// Verify Bettembourg sorting center and local distribution events exist
	hasBettembourg := false
	hasDistribution := false

	for _, ev := range shipment.Events {
		locDesc := ev.Location + " " + ev.Description
		if strings.Contains(locDesc, "Bettembourg") {
			hasBettembourg = true
		}
		if strings.Contains(locDesc, "distribution local") || strings.Contains(locDesc, "distribution") || strings.Contains(locDesc, "Zustellung") {
			hasDistribution = true
		}
	}

	if !hasBettembourg {
		t.Errorf("expected an event referencing Centre Postal Bettembourg in events: %+v", shipment.Events)
	}
	if !hasDistribution {
		t.Errorf("expected an event referencing local distribution in events: %+v", shipment.Events)
	}

	if shipment.Status == "" {
		t.Errorf("expected non-empty status")
	}
	if shipment.Progress <= 0 {
		t.Errorf("expected progress > 0, got %d", shipment.Progress)
	}
}

func TestLuxembourgPostCarrier_Track_DestinationOptions(t *testing.T) {
	c := NewLuxembourgPostCarrier()
	ctx := context.Background()

	testCases := []struct {
		destCode     string
		expectedName string
	}{
		{"", "Luxembourg"},
		{"LU", "Luxembourg"},
		{"BE", "Belgium"},
		{"NL", "Netherlands"},
		{"FR", "France"},
		{"DE", "Germany"},
		{"ES", "Spain"},
		{"US", "US"},
	}

	for _, tc := range testCases {
		opts := models.TrackOptions{DestinationCountry: tc.destCode}
		shipment, err := c.Track(ctx, "LU987654321LU", opts)
		if err != nil {
			t.Fatalf("unexpected error for dest %s: %v", tc.destCode, err)
		}
		if tc.destCode != "" && shipment.Destination.Code != strings.ToUpper(tc.destCode) {
			t.Errorf("expected destination code %s, got %s", strings.ToUpper(tc.destCode), shipment.Destination.Code)
		}
		if shipment.Destination.Name != tc.expectedName {
			t.Errorf("expected destination name %s, got %s", tc.expectedName, shipment.Destination.Name)
		}
	}
}

func TestLuxembourgPostRegistryAndAliases(t *testing.T) {
	reg := GetRegistry()

	// Verify retrieval by primary slug
	c, ok := reg.Get("luxembourg-post")
	if !ok || c == nil {
		t.Fatalf("expected carrier 'luxembourg-post' in registry")
	}
	if c.Slug() != "luxembourg-post" {
		t.Errorf("expected slug 'luxembourg-post', got '%s'", c.Slug())
	}

	// Verify retrieval by alias
	aliasCarrier, ok := reg.Get("post-luxembourg")
	if !ok || aliasCarrier == nil {
		t.Fatalf("expected carrier by alias 'post-luxembourg'")
	}
	if aliasCarrier.Slug() != "luxembourg-post" {
		t.Errorf("expected slug 'luxembourg-post' for alias 'post-luxembourg', got '%s'", aliasCarrier.Slug())
	}

	// Case-insensitivity check
	upperCarrier, ok := reg.Get("LUXEMBOURG-POST")
	if !ok || upperCarrier == nil {
		t.Fatalf("expected carrier by uppercase 'LUXEMBOURG-POST'")
	}

	upperAlias, ok := reg.Get("POST-LUXEMBOURG")
	if !ok || upperAlias == nil {
		t.Fatalf("expected carrier by uppercase alias 'POST-LUXEMBOURG'")
	}

	// GetAlias directly
	directAlias, ok := reg.GetAlias("post-luxembourg")
	if !ok || directAlias == nil {
		t.Fatalf("expected carrier via GetAlias('post-luxembourg')")
	}

	// Verify carrier appears in registry listing
	results := reg.List("luxembourg")
	if len(results) == 0 {
		t.Errorf("expected 'luxembourg-post' in reg.List('luxembourg'), got 0 results")
	}
}
