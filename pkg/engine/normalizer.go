// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package engine

import (
	"strings"

	"open-parcels/pkg/models"
)

// NormalizeEventStatus inspects an event's text and classifies its status and lifecycle stage.
func NormalizeEventStatus(desc string) (models.Status, models.Stage) {
	d := strings.ToLower(desc)

	// Failed attempt (checked early so phrases like 'não entregue' aren't misclassified as delivered)
	if strings.Contains(d, "failed attempt") ||
		strings.Contains(d, "ausente") ||
		strings.Contains(d, "delivery attempted") ||
		strings.Contains(d, "destinatario ausente") ||
		strings.Contains(d, "destinatário ausente") ||
		strings.Contains(d, "missed delivery") ||
		strings.Contains(d, "notice left") ||
		strings.Contains(d, "tentativa de entrega") ||
		strings.Contains(d, "entrega falhada") ||
		strings.Contains(d, "sem sucesso") ||
		strings.Contains(d, "aviso deixado") ||
		strings.Contains(d, "não entregue") ||
		strings.Contains(d, "nao entregue") ||
		strings.Contains(d, "não conseguida") ||
		strings.Contains(d, "nao conseguida") {
		return models.StatusFailedAttempt, models.StageLastMile
	}

	// Exception / Delay / Customs issue (checked before normal customs clearance)
	if strings.Contains(d, "exception") ||
		strings.Contains(d, "delay") ||
		strings.Contains(d, "retenido") ||
		strings.Contains(d, "retido") ||
		strings.Contains(d, "incidencia") ||
		strings.Contains(d, "incidência") ||
		strings.Contains(d, "customs inspection") ||
		strings.Contains(d, "undeliverable") ||
		strings.Contains(d, "held in customs") ||
		strings.Contains(d, "atraso") ||
		strings.Contains(d, "morada incorreta") ||
		strings.Contains(d, "morada incorrecta") ||
		strings.Contains(d, "morada errada") ||
		strings.Contains(d, "endereço incorreto") ||
		strings.Contains(d, "endereco incorreto") ||
		strings.Contains(d, "devolvido") ||
		strings.Contains(d, "devolução") ||
		strings.Contains(d, "devolucao") ||
		strings.Contains(d, "extraviado") ||
		strings.Contains(d, "avaria") ||
		strings.Contains(d, "dano") {
		return models.StatusException, models.StageException
	}

	// Handover / transfer to partner (checked before delivered so 'entregue à transportadora' maps to handover)
	if strings.Contains(d, "hand over to local") ||
		strings.Contains(d, "handed over") ||
		strings.Contains(d, "transfer to partner") ||
		strings.Contains(d, "transferred to") ||
		strings.Contains(d, "transferido para") ||
		strings.Contains(d, "encaminhado para") ||
		strings.Contains(d, "entregue à transportadora") ||
		strings.Contains(d, "entregue a transportadora") ||
		strings.Contains(d, "entregue ao parceiro") ||
		strings.Contains(d, "entregue ao operador") {
		return models.StatusInTransit, models.StageHandover
	}

	// Delivered
	if strings.Contains(d, "delivered") ||
		strings.Contains(d, "entregado") ||
		strings.Contains(d, "entregue") ||
		strings.Contains(d, "entrega efetuada") ||
		strings.Contains(d, "entrega efectuada") ||
		strings.Contains(d, "livré") ||
		strings.Contains(d, "zugestellt") ||
		strings.Contains(d, "in/at mailbox") ||
		strings.Contains(d, "signed by") ||
		strings.Contains(d, "assinado por") ||
		strings.Contains(d, "distribuído") ||
		strings.Contains(d, "distribuido") {
		return models.StatusDelivered, models.StageDelivery
	}

	// Out for delivery
	if strings.Contains(d, "out for delivery") ||
		strings.Contains(d, "en reparto") ||
		strings.Contains(d, "en cours de livraison") ||
		strings.Contains(d, "in zustellung") ||
		strings.Contains(d, "with delivery courier") ||
		strings.Contains(d, "loaded onto delivery vehicle") ||
		strings.Contains(d, "repartidor") ||
		strings.Contains(d, "em distribuição") ||
		strings.Contains(d, "em distribuicao") ||
		strings.Contains(d, "em rota de entrega") ||
		strings.Contains(d, "em rota para entrega") ||
		strings.Contains(d, "em rota") ||
		strings.Contains(d, "saída para entrega") ||
		strings.Contains(d, "saida para entrega") ||
		strings.Contains(d, "com o estafeta") ||
		strings.Contains(d, "estafeta") {
		return models.StatusOutForDelivery, models.StageLastMile
	}

	// Customs clearance
	if strings.Contains(d, "customs") ||
		strings.Contains(d, "aduana") ||
		strings.Contains(d, "clearance") ||
		strings.Contains(d, "despacho") ||
		strings.Contains(d, "alfândega") ||
		strings.Contains(d, "alfandega") ||
		strings.Contains(d, "desalfandegamento") ||
		strings.Contains(d, "controlo aduaneiro") ||
		strings.Contains(d, "aduaneiro") {
		return models.StatusInTransit, models.StageCustoms
	}

	// Info received / pre-registered (checked before generic transit to match 'informação sobre o envio recebida')
	if strings.Contains(d, "information received") ||
		strings.Contains(d, "pre-registrado") ||
		strings.Contains(d, "pre-registered") ||
		strings.Contains(d, "label created") ||
		strings.Contains(d, "datos del envío") ||
		strings.Contains(d, "sin recibir") ||
		strings.Contains(d, "electronic") ||
		strings.Contains(d, "eletronicamente") ||
		strings.Contains(d, "eletrónico") ||
		strings.Contains(d, "eletronico") ||
		strings.Contains(d, "pré-registado") ||
		strings.Contains(d, "pre-registado") ||
		strings.Contains(d, "pré-registada") ||
		strings.Contains(d, "pre-registada") ||
		strings.Contains(d, "informação sobre o envio") ||
		strings.Contains(d, "informacao sobre o envio") ||
		strings.Contains(d, "informação recebida") ||
		strings.Contains(d, "informacao recebida") ||
		strings.Contains(d, "dados do envio") ||
		strings.Contains(d, "guia registada") ||
		strings.Contains(d, "envio criado") ||
		strings.Contains(d, "etiqueta criada") ||
		strings.Contains(d, "aguarda receção") ||
		strings.Contains(d, "aguarda recepção") ||
		strings.Contains(d, "aguarda envio") {
		return models.StatusInfoReceived, models.StageCreated
	}

	// Picked up / In transit
	if strings.Contains(d, "picked up") ||
		strings.Contains(d, "admitido") ||
		strings.Contains(d, "departed") ||
		strings.Contains(d, "arrived at") ||
		strings.Contains(d, "in transit") ||
		strings.Contains(d, "en tránsito") ||
		strings.Contains(d, "en transito") ||
		strings.Contains(d, "sorting center") ||
		strings.Contains(d, "en camino") ||
		strings.Contains(d, "hub") ||
		strings.Contains(d, "em trânsito") ||
		strings.Contains(d, "em transito") ||
		strings.Contains(d, "aceite") ||
		strings.Contains(d, "recebido") ||
		strings.Contains(d, "recebida") ||
		strings.Contains(d, "recolha") ||
		strings.Contains(d, "expedição") ||
		strings.Contains(d, "expedicao") ||
		strings.Contains(d, "em transferência") ||
		strings.Contains(d, "em transferencia") ||
		strings.Contains(d, "em viagem") ||
		strings.Contains(d, "a caminho") {
		return models.StatusInTransit, models.StageTransit
	}

	return models.StatusInTransit, models.StageTransit
}

