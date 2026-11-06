package engine

import (
	"sort"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

// MergeEvents combines events from multiple carriers into a clean, deduplicated, chronologically sorted timeline.
func MergeEvents(primaryEvents []models.TrackingEvent, secondaryEvents []models.TrackingEvent) []models.TrackingEvent {
	all := make([]models.TrackingEvent, 0, len(primaryEvents)+len(secondaryEvents))

	// Tag primary events (carrier index 0 if not already set)
	for _, ev := range primaryEvents {
		if ev.Status == "" || ev.Stage == "" {
			ev.Status, ev.Stage = NormalizeEventStatus(ev.Description)
		}
		descLower := strings.ToLower(ev.Description)
		if strings.Contains(descLower, "entregado") || strings.Contains(descLower, "delivered") {
			ev.Status = models.StatusDelivered
			ev.Stage = models.StageDelivery
		}
		all = append(all, ev)
	}

	// Tag secondary events (carrier index 1 if not already set)
	for _, ev := range secondaryEvents {
		if ev.CarrierIndex == 0 {
			ev.CarrierIndex = 1
		}
		if ev.Status == "" || ev.Stage == "" {
			ev.Status, ev.Stage = NormalizeEventStatus(ev.Description)
		}
		descLower := strings.ToLower(ev.Description)
		if strings.Contains(descLower, "entregado") || strings.Contains(descLower, "delivered") {
			ev.Status = models.StatusDelivered
			ev.Stage = models.StageDelivery
		}
		all = append(all, ev)
	}

	// If shipment has a delivered event, discard any non-delivered anomalies/mocks with timestamps after delivery
	var deliveredDate *time.Time
	for _, ev := range all {
		if ev.Status == models.StatusDelivered {
			d := ev.Date
			if deliveredDate == nil || d.After(*deliveredDate) {
				deliveredDate = &d
			}
		}
	}

	if deliveredDate != nil {
		filtered := make([]models.TrackingEvent, 0, len(all))
		for _, ev := range all {
			if ev.Date.After(*deliveredDate) && ev.Status != models.StatusDelivered {
				continue
			}
			filtered = append(filtered, ev)
		}
		all = filtered
	}

	// Sort events chronologically descending (newest first)
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Date.After(all[j].Date)
	})

	// Deduplication: remove redundant/duplicate events within a 2-hour window
	var deduplicated []models.TrackingEvent
	for i, ev := range all {
		if i == 0 {
			deduplicated = append(deduplicated, ev)
			continue
		}

		prev := deduplicated[len(deduplicated)-1]
		timeDiff := prev.Date.Sub(ev.Date)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		// Check if it's a duplicate: within 2 hours with same normalized status and stage
		isSameStatus := ev.Status == prev.Status && ev.Stage == prev.Stage
		isIdenticalText := strings.EqualFold(strings.TrimSpace(ev.Description), strings.TrimSpace(prev.Description))
		isBothDelivered := ev.Status == models.StatusDelivered && prev.Status == models.StatusDelivered

		if (isIdenticalText || isSameStatus || isBothDelivered) && timeDiff < 2*time.Hour {
			// Skip duplicate
			continue
		}

		deduplicated = append(deduplicated, ev)
	}

	return deduplicated
}

// DeriveOverallStatus determines the highest-order latest status from events.
func DeriveOverallStatus(events []models.TrackingEvent) models.Status {
	if len(events) == 0 {
		return models.StatusPending
	}

	// If any event in the history is Delivered, shipment is delivered
	for _, ev := range events {
		if ev.Status == models.StatusDelivered {
			return models.StatusDelivered
		}
		descLower := strings.ToLower(ev.Description)
		if strings.Contains(descLower, "entregado") || strings.Contains(descLower, "delivered") {
			return models.StatusDelivered
		}
	}

	latest := events[0]

	// Check if latest is Out for delivery
	if latest.Status == models.StatusOutForDelivery {
		return models.StatusOutForDelivery
	}

	// Check if latest is Failed attempt or Exception
	if latest.Status == models.StatusFailedAttempt {
		return models.StatusFailedAttempt
	}
	if latest.Status == models.StatusException {
		return models.StatusException
	}

	// If latest has in_transit or info_received
	if latest.Status != "" {
		return latest.Status
	}

	return models.StatusInTransit
}
