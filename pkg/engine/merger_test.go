// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestMergeEventsDeduplicationAndOrder(t *testing.T) {
	now := time.Now()

	primary := []models.TrackingEvent{
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "Departed sorting facility",
			CarrierSlug: "carrier-a",
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "Gate in full at origin port terminal",
			CarrierSlug: "carrier-a",
		},
	}

	secondary := []models.TrackingEvent{
		{
			// Duplicate of primary[0] within 30 minutes, same carrier
			Date:        now.Add(-90 * time.Minute),
			Description: "Departed sorting facility",
			CarrierSlug: "carrier-a",
		},
		{
			// Not a duplicate (different carrier)
			Date:        now.Add(-1 * time.Hour),
			Description: "Package in transit",
			CarrierSlug: "carrier-b",
		},
	}

	merged := MergeEvents(primary, secondary)
	if len(merged) != 3 {
		t.Fatalf("expected 3 merged events after deduplicating 1, got %d", len(merged))
	}

	// Verify chronological descending order (newest first)
	for i := 0; i < len(merged)-1; i++ {
		if merged[i].Date.Before(merged[i+1].Date) {
			t.Errorf("merged events not descending at index %d", i)
		}
	}
}

func TestDeriveOverallStatus(t *testing.T) {
	now := time.Now()

	// Empty events
	if s := DeriveOverallStatus(nil); s != models.StatusPending {
		t.Errorf("expected pending for empty, got %s", s)
	}

	// Delivered anywhere in history
	deliveredEvents := []models.TrackingEvent{
		{Date: now, Status: models.StatusInTransit},
		{Date: now.Add(-1 * time.Hour), Status: models.StatusDelivered},
	}
	if s := DeriveOverallStatus(deliveredEvents); s != models.StatusDelivered {
		t.Errorf("expected delivered, got %s", s)
	}

	// Out for delivery as latest
	ofdEvents := []models.TrackingEvent{
		{Date: now, Status: models.StatusOutForDelivery},
		{Date: now.Add(-1 * time.Hour), Status: models.StatusInTransit},
	}
	if s := DeriveOverallStatus(ofdEvents); s != models.StatusOutForDelivery {
		t.Errorf("expected out_for_delivery, got %s", s)
	}

	// Failed attempt as latest
	failedEvents := []models.TrackingEvent{
		{Date: now, Status: models.StatusFailedAttempt},
		{Date: now.Add(-1 * time.Hour), Status: models.StatusInTransit},
	}
	if s := DeriveOverallStatus(failedEvents); s != models.StatusFailedAttempt {
		t.Errorf("expected failed_attempt, got %s", s)
	}

	// Exception as latest
	exEvents := []models.TrackingEvent{
		{Date: now, Status: models.StatusException},
		{Date: now.Add(-1 * time.Hour), Status: models.StatusInTransit},
	}
	if s := DeriveOverallStatus(exEvents); s != models.StatusException {
		t.Errorf("expected exception, got %s", s)
	}

	// In transit as latest
	itEvents := []models.TrackingEvent{
		{Date: now, Status: models.StatusInTransit},
	}
	if s := DeriveOverallStatus(itEvents); s != models.StatusInTransit {
		t.Errorf("expected in_transit, got %s", s)
	}

	// Latest with empty status
	emptyStatEvents := []models.TrackingEvent{
		{Date: now, Status: ""},
	}
	if s := DeriveOverallStatus(emptyStatEvents); s != models.StatusInTransit {
		t.Errorf("expected in_transit fallback, got %s", s)
	}
}
