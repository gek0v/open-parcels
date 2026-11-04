// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

func TestRegistry(t *testing.T) {
	reg := GetRegistry()
	list := reg.List("")
	if len(list) < 15 {
		t.Fatalf("expected at least 15 registered carriers, got %d", len(list))
	}

	cainiao, ok := reg.Get("cainiao")
	if !ok || cainiao == nil {
		t.Fatalf("expected cainiao carrier to be found")
	}

	shipment, err := cainiao.Track(context.Background(), "LP00561234567890", models.TrackOptions{Language: "en"})
	if err != nil {
		t.Fatalf("unexpected error tracking cainiao: %v", err)
	}
	if shipment.TrackingNumber != "LP00561234567890" {
		t.Errorf("expected tracking number LP00561234567890, got %s", shipment.TrackingNumber)
	}
	if len(shipment.Events) == 0 {
		t.Errorf("expected tracking events for cainiao shipment")
	}
	if shipment.LastMileCarrier == nil {
		t.Errorf("expected cainiao to provide last-mile carrier information")
	}
}

func TestPostNLCarrier_Interface(t *testing.T) {
	c := NewPostNLCarrier()
	var _ Carrier = c

	if c.Slug() != "postnl" {
		t.Errorf("expected slug 'postnl', got '%s'", c.Slug())
	}
	if c.Name() != "PostNL" {
		t.Errorf("expected name 'PostNL', got '%s'", c.Name())
	}
	if c.Country() != "NL" {
		t.Errorf("expected country 'NL', got '%s'", c.Country())
	}
	if c.CountryName() != "Netherlands" {
		t.Errorf("expected country name 'Netherlands', got '%s'", c.CountryName())
	}
	if c.Type() != models.CarrierTypePostal {
		t.Errorf("expected type '%s', got '%s'", models.CarrierTypePostal, c.Type())
	}
	if c.Website() != "https://www.postnl.nl" {
		t.Errorf("expected website 'https://www.postnl.nl', got '%s'", c.Website())
	}

	trackingURL := c.TrackingURL("3SABCD1234567")
	expectedURL := "https://jouw.postnl.nl/track-and-trace/3SABCD1234567"
	if trackingURL != expectedURL {
		t.Errorf("expected tracking URL '%s', got '%s'", expectedURL, trackingURL)
	}

	info := c.Info()
	if info.Slug != "postnl" || info.Country != "NL" || info.CountryName != "Netherlands" {
		t.Errorf("unexpected carrier info: %+v", info)
	}
}

func TestPostNLInternationalCarrier_Interface(t *testing.T) {
	c := NewPostNLInternationalCarrier()
	var _ Carrier = c

	if c.Slug() != "postnl-international" {
		t.Errorf("expected slug 'postnl-international', got '%s'", c.Slug())
	}
	if c.Name() != "PostNL International" {
		t.Errorf("expected name 'PostNL International', got '%s'", c.Name())
	}
	if c.Country() != "NL" {
		t.Errorf("expected country 'NL', got '%s'", c.Country())
	}
	if c.CountryName() != "Netherlands" {
		t.Errorf("expected country name 'Netherlands', got '%s'", c.CountryName())
	}
	if c.Type() != models.CarrierTypePostal {
		t.Errorf("expected type '%s', got '%s'", models.CarrierTypePostal, c.Type())
	}
	if c.Website() != "https://www.postnl.nl" {
		t.Errorf("expected website 'https://www.postnl.nl', got '%s'", c.Website())
	}

	trackingURL := c.TrackingURL("CC123456789NL")
	expectedURL := "https://jouw.postnl.nl/track-and-trace/CC123456789NL"
	if trackingURL != expectedURL {
		t.Errorf("expected tracking URL '%s', got '%s'", expectedURL, trackingURL)
	}

	info := c.Info()
	if info.Slug != "postnl-international" || info.Country != "NL" || info.CountryName != "Netherlands" {
		t.Errorf("unexpected carrier info: %+v", info)
	}
}

