// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"testing"

	"open-parcels/pkg/models"
)

func TestNormalizeEventStatusItalianKeywords(t *testing.T) {
	tests := []struct {
		desc           string
		expectedStatus models.Status
		expectedStage  models.Stage
	}{
		// Delivered (consegnato)
		{
			desc:           "Pacco consegnato al destinatario",
			expectedStatus: models.StatusDelivered,
			expectedStage:  models.StageDelivery,
		},
		{
			desc:           "Spedizione consegnata con successo",
			expectedStatus: models.StatusDelivered,
			expectedStage:  models.StageDelivery,
		},
		{
			desc:           "CONSEGNATO",
			expectedStatus: models.StatusDelivered,
			expectedStage:  models.StageDelivery,
		},

		// Out for delivery (in consegna)
		{
			desc:           "Spedizione in consegna nella giornata odierna",
			expectedStatus: models.StatusOutForDelivery,
			expectedStage:  models.StageLastMile,
		},
		{
			desc:           "In consegna con incaricato BRT",
			expectedStatus: models.StatusOutForDelivery,
			expectedStage:  models.StageLastMile,
		},
		{
			desc:           "La spedizione è in consegna con l'autista GLS",
			expectedStatus: models.StatusOutForDelivery,
			expectedStage:  models.StageLastMile,
		},

		// Picked up (preso in carico / presa in carico)
		{
			desc:           "Preso in carico all'ufficio postale",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StagePickedUp,
		},
		{
			desc:           "Preso in carico - Spedizione ritirata dal corriere",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StagePickedUp,
		},
		{
			desc:           "La spedizione è stata presa in carico da SDA",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StagePickedUp,
		},

		// In transit / In lavorazione
		{
			desc:           "In lavorazione presso il centro operativo postale",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StageTransit,
		},
		{
			desc:           "In lavorazione - Arrivata nella filiale di smistamento",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StageTransit,
		},
		{
			desc:           "In lavorazione presso il CMP di San Marino",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StageTransit,
		},

		// Customs clearance (dogana / sdoganamento)
		{
			desc:           "Spedizione arrivata in dogana",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StageCustoms,
		},
		{
			desc:           "Controllo dogana completato",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StageCustoms,
		},
		{
			desc:           "Pratica di sdoganamento completata",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StageCustoms,
		},
		{
			desc:           "Operazioni di sdoganamento in corso",
			expectedStatus: models.StatusInTransit,
			expectedStage:  models.StageCustoms,
		},

		// Failed attempt (destinatario assente)
		{
			desc:           "Mancata consegna: destinatario assente",
			expectedStatus: models.StatusFailedAttempt,
			expectedStage:  models.StageLastMile,
		},
		{
			desc:           "Tentativo fallito - destinatario assente al passaggio",
			expectedStatus: models.StatusFailedAttempt,
			expectedStage:  models.StageLastMile,
		},

		// Exception (giacenza)
		{
			desc:           "Spedizione in giacenza presso l'ufficio postale",
			expectedStatus: models.StatusException,
			expectedStage:  models.StageException,
		},
		{
			desc:           "Apertura pratica di giacenza per mancata consegna",
			expectedStatus: models.StatusException,
			expectedStage:  models.StageException,
		},
		{
			desc:           "Merce in giacenza svincolata",
			expectedStatus: models.StatusException,
			expectedStage:  models.StageException,
		},
	}

	for _, tc := range tests {
		status, stage := NormalizeEventStatus(tc.desc)
		if status != tc.expectedStatus || stage != tc.expectedStage {
			t.Errorf("NormalizeEventStatus(%q) = (%v, %v); expected (%v, %v)",
				tc.desc, status, stage, tc.expectedStatus, tc.expectedStage)
		}
	}
}

func TestLocalizedStatusTextItalian(t *testing.T) {
	// Acceptance criterion: LocalizedStatusText(models.StatusDelivered, "it") retorna 'Consegnato'
	if got := LocalizedStatusText(models.StatusDelivered, "it"); got != "Consegnato" {
		t.Errorf("LocalizedStatusText(StatusDelivered, 'it') = %q, expected 'Consegnato'", got)
	}

	// Verify all other statuses return non-empty polished Italian text
	statuses := []models.Status{
		models.StatusPending,
		models.StatusInfoReceived,
		models.StatusInTransit,
		models.StatusOutForDelivery,
		models.StatusDelivered,
		models.StatusFailedAttempt,
		models.StatusException,
		models.StatusExpired,
		models.Status("unknown_status"),
	}

	for _, s := range statuses {
		textIT := LocalizedStatusText(s, "it")
		if textIT == "" {
			t.Errorf("LocalizedStatusText(%q, 'it') returned empty string", s)
		}

		// Check uppercase "IT" and locale "it-IT"
		if textUpper := LocalizedStatusText(s, "IT"); textUpper != textIT {
			t.Errorf("LocalizedStatusText(%q, 'IT') = %q, expected %q", s, textUpper, textIT)
		}
		if textLocale := LocalizedStatusText(s, "it-IT"); textLocale != textIT {
			t.Errorf("LocalizedStatusText(%q, 'it-IT') = %q, expected %q", s, textLocale, textIT)
		}

		// Ensure it differs from English fallback for recognized statuses
		textEN := LocalizedStatusText(s, "en")
		if s != models.Status("unknown_status") && textIT == textEN {
			t.Errorf("LocalizedStatusText(%q, 'it') returned English fallback %q", s, textIT)
		}
	}

	// Check specific expected Italian translations
	expectedIT := map[models.Status]string{
		models.StatusPending:        "In attesa di aggiornamento",
		models.StatusInfoReceived:   "Informazioni ricevute / Spedizione registrata",
		models.StatusInTransit:      "In transito",
		models.StatusOutForDelivery: "In consegna",
		models.StatusDelivered:      "Consegnato",
		models.StatusFailedAttempt:  "Tentativo di consegna fallito / Destinatario assente",
		models.StatusException:      "Eccezione di spedizione / In giacenza",
		models.StatusExpired:        "Scaduto / Nessun movimento",
	}

	for s, expected := range expectedIT {
		if got := LocalizedStatusText(s, "it"); got != expected {
			t.Errorf("LocalizedStatusText(%q, 'it') = %q, expected %q", s, got, expected)
		}
	}
}

