// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"testing"

	"open-parcels/pkg/models"
)

func TestNormalizeEventStatusPortuguese(t *testing.T) {
	tests := []struct {
		desc           string
		expectedStatus models.Status
		expectedStage  models.Stage
	}{
		// Delivered
		{"Objeto entregue", models.StatusDelivered, models.StageDelivery},
		{"Encomenda entregue ao destinatário", models.StatusDelivered, models.StageDelivery},
		{"Entrega efetuada com sucesso", models.StatusDelivered, models.StageDelivery},
		{"Objeto distribuído ao cliente", models.StatusDelivered, models.StageDelivery},
		{"Assinado por Silva", models.StatusDelivered, models.StageDelivery},

		// Out for delivery
		{"Objeto em distribuição - em rota de entrega postal", models.StatusOutForDelivery, models.StageLastMile},
		{"Em distribuição - Encomenda com o estafeta DPD para entrega", models.StatusOutForDelivery, models.StageLastMile},
		{"Em distribuição - Em rota para entrega ao destinatário / Ponto K", models.StatusOutForDelivery, models.StageLastMile},
		{"Em distribuição - Encomenda em rota de entrega final", models.StatusOutForDelivery, models.StageLastMile},
		{"Saída para entrega ao domicílio", models.StatusOutForDelivery, models.StageLastMile},
		{"Encomenda em rota de entrega", models.StatusOutForDelivery, models.StageLastMile},

		// Failed attempt
		{"Tentativa de entrega falhada - destinatário ausente", models.StatusFailedAttempt, models.StageLastMile},
		{"Tentativa de entrega sem sucesso", models.StatusFailedAttempt, models.StageLastMile},
		{"Destinatário ausente na morada indicada", models.StatusFailedAttempt, models.StageLastMile},
		{"Aviso deixado na caixa de correio", models.StatusFailedAttempt, models.StageLastMile},
		{"Objeto não entregue", models.StatusFailedAttempt, models.StageLastMile},

		// Exception / Delay / Customs issue
		{"Objeto retido na alfândega para verificação", models.StatusException, models.StageException},
		{"Envio retido para inspeção", models.StatusException, models.StageException},
		{"Incidência no processo de entrega", models.StatusException, models.StageException},
		{"Atraso no transporte operacional", models.StatusException, models.StageException},
		{"Morada incorreta ou incompleta", models.StatusException, models.StageException},
		{"Devolvido ao remetente", models.StatusException, models.StageException},

		// Customs clearance
		{"Desalfandegamento concluído com sucesso", models.StatusInTransit, models.StageCustoms},
		{"Objeto em controlo aduaneiro", models.StatusInTransit, models.StageCustoms},
		{"Em desalfandegamento no aeroporto de Lisboa", models.StatusInTransit, models.StageCustoms},
		{"Despacho aduaneiro finalizado", models.StatusInTransit, models.StageCustoms},

		// Handover
		{"Transferido para operador local", models.StatusInTransit, models.StageHandover},
		{"Encaminhado para parceiro de distribuição", models.StatusInTransit, models.StageHandover},
		{"Entregue à transportadora de destino", models.StatusInTransit, models.StageHandover},

		// Picked up / In transit
		{"Objeto aceite no posto de correios", models.StatusInTransit, models.StageTransit},
		{"Objeto em trânsito no centro de processamento e logística", models.StatusInTransit, models.StageTransit},
		{"Encomenda recebida no centro de triagem da DPD", models.StatusInTransit, models.StageTransit},
		{"Encomenda em trânsito para o centro operacional de entrega", models.StatusInTransit, models.StageTransit},
		{"Expedição recebida na plataforma central VASP", models.StatusInTransit, models.StageTransit},
		{"Em transferência para o centro operacional regional", models.StatusInTransit, models.StageTransit},
		{"Recolha efetuada e encomenda recebida no armazém Delnext", models.StatusInTransit, models.StageTransit},
		{"Em trânsito para o centro de distribuição da zona de entrega", models.StatusInTransit, models.StageTransit},
		{"Em viagem para Portugal", models.StatusInTransit, models.StageTransit},

		// Info received / pre-registered
		{"Informação sobre o envio recebida eletronicamente", models.StatusInfoReceived, models.StageCreated},
		{"Dados do envio transmitidos eletronicamente à DPD", models.StatusInfoReceived, models.StageCreated},
		{"Guia registada no sistema VASP Expresso", models.StatusInfoReceived, models.StageCreated},
		{"Envio criado no portal Delnext", models.StatusInfoReceived, models.StageCreated},
		{"Envio pré-registado no sistema", models.StatusInfoReceived, models.StageCreated},
		{"Etiqueta criada pelo remetente", models.StatusInfoReceived, models.StageCreated},
		{"Aguarda receção no centro de distribuição", models.StatusInfoReceived, models.StageCreated},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			status, stage := NormalizeEventStatus(tt.desc)
			if status != tt.expectedStatus {
				t.Errorf("NormalizeEventStatus(%q) status = %v, want %v", tt.desc, status, tt.expectedStatus)
			}
			if stage != tt.expectedStage {
				t.Errorf("NormalizeEventStatus(%q) stage = %v, want %v", tt.desc, stage, tt.expectedStage)
			}
		})
	}
}

