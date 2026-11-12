package engine

import (
	"sort"
	"strings"
	"time"

	"open-parcels/pkg/carriers"
	"open-parcels/pkg/models"
)

// NormalizeEventTime converts a timestamp representing local clock time in zoneOrCountry into its exact UTC instant.
func NormalizeEventTime(t time.Time, zoneOrCountry string) time.Time {
	return carriers.NormalizeEventTime(t, zoneOrCountry)
}

// ParseLocalTimeWithZone parses a date string in the given timezone or country and returns the UTC instant.
func ParseLocalTimeWithZone(layout, value, zoneOrCountry string) (time.Time, error) {
	return carriers.ParseLocalTimeWithZone(layout, value, zoneOrCountry)
}

// NormalizeTrackingEvent ensures that the event date is normalized to UTC based on
// its location, carrier slug, raw date, or fallback timezone, without double-shifting.
func NormalizeTrackingEvent(ev *models.TrackingEvent, fallbackZone string) {
	if ev.Date.IsZero() && ev.RawDate == "" {
		return
	}

	zone := resolveEventZone(*ev, fallbackZone)

	if ev.RawDate != "" {
		if ev.Date.IsZero() {
			if t, err := carriers.ParseLocalTime(ev.RawDate, zone); err == nil {
				ev.Date = t
			}
			return
		}
		if naiveUTC, err := carriers.ParseLocalTime(ev.RawDate, "UTC"); err == nil && ev.Date.Equal(naiveUTC) {
			if t, err := carriers.ParseLocalTime(ev.RawDate, zone); err == nil {
				ev.Date = t
				return
			}
		}
		ev.Date = ev.Date.UTC()
		return
	}

	if !ev.Date.IsZero() {
		if ev.Date.Location() != time.UTC {
			ev.Date = ev.Date.UTC()
		} else {
			ev.Date = carriers.NormalizeEventTime(ev.Date, zone)
		}
	}
}

// resolveEventZone determines the timezone for a tracking event using location, carrier slug, or fallback zone.
func resolveEventZone(ev models.TrackingEvent, fallbackZone string) string {
	locLower := strings.ToLower(ev.Location)
	switch {
	case strings.Contains(locLower, "china") || strings.Contains(locLower, "shanghai") || strings.Contains(locLower, "shenzhen") || strings.Contains(locLower, "hangzhou") || strings.Contains(locLower, "guangzhou") || strings.Contains(locLower, "beijing"):
		return "Asia/Shanghai"
	case strings.Contains(locLower, "spain") || strings.Contains(locLower, "españa") || strings.Contains(locLower, "espana") || strings.Contains(locLower, "madrid") || strings.Contains(locLower, "barcelona") || strings.Contains(locLower, "valencia") || strings.Contains(locLower, "sevilla") || strings.Contains(locLower, "coslada"):
		return "Europe/Madrid"
	case strings.Contains(locLower, "italy") || strings.Contains(locLower, "italia") || strings.Contains(locLower, "roma") || strings.Contains(locLower, "milano"):
		return "Europe/Rome"
	case strings.Contains(locLower, "portugal") || strings.Contains(locLower, "lisboa") || strings.Contains(locLower, "porto"):
		return "Europe/Lisbon"
	case strings.Contains(locLower, "france") || strings.Contains(locLower, "paris"):
		return "Europe/Paris"
	case strings.Contains(locLower, "germany") || strings.Contains(locLower, "berlin") || strings.Contains(locLower, "frankfurt"):
		return "Europe/Berlin"
	case strings.Contains(locLower, "united kingdom") || strings.Contains(locLower, "london"):
		return "Europe/London"
	}

	if ev.CarrierSlug != "" {
		z := carriers.CarrierZone(ev.CarrierSlug)
		if z != "" && z != "UTC" {
			return z
		}
	}

	if fallbackZone != "" {
		return fallbackZone
	}

	return "UTC"
}