func TestPostNLCarrier_Track_Events(t *testing.T) {
	c := NewPostNLCarrier()
	ctx := context.Background()
	trackingNumber := "3SABCD1234567"

	shipment, err := c.Track(ctx, trackingNumber, models.TrackOptions{Language: "nl"})
	if err != nil {
		t.Fatalf("unexpected error tracking postnl: %v", err)
	}

	if shipment.TrackingNumber != trackingNumber {
		t.Errorf("expected tracking number %s, got %s", trackingNumber, shipment.TrackingNumber)
	}
	if shipment.Carrier.Slug != "postnl" {
		t.Errorf("expected carrier slug postnl, got %s", shipment.Carrier.Slug)
	}
	if shipment.Origin.Code != "NL" || shipment.Origin.Name != "Netherlands" {
		t.Errorf("expected origin NL/Netherlands, got %+v", shipment.Origin)
	}
	if len(shipment.Events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(shipment.Events))
	}

	var hasAmsterdam, hasUtrecht, hasRotterdam bool
	var hasSorting, hasDelivery bool

	for _, ev := range shipment.Events {
		locLower := strings.ToLower(ev.Location)
		if strings.Contains(locLower, "amsterdam") || strings.Contains(locLower, "ámsterdam") {
			hasAmsterdam = true
		}
		if strings.Contains(locLower, "utrecht") {
			hasUtrecht = true
		}
		if strings.Contains(locLower, "rotterdam") {
			hasRotterdam = true
		}

		descLower := strings.ToLower(ev.Description)
		if strings.Contains(descLower, "sorteer") || strings.Contains(descLower, "clasifica") || strings.Contains(descLower, "sorted") || ev.Stage == models.StageTransit || ev.Stage == models.StagePickedUp {
			hasSorting = true
		}
		if strings.Contains(descLower, "bezorg") || strings.Contains(descLower, "reparto") || strings.Contains(descLower, "delivery") || ev.Status == models.StatusOutForDelivery || ev.Stage == models.StageLastMile {
			hasDelivery = true
		}
	}

	if !hasAmsterdam {
		t.Errorf("expected events to include Ámsterdam hub/center")
	}
	if !hasUtrecht {
		t.Errorf("expected events to include Utrecht hub/center")
	}
	if !hasRotterdam {
		t.Errorf("expected events to include Rotterdam hub/center")
	}
	if !hasSorting {
		t.Errorf("expected classification/sorting events in chronology")
	}
	if !hasDelivery {
		t.Errorf("expected delivery/reparto events in chronology")
	}
}

func TestPostNLInternationalCarrier_Track_Events(t *testing.T) {
	c := NewPostNLInternationalCarrier()
	ctx := context.Background()
	trackingNumber := "CC123456789NL"

	shipment, err := c.Track(ctx, trackingNumber, models.TrackOptions{DestinationCountry: "ES"})
	if err != nil {
		t.Fatalf("unexpected error tracking postnl international: %v", err)
	}

	if shipment.TrackingNumber != trackingNumber {
		t.Errorf("expected tracking number %s, got %s", trackingNumber, shipment.TrackingNumber)
	}
	if shipment.Carrier.Slug != "postnl-international" {
		t.Errorf("expected carrier slug postnl-international, got %s", shipment.Carrier.Slug)
	}
	if shipment.Origin.Code != "NL" || shipment.Origin.Name != "Netherlands" {
		t.Errorf("expected origin NL/Netherlands, got %+v", shipment.Origin)
	}
	if len(shipment.Events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(shipment.Events))
	}

	var hasAmsterdam, hasUtrecht, hasRotterdam bool
	for _, ev := range shipment.Events {
		locLower := strings.ToLower(ev.Location)
		if strings.Contains(locLower, "amsterdam") || strings.Contains(locLower, "ámsterdam") {
			hasAmsterdam = true
		}
		if strings.Contains(locLower, "utrecht") {
			hasUtrecht = true
		}
		if strings.Contains(locLower, "rotterdam") {
			hasRotterdam = true
		}
	}

	if !hasAmsterdam {
		t.Errorf("expected events to include Ámsterdam hub/center")
	}
	if !hasUtrecht {
		t.Errorf("expected events to include Utrecht hub/center")
	}
	if !hasRotterdam {
		t.Errorf("expected events to include Rotterdam hub/center")
	}
}

func TestPostNLRegistryAndAliases(t *testing.T) {
	reg := GetRegistry()

	postnl, ok := reg.Get("postnl")
	if !ok || postnl == nil {
		t.Fatalf("expected 'postnl' to be found in registry")
	}
	if postnl.Slug() != "postnl" {
		t.Errorf("expected slug 'postnl', got '%s'", postnl.Slug())
	}

	postnlIntl, ok := reg.Get("postnl-international")
	if !ok || postnlIntl == nil {
		t.Fatalf("expected 'postnl-international' to be found in registry")
	}

	// Test aliases
	aliasCarrier, ok := reg.Get("post-nl")
	if !ok || aliasCarrier == nil {
		t.Fatalf("expected alias 'post-nl' to resolve to a carrier")
	}
	if aliasCarrier.Country() != "NL" {
		t.Errorf("expected alias carrier country NL, got %s", aliasCarrier.Country())
	}

	// Test RegisterAlias dynamic registration
	customCarrier := NewPostNLCarrier()
	reg.RegisterAlias("postnl-custom-alias", customCarrier)
	found, ok := reg.Get("postnl-custom-alias")
	if !ok || found == nil {
		t.Fatalf("expected dynamically registered alias to resolve")
	}

	// Test List searching for PostNL
	results := reg.List("postnl")
	if len(results) == 0 {
		t.Fatalf("expected List('postnl') to return at least 1 result")
	}

	// Test GetAlias
	aliasC, ok := reg.GetAlias("post-nl")
	if !ok || aliasC == nil {
		t.Errorf("expected GetAlias('post-nl') to succeed")
	}
	_, notFound := reg.GetAlias("nonexistent-carrier-xyz")
	if notFound {
		t.Errorf("expected GetAlias for nonexistent to return false")
	}

	// Test Aliases()
	allAliases := reg.Aliases()
	if len(allAliases) == 0 {
		t.Errorf("expected at least one registered alias")
	}

	// Test AllCarriers()
	allCarriers := reg.AllCarriers()
	if len(allCarriers) < 15 {
		t.Errorf("expected at least 15 carriers in AllCarriers, got %d", len(allCarriers))
	}
}

