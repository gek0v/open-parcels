// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

// Helper to validate UPU S10 Modulo 11 check digit
func checkUPUS10(num string) bool {
	if len(num) != 13 {
		return false
	}
	weights := []int{8, 6, 4, 2, 3, 5, 9, 7}
	sum := 0
	for i := 0; i < 8; i++ {
		d := int(num[2+i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		sum += d * weights[i]
	}
	rem := sum % 11
	chk := 11 - rem
	if chk == 10 {
		chk = 0
	} else if chk == 11 {
		chk = 5
	}
	return chk == int(num[10]-'0')
}

func TestCainiaoPortugalHandover(t *testing.T) {
	c := NewCainiaoCarrier()
	ctx := context.Background()

	// 1. Destination PT should assign portugal-post instead of Spanish operator
	shipment, err := c.Track(ctx, "LP00561234567890", models.TrackOptions{
		DestinationCountry: "PT",
		Language:           "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking cainiao with PT destination: %v", err)
	}

	if shipment.Destination.Code != "PT" {
		t.Errorf("expected destination code PT, got %s", shipment.Destination.Code)
	}
	if shipment.Destination.Name != "Portugal" {
		t.Errorf("expected destination name Portugal, got %s", shipment.Destination.Name)
	}

	if shipment.LastMileCarrier == nil {
		t.Fatalf("expected last-mile carrier to be set for PT destination")
	}

	// Must NOT assign Spanish operators
	if shipment.LastMileCarrier.Slug == "correos-express" || shipment.LastMileCarrier.Slug == "correos-spain" {
		t.Errorf("expected portugal carrier, got Spanish carrier %s", shipment.LastMileCarrier.Slug)
	}

	// Must assign portugal-post
	if shipment.LastMileCarrier.Slug != "portugal-post" {
		t.Errorf("expected LastMileCarrier portugal-post, got %s", shipment.LastMileCarrier.Slug)
	}

	// Last-mile tracking number must follow valid UPU S10 PT format (e.g. RR...PT)
	upuRegex := regexp.MustCompile(`^[A-Z]{2}\d{9}PT$`)
	if !upuRegex.MatchString(shipment.LastMileTrackingNumber) {
		t.Errorf("expected UPU S10 PT tracking format, got %s", shipment.LastMileTrackingNumber)
	}
	if !checkUPUS10(shipment.LastMileTrackingNumber) {
		t.Errorf("expected valid UPU S10 check digit for %s", shipment.LastMileTrackingNumber)
	}

	// Milestones in Lisboa and Oporto
	hasLisboa := false
	hasOporto := false
	for _, ev := range shipment.Events {
		loc := strings.ToLower(ev.Location)
		if strings.Contains(loc, "lisboa") || strings.Contains(loc, "lisbon") {
			hasLisboa = true
		}
		if strings.Contains(loc, "porto") || strings.Contains(loc, "oporto") {
			hasOporto = true
		}
	}
	if !hasLisboa {
		t.Errorf("expected tracking events to include Lisboa milestone")
	}
	if !hasOporto {
		t.Errorf("expected tracking events to include Oporto milestone")
	}

	// Case-insensitivity check (e.g. "pt")
	shipmentLower, err := c.Track(ctx, "LP00561234567890", models.TrackOptions{
		DestinationCountry: "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking cainiao with lower 'pt': %v", err)
	}
	if shipmentLower.LastMileCarrier == nil || shipmentLower.LastMileCarrier.Slug != "portugal-post" {
		t.Errorf("expected portugal-post for lowercase 'pt', got %+v", shipmentLower.LastMileCarrier)
	}

	// Carrier hint override to dpd-portugal
	shipmentDPD, err := c.Track(ctx, "LP00561234567890", models.TrackOptions{
		DestinationCountry: "PT",
		CarrierHint:        "dpd-portugal",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking cainiao with dpd hint: %v", err)
	}
	if shipmentDPD.LastMileCarrier == nil || shipmentDPD.LastMileCarrier.Slug != "dpd-portugal" {
		t.Errorf("expected dpd-portugal when hinted, got %+v", shipmentDPD.LastMileCarrier)
	}

	// Regression: ES destination must continue assigning Correos Express
	shipmentES, err := c.Track(ctx, "LP00561234567890", models.TrackOptions{
		DestinationCountry: "ES",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking cainiao with ES: %v", err)
	}
	if shipmentES.LastMileCarrier == nil || shipmentES.LastMileCarrier.Slug != "correos-express" {
		t.Errorf("expected correos-express for ES, got %+v", shipmentES.LastMileCarrier)
	}
}

func TestYunExpressPortugalHandover(t *testing.T) {
	y := NewYunExpressCarrier()
	ctx := context.Background()

	// 1. Destination PT should assign portuguese carrier (dpd-portugal or portugal-post)
	shipment, err := y.Track(ctx, "YT2401234567890123", models.TrackOptions{
		DestinationCountry: "PT",
		Language:           "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking yunexpress with PT destination: %v", err)
	}

	if shipment.Destination.Code != "PT" {
		t.Errorf("expected destination code PT, got %s", shipment.Destination.Code)
	}
	if shipment.Destination.Name != "Portugal" {
		t.Errorf("expected destination name Portugal, got %s", shipment.Destination.Name)
	}

	if shipment.LastMileCarrier == nil {
		t.Fatalf("expected last-mile carrier to be set for PT destination")
	}

	// Must NOT assign Spanish operators
	if shipment.LastMileCarrier.Slug == "correos-express" || shipment.LastMileCarrier.Slug == "correos-spain" {
		t.Errorf("expected portugal carrier, got Spanish carrier %s", shipment.LastMileCarrier.Slug)
	}

	// Must assign dpd-portugal (or portugal-post)
	if shipment.LastMileCarrier.Slug != "dpd-portugal" && shipment.LastMileCarrier.Slug != "portugal-post" {
		t.Errorf("expected dpd-portugal or portugal-post, got %s", shipment.LastMileCarrier.Slug)
	}

	// Milestones in Lisboa and Oporto
	hasLisboa := false
	hasOporto := false
	for _, ev := range shipment.Events {
		loc := strings.ToLower(ev.Location)
		if strings.Contains(loc, "lisboa") || strings.Contains(loc, "lisbon") {
			hasLisboa = true
		}
		if strings.Contains(loc, "porto") || strings.Contains(loc, "oporto") {
			hasOporto = true
		}
	}
	if !hasLisboa {
		t.Errorf("expected tracking events to include Lisboa milestone")
	}
	if !hasOporto {
		t.Errorf("expected tracking events to include Oporto milestone")
	}

	// Case-insensitivity check
	shipmentLower, err := y.Track(ctx, "YT2401234567890123", models.TrackOptions{
		DestinationCountry: "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking yunexpress with lower 'pt': %v", err)
	}
	if shipmentLower.LastMileCarrier == nil || shipmentLower.LastMileCarrier.Slug != "dpd-portugal" {
		t.Errorf("expected dpd-portugal for lowercase 'pt', got %+v", shipmentLower.LastMileCarrier)
	}

	// Carrier hint override to portugal-post with UPU S10 format
	shipmentPTPost, err := y.Track(ctx, "YT2401234567890123", models.TrackOptions{
		DestinationCountry: "PT",
		CarrierHint:        "portugal-post",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking yunexpress with portugal-post hint: %v", err)
	}
	if shipmentPTPost.LastMileCarrier == nil || shipmentPTPost.LastMileCarrier.Slug != "portugal-post" {
		t.Errorf("expected portugal-post when hinted, got %+v", shipmentPTPost.LastMileCarrier)
	}
	if !checkUPUS10(shipmentPTPost.LastMileTrackingNumber) {
		t.Errorf("expected valid UPU S10 check digit for %s", shipmentPTPost.LastMileTrackingNumber)
	}

	// Regression: ES destination must continue assigning Correos Spain
	shipmentES, err := y.Track(ctx, "YT2401234567890123", models.TrackOptions{
		DestinationCountry: "ES",
	})
	if err != nil {
		t.Fatalf("unexpected error tracking yunexpress with ES: %v", err)
	}
	if shipmentES.LastMileCarrier == nil || shipmentES.LastMileCarrier.Slug != "correos-spain" {
		t.Errorf("expected correos-spain for ES, got %+v", shipmentES.LastMileCarrier)
	}
}
