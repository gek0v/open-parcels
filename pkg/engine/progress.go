// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"strings"
	"time"

	"open-parcels/pkg/models"
)

// CalculateProgress estimates the overall journey completion percentage (0 - 100%).
func CalculateProgress(status models.Status, events []models.TrackingEvent) int {
	switch status {
	case models.StatusDelivered:
		return 100
	case models.StatusExpired:
		return 100
	case models.StatusInfoReceived:
		return 15
	case models.StatusPending:
		return 5
	case models.StatusException:
		return 60
	case models.StatusFailedAttempt:
		return 80
	case models.StatusOutForDelivery:
		// Check if maritime gate-out milestone is present
		if detectMaritimeProgress(events) >= 90 {
			return 90
		}
		return 85
	case models.StatusInTransit:
		// Check if maritime milestones are present
		if mProg := detectMaritimeProgress(events); mProg > 0 {
			return mProg
		}

		// Analyze stages present in events for standard parcels
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

// detectMaritimeProgress inspects events for maritime freight milestones and returns highest progress (20-100).
func detectMaritimeProgress(events []models.TrackingEvent) int {
	highest := 0

	for _, ev := range events {
		code := strings.ToUpper(ev.EventCode)
		d := strings.ToLower(ev.Description)

		switch {
		case code == "EMPTY_RETURN" || strings.Contains(d, "empty return") || strings.Contains(d, "empty returned") ||
			strings.Contains(d, "returned empty") || strings.Contains(d, "empty container returned") ||
			strings.Contains(d, "devolución de vacío") || strings.Contains(d, "retorno de vacío"):
			if highest < 100 {
				highest = 100
			}

		case code == "GATE_OUT" || strings.Contains(d, "gate-out") || strings.Contains(d, "gate out") ||
			strings.Contains(d, "puerta de salida") || strings.Contains(d, "salida de terminal"):
			if highest < 90 {
				highest = 90
			}

		case code == "CUSTOMS" || (strings.Contains(d, "customs") && (strings.Contains(d, "clear") || strings.Contains(d, "release") || strings.Contains(d, "despacho"))) ||
			strings.Contains(d, "despacho de aduanas"):
			if highest < 80 {
				highest = 80
			}

		case code == "DISCHARGED" || strings.Contains(d, "discharged") || strings.Contains(d, "descargado"):
			if highest < 70 {
				highest = 70
			}

		case code == "TRANSSHIPMENT" || strings.Contains(d, "transshipment") || strings.Contains(d, "transbordo") ||
			strings.Contains(d, "high seas"):
			if highest < 60 {
				highest = 60
			}

		case code == "DEPARTED" || code == "UNDERWAY" || strings.Contains(d, "vessel departed") ||
			strings.Contains(d, "underway") || strings.Contains(d, "zarpó"):
			if highest < 50 {
				highest = 50
			}

		case code == "LOADED" || strings.Contains(d, "loaded on board") || strings.Contains(d, "laden on board") ||
			strings.Contains(d, "cargado a bordo") || strings.Contains(d, "loaded onto vessel") ||
			strings.Contains(d, "on board"):
			if highest < 40 {
				highest = 40
			}

		case code == "GATE_IN" || strings.Contains(d, "gate-in") || strings.Contains(d, "gate in") ||
			strings.Contains(d, "ingreso a terminal") || strings.Contains(d, "puerta de entrada"):
			if highest < 20 {
				highest = 20
			}
		}
	}

	return highest
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
