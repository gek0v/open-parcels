// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package models

import (
	"encoding/json"
	"testing"
)

func TestStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{StatusPending, false},
		{StatusInfoReceived, false},
		{StatusInTransit, false},
		{StatusOutForDelivery, false},
		{StatusDelivered, true},
		{StatusFailedAttempt, false},
		{StatusException, false},
		{StatusExpired, true},
	}

	for _, tt := range tests {
		if got := tt.status.IsTerminal(); got != tt.expected {
			t.Errorf("Status(%s).IsTerminal() = %v, expected %v", tt.status, got, tt.expected)
		}
	}
}

func TestVesselInfoJSON(t *testing.T) {
	vessel := VesselInfo{
		Name:     "MSC OSCAR",
		IMO:      "9703291",
		MMSI:     "355906000",
		CallSign: "3FDA9",
		Flag:     "PA",
	}

	data, err := json.Marshal(vessel)
	if err != nil {
		t.Fatalf("failed to marshal VesselInfo: %v", err)
	}

	var parsed VesselInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal VesselInfo: %v", err)
	}

	if parsed.Name != vessel.Name || parsed.IMO != vessel.IMO || parsed.MMSI != vessel.MMSI ||
		parsed.CallSign != vessel.CallSign || parsed.Flag != vessel.Flag {
		t.Errorf("mismatch in VesselInfo roundtrip: got %+v, want %+v", parsed, vessel)
	}
}

func TestContainerDetailsJSON(t *testing.T) {
	container := ContainerDetails{
		Number:        "MSCU6543210",
		Type:          "40HC",
		ISOType:       "45G1",
		Status:        "loaded",
		SealNumber:    "SL-987654",
		TareWeight:    "3,800 kg",
		PayloadWeight: "26,680 kg",
		GrossWeight:   "30,480 kg",
	}

	data, err := json.Marshal(container)
	if err != nil {
		t.Fatalf("failed to marshal ContainerDetails: %v", err)
	}

	var parsed ContainerDetails
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal ContainerDetails: %v", err)
	}

	if parsed.Number != container.Number || parsed.Type != container.Type ||
		parsed.ISOType != container.ISOType || parsed.Status != container.Status ||
		parsed.SealNumber != container.SealNumber || parsed.TareWeight != container.TareWeight ||
		parsed.PayloadWeight != container.PayloadWeight || parsed.GrossWeight != container.GrossWeight {
		t.Errorf("mismatch in ContainerDetails roundtrip: got %+v, want %+v", parsed, container)
	}
}

func TestSeaLegJSON(t *testing.T) {
	leg := SeaLeg{
		Vessel: &VesselInfo{
			Name: "MAERSK MC-KINNEY MOLLER",
			IMO:  "9619907",
			Flag: "DK",
		},
		VoyageNumber:           "2401E",
		OriginPort:             "Port of Shanghai",
		DestinationPort:        "Port of Rotterdam",
		OriginUNLOCODE:         "CNSHA",
		DestinationUNLOCODE:    "NLRTM",
		Status:                 "underway",
		ScheduledDepartureDate: "2026-03-01",
		ScheduledDeparture:     "2026-03-01T08:00:00Z",
		ScheduledArrival:       "2026-03-25T14:00:00Z",
		ActualDeparture:        "2026-03-01T09:30:00Z",
		ActualArrival:          "",
	}

	data, err := json.Marshal(leg)
	if err != nil {
		t.Fatalf("failed to marshal SeaLeg: %v", err)
	}

	var parsed SeaLeg
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal SeaLeg: %v", err)
	}

	if parsed.VoyageNumber != leg.VoyageNumber || parsed.OriginUNLOCODE != "CNSHA" ||
		parsed.DestinationUNLOCODE != "NLRTM" || parsed.Vessel == nil ||
		parsed.Vessel.Name != "MAERSK MC-KINNEY MOLLER" {
		t.Errorf("mismatch in SeaLeg roundtrip: got %+v, want %+v", parsed, leg)
	}
}

