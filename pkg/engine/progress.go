package engine

import (
	"time"

	"open-parcels/pkg/models"
)

// CalculateProgress estimates the overall journey completion percentage (0 - 100%).
func CalculateProgress(status models.Status, events []models.TrackingEvent) int {
	switch status {
	case models.StatusDelivered:
		return 100
	case models.StatusOutForDelivery:
		return 85
	case models.StatusFailedAttempt:
		return 80
	case models.StatusInfoReceived:
		return 15
	case models.StatusPending:
		return 5
	case models.StatusException:
		return 60
	case models.StatusInTransit:
		// Analyze stages present in events to provide refined granular progress
		hasHandover := false
		hasCustoms := false
		hasDepartedOrigin := false

		for _, ev := range events {
			switch ev.Stage {
			case models.StageHandover:
				hasHandover = true
			case models.StageCustoms:
				hasCustoms = true
			case models.StageTransit, models.StagePickedUp:
				hasDepartedOrigin = true
			}
		}

		if hasHandover {
			return 75
		}
		if hasCustoms {
			return 55
		}
		if hasDepartedOrigin {
			return 35
		}
		return 25
	default:
		return 30
	}
}

// CalculateDaysInTransit computes total days elapsed since the earliest event.
func CalculateDaysInTransit(events []models.TrackingEvent, deliveredAt *time.Time) int {
	if len(events) == 0 {
		return 0
	}

	earliest := events[0].Date
	for _, ev := range events {
		if !ev.Date.IsZero() && ev.Date.Before(earliest) {
			earliest = ev.Date
		}
	}

	endTime := time.Now()
	if deliveredAt != nil && !deliveredAt.IsZero() {
		endTime = *deliveredAt
	}

	diff := endTime.Sub(earliest)
	days := int(diff.Hours() / 24)
	if days < 1 && diff > 0 {
		return 1
	}
	return days
}