func TestNormalizeEventStatusOtherLanguages(t *testing.T) {
	tests := []struct {
		desc           string
		expectedStatus models.Status
		expectedStage  models.Stage
	}{
		{"Delivered to front porch", models.StatusDelivered, models.StageDelivery},
		{"Signed by John Doe", models.StatusDelivered, models.StageDelivery},
		{"In/at mailbox", models.StatusDelivered, models.StageDelivery},
		{"Colis livré", models.StatusDelivered, models.StageDelivery},
		{"Paket zugestellt", models.StatusDelivered, models.StageDelivery},
		{"Envío entregado al destinatario", models.StatusDelivered, models.StageDelivery},

		{"Out for delivery today", models.StatusOutForDelivery, models.StageLastMile},
		{"En reparto", models.StatusOutForDelivery, models.StageLastMile},
		{"En cours de livraison", models.StatusOutForDelivery, models.StageLastMile},
		{"In Zustellung", models.StatusOutForDelivery, models.StageLastMile},
		{"Loaded onto delivery vehicle", models.StatusOutForDelivery, models.StageLastMile},
		{"Asignado a repartidor", models.StatusOutForDelivery, models.StageLastMile},
		{"With delivery courier", models.StatusOutForDelivery, models.StageLastMile},

		{"Failed attempt to deliver", models.StatusFailedAttempt, models.StageLastMile},
		{"Destinatario ausente", models.StatusFailedAttempt, models.StageLastMile},
		{"Delivery attempted", models.StatusFailedAttempt, models.StageLastMile},
		{"Missed delivery", models.StatusFailedAttempt, models.StageLastMile},
		{"Notice left for customer", models.StatusFailedAttempt, models.StageLastMile},

		{"Shipment exception occurred", models.StatusException, models.StageException},
		{"Package delay in transit", models.StatusException, models.StageException},
		{"Paquete retenido", models.StatusException, models.StageException},
		{"Incidencia en el envío", models.StatusException, models.StageException},
		{"Customs inspection required", models.StatusException, models.StageException},
		{"Undeliverable parcel", models.StatusException, models.StageException},
		{"Held in customs", models.StatusException, models.StageException},

		{"Customs clearance processing", models.StatusInTransit, models.StageCustoms},
		{"Llegada a aduana", models.StatusInTransit, models.StageCustoms},
		{"Despacho aduanero", models.StatusInTransit, models.StageCustoms},

		{"Hand over to local courier", models.StatusInTransit, models.StageHandover},
		{"Handed over to carrier", models.StatusInTransit, models.StageHandover},
		{"Transfer to partner carrier", models.StatusInTransit, models.StageHandover},
		{"Transferred to destination post", models.StatusInTransit, models.StageHandover},

		{"Picked up from sender", models.StatusInTransit, models.StagePickedUp},
		{"Admitido en oficina", models.StatusInTransit, models.StagePickedUp},
		{"Departed from facility", models.StatusInTransit, models.StageTransit},
		{"Arrived at distribution center", models.StatusInTransit, models.StageTransit},
		{"Sorting center processing", models.StatusInTransit, models.StageTransit},
		{"En camino a destino", models.StatusInTransit, models.StageTransit},
		{"Package at hub", models.StatusInTransit, models.StageTransit},

		{"Electronic information received", models.StatusInfoReceived, models.StageCreated},
		{"Pre-registrado en el sistema", models.StatusInfoReceived, models.StageCreated},
		{"Pre-registered shipment", models.StatusInfoReceived, models.StageCreated},
		{"Shipping label created", models.StatusInfoReceived, models.StageCreated},
		{"Datos del envío registrados", models.StatusInfoReceived, models.StageCreated},
		{"Sin recibir por el transportista", models.StatusInfoReceived, models.StageCreated},

		{"Something unusual and completely unknown", models.StatusInTransit, models.StageTransit},
	}

	for _, tc := range tests {
		status, stage := NormalizeEventStatus(tc.desc)
		if status != tc.expectedStatus || stage != tc.expectedStage {
			t.Errorf("NormalizeEventStatus(%q) = (%v, %v); expected (%v, %v)",
				tc.desc, status, stage, tc.expectedStatus, tc.expectedStage)
		}
	}
}

func TestLocalizedStatusTextEnglishAndSpanish(t *testing.T) {
	statuses := []models.Status{
		models.StatusPending,
		models.StatusInfoReceived,
		models.StatusInTransit,
		models.StatusOutForDelivery,
		models.StatusDelivered,
		models.StatusFailedAttempt,
		models.StatusException,
		models.StatusExpired,
		models.Status("other"),
	}

	for _, s := range statuses {
		en := LocalizedStatusText(s, "en")
		es := LocalizedStatusText(s, "es")
		esUpper := LocalizedStatusText(s, "ES")
		esLocale := LocalizedStatusText(s, "es-ES")

		if en == "" {
			t.Errorf("expected non-empty EN text for status %q", s)
		}
		if es == "" {
			t.Errorf("expected non-empty ES text for status %q", s)
		}
		if es != esUpper || es != esLocale {
			t.Errorf("expected case-insensitive and locale matching for ES: %q vs %q vs %q", es, esUpper, esLocale)
		}
	}
}
