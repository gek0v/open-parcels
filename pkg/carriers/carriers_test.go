// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestRegistry(t *testing.T) {
	reg := GetRegistry()
	list := reg.List("")
	if len(list) < 15 {
		t.Fatalf("expected at least 15 registered carriers, got %d", len(list))
	}

	// Verify Portuguese carriers are included in List("")
	expectedSlugs := []string{
		"portugal-post",
		"dpd-portugal",
		"vasp-expresso",
		"delnext",
	}
	foundSlugs := make(map[string]bool)
	for _, c := range list {
		foundSlugs[c.Slug] = true
	}
	for _, slug := range expectedSlugs {
		if !foundSlugs[slug] {
			t.Errorf("expected List(\"\") to include Portuguese carrier %q", slug)
		}
	}

	// Verify Get('portugal-post') returns CTT Portugal
	ptPost, ok := reg.Get("portugal-post")
	if !ok || ptPost == nil {
		t.Fatalf("expected portugal-post carrier to be found")
	}
	if ptPost.Slug() != "portugal-post" || ptPost.Name() != "CTT Portugal" {
		t.Errorf("expected CTT Portugal, got slug=%s name=%s", ptPost.Slug(), ptPost.Name())
	}

	// Verify alias 'ctt-portugal' resolves to CTT Portugal
	cttPT, ok := reg.Get("ctt-portugal")
	if !ok || cttPT == nil {
		t.Fatalf("expected ctt-portugal alias to be found")
	}
	if cttPT.Slug() != "portugal-post" || cttPT.Name() != "CTT Portugal" {
		t.Errorf("expected ctt-portugal to resolve to CTT Portugal, got slug=%s name=%s", cttPT.Slug(), cttPT.Name())
	}

	// Verify alias 'ctt' resolves to CTT Portugal
	ctt, ok := reg.Get("ctt")
	if !ok || ctt == nil {
		t.Fatalf("expected ctt alias to be found and resolve to portugal-post")
	}
	if ctt.Slug() != "portugal-post" || ctt.Name() != "CTT Portugal" {
		t.Errorf("expected ctt to resolve to CTT Portugal, got slug=%s name=%s", ctt.Slug(), ctt.Name())
	}

	// Verify other Portuguese carriers are retrievable by slug
	for _, slug := range []string{"dpd-portugal", "vasp-expresso", "delnext"} {
		c, ok := reg.Get(slug)
		if !ok || c == nil {
			t.Errorf("expected %q carrier to be found in registry", slug)
		} else if c.Slug() != slug {
			t.Errorf("expected slug %q, got %q", slug, c.Slug())
		}
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

func TestAllCarriersTrack(t *testing.T) {
	reg := GetRegistry()
	ctx := context.Background()

	for slug, c := range reg.AllCarriers() {
		if slug == "cainiao" {
			continue // Tested above
		}
		t.Run(slug, func(t *testing.T) {
			shipment, err := c.Track(ctx, "TRACK123456", models.TrackOptions{Language: "en"})
			if err != nil {
				t.Fatalf("unexpected error tracking %s: %v", slug, err)
			}
			if shipment == nil {
				t.Fatalf("expected non-nil shipment for %s", slug)
			}
			if shipment.TrackingNumber != "TRACK123456" {
				t.Errorf("expected tracking number TRACK123456, got %s", shipment.TrackingNumber)
			}
			if len(shipment.Events) == 0 {
				t.Errorf("expected tracking events for %s", slug)
			}
		})
	}
}
