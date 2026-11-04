// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

func TestBpostCarrier_Interface(t *testing.T) {
	c := NewBpostCarrier()
	var _ Carrier = c
	var _ CarrierWithAliases = c

	if c.Slug() != "bpost" {
		t.Errorf("expected slug 'bpost', got '%s'", c.Slug())
	}
	if !strings.EqualFold(c.Name(), "Bpost") {
		t.Errorf("expected name to be Bpost, got '%s'", c.Name())
	}
	if c.Country() != "BE" {
		t.Errorf("expected country 'BE', got '%s'", c.Country())
	}
	if c.CountryName() != "Belgium" {
		t.Errorf("expected country name 'Belgium', got '%s'", c.CountryName())
	}
	if c.Type() != models.CarrierTypePostal {
		t.Errorf("expected type '%s', got '%s'", models.CarrierTypePostal, c.Type())
	}
	if c.Website() != "https://www.bpost.be" {
		t.Errorf("expected website 'https://www.bpost.be', got '%s'", c.Website())
	}

	trackingURL := c.TrackingURL("323212345678901234567890")
	expectedURL := "https://track.bpost.cloud/btr/web/#/search?itemCode=323212345678901234567890"
	if trackingURL != expectedURL {
		t.Errorf("expected tracking URL '%s', got '%s'", expectedURL, trackingURL)
	}

	info := c.Info()
	if info.Slug != "bpost" || info.Country != "BE" || info.CountryName != "Belgium" {
		t.Errorf("unexpected carrier info: %+v", info)
	}

	aliases := c.Aliases()
	var foundLocal bool
	for _, alias := range aliases {
		if alias == "bpost-local" {
			foundLocal = true
		}
	}
	if !foundLocal {
		t.Errorf("expected aliases to contain 'bpost-local', got %v", aliases)
	}
}

func TestBpostLocalCarrier_Interface(t *testing.T) {
	c := NewBpostLocalCarrier()
	var _ Carrier = c

	if c.Slug() != "bpost-local" {
		t.Errorf("expected slug 'bpost-local', got '%s'", c.Slug())
	}
	if c.Country() != "BE" {
		t.Errorf("expected country 'BE', got '%s'", c.Country())
	}
	if c.CountryName() != "Belgium" {
		t.Errorf("expected country name 'Belgium', got '%s'", c.CountryName())
	}
	if c.Type() != models.CarrierTypePostal {
		t.Errorf("expected type '%s', got '%s'", models.CarrierTypePostal, c.Type())
	}
	if c.Website() != "https://www.bpost.be" {
		t.Errorf("expected website 'https://www.bpost.be', got '%s'", c.Website())
	}

	trackingURL := c.TrackingURL("CD123456789BE")
	expectedURL := "https://track.bpost.cloud/btr/web/#/search?itemCode=CD123456789BE"
	if trackingURL != expectedURL {
		t.Errorf("expected tracking URL '%s', got '%s'", expectedURL, trackingURL)
	}

	info := c.Info()
	if info.Slug != "bpost-local" || info.Country != "BE" || info.CountryName != "Belgium" {
		t.Errorf("unexpected carrier info: %+v", info)
	}
}

func TestBpostCarrier_Track_Events(t *testing.T) {
	c := NewBpostCarrier()
	ctx := context.Background()
	trackingNumber := "323212345678901234567890"

	shipment, err := c.Track(ctx, trackingNumber, models.TrackOptions{Language: "nl"})
	if err != nil {
		t.Fatalf("unexpected error tracking bpost: %v", err)
	}

	if shipment.TrackingNumber != trackingNumber {
		t.Errorf("expected tracking number %s, got %s", trackingNumber, shipment.TrackingNumber)
	}
	if shipment.Carrier.Slug != "bpost" {
		t.Errorf("expected carrier slug bpost, got %s", shipment.Carrier.Slug)
	}
	if shipment.Origin.Code != "BE" || shipment.Origin.Name != "Belgium" {
		t.Errorf("expected origin BE/Belgium, got %+v", shipment.Origin)
	}
	if shipment.Destination.Code != "BE" || shipment.Destination.Name != "Belgium" {
		t.Errorf("expected destination BE/Belgium, got %+v", shipment.Destination)
	}
	if len(shipment.Events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(shipment.Events))
	}

	var hasBrusselX, hasAntwerpen, hasGent bool
	var hasSorting, hasDelivery bool

	for _, ev := range shipment.Events {
		locLower := strings.ToLower(ev.Location)
		if strings.Contains(locLower, "brussel x") || strings.Contains(locLower, "bruxelles x") {
			hasBrusselX = true
		}
		if strings.Contains(locLower, "antwerpen") || strings.Contains(locLower, "anvers") {
			hasAntwerpen = true
		}
		if strings.Contains(locLower, "gent") || strings.Contains(locLower, "gand") {
			hasGent = true
		}

		descLower := strings.ToLower(ev.Description)
		if strings.Contains(descLower, "sorteer") || strings.Contains(descLower, "tri") || strings.Contains(descLower, "sorted") || ev.Stage == models.StageTransit || ev.Stage == models.StagePickedUp {
			hasSorting = true
		}
		if strings.Contains(descLower, "bezorg") || strings.Contains(descLower, "livraison") || strings.Contains(descLower, "distribution") || strings.Contains(descLower, "delivery") || ev.Status == models.StatusOutForDelivery || ev.Stage == models.StageLastMile {
			hasDelivery = true
		}
	}

	if !hasBrusselX {
		t.Errorf("expected events to include Brussel X sorting center")
	}
	if !hasAntwerpen {
		t.Errorf("expected events to include Antwerpen sorting center")
	}
	if !hasGent {
		t.Errorf("expected events to include Gent sorting center")
	}
	if !hasSorting {
		t.Errorf("expected sorting/tri events in chronology")
	}
	if !hasDelivery {
		t.Errorf("expected delivery/livraison events in chronology")
	}
}