func TestNormalizeEventStatusRegressions(t *testing.T) {
	tests := []struct {
		desc           string
		expectedStatus models.Status
		expectedStage  models.Stage
	}{
		{"Delivered to front door", models.StatusDelivered, models.StageDelivery},
		{"Entregado en buzón", models.StatusDelivered, models.StageDelivery},
		{"Colis livré", models.StatusDelivered, models.StageDelivery},
		{"Sendung zugestellt", models.StatusDelivered, models.StageDelivery},
		{"Out for delivery with courier", models.StatusOutForDelivery, models.StageLastMile},
		{"En reparto con el cartero", models.StatusOutForDelivery, models.StageLastMile},
		{"Delivery attempted - notice left", models.StatusFailedAttempt, models.StageLastMile},
		{"Destinatario ausente", models.StatusFailedAttempt, models.StageLastMile},
		{"Held in customs inspection", models.StatusException, models.StageException},
		{"Incidencia en el envío", models.StatusException, models.StageException},
		{"Customs clearance completed", models.StatusInTransit, models.StageCustoms},
		{"Handed over to local delivery partner", models.StatusInTransit, models.StageHandover},
		{"In transit to sorting center", models.StatusInTransit, models.StageTransit},
		{"Pre-registered electronically", models.StatusInfoReceived, models.StageCreated},
		{"Random unrecognized status message", models.StatusInTransit, models.StageTransit},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			status, stage := NormalizeEventStatus(tt.desc)
			if status != tt.expectedStatus {
				t.Errorf("NormalizeEventStatus(%q) status = %v, want %v", tt.desc, status, tt.expectedStatus)
			}
			if stage != tt.expectedStage {
				t.Errorf("NormalizeEventStatus(%q) stage = %v, want %v", tt.desc, stage, tt.expectedStage)
			}
		})
	}
}

func TestLocalizedStatusTextPortuguese(t *testing.T) {
	tests := []struct {
		status   models.Status
		lang     string
		expected string
	}{
		{models.StatusPending, "pt", "Pendente de atualização"},
		{models.StatusPending, "PT", "Pendente de atualização"},
		{models.StatusPending, "pt-PT", "Pendente de atualização"},
		{models.StatusInfoReceived, "pt", "Informação recebida / Pré-registado"},
		{models.StatusInTransit, "pt", "Em trânsito"},
		{models.StatusOutForDelivery, "pt", "Em distribuição"},
		{models.StatusDelivered, "pt", "Entregue"},
		{models.StatusFailedAttempt, "pt", "Tentativa de entrega falhada / Ausente"},
		{models.StatusException, "pt", "Incidência / Retido"},
		{models.StatusExpired, "pt", "Expirado / Sem movimento"},
		{models.Status("unknown_status"), "pt", "Em trânsito"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status)+"_"+tt.lang, func(t *testing.T) {
			got := LocalizedStatusText(tt.status, tt.lang)
			if got != tt.expected {
				t.Errorf("LocalizedStatusText(%q, %q) = %q, want %q", tt.status, tt.lang, got, tt.expected)
			}
		})
	}
}

func TestLocalizedStatusTextLanguages(t *testing.T) {
	// Spanish
	if got := LocalizedStatusText(models.StatusDelivered, "es"); got != "Entregado" {
		t.Errorf("expected 'Entregado', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusPending, "es"); got != "Pendiente de actualización" {
		t.Errorf("expected 'Pendiente de actualización', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusOutForDelivery, "es"); got != "En reparto" {
		t.Errorf("expected 'En reparto', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusInfoReceived, "es"); got != "Información recibida / Pre-registrado" {
		t.Errorf("expected 'Información recibida / Pre-registrado', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusFailedAttempt, "es"); got != "Intento de entrega fallido / Ausente" {
		t.Errorf("expected 'Intento de entrega fallido / Ausente', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusException, "es"); got != "Incidencia / Retenido" {
		t.Errorf("expected 'Incidencia / Retenido', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusExpired, "es"); got != "Expirado / Sin movimiento" {
		t.Errorf("expected 'Expirado / Sin movimiento', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusInTransit, "es"); got != "En tránsito" {
		t.Errorf("expected 'En tránsito', got %q", got)
	}
	if got := LocalizedStatusText("other", "es"); got != "En camino" {
		t.Errorf("expected 'En camino', got %q", got)
	}

	// English (default)
	if got := LocalizedStatusText(models.StatusDelivered, "en"); got != "Delivered" {
		t.Errorf("expected 'Delivered', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusPending, "en"); got != "Pending update" {
		t.Errorf("expected 'Pending update', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusOutForDelivery, "en"); got != "Out for delivery" {
		t.Errorf("expected 'Out for delivery', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusInfoReceived, "en"); got != "Information received / Label created" {
		t.Errorf("expected 'Information received / Label created', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusFailedAttempt, "en"); got != "Failed delivery attempt / Notice left" {
		t.Errorf("expected 'Failed delivery attempt / Notice left', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusException, "en"); got != "Shipment exception / Delayed" {
		t.Errorf("expected 'Shipment exception / Delayed', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusExpired, "en"); got != "Expired / Stale" {
		t.Errorf("expected 'Expired / Stale', got %q", got)
	}
	if got := LocalizedStatusText(models.StatusInTransit, "en"); got != "In transit" {
		t.Errorf("expected 'In transit', got %q", got)
	}
	if got := LocalizedStatusText("other", "en"); got != "In transit" {
		t.Errorf("expected 'In transit', got %q", got)
	}
}
