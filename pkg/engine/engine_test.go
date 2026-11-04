// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"context"
	"testing"

	"open-parcels/pkg/models"
)

func TestEngineTrackChaining(t *testing.T) {
	eng := New()

	// Track Cainiao number with handover to Correos Express
	shipment, err := eng.Track(context.Background(), "LP00561234567890", models.TrackOptions{
		DestinationCountry: "ES",
		Language:           "es",
	})
	if err != nil {
		t.Fatalf("expected successful tracking, got err: %v", err)
	}

	if shipment.Carrier.Slug != "cainiao" {
		t.Errorf("expected primary carrier cainiao, got %s", shipment.Carrier.Slug)
	}

	if len(shipment.AdditionalCarriers) == 0 {
		t.Errorf("expected additional carrier (Correos Express) chained for last mile")
	}

	if len(shipment.Events) < 5 {
		t.Errorf("expected merged timeline with events, got %d", len(shipment.Events))
	}

	// Verify chronological ordering: newest event first
	for i := 0; i < len(shipment.Events)-1; i++ {
		if shipment.Events[i].Date.Before(shipment.Events[i+1].Date) {
			t.Errorf("events not sorted in descending chronological order at index %d", i)
		}
	}

	// Check status normalization and progress
	if shipment.Progress <= 0 || shipment.Progress > 100 {
		t.Errorf("progress should be between 1 and 100, got %d", shipment.Progress)
	}

	if shipment.StatusText == "" {
		t.Errorf("expected localized status text, got empty")
	}

	// Test ParcelsApp v4 response conversion
	pResp := eng.ToParcelsAppResponse(shipment, "test-req-123")
	if pResp == nil || len(pResp.Results) == 0 {
		t.Fatalf("expected ParcelsApp response with results")
	}
	if pResp.Results[0].Shipment.TrackingID != "LP00561234567890" {
		t.Errorf("expected tracking ID in ParcelsApp shipment, got %s", pResp.Results[0].Shipment.TrackingID)
	}
	if len(pResp.Results[0].Shipment.Services) < 2 {
		t.Errorf("expected multiple services listed in ParcelsApp response for chained journey")
	}
}

func TestEngineAirCargo(t *testing.T) {
	eng := New()
	shipment, err := eng.Track(context.Background(), "157-52471764", models.TrackOptions{})
	if err != nil {
		t.Fatalf("unexpected error tracking AWB: %v", err)
	}

	if shipment.Carrier.Slug != "air-cargo" {
		t.Errorf("expected air-cargo carrier, got %s", shipment.Carrier.Slug)
	}

	if len(shipment.FlightLegs) == 0 {
		t.Errorf("expected flight legs for air cargo tracking")
	}
}

