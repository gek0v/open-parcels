// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("empty or invalid tracking number")
	}

	var primaryCarrier carriers.Carrier
	var carrierInfo models.CarrierInfo

	// 1. Resolve carrier by hint or auto-detection
	if opts.CarrierHint != "" {
		c, ok := e.registry.Get(opts.CarrierHint)
		if !ok {
			return nil, fmt.Errorf("unsupported carrier hint: %s", opts.CarrierHint)
		}
		primaryCarrier = c
		carrierInfo = c.Info()
	} else {
		best := e.detector.BestMatch(cleanNumber, opts.DestinationCountry)
		if best == nil {
			return nil, fmt.Errorf("could not automatically detect carrier for tracking number: %s", cleanNumber)
		}
		c, ok := e.registry.Get(best.Carrier.Slug)
		if !ok {
			// If not specifically registered, try base fallback or generic
			return nil, fmt.Errorf("carrier detected (%s) but adapter not registered", best.Carrier.Slug)
		}
		primaryCarrier = c
		carrierInfo = c.Info()
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
				finalShipment.AdditionalCarriers = append(finalShipment.AdditionalCarriers, secCarrier.Info())
				if secShipment.Destination.Name != "" && finalShipment.Destination.Name == "" {
					finalShipment.Destination = secShipment.Destination
				}
			}
		}
	}

	// 4. Merge events into a single unified timeline
	mergedEvents := MergeEvents(primaryShipment.Events, secondaryEvents)
	finalShipment.Events = mergedEvents

	// 5. Derive unified status and normalize
	finalShipment.Status = DeriveOverallStatus(mergedEvents)
	finalShipment.StatusText = LocalizedStatusText(finalShipment.Status, opts.Language)

	// Check if delivered
	for _, ev := range mergedEvents {
		if ev.Status == models.StatusDelivered {
			finalShipment.DeliveredAt = &ev.Date
			break
		}
	}

	// 6. Calculate progress percentage and transit days
	finalShipment.Progress = CalculateProgress(finalShipment.Status, mergedEvents)
	finalShipment.DaysInTransit = CalculateDaysInTransit(mergedEvents, finalShipment.DeliveredAt)
	finalShipment.UpdatedAt = time.Now()

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

	if shipment.FromPort != "" {
		attrs = append(attrs, models.ParcelsAppAttribute{Label: "from_port", Val: shipment.FromPort, Code: shipment.FromPort})
	}
	if shipment.ToPort != "" {
		attrs = append(attrs, models.ParcelsAppAttribute{Label: "to_port", Val: shipment.ToPort, Code: shipment.ToPort})
	}

	// Enrich with maritime attributes (vessel, voyage, pol, pod, UN/LOCODEs)
	originUNLOCODE := shipment.FromPort
	destUNLOCODE := shipment.ToPort

	if len(shipment.SeaLegs) > 0 {
		firstLeg := shipment.SeaLegs[0]
		if firstLeg.OriginUNLOCODE != "" && originUNLOCODE == "" {
			originUNLOCODE = firstLeg.OriginUNLOCODE
		}
		if firstLeg.DestinationUNLOCODE != "" && destUNLOCODE == "" {
			destUNLOCODE = firstLeg.DestinationUNLOCODE
		}

		if firstLeg.Vessel != nil && firstLeg.Vessel.Name != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{
				Label: "vessel",
				Val:   firstLeg.Vessel.Name,
				Code:  firstLeg.Vessel.IMO,
			})
		}

		if firstLeg.VoyageNumber != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{
				Label: "voyage",
				Val:   firstLeg.VoyageNumber,
			})
		}

		polName := firstLeg.OriginPort
		if polName == "" {
			polName = firstLeg.Origin
		}
		polCode := firstLeg.OriginUNLOCODE
		if polCode == "" {
			polCode = originUNLOCODE
		}
		if polName == "" {
			polName = polCode
		}
		if polName != "" || polCode != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{
				Label: "pol",
				Val:   polName,
				Code:  polCode,
			})
		}

		podName := firstLeg.DestinationPort
		if podName == "" {
			podName = firstLeg.Destination
		}
		podCode := firstLeg.DestinationUNLOCODE
		if podCode == "" {
			podCode = destUNLOCODE
		}
		if podName == "" {
			podName = podCode
		}
		if podName != "" || podCode != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{
				Label: "pod",
				Val:   podName,
				Code:  podCode,
			})
		}
	} else {
		if originUNLOCODE != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{
				Label: "pol",
				Val:   originUNLOCODE,
				Code:  originUNLOCODE,
			})
		}
		if destUNLOCODE != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{
				Label: "pod",
				Val:   destUNLOCODE,
				Code:  destUNLOCODE,
			})
		}
	}

	if originUNLOCODE != "" {
		attrs = append(attrs, models.ParcelsAppAttribute{
			Label: "origin_unlocode",
			Val:   originUNLOCODE,
			Code:  originUNLOCODE,
		})
	}
	if destUNLOCODE != "" {
		attrs = append(attrs, models.ParcelsAppAttribute{
			Label: "destination_unlocode",
			Val:   destUNLOCODE,
			Code:  destUNLOCODE,
		})
	}

	if shipment.Container != nil {
		if shipment.Container.Number != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{Label: "container", Val: shipment.Container.Number})
		}
		if shipment.Container.Type != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{Label: "container_type", Val: shipment.Container.Type})
		}
		if shipment.Container.SealNumber != "" {
			attrs = append(attrs, models.ParcelsAppAttribute{Label: "seal_number", Val: shipment.Container.SealNumber})
		}
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
		FromPort:        originUNLOCODE,
		ToPort:          destUNLOCODE,
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
		SeaLegs:          shipment.SeaLegs,
		Container:        shipment.Container,
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
