package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"open-parcels/pkg/carriers"
	"open-parcels/pkg/detector"
	"open-parcels/pkg/models"
)

// Engine is the central multi-carrier parcel tracking and journey merging orchestrator.
type Engine struct {
	detector *detector.Detector
	registry *carriers.Registry
}

// New creates an initialized Engine instance.
func New() *Engine {
	return &Engine{
		detector: detector.New(),
		registry: carriers.GetRegistry(),
	}
}

// Detector returns the underlying carrier detector engine.
func (e *Engine) Detector() *detector.Detector {
	return e.detector
}

// Registry returns the underlying carrier registry.
func (e *Engine) Registry() *carriers.Registry {
	return e.registry
}

// Track performs multi-carrier parcel tracking with automatic carrier detection, last-mile handover chaining, and milestone merging.
func (e *Engine) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNumber := detector.CleanTrackingNumber(trackingNumber)
	if cleanNumber == "" {
		return nil, fmt.Errorf("%w: empty or invalid tracking number", models.ErrShipmentNotFound)
	}

	var primaryCarrier carriers.Carrier
	var carrierInfo models.CarrierInfo

	// 1. Resolve carrier by hint or auto-detection
	if opts.CarrierHint != "" {
		c, ok := e.registry.Get(opts.CarrierHint)
		if !ok {
			return nil, fmt.Errorf("%w: unsupported carrier hint: %s", models.ErrCarrierNotImplemented, opts.CarrierHint)
		}
		primaryCarrier = c
		carrierInfo = c.Info()
	} else {
		best := e.detector.BestMatch(cleanNumber, opts.DestinationCountry)
		if best == nil {
			return nil, fmt.Errorf("%w: could not automatically detect carrier for tracking number: %s", models.ErrShipmentNotFound, cleanNumber)
		}
		c, ok := e.registry.Get(best.Carrier.Slug)
		if !ok {
			// If not specifically registered, try base fallback or generic
			return nil, fmt.Errorf("%w: carrier detected (%s) but adapter not registered", models.ErrCarrierNotImplemented, best.Carrier.Slug)
		}
		primaryCarrier = c
		carrierInfo = c.Info()
	}

	// Strictly reject simulated fallbacks unless AllowSimulated is explicitly true
	if !opts.AllowSimulated && !carriers.HasLiveTracking(primaryCarrier) {
		return nil, fmt.Errorf("%w: carrier %s does not have an active live tracking adapter", models.ErrCarrierNotImplemented, primaryCarrier.Name())
	}

	// 2. Track primary carrier
	primaryShipment, err := primaryCarrier.Track(ctx, cleanNumber, opts)
	if err != nil {
		return nil, fmt.Errorf("failed tracking with primary carrier %s: %w", primaryCarrier.Name(), err)
	}

	finalShipment := primaryShipment
	finalShipment.Carrier = carrierInfo

	// 3. Multi-carrier chaining: Check if last-mile carrier is present
	var secondaryEvents []models.TrackingEvent
	if primaryShipment.LastMileCarrier != nil && primaryShipment.LastMileTrackingNumber != "" {
		secSlug := primaryShipment.LastMileCarrier.Slug
		if secCarrier, ok := e.registry.Get(secSlug); ok {
			// Track secondary carrier concurrently with bounded timeout
			secCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()

			secOpts := opts
			if secOpts.DestinationCountry == "" {
				secOpts.DestinationCountry = primaryShipment.LastMileCarrier.Country
			}

			secShipment, err := secCarrier.Track(secCtx, primaryShipment.LastMileTrackingNumber, secOpts)
			if err == nil && secShipment != nil && len(secShipment.Events) > 0 {
				secondaryEvents = secShipment.Events
				hasCarrier := false
				for _, ac := range finalShipment.AdditionalCarriers {
					if strings.EqualFold(ac.Slug, secCarrier.Slug()) {
						hasCarrier = true
						break
					}
				}
				if !hasCarrier {
					finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, secCarrier.Info())
				}
				if secShipment.Destination.Name != "" && finalShipment.Destination.Name == "" {
					finalShipment.Destination = secShipment.Destination
				}
			}
		}
	}

	// Ensure LastMileCarrier is in AdditionalCarriers
	if finalShipment.LastMileCarrier != nil {
		hasCarrier := false
		for _, ac := range finalShipment.AdditionalCarriers {
			if strings.EqualFold(ac.Slug, finalShipment.LastMileCarrier.Slug) {
				hasCarrier = true
				break
			}
		}
		if !hasCarrier {
			finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, *finalShipment.LastMileCarrier)
		}
	}

	// 4. Merge events into a single unified timeline
	mergedEvents := MergeEvents(primaryShipment.Events, secondaryEvents)
	finalShipment.Events = mergedEvents

	// 5. Derive unified status and normalize
	finalShipment.Status = DeriveOverallStatus(mergedEvents)

	// Check if delivered in events or primary shipment
	for _, ev := range mergedEvents {
		if ev.Status == models.StatusDelivered {
			if finalShipment.DeliveredAt == nil {
				d := ev.Date
				finalShipment.DeliveredAt = &d
			}
			break
		}
	}
	if primaryShipment.Status == models.StatusDelivered || finalShipment.DeliveredAt != nil {
		finalShipment.Status = models.StatusDelivered
	}

	finalShipment.StatusText = LocalizedStatusText(finalShipment.Status, opts.Language)

	// 6. Calculate progress percentage and transit days
	if finalShipment.Status == models.StatusDelivered {
		finalShipment.Progress = 100
	} else {
		finalShipment.Progress = CalculateProgress(finalShipment.Status, mergedEvents)
	}
	finalShipment.DaysInTransit = CalculateDaysInTransit(mergedEvents, finalShipment.DeliveredAt)
	finalShipment.UpdatedAt = time.Now()

	// 7. Multi-carrier Journey: Ensure both Cainiao Global and Correos Express are represented for 323... guides
	if strings.HasPrefix(cleanNumber, "323") && len(cleanNumber) == 23 {
		if strings.EqualFold(finalShipment.Carrier.Slug, "cainiao") {
			hasCE := false
			for _, ac := range finalShipment.AdditionalCarriers {
				if strings.EqualFold(ac.Slug, "correosexpress") {
					hasCE = true
					break
				}
			}
			if !hasCE {
				if ceCarrier, ok := e.registry.Get("correosexpress"); ok {
					finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, ceCarrier.Info())
				}
			}
		} else if strings.EqualFold(finalShipment.Carrier.Slug, "correosexpress") {
			hasCainiao := false
			for _, ac := range finalShipment.AdditionalCarriers {
				if strings.EqualFold(ac.Slug, "cainiao") {
					hasCainiao = true
					break
				}
			}
			if !hasCainiao {
				if cainiaoCarrier, ok := e.registry.Get("cainiao"); ok {
					finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, cainiaoCarrier.Info())
				}
			}
		}
	}

	// Also ensure any distinct carrier present in events is represented in AdditionalCarriers
	for _, ev := range finalShipment.Events {
		if ev.CarrierSlug != "" && !strings.EqualFold(ev.CarrierSlug, finalShipment.Carrier.Slug) {
			hasSlug := false
			for _, ac := range finalShipment.AdditionalCarriers {
				if strings.EqualFold(ac.Slug, ev.CarrierSlug) {
					hasSlug = true
					break
				}
			}
			if !hasSlug {
				if c, ok := e.registry.Get(ev.CarrierSlug); ok {
					finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, c.Info())
				} else {
					finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, models.CarrierInfo{
						Slug: ev.CarrierSlug,
						Name: ev.CarrierName,
					})
				}
			}
		}
	}

	return finalShipment, nil
}