func TestBpostCarrier_Track_DestinationOptions(t *testing.T) {
	c := NewBpostCarrier()
	ctx := context.Background()

	// Destination ES
	sES, err := c.Track(ctx, "CD123456789BE", models.TrackOptions{DestinationCountry: "ES"})
	if err != nil || sES.Destination.Code != "ES" || sES.Destination.Name != "Spain" {
		t.Errorf("expected destination ES/Spain, got %+v (err: %v)", sES.Destination, err)
	}

	// Destination NL
	sNL, err := c.Track(ctx, "CD123456789BE", models.TrackOptions{DestinationCountry: "NL"})
	if err != nil || sNL.Destination.Code != "NL" || sNL.Destination.Name != "Netherlands" {
		t.Errorf("expected destination NL/Netherlands, got %+v (err: %v)", sNL.Destination, err)
	}

	// Destination DE
	sDE, err := c.Track(ctx, "CD123456789BE", models.TrackOptions{DestinationCountry: "DE"})
	if err != nil || sDE.Destination.Code != "DE" || sDE.Destination.Name != "DE" {
		t.Errorf("expected destination DE/DE, got %+v (err: %v)", sDE.Destination, err)
	}
}

func TestBpostLocalCarrier_Track_Events(t *testing.T) {
	c := NewBpostLocalCarrier()
	ctx := context.Background()
	trackingNumber := "218212345678901234567890"

	shipment, err := c.Track(ctx, trackingNumber, models.TrackOptions{Language: "fr"})
	if err != nil {
		t.Fatalf("unexpected error tracking bpost-local: %v", err)
	}

	if shipment.TrackingNumber != trackingNumber {
		t.Errorf("expected tracking number %s, got %s", trackingNumber, shipment.TrackingNumber)
	}
	if shipment.Carrier.Slug != "bpost-local" {
		t.Errorf("expected carrier slug bpost-local, got %s", shipment.Carrier.Slug)
	}
	if shipment.Origin.Code != "BE" || shipment.Origin.Name != "Belgium" {
		t.Errorf("expected origin BE/Belgium, got %+v", shipment.Origin)
	}
	if len(shipment.Events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(shipment.Events))
	}

	var hasBrusselX, hasAntwerpen, hasGent bool
	for _, ev := range shipment.Events {
		locLower := strings.ToLower(ev.Location)
		if strings.Contains(locLower, "brussel x") || strings.Contains(locLower, "bruxelles x") {
			hasBrusselX = true
		}
		if strings.Contains(locLower, "antwerpen") || strings.Contains(locLower, "anvers") {
			hasAntwerpen = true
		}
		if strings.Contains(locLower, "gent") || strings.Contains(locLower, "gand") {
			hasGent = true
		}
	}

	if !hasBrusselX {
		t.Errorf("expected events to include Brussel X sorting center")
	}
	if !hasAntwerpen {
		t.Errorf("expected events to include Antwerpen sorting center")
	}
	if !hasGent {
		t.Errorf("expected events to include Gent sorting center")
	}

	// Destination option test
	sES, err := c.Track(ctx, trackingNumber, models.TrackOptions{DestinationCountry: "ES"})
	if err != nil || sES.Destination.Code != "ES" || sES.Destination.Name != "Spain" {
		t.Errorf("expected destination ES/Spain, got %+v (err: %v)", sES.Destination, err)
	}
	sBE, err := c.Track(ctx, trackingNumber, models.TrackOptions{DestinationCountry: "BE"})
	if err != nil || sBE.Destination.Code != "BE" || sBE.Destination.Name != "Belgium" {
		t.Errorf("expected destination BE/Belgium, got %+v (err: %v)", sBE.Destination, err)
	}
	sOther, err := c.Track(ctx, trackingNumber, models.TrackOptions{DestinationCountry: "FR"})
	if err != nil || sOther.Destination.Code != "FR" || sOther.Destination.Name != "FR" {
		t.Errorf("expected destination FR/FR, got %+v (err: %v)", sOther.Destination, err)
	}
}

func TestBpostRegistryAndAliases(t *testing.T) {
	reg := GetRegistry()

	bpost, ok := reg.Get("bpost")
	if !ok || bpost == nil {
		t.Fatalf("expected 'bpost' to be found in registry")
	}
	if bpost.Slug() != "bpost" {
		t.Errorf("expected slug 'bpost', got '%s'", bpost.Slug())
	}

	// Check bpost-local via Get
	bpostLocal, ok := reg.Get("bpost-local")
	if !ok || bpostLocal == nil {
		t.Fatalf("expected 'bpost-local' to be found in registry")
	}

	// Check bpost-local via GetAlias
	aliasC, ok := reg.GetAlias("bpost-local")
	if !ok || aliasC == nil {
		t.Errorf("expected GetAlias('bpost-local') to find carrier")
	}

	// Check aliases map contains bpost-local
	aliases := reg.Aliases()
	if _, ok := aliases["bpost-local"]; !ok {
		t.Errorf("expected 'bpost-local' in reg.Aliases()")
	}

	// Check List filtering
	results := reg.List("bpost")
	if len(results) == 0 {
		t.Fatalf("expected List('bpost') to return at least 1 result")
	}

	beResults := reg.List("Belgium")
	if len(beResults) == 0 {
		t.Fatalf("expected List('Belgium') to return at least 1 result")
	}
}
