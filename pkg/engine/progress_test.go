// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestCalculateProgressMaritimeProgression(t *testing.T) {
	now := time.Now()

	// 1. Gate-in only
	gateInEvents := []models.TrackingEvent{
		{
			Date:        now.Add(-10 * 24 * time.Hour),
			Description: "Gate-in full at origin port terminal",
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
			EventCode:   "GATE_IN",
		},
	}
	progGateIn := CalculateProgress(models.StatusInTransit, gateInEvents)

	// 2. Gate-in + Loaded on board
	loadedEvents := append([]models.TrackingEvent{
		{
			Date:        now.Add(-8 * 24 * time.Hour),
			Description: "Loaded on board vessel MSC OSCAR",
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "LOADED",
		},
	}, gateInEvents...)
	progLoaded := CalculateProgress(models.StatusInTransit, loadedEvents)

	// 3. + Vessel Departed / Underway
	departedEvents := append([]models.TrackingEvent{
		{
			Date:        now.Add(-7 * 24 * time.Hour),
			Description: "Vessel departed origin port / Underway",
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "DEPARTED",
		},
	}, loadedEvents...)
	progDeparted := CalculateProgress(models.StatusInTransit, departedEvents)

	// 4. + Transshipment / high seas
	transEvents := append([]models.TrackingEvent{
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Vessel in transit on high seas / Transshipment leg",
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "TRANSSHIPMENT",
		},
	}, departedEvents...)
	progTrans := CalculateProgress(models.StatusInTransit, transEvents)

	// 5. + Discharged at POD
	dischargedEvents := append([]models.TrackingEvent{
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Container discharged at destination port terminal",
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "DISCHARGED",
		},
	}, transEvents...)
	progDischarged := CalculateProgress(models.StatusInTransit, dischargedEvents)

	// 6. + Customs clearance
	customsEvents := append([]models.TrackingEvent{
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Customs clearance completed at destination port",
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
			EventCode:   "CUSTOMS",
		},
	}, dischargedEvents...)
	progCustoms := CalculateProgress(models.StatusInTransit, customsEvents)

	// 7. + Gate-out full
	gateOutEvents := append([]models.TrackingEvent{
		{
			Date:        now.Add(-1 * 24 * time.Hour),
			Description: "Gate-out full from terminal for delivery to consignee",
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
			EventCode:   "GATE_OUT",
		},
	}, customsEvents...)
	progGateOut := CalculateProgress(models.StatusOutForDelivery, gateOutEvents)

	// 8. + Empty Return (Delivered)
	emptyReturnEvents := append([]models.TrackingEvent{
		{
			Date:        now,
			Description: "Empty return to container depot",
			Status:      models.StatusDelivered,
			Stage:       models.StageDelivery,
			EventCode:   "EMPTY_RETURN",
		},
	}, gateOutEvents...)
	progDelivered := CalculateProgress(models.StatusDelivered, emptyReturnEvents)

	// Verify all percentages are within [20, 100]
	percentages := []int{progGateIn, progLoaded, progDeparted, progTrans, progDischarged, progCustoms, progGateOut, progDelivered}
	for i, p := range percentages {
		if p < 20 || p > 100 {
			t.Errorf("step %d progress %d not between 20%% and 100%%", i, p)
		}
	}

	// Verify strictly monotonic advancement across maritime stages:
	// Gate-in (20%) < Loaded < Departed < Transshipment < Discharged < Customs < Gate-out < Delivered (100%)
	if !(progGateIn < progLoaded &&
		progLoaded < progDeparted &&
		progDeparted < progTrans &&
		progTrans < progDischarged &&
		progDischarged < progCustoms &&
		progCustoms < progGateOut &&
		progGateOut <= progDelivered) {
		t.Errorf("expected strictly monotonic progression across maritime milestones, got: "+
			"GateIn=%d, Loaded=%d, Departed=%d, Trans=%d, Discharged=%d, Customs=%d, GateOut=%d, Delivered=%d",
			progGateIn, progLoaded, progDeparted, progTrans, progDischarged, progCustoms, progGateOut, progDelivered)
	}

	if progGateIn != 20 {
		t.Errorf("expected Gate-in progress to be 20%%, got %d%%", progGateIn)
	}
	if progDelivered != 100 {
		t.Errorf("expected Empty return / Delivered progress to be 100%%, got %d%%", progDelivered)
	}
}

func TestCalculateProgressStandardParcels(t *testing.T) {
	if p := CalculateProgress(models.StatusDelivered, nil); p != 100 {
		t.Errorf("expected 100 for delivered, got %d", p)
	}
	if p := CalculateProgress(models.StatusExpired, nil); p != 100 {
		t.Errorf("expected 100 for expired, got %d", p)
	}
	if p := CalculateProgress(models.StatusOutForDelivery, nil); p != 85 {
		t.Errorf("expected 85 for out for delivery, got %d", p)
	}
	if p := CalculateProgress(models.StatusFailedAttempt, nil); p != 80 {
		t.Errorf("expected 80 for failed attempt, got %d", p)
	}
	if p := CalculateProgress(models.StatusInfoReceived, nil); p != 15 {
		t.Errorf("expected 15 for info received, got %d", p)
	}
	if p := CalculateProgress(models.StatusPending, nil); p != 5 {
		t.Errorf("expected 5 for pending, got %d", p)
	}
	if p := CalculateProgress(models.StatusException, nil); p != 60 {
		t.Errorf("expected 60 for exception, got %d", p)
	}

	// Standard in transit with different parcel stages
	handoverEvents := []models.TrackingEvent{{Stage: models.StageHandover}}
	if p := CalculateProgress(models.StatusInTransit, handoverEvents); p != 75 {
		t.Errorf("expected 75 for handover, got %d", p)
	}

	customsEvents := []models.TrackingEvent{{Stage: models.StageCustoms}}
	if p := CalculateProgress(models.StatusInTransit, customsEvents); p != 55 {
		t.Errorf("expected 55 for parcel customs, got %d", p)
	}

	departedEvents := []models.TrackingEvent{{Stage: models.StagePickedUp}}
	if p := CalculateProgress(models.StatusInTransit, departedEvents); p != 35 {
		t.Errorf("expected 35 for departed origin, got %d", p)
	}

	baselineEvents := []models.TrackingEvent{{Stage: models.StageCreated}}
	if p := CalculateProgress(models.StatusInTransit, baselineEvents); p != 25 {
		t.Errorf("expected 25 for baseline transit, got %d", p)
	}

	if p := CalculateProgress(models.Status("other"), nil); p != 30 {
		t.Errorf("expected 30 for default, got %d", p)
	}
}

func TestCalculateDaysInTransit(t *testing.T) {
	now := time.Now()

	// Empty events
	if d := CalculateDaysInTransit(nil, nil); d != 0 {
		t.Errorf("expected 0 days for empty events, got %d", d)
	}

	// Delivered 3 days after start
	events := []models.TrackingEvent{
		{Date: now.Add(-3 * 24 * time.Hour)},
		{Date: now.Add(-1 * 24 * time.Hour)},
	}
	del := now
	if d := CalculateDaysInTransit(events, &del); d != 3 {
		t.Errorf("expected 3 days, got %d", d)
	}

	// Recent event (< 24h)
	recentEvents := []models.TrackingEvent{
		{Date: now.Add(-2 * time.Hour)},
	}
	if d := CalculateDaysInTransit(recentEvents, nil); d != 1 {
		t.Errorf("expected 1 day for recent event, got %d", d)
	}
}