func TestEngineOceanShipmentConversion(t *testing.T) {
	eng := New()
	shipment := &models.Shipment{
		TrackingNumber: "MSCU1234567",
		Carrier: models.CarrierInfo{
			Slug: "ocean-container",
			Name: "ISO Ocean Container",
		},
		Status:   models.StatusInTransit,
		FromPort: "CNSHA",
		ToPort:   "ESVLC",
		Container: &models.ContainerDetails{
			Number:     "MSCU1234567",
			Type:       "40HC",
			SealNumber: "SEAL-001",
		},
		SeaLegs: []models.SeaLeg{
			{
				Vessel: &models.VesselInfo{
					Name: "MSC OSCAR",
					IMO:  "9703291",
				},
				VoyageNumber:        "V100",
				OriginPort:          "Shanghai",
				DestinationPort:     "Valencia",
				OriginUNLOCODE:      "CNSHA",
				DestinationUNLOCODE: "ESVLC",
			},
		},
	}

	pResp := eng.ToParcelsAppResponse(shipment, "ocean-req-1")
	if pResp == nil || len(pResp.Results) == 0 {
		t.Fatalf("expected valid ParcelsAppResponse")
	}

	pShip := pResp.Results[0].Shipment
	if pShip == nil {
		t.Fatalf("expected non-nil ParcelsAppShipment")
	}

	if pShip.FromPort != "CNSHA" || pShip.ToPort != "ESVLC" {
		t.Errorf("expected ports CNSHA/ESVLC, got %s/%s", pShip.FromPort, pShip.ToPort)
	}
	if pShip.Container == nil || pShip.Container.Number != "MSCU1234567" {
		t.Errorf("expected container MSCU1234567, got %+v", pShip.Container)
	}
	if len(pShip.SeaLegs) != 1 || pShip.SeaLegs[0].VoyageNumber != "V100" {
		t.Errorf("expected 1 sea leg with V100, got %+v", pShip.SeaLegs)
	}

	// Verify UN/LOCODE and container attributes
	attrPort, okPort := pShip.FindAttribute("from_port")
	if !okPort || attrPort.Code != "CNSHA" {
		t.Errorf("expected from_port attribute with CNSHA, got ok=%v, attr=%+v", okPort, attrPort)
	}
	attrCont, okCont := pShip.FindAttribute("container")
	if !okCont || attrCont.Val != "MSCU1234567" {
		t.Errorf("expected container attribute MSCU1234567, got ok=%v, attr=%+v", okCont, attrCont)
	}

	// Verify enriched maritime attributes: vessel, voyage, pol, pod, UN/LOCODEs
	attrVessel, okVessel := pShip.FindAttribute("vessel")
	if !okVessel || attrVessel.Val != "MSC OSCAR" || attrVessel.Code != "9703291" {
		t.Errorf("expected vessel attribute MSC OSCAR (9703291), got ok=%v, attr=%+v", okVessel, attrVessel)
	}

	attrVoyage, okVoyage := pShip.FindAttribute("voyage")
	if !okVoyage || attrVoyage.Val != "V100" {
		t.Errorf("expected voyage attribute V100, got ok=%v, attr=%+v", okVoyage, attrVoyage)
	}

	attrPOL, okPOL := pShip.FindAttribute("pol")
	if !okPOL || attrPOL.Code != "CNSHA" {
		t.Errorf("expected pol attribute with CNSHA, got ok=%v, attr=%+v", okPOL, attrPOL)
	}

	attrPOD, okPOD := pShip.FindAttribute("pod")
	if !okPOD || attrPOD.Code != "ESVLC" {
		t.Errorf("expected pod attribute with ESVLC, got ok=%v, attr=%+v", okPOD, attrPOD)
	}

	attrOrigLoc, okOrigLoc := pShip.FindAttribute("origin_unlocode")
	if !okOrigLoc || attrOrigLoc.Val != "CNSHA" {
		t.Errorf("expected origin_unlocode attribute with CNSHA, got ok=%v, attr=%+v", okOrigLoc, attrOrigLoc)
	}

	attrDestLoc, okDestLoc := pShip.FindAttribute("destination_unlocode")
	if !okDestLoc || attrDestLoc.Val != "ESVLC" {
		t.Errorf("expected destination_unlocode attribute with ESVLC, got ok=%v, attr=%+v", okDestLoc, attrDestLoc)
	}
}

func TestEngineTrackErrorConditions(t *testing.T) {
	eng := New()
	ctx := context.Background()

	// Empty tracking number
	_, err := eng.Track(ctx, "   ", models.TrackOptions{})
	if err == nil {
		t.Errorf("expected error tracking empty tracking number, got nil")
	}

	// Unsupported carrier hint
	_, err = eng.Track(ctx, "12345678", models.TrackOptions{CarrierHint: "non-existent-carrier"})
	if err == nil {
		t.Errorf("expected error tracking unsupported carrier hint, got nil")
	}

	// Undetectable carrier number
	_, err = eng.Track(ctx, "???invalid_number???", models.TrackOptions{})
	if err == nil {
		t.Errorf("expected error for undetectable tracking number, got nil")
	}

	// Getters
	if eng.Detector() == nil {
		t.Errorf("expected non-nil detector")
	}
	if eng.Registry() == nil {
		t.Errorf("expected non-nil registry")
	}
}