func TestShipmentOceanFields(t *testing.T) {
	shipment := Shipment{
		TrackingNumber: "MSCU6543210",
		Carrier: CarrierInfo{
			Slug: "msc",
			Name: "Mediterranean Shipping Company",
			Type: CarrierTypeFreight,
		},
		Status:     StatusInTransit,
		StatusText: "In Transit - Underway",
		Progress:   50,
		Container: &ContainerDetails{
			Number:  "MSCU6543210",
			Type:    "40HC",
			ISOType: "45G1",
			Status:  "underway",
		},
		SeaLegs: []SeaLeg{
			{
				Vessel: &VesselInfo{
					Name: "MSC OSCAR",
					IMO:  "9703291",
				},
				VoyageNumber:        "2401W",
				OriginPort:          "Port of Shanghai",
				DestinationPort:     "Port of Valencia",
				OriginUNLOCODE:      "CNSHA",
				DestinationUNLOCODE: "ESVLC",
				Status:              "underway",
			},
		},
		FromPort: "CNSHA",
		ToPort:   "ESVLC",
	}

	if !shipment.HasOceanData() {
		t.Errorf("expected shipment.HasOceanData() to return true")
	}

	data, err := json.Marshal(shipment)
	if err != nil {
		t.Fatalf("failed to marshal Shipment: %v", err)
	}

	var parsed Shipment
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal Shipment: %v", err)
	}

	if parsed.Container == nil || parsed.Container.Number != "MSCU6543210" {
		t.Fatalf("expected container MSCU6543210, got %+v", parsed.Container)
	}
	if len(parsed.SeaLegs) != 1 || parsed.SeaLegs[0].OriginUNLOCODE != "CNSHA" {
		t.Fatalf("expected 1 sea leg with CNSHA, got %+v", parsed.SeaLegs)
	}
	if parsed.FromPort != "CNSHA" || parsed.ToPort != "ESVLC" {
		t.Errorf("expected FromPort=CNSHA, ToPort=ESVLC, got %s, %s", parsed.FromPort, parsed.ToPort)
	}

	emptyShipment := Shipment{TrackingNumber: "123"}
	if emptyShipment.HasOceanData() {
		t.Errorf("expected empty shipment.HasOceanData() to return false")
	}
}

func TestParcelsAppShipmentOceanAndUNLOCODE(t *testing.T) {
	pShipment := ParcelsAppShipment{
		TrackingID: "MSCU6543210",
		Status:     "in_transit",
		FromPort:   "CNSHA",
		ToPort:     "ESVLC",
		Container: &ContainerDetails{
			Number:  "MSCU6543210",
			Type:    "40HC",
			ISOType: "45G1",
		},
		SeaLegs: []SeaLeg{
			{
				Vessel: &VesselInfo{
					Name: "MSC OSCAR",
					IMO:  "9703291",
				},
				VoyageNumber:        "2401W",
				OriginPort:          "Shanghai",
				DestinationPort:     "Valencia",
				OriginUNLOCODE:      "CNSHA",
				DestinationUNLOCODE: "ESVLC",
				Status:              "underway",
			},
		},
	}

	// Add UN/LOCODE attributes and container attributes
	pShipment.AddAttribute("from_port", "Port of Shanghai", "CNSHA")
	pShipment.AddAttribute("to_port", "Port of Valencia", "ESVLC")
	pShipment.AddAttribute("container", "MSCU6543210", "")
	pShipment.AddAttribute("container_type", "40HC", "")

	// Test updating existing attributes (with and without code update)
	pShipment.AddAttribute("from_port", "Port of Shanghai Terminal 1", "CNSHA-T1")
	pShipment.AddAttribute("container_type", "40HC High Cube", "")

	attr, found := pShipment.FindAttribute("from_port")
	if !found || attr.Code != "CNSHA-T1" || attr.Val != "Port of Shanghai Terminal 1" {
		t.Errorf("expected updated from_port attribute, got found=%v, attr=%+v", found, attr)
	}

	attrType, foundType := pShipment.FindAttribute("container_type")
	if !foundType || attrType.Val != "40HC High Cube" {
		t.Errorf("expected updated container_type attribute, got found=%v, attr=%+v", foundType, attrType)
	}

	attrCont, foundCont := pShipment.FindAttribute("container")
	if !foundCont || attrCont.Val != "MSCU6543210" {
		t.Errorf("expected container attribute with MSCU6543210, got found=%v, attr=%+v", foundCont, attrCont)
	}

	if !pShipment.HasOceanData() {
		t.Errorf("expected pShipment.HasOceanData() to return true")
	}

	data, err := json.Marshal(pShipment)
	if err != nil {
		t.Fatalf("failed to marshal ParcelsAppShipment: %v", err)
	}

	var parsed ParcelsAppShipment
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal ParcelsAppShipment: %v", err)
	}

	if parsed.FromPort != "CNSHA" || parsed.ToPort != "ESVLC" {
		t.Errorf("expected FromPort=CNSHA, ToPort=ESVLC, got %s, %s", parsed.FromPort, parsed.ToPort)
	}
	if parsed.Container == nil || parsed.Container.Number != "MSCU6543210" {
		t.Errorf("expected container MSCU6543210, got %+v", parsed.Container)
	}
	if len(parsed.SeaLegs) != 1 || parsed.SeaLegs[0].VoyageNumber != "2401W" {
		t.Errorf("expected 1 sea leg with voyage 2401W, got %+v", parsed.SeaLegs)
	}
	if len(parsed.Attributes) != 4 {
		t.Errorf("expected 4 attributes, got %d", len(parsed.Attributes))
	}

	emptyPShipment := ParcelsAppShipment{TrackingID: "123"}
	if emptyPShipment.HasOceanData() {
		t.Errorf("expected empty ParcelsAppShipment.HasOceanData() to return false")
	}
	if _, found := emptyPShipment.FindAttribute("missing"); found {
		t.Errorf("expected false for missing attribute")
	}
}