// LocalizedStatusText returns human-readable status in ES, PT, or EN.
func LocalizedStatusText(status models.Status, lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	isPT := l == "pt" || strings.HasPrefix(l, "pt-") || strings.HasPrefix(l, "pt_")
	isES := l == "es" || strings.HasPrefix(l, "es-") || strings.HasPrefix(l, "es_")

	switch status {
	case models.StatusPending:
		if isPT {
			return "Pendente de atualização"
		}
		if isES {
			return "Pendiente de actualización"
		}
		return "Pending update"
	case models.StatusInfoReceived:
		if isPT {
			return "Informação recebida / Pré-registado"
		}
		if isES {
			return "Información recibida / Pre-registrado"
		}
		return "Information received / Label created"
	case models.StatusInTransit:
		if isPT {
			return "Em trânsito"
		}
		if isES {
			return "En tránsito"
		}
		return "In transit"
	case models.StatusOutForDelivery:
		if isPT {
			return "Em distribuição"
		}
		if isES {
			return "En reparto"
		}
		return "Out for delivery"
	case models.StatusDelivered:
		if isPT {
			return "Entregue"
		}
		if isES {
			return "Entregado"
		}
		return "Delivered"
	case models.StatusFailedAttempt:
		if isPT {
			return "Tentativa de entrega falhada / Ausente"
		}
		if isES {
			return "Intento de entrega fallido / Ausente"
		}
		return "Failed delivery attempt / Notice left"
	case models.StatusException:
		if isPT {
			return "Incidência / Retido"
		}
		if isES {
			return "Incidencia / Retenido"
		}
		return "Shipment exception / Delayed"
	case models.StatusExpired:
		if isPT {
			return "Expirado / Sem movimento"
		}
		if isES {
			return "Expirado / Sin movimiento"
		}
		return "Expired / Stale"
	default:
		if isPT {
			return "Em trânsito"
		}
		if isES {
			return "En camino"
		}
		return "In transit"
	}
}
