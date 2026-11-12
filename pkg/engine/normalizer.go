package engine

import (
	"strings"

	"open-parcels/pkg/models"
)

// IsDeliveredText checks if a status text represents an already completed delivery,
// being careful to avoid false positives like "being delivered", "to be delivered", "undelivered", etc.
func IsDeliveredText(desc string) bool {
	d := strings.ToLower(desc)

	// Filter out non-delivered contexts first
	if strings.Contains(d, "being delivered") ||
		strings.Contains(d, "to be delivered") ||
		strings.Contains(d, "undelivered") ||
		strings.Contains(d, "undeliverable") ||
		strings.Contains(d, "cannot be delivered") ||
		strings.Contains(d, "not delivered") ||
		strings.Contains(d, "could not be delivered") ||
		strings.Contains(d, "attempted") ||
		strings.Contains(d, "intentando") ||
		strings.Contains(d, "pendiente de entrega") ||
		strings.Contains(d, "en reparto") ||
		strings.Contains(d, "out for delivery") ||
		strings.Contains(d, "in consegna") ||
		strings.Contains(d, "recogido por el mensajero") ||
		strings.Contains(d, "recogido por mensajero") ||
		strings.Contains(d, "collected by the courier") ||
		strings.Contains(d, "collected by courier") ||
		strings.Contains(d, "será recogido") ||
		strings.Contains(d, "sera recogido") ||
		strings.Contains(d, "listo para ser recogido") ||
		strings.Contains(d, "courier will pick it up") ||
		strings.Contains(d, "soon our courier") ||
		strings.Contains(d, "will pick it up") {
		return false
	}

	return strings.Contains(d, "delivered") ||
		strings.Contains(d, "entregado") ||
		strings.Contains(d, "consegnato") ||
		strings.Contains(d, "consegnata") ||
		strings.Contains(d, "livré") ||
		strings.Contains(d, "zugestellt") ||
		strings.Contains(d, "in/at mailbox") ||
		strings.Contains(d, "signed by")
}

// NormalizeEventStatus inspects an event's text and classifies its status and lifecycle stage.
func NormalizeEventStatus(desc string) (models.Status, models.Stage) {
	d := strings.ToLower(desc)

	// Out for delivery / in delivery process (must be checked before delivered to catch "being delivered", etc.)
	if strings.Contains(d, "being delivered") ||
		strings.Contains(d, "to be delivered") ||
		strings.Contains(d, "out for delivery") ||
		strings.Contains(d, "en reparto") ||
		strings.Contains(d, "in consegna") ||
		strings.Contains(d, "en cours de livraison") ||
		strings.Contains(d, "in zustellung") ||
		strings.Contains(d, "with delivery courier") ||
		strings.Contains(d, "loaded onto delivery vehicle") ||
		strings.Contains(d, "repartidor") {
		return models.StatusOutForDelivery, models.StageLastMile
	}

	// Delivered
	if IsDeliveredText(d) {
		return models.StatusDelivered, models.StageDelivery
	}

	// Failed attempt
	if strings.Contains(d, "failed attempt") ||
		strings.Contains(d, "ausente") ||
		strings.Contains(d, "destinatario assente") ||
		strings.Contains(d, "assente") ||
		strings.Contains(d, "delivery attempted") ||
		strings.Contains(d, "destinatario ausente") ||
		strings.Contains(d, "missed delivery") ||
		strings.Contains(d, "notice left") {
		return models.StatusFailedAttempt, models.StageLastMile
	}

	// Exception / Delay / Customs issue
	if strings.Contains(d, "exception") ||
		strings.Contains(d, "delay") ||
		strings.Contains(d, "giacenza") ||
		strings.Contains(d, "retenido") ||
		strings.Contains(d, "incidencia") ||
		strings.Contains(d, "customs inspection") ||
		strings.Contains(d, "undeliverable") ||
		strings.Contains(d, "held in customs") {
		return models.StatusException, models.StageException
	}

	// Customs clearance
	if strings.Contains(d, "customs") ||
		strings.Contains(d, "aduana") ||
		strings.Contains(d, "dogana") ||
		strings.Contains(d, "sdoganamento") ||
		strings.Contains(d, "clearance") ||
		strings.Contains(d, "despacho") {
		return models.StatusInTransit, models.StageCustoms
	}

	// Handover / transfer to partner
	if strings.Contains(d, "hand over to local") ||
		strings.Contains(d, "handed over") ||
		strings.Contains(d, "transfer to partner") ||
		strings.Contains(d, "transferred to") {
		return models.StatusInTransit, models.StageHandover
	}

	// Picked up
	if strings.Contains(d, "picked up") ||
		strings.Contains(d, "admitido") ||
		strings.Contains(d, "preso in carico") ||
		strings.Contains(d, "presa in carico") ||
		strings.Contains(d, "recogido por el mensajero") ||
		strings.Contains(d, "recogido por mensajero") ||
		strings.Contains(d, "collected by the courier") ||
		strings.Contains(d, "collected by courier") {
		return models.StatusInTransit, models.StagePickedUp
	}

	// In transit
	if strings.Contains(d, "departed") ||
		strings.Contains(d, "arrived at") ||
		strings.Contains(d, "in transit") ||
		strings.Contains(d, "en tránsito") ||
		strings.Contains(d, "in lavorazione") ||
		strings.Contains(d, "sorting center") ||
		strings.Contains(d, "en camino") ||
		strings.Contains(d, "hub") {
		return models.StatusInTransit, models.StageTransit
	}

	// Info received / pre-registered
	if strings.Contains(d, "information received") ||
		strings.Contains(d, "pre-registrado") ||
		strings.Contains(d, "pre-registered") ||
		strings.Contains(d, "label created") ||
		strings.Contains(d, "datos del envío") ||
		strings.Contains(d, "sin recibir") ||
		strings.Contains(d, "electronic") {
		return models.StatusInfoReceived, models.StageCreated
	}

	return models.StatusInTransit, models.StageTransit
}

