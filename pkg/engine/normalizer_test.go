// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"testing"

	"open-parcels/pkg/models"
)

func TestNormalizeEventStatusMaritime(t *testing.T) {
	tests := []struct {
		desc          string
		expectedStat  models.Status
		expectedStage models.Stage
	}{
		{
			desc:          "Gate in full at origin port terminal",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StagePickedUp,
		},
		{
			desc:          "Gate-in full at Shanghai terminal",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StagePickedUp,
		},
		{
			desc:          "Loaded on board vessel MSC OSCAR",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StageTransit,
		},
		{
			desc:          "Laden on board vessel EVER GIVEN",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StageTransit,
		},
		{
			desc:          "Vessel departed origin port / Underway",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StageTransit,
		},
		{
			desc:          "Vessel in transit on high seas / Transshipment leg",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StageTransit,
		},
		{
			desc:          "Container discharged from vessel at destination port",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StageTransit,
		},
		{
			desc:          "Customs clearance completed at destination port",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StageCustoms,
		},
		{
			desc:          "Despacho de aduanas completado",
			expectedStat:  models.StatusInTransit,
			expectedStage: models.StageCustoms,
		},
		{
			desc:          "Gate-out full from terminal for delivery to consignee",
			expectedStat:  models.StatusOutForDelivery,
			expectedStage: models.StageLastMile,
		},
		{
			desc:          "Gate out full",
			expectedStat:  models.StatusOutForDelivery,
			expectedStage: models.StageLastMile,
		},
		{
			desc:          "Empty return to container depot",
			expectedStat:  models.StatusDelivered,
			expectedStage: models.StageDelivery,
		},
		{
			desc:          "Empty container returned to carrier yard",
			expectedStat:  models.StatusDelivered,
			expectedStage: models.StageDelivery,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			stat, stage := NormalizeEventStatus(tc.desc)
			if stat != tc.expectedStat {
				t.Errorf("for %q, expected status %s, got %s", tc.desc, tc.expectedStat, stat)
			}
			if stage != tc.expectedStage {
				t.Errorf("for %q, expected stage %s, got %s", tc.desc, tc.expectedStage, stage)
			}
		})
	}
}

func TestNormalizeEventStatusStandard(t *testing.T) {
	tests := []struct {
		desc          string
		expectedStat  models.Status
		expectedStage models.Stage
	}{
		{"Delivered to recipient mailbox", models.StatusDelivered, models.StageDelivery},
		{"Entregado al destinatario", models.StatusDelivered, models.StageDelivery},
		{"Colis livré", models.StatusDelivered, models.StageDelivery},
		{"Zugestellt an Empfänger", models.StatusDelivered, models.StageDelivery},
		{"In/at mailbox", models.StatusDelivered, models.StageDelivery},
		{"Signed by John Doe", models.StatusDelivered, models.StageDelivery},

		{"Out for delivery", models.StatusOutForDelivery, models.StageLastMile},
		{"En reparto con el repartidor", models.StatusOutForDelivery, models.StageLastMile},
		{"En cours de livraison", models.StatusOutForDelivery, models.StageLastMile},
		{"In Zustellung", models.StatusOutForDelivery, models.StageLastMile},
		{"With delivery courier", models.StatusOutForDelivery, models.StageLastMile},
		{"Loaded onto delivery vehicle", models.StatusOutForDelivery, models.StageLastMile},

		{"Delivery attempted - Notice left", models.StatusFailedAttempt, models.StageLastMile},
		{"Destinatario ausente", models.StatusFailedAttempt, models.StageLastMile},
		{"Delivery failed attempt", models.StatusFailedAttempt, models.StageLastMile},
		{"Missed delivery", models.StatusFailedAttempt, models.StageLastMile},

		{"Shipment exception - address incorrect", models.StatusException, models.StageException},
		{"Customs inspection delay", models.StatusException, models.StageException},
		{"Paquete retenido en aduana", models.StatusException, models.StageException},
		{"Incidencia en el envío", models.StatusException, models.StageException},
		{"Undeliverable package", models.StatusException, models.StageException},
		{"Held in customs", models.StatusException, models.StageException},

		{"Hand over to local partner", models.StatusInTransit, models.StageHandover},
		{"Handed over to carrier", models.StatusInTransit, models.StageHandover},
		{"Transfer to partner network", models.StatusInTransit, models.StageHandover},
		{"Transferred to local hub", models.StatusInTransit, models.StageHandover},

		{"Package picked up by courier", models.StatusInTransit, models.StagePickedUp},
		{"Envío admitido en origen", models.StatusInTransit, models.StagePickedUp},
		{"Departed from sorting facility", models.StatusInTransit, models.StageTransit},
		{"Arrived at central hub", models.StatusInTransit, models.StageTransit},
		{"In transit to destination country", models.StatusInTransit, models.StageTransit},
		{"En tránsito hacia la delegación", models.StatusInTransit, models.StageTransit},
		{"En camino a destino", models.StatusInTransit, models.StageTransit},

		{"Electronic information received", models.StatusInfoReceived, models.StageCreated},
		{"Envío pre-registrado en el sistema", models.StatusInfoReceived, models.StageCreated},
		{"Pre-registered shipping label", models.StatusInfoReceived, models.StageCreated},
		{"Shipping label created", models.StatusInfoReceived, models.StageCreated},
		{"Datos del envío registrados", models.StatusInfoReceived, models.StageCreated},
		{"Envío sin recibir", models.StatusInfoReceived, models.StageCreated},

		{"Unknown carrier update message", models.StatusInTransit, models.StageTransit},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			stat, stage := NormalizeEventStatus(tc.desc)
			if stat != tc.expectedStat {
				t.Errorf("for %q, expected status %s, got %s", tc.desc, tc.expectedStat, stat)
			}
			if stage != tc.expectedStage {
				t.Errorf("for %q, expected stage %s, got %s", tc.desc, tc.expectedStage, stage)
			}
		})
	}
}

func TestLocalizedStatusText(t *testing.T) {
	statuses := []models.Status{
		models.StatusPending,
		models.StatusInfoReceived,
		models.StatusInTransit,
		models.StatusOutForDelivery,
		models.StatusDelivered,
		models.StatusFailedAttempt,
		models.StatusException,
		models.StatusExpired,
		models.Status("unknown_stat"),
	}

	for _, st := range statuses {
		enText := LocalizedStatusText(st, "en")
		esText := LocalizedStatusText(st, "es")

		if enText == "" {
			t.Errorf("expected non-empty EN text for status %s", st)
		}
		if esText == "" {
			t.Errorf("expected non-empty ES text for status %s", st)
		}
		if st != "unknown_stat" && enText == esText {
			t.Errorf("expected EN and ES text to differ for %s: en=%q, es=%q", st, enText, esText)
		}
	}
}