func TestPostNLCarrier_Track_DestinationOptions(t *testing.T) {
	c := NewPostNLCarrier()
	ctx := context.Background()

	// Destination ES
	sES, err := c.Track(ctx, "3S123", models.TrackOptions{DestinationCountry: "ES"})
	if err != nil || sES.Destination.Code != "ES" || sES.Destination.Name != "Spain" {
		t.Errorf("expected destination ES/Spain, got %+v (err: %v)", sES.Destination, err)
	}

	// Destination DE
	sDE, err := c.Track(ctx, "3S123", models.TrackOptions{DestinationCountry: "DE"})
	if err != nil || sDE.Destination.Code != "DE" || sDE.Destination.Name != "DE" {
		t.Errorf("expected destination DE/DE, got %+v (err: %v)", sDE.Destination, err)
	}

	// Aliases check
	aliases := c.Aliases()
	if len(aliases) == 0 {
		t.Errorf("expected PostNLCarrier aliases to be non-empty")
	}
}

func TestPostNLInternationalCarrier_Track_DestinationOptions(t *testing.T) {
	c := NewPostNLInternationalCarrier()
	ctx := context.Background()

	// Default destination (empty)
	sDef, err := c.Track(ctx, "CC123NL", models.TrackOptions{})
	if err != nil || sDef.Destination.Code != "GLOBAL" || sDef.Destination.Name != "International" {
		t.Errorf("expected default destination GLOBAL/International, got %+v (err: %v)", sDef.Destination, err)
	}

	// Destination NL
	sNL, err := c.Track(ctx, "CC123NL", models.TrackOptions{DestinationCountry: "NL"})
	if err != nil || sNL.Destination.Code != "NL" || sNL.Destination.Name != "Netherlands" {
		t.Errorf("expected destination NL/Netherlands, got %+v (err: %v)", sNL.Destination, err)
	}

	// Destination DE
	sDE, err := c.Track(ctx, "CC123NL", models.TrackOptions{DestinationCountry: "DE"})
	if err != nil || sDE.Destination.Code != "DE" || sDE.Destination.Name != "DE" {
		t.Errorf("expected destination DE/DE, got %+v (err: %v)", sDE.Destination, err)
	}

	// Aliases check
	aliases := c.Aliases()
	if len(aliases) == 0 {
		t.Errorf("expected PostNLInternationalCarrier aliases to be non-empty")
	}
}

func TestBaseCarrier_EmptyTrackingURL(t *testing.T) {
	bc := NewBaseCarrier("test", "Test", "NL", "Netherlands", models.CarrierTypePostal, "https://test.com", "")
	if bc.TrackingURL("123") != "" {
		t.Errorf("expected empty string for empty urlTemplate")
	}
}

func TestRegistry_NilAliases(t *testing.T) {
	reg := &Registry{
		carriers: make(map[string]Carrier),
	}
	if _, ok := reg.GetAlias("test"); ok {
		t.Errorf("expected false for uninitialized aliases")
	}

	c := NewPostNLCarrier()
	reg.RegisterAlias("nl", c)
	if found, ok := reg.GetAlias("nl"); !ok || found == nil {
		t.Errorf("expected to find registered alias in fresh registry")
	}

	reg2 := &Registry{
		carriers: make(map[string]Carrier),
	}
	reg2.Register(c)
	if found, ok := reg2.Get("postnl"); !ok || found == nil {
		t.Errorf("expected to find registered carrier in fresh registry")
	}
}

func TestAllCarriers_Track(t *testing.T) {
	reg := GetRegistry()
	for slug, carrier := range reg.AllCarriers() {
		t.Run(slug, func(t *testing.T) {
			ctx := context.Background()
			res, err := carrier.Track(ctx, "TRACK123456", models.TrackOptions{Language: "en"})
			if err != nil {
				t.Fatalf("unexpected error tracking %s: %v", slug, err)
			}
			if res == nil {
				t.Fatalf("expected non-nil shipment for %s", slug)
			}
			if res.TrackingNumber != "TRACK123456" {
				t.Errorf("expected tracking number TRACK123456, got %s", res.TrackingNumber)
			}
		})
	}
}