// LocalizedStatusText returns human-readable status in IT, ES, or EN.
func LocalizedStatusText(status models.Status, lang string) string {
	normalizedLang := strings.ToLower(strings.TrimSpace(lang))
	isIT := normalizedLang == "it" || strings.HasPrefix(normalizedLang, "it-") || strings.HasPrefix(normalizedLang, "it_")
	isES := normalizedLang == "es" || strings.HasPrefix(normalizedLang, "es-") || strings.HasPrefix(normalizedLang, "es_")

	switch status {
	case models.StatusPending:
		if isIT {
			return "In attesa di aggiornamento"
		}
		if isES {
			return "Pendiente de actualización"
		}
		return "Pending update"
	case models.StatusInfoReceived:
		if isIT {
			return "Informazioni ricevute / Spedizione registrata"
		}
		if isES {
			return "Información recibida / Pre-registrado"
		}
		return "Information received / Label created"
	case models.StatusInTransit:
		if isIT {
			return "In transito"
		}
		if isES {
			return "En tránsito"
		}
		return "In transit"
	case models.StatusOutForDelivery:
		if isIT {
			return "In consegna"
		}
		if isES {
			return "En reparto"
		}
		return "Out for delivery"
	case models.StatusDelivered:
		if isIT {
			return "Consegnato"
		}
		if isES {
			return "Entregado"
		}
		return "Delivered"
	case models.StatusFailedAttempt:
		if isIT {
			return "Tentativo di consegna fallito / Destinatario assente"
		}
		if isES {
			return "Intento de entrega fallido / Ausente"
		}
		return "Failed delivery attempt / Notice left"
	case models.StatusException:
		if isIT {
			return "Eccezione di spedizione / In giacenza"
		}
		if isES {
			return "Incidencia / Retenido"
		}
		return "Shipment exception / Delayed"
	case models.StatusExpired:
		if isIT {
			return "Scaduto / Nessun movimento"
		}
		if isES {
			return "Expirado / Sin movimiento"
		}
		return "Expired / Stale"
	default:
		if isIT {
			return "In transito"
		}
		if isES {
			return "En camino"
		}
		return "In transit"
	}
}