// MergeEvents combines events from multiple carriers into a clean, deduplicated, chronologically sorted timeline in UTC.
func MergeEvents(primaryEvents []models.TrackingEvent, secondaryEvents []models.TrackingEvent) []models.TrackingEvent {
	normPrimary := make([]models.TrackingEvent, len(primaryEvents))
	for i, ev := range primaryEvents {
		normPrimary[i] = ev
		NormalizeTrackingEvent(&normPrimary[i], "")
	}

	normSecondary := make([]models.TrackingEvent, len(secondaryEvents))
	for i, ev := range secondaryEvents {
		normSecondary[i] = ev
		NormalizeTrackingEvent(&normSecondary[i], "")
	}

	return mergeNormalizedEvents(normPrimary, normSecondary)
}

func mergeNormalizedEvents(primaryEvents []models.TrackingEvent, secondaryEvents []models.TrackingEvent) []models.TrackingEvent {
	all := make([]models.TrackingEvent, 0, len(primaryEvents)+len(secondaryEvents))

	// Tag primary events (carrier index 0 if not already set)
	for _, ev := range primaryEvents {
		if ev.Status == "" || ev.Stage == "" {
			ev.Status, ev.Stage = NormalizeEventStatus(ev.Description)
		} else if ev.Status != models.StatusDelivered && IsDeliveredText(ev.Description) {
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
		} else if ev.Status != models.StatusDelivered && IsDeliveredText(ev.Description) {
			ev.Status = models.StatusDelivered
			ev.Stage = models.StageDelivery
		}
		all = append(all, ev)
	}

	// Sort events chronologically descending (newest first) by exact UTC instant
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

// MergeShipments merges two shipments (typically primary international journey and secondary last-mile journey)
// into a single unified shipment with chronologically normalized UTC events, unified status, progress calculation,
// and combined carrier metadata.
func MergeShipments(primary *models.Shipment, secondary *models.Shipment) *models.Shipment {
	if primary == nil && secondary == nil {
		return nil
	}
	if primary == nil {
		res := *secondary
		secZone := carriers.ResolveCountryZone(secondary.Carrier.Country)
		if secZone == "UTC" && secondary.Destination.Code != "" {
			secZone = carriers.ResolveCountryZone(secondary.Destination.Code)
		}
		if secZone == "UTC" && secondary.Carrier.Slug != "" {
			secZone = carriers.CarrierZone(secondary.Carrier.Slug)
		}
		normSec := make([]models.TrackingEvent, len(secondary.Events))
		for i, ev := range secondary.Events {
			normSec[i] = ev
			NormalizeTrackingEvent(&normSec[i], secZone)
		}
		res.Events = mergeNormalizedEvents(nil, normSec)
		res.Status = DeriveOverallStatus(res.Events)
		return &res
	}
	if secondary == nil {
		res := *primary
		primZone := carriers.ResolveCountryZone(primary.Carrier.Country)
		if primZone == "UTC" && primary.Origin.Code != "" {
			primZone = carriers.ResolveCountryZone(primary.Origin.Code)
		}
		if primZone == "UTC" && primary.Carrier.Slug != "" {
			primZone = carriers.CarrierZone(primary.Carrier.Slug)
		}
		normPrim := make([]models.TrackingEvent, len(primary.Events))
		for i, ev := range primary.Events {
			normPrim[i] = ev
			NormalizeTrackingEvent(&normPrim[i], primZone)
		}
		res.Events = mergeNormalizedEvents(normPrim, nil)
		res.Status = DeriveOverallStatus(res.Events)
		if primary.Status == models.StatusDelivered {
			res.Status = models.StatusDelivered
		}
		for _, ev := range res.Events {
			if ev.Status == models.StatusDelivered {
				if res.DeliveredAt == nil {
					d := ev.Date
					res.DeliveredAt = &d
				}
				break
			}
		}
		if res.Status == models.StatusDelivered {
			res.Progress = 100
		} else {
			res.Progress = CalculateProgress(res.Status, res.Events)
		}
		res.DaysInTransit = CalculateDaysInTransit(res.Events, res.DeliveredAt)
		res.UpdatedAt = time.Now().UTC()
		return &res
	}

	finalShipment := *primary
	finalShipment.AdditionalCarriers = append([]models.CarrierInfo(nil), primary.AdditionalCarriers...)

	// Resolve fallback timezones based on carrier / country
	primaryZone := carriers.ResolveCountryZone(primary.Carrier.Country)
	if primaryZone == "UTC" && primary.Origin.Code != "" {
		primaryZone = carriers.ResolveCountryZone(primary.Origin.Code)
	}
	if primaryZone == "UTC" && primary.Carrier.Slug != "" {
		primaryZone = carriers.CarrierZone(primary.Carrier.Slug)
	}

	secondaryZone := carriers.ResolveCountryZone(secondary.Carrier.Country)
	if secondaryZone == "UTC" && secondary.Destination.Code != "" {
		secondaryZone = carriers.ResolveCountryZone(secondary.Destination.Code)
	}
	if secondaryZone == "UTC" && secondary.Carrier.Slug != "" {
		secondaryZone = carriers.CarrierZone(secondary.Carrier.Slug)
	}

	// Normalize events of both legs
	normPrimaryEvents := make([]models.TrackingEvent, len(primary.Events))
	for i, ev := range primary.Events {
		normEv := ev
		NormalizeTrackingEvent(&normEv, primaryZone)
		normPrimaryEvents[i] = normEv
	}

	normSecondaryEvents := make([]models.TrackingEvent, len(secondary.Events))
	for i, ev := range secondary.Events {
		normEv := ev
		NormalizeTrackingEvent(&normEv, secondaryZone)
		normSecondaryEvents[i] = normEv
	}

	// Merge events into a single unified timeline
	mergedEvents := mergeNormalizedEvents(normPrimaryEvents, normSecondaryEvents)
	finalShipment.Events = mergedEvents

	// Ensure secondary carrier is represented in AdditionalCarriers
	if secondary.Carrier.Slug != "" && !strings.EqualFold(secondary.Carrier.Slug, finalShipment.Carrier.Slug) {
		hasSec := false
		for _, ac := range finalShipment.AdditionalCarriers {
			if strings.EqualFold(ac.Slug, secondary.Carrier.Slug) {
				hasSec = true
				break
			}
		}
		if !hasSec {
			finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, secondary.Carrier)
		}
	}
	for _, ac := range secondary.AdditionalCarriers {
		hasAc := false
		for _, fac := range finalShipment.AdditionalCarriers {
			if strings.EqualFold(fac.Slug, ac.Slug) {
				hasAc = true
				break
			}
		}
		if !hasAc && !strings.EqualFold(ac.Slug, finalShipment.Carrier.Slug) {
			finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, ac)
		}
	}

	// Inherit destination or origin if primary lacks them
	if finalShipment.Destination.Name == "" && secondary.Destination.Name != "" {
		finalShipment.Destination = secondary.Destination
	} else if finalShipment.Destination.Code == "" && secondary.Destination.Code != "" {
		finalShipment.Destination.Code = secondary.Destination.Code
	}
	if finalShipment.Origin.Name == "" && secondary.Origin.Name != "" {
		finalShipment.Origin = secondary.Origin
	}

	// Derive unified status
	finalShipment.Status = DeriveOverallStatus(mergedEvents)
	if primary.Status == models.StatusDelivered || secondary.Status == models.StatusDelivered {
		finalShipment.Status = models.StatusDelivered
	}

	// Check if delivered in events or primary/secondary shipment
	for _, ev := range mergedEvents {
		if ev.Status == models.StatusDelivered {
			if finalShipment.DeliveredAt == nil {
				d := ev.Date
				finalShipment.DeliveredAt = &d
			}
			break
		}
	}
	if finalShipment.Status == models.StatusDelivered && finalShipment.DeliveredAt == nil {
		if secondary.DeliveredAt != nil {
			finalShipment.DeliveredAt = secondary.DeliveredAt
		} else if primary.DeliveredAt != nil {
			finalShipment.DeliveredAt = primary.DeliveredAt
		}
	}

	// Calculate progress
	if finalShipment.Status == models.StatusDelivered {
		finalShipment.Progress = 100
	} else {
		finalShipment.Progress = CalculateProgress(finalShipment.Status, mergedEvents)
	}

	finalShipment.DaysInTransit = CalculateDaysInTransit(mergedEvents, finalShipment.DeliveredAt)
	finalShipment.UpdatedAt = time.Now().UTC()

	return &finalShipment
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
		if IsDeliveredText(ev.Description) {
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