// ToParcelsAppResponse converts a Shipment into the ParcelsApp v4 API compatibility response structure.
func (e *Engine) ToParcelsAppResponse(shipment *models.Shipment, reqID string) *models.ParcelsAppResponse {
	if reqID == "" {
		reqID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}

	var states []models.ParcelsAppStateItem
	for _, ev := range shipment.Events {
		states = append(states, models.ParcelsAppStateItem{
			State:    ev.Description,
			Location: ev.Location,
			Date:     ev.Date.UTC().Format(time.RFC3339),
			Carrier:  ev.CarrierIndex,
		})
	}

	services := []models.ParcelsAppServiceItem{
		{Slug: shipment.Carrier.Slug, Name: shipment.Carrier.Name},
	}
	for _, ac := range shipment.AdditionalCarriers {
		services = append(services, models.ParcelsAppServiceItem{Slug: ac.Slug, Name: ac.Name})
	}

	var attrs []models.ParcelsAppAttribute
	if shipment.Origin.Name != "" {
		attrs = append(attrs, models.ParcelsAppAttribute{Label: "origin", Val: shipment.Origin.Name, Code: shipment.Origin.Code})
	}
	if shipment.Destination.Name != "" {
		attrs = append(attrs, models.ParcelsAppAttribute{Label: "destination", Val: shipment.Destination.Name, Code: shipment.Destination.Code})
	}

	var extTrack []models.ParcelsAppExternalTrack
	if shipment.Carrier.TrackingURL != "" {
		extTrack = append(extTrack, models.ParcelsAppExternalTrack{
			Slug:       shipment.Carrier.Slug,
			URL:        fmt.Sprintf(shipment.Carrier.TrackingURL, shipment.TrackingNumber),
			TrackingID: shipment.TrackingNumber,
			Method:     "GET",
		})
	}

	detectedIndices := []int{0}
	if len(shipment.AdditionalCarriers) > 0 {
		for i := 1; i <= len(shipment.AdditionalCarriers); i++ {
			detectedIndices = append(detectedIndices, i)
		}
	}

	pShipment := &models.ParcelsAppShipment{
		TrackingID:      shipment.TrackingNumber,
		Status:          string(shipment.Status),
		Origin:          shipment.Origin.Name,
		Destination:     shipment.Destination.Name,
		OriginCode:      shipment.Origin.Code,
		DestinationCode: shipment.Destination.Code,
		States:          states,
		Services:        services,
		DetectedCarrier: models.ParcelsAppServiceItem{
			Slug: shipment.Carrier.Slug,
			Name: shipment.Carrier.Name,
		},
		Detected:         detectedIndices,
		DaysInTransit:    shipment.DaysInTransit,
		Attributes:       attrs,
		ExternalTracking: extTrack,
		FlightLegs:       shipment.FlightLegs,
		Error:            shipment.Error,
	}

	return &models.ParcelsAppResponse{
		RequestID: reqID,
		Status:    "completed",
		Done:      true,
		ExpiresAt: time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339),
		Results: []models.ParcelsAppResultItem{
			{
				TrackingNumber: shipment.TrackingNumber,
				State:          "completed",
				FromCache:      shipment.FromCache,
				Shipment:       pShipment,
			},
		},
	}
}
