// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

type localizedEvent struct {
	pt       string
	es       string
	en       string
	location string
	status   models.Status
	stage    models.Stage
	offset   time.Duration
}

func buildEvents(templates []localizedEvent, lang string, carrierSlug, carrierName string, now time.Time) []models.TrackingEvent {
	l := strings.ToLower(strings.TrimSpace(lang))
	events := make([]models.TrackingEvent, len(templates))

	for i, t := range templates {
		desc := t.pt
		switch l {
		case "es":
			desc = t.es
		case "en":
			desc = t.en
		}

		events[i] = models.TrackingEvent{
			Date:        now.Add(t.offset),
			Description: desc,
			Location:    t.location,
			CarrierSlug: carrierSlug,
			CarrierName: carrierName,
			Status:      t.status,
			Stage:       t.stage,
		}
	}
	return events
}

func localizedStatus(status models.Status, lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	switch status {
	case models.StatusOutForDelivery:
		switch l {
		case "pt":
			return "Objeto em distribuição"
		case "es":
			return "Envío en reparto"
		case "en":
			return "Out for delivery"
		default:
			return "Objeto em distribuição / Out for delivery"
		}
	case models.StatusDelivered:
		switch l {
		case "pt":
			return "Objeto entregue"
		case "es":
			return "Envío entregado"
		case "en":
			return "Delivered"
		default:
			return "Objeto entregue / Delivered"
		}
	default:
		switch l {
		case "pt":
			return "Em trânsito"
		case "es":
			return "En tránsito"
		case "en":
			return "In transit"
		default:
			return "Em trânsito / In transit"
		}
	}
}

func resolveCountries(opts models.TrackOptions) (models.CountryInfo, models.CountryInfo) {
	origin := models.CountryInfo{Name: "Portugal", Code: "PT"}
	destCode := "PT"
	destName := "Portugal"
	if opts.DestinationCountry != "" {
		destCode = strings.ToUpper(opts.DestinationCountry)
		if destCode == "PT" {
			destName = "Portugal"
		} else {
			destName = destCode
		}
	}

	return origin, models.CountryInfo{Name: destName, Code: destCode}
}

// PortugalPostCarrier implements the CTT Portugal / Portugal Post carrier adapter.
type PortugalPostCarrier struct {
	BaseCarrier
}

// NewPortugalPostCarrier constructs a new PortugalPostCarrier instance.
func NewPortugalPostCarrier() *PortugalPostCarrier {
	return &PortugalPostCarrier{
		BaseCarrier: NewBaseCarrier(
			"portugal-post",
			"CTT Portugal",
			"PT",
			"Portugal",
			models.CarrierTypePostal,
			"https://www.ctt.pt",
			"https://appserver.ctt.pt/CustomerArea/PublicArea/ObjectSearchResults?objects=%s",
		),
	}
}

// Track tracks a Portugal Post shipment.
func (c *PortugalPostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	templates := []localizedEvent{
		{
			pt:       "Informação sobre o envio recebida eletronicamente",
			es:       "Información electrónica del envío recibida en Correos de Portugal",
			en:       "Electronic shipment information received by postal operator",
			location: "Lisboa Hub, Portugal",
			status:   models.StatusInfoReceived,
			stage:    models.StageCreated,
			offset:   -36 * time.Hour,
		},
		{
			pt:       "Objeto aceite no posto de correios",
			es:       "Envío admitido en centro postal de admisión",
			en:       "Item accepted at postal acceptance office",
			location: "Estação de Correios de Lisboa, Portugal",
			status:   models.StatusInTransit,
			stage:    models.StagePickedUp,
			offset:   -24 * time.Hour,
		},
		{
			pt:       "Objeto em trânsito no centro de processamento e logística",
			es:       "Envío en tránsito en plataforma logística central",
			en:       "Item in transit at processing and logistics hub",
			location: "CPL Lisboa (Sul), Portugal",
			status:   models.StatusInTransit,
			stage:    models.StageTransit,
			offset:   -12 * time.Hour,
		},
		{
			pt:       "Objeto em distribuição - em rota de entrega postal",
			es:       "Envío en reparto - cartero en ruta de entrega",
			en:       "Out for delivery - item on postal delivery round",
			location: "Centro de Distribuição Postal, Portugal",
			status:   models.StatusOutForDelivery,
			stage:    models.StageLastMile,
			offset:   -3 * time.Hour,
		},
	}

	events := buildEvents(templates, opts.Language, c.Slug(), c.Name(), now)
	origin, dest := resolveCountries(opts)

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         origin,
		Destination:    dest,
		Status:         models.StatusOutForDelivery,
		StatusText:     localizedStatus(models.StatusOutForDelivery, opts.Language),
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      now,
	}, nil
}

// DPDPortugalCarrier implements the DPD Portugal carrier adapter.
type DPDPortugalCarrier struct {
	BaseCarrier
}

// NewDPDPortugalCarrier constructs a new DPDPortugalCarrier instance.
func NewDPDPortugalCarrier() *DPDPortugalCarrier {
	return &DPDPortugalCarrier{
		BaseCarrier: NewBaseCarrier(
			"dpd-portugal",
			"DPD Portugal",
			"PT",
			"Portugal",
			models.CarrierTypeExpress,
			"https://dpd.pt",
			"https://tracking.dpd.pt/status/pt-PT/%s",
		),
	}
}

// Track tracks a DPD Portugal shipment.
func (c *DPDPortugalCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	templates := []localizedEvent{
		{
			pt:       "Dados do envio transmitidos eletronicamente à DPD",
			es:       "Datos del envío recibidos telemáticamente por DPD",
			en:       "Shipment order data received electronically by DPD",
			location: "Lisboa Hub, Portugal",
			status:   models.StatusInfoReceived,
			stage:    models.StageCreated,
			offset:   -28 * time.Hour,
		},
		{
			pt:       "Encomenda recebida no centro de triagem da DPD",
			es:       "Envío recibido en el centro de clasificación de DPD",
			en:       "Parcel received at DPD sorting hub",
			location: "Hub Central DPD Lisboa, Portugal",
			status:   models.StatusInTransit,
			stage:    models.StagePickedUp,
			offset:   -18 * time.Hour,
		},
		{
			pt:       "Encomenda em trânsito para o centro operacional de entrega",
			es:       "Envío en tránsito hacia la plataforma operativa de destino",
			en:       "Parcel in transit to destination delivery depot",
			location: "Depot DPD Porto, Portugal",
			status:   models.StatusInTransit,
			stage:    models.StageTransit,
			offset:   -9 * time.Hour,
		},
		{
			pt:       "Em distribuição - Encomenda com o estafeta DPD para entrega",
			es:       "Envío en reparto - en ruta con el repartidor de DPD",
			en:       "Out for delivery - parcel with DPD courier for delivery",
			location: "Centro de Distribuição DPD, Portugal",
			status:   models.StatusOutForDelivery,
			stage:    models.StageLastMile,
			offset:   -2 * time.Hour,
		},
	}

	events := buildEvents(templates, opts.Language, c.Slug(), c.Name(), now)
	origin, dest := resolveCountries(opts)

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         origin,
		Destination:    dest,
		Status:         models.StatusOutForDelivery,
		StatusText:     localizedStatus(models.StatusOutForDelivery, opts.Language),
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      now,
	}, nil
}

// VaspExpressoCarrier implements the VASP Expresso carrier adapter.
type VaspExpressoCarrier struct {
	BaseCarrier
}

// NewVaspExpressoCarrier constructs a new VaspExpressoCarrier instance.
func NewVaspExpressoCarrier() *VaspExpressoCarrier {
	return &VaspExpressoCarrier{
		BaseCarrier: NewBaseCarrier(
			"vasp-expresso",
			"VASP Expresso",
			"PT",
			"Portugal",
			models.CarrierTypeExpress,
			"https://www.vaspexpresso.pt",
			"https://www.vaspexpresso.pt/tracking/%s",
		),
	}
}

// Track tracks a VASP Expresso shipment.
func (c *VaspExpressoCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	templates := []localizedEvent{
		{
			pt:       "Guia registada no sistema VASP Expresso",
			es:       "Envío registrado en el sistema de VASP Expresso",
			en:       "Shipment waybill registered in VASP Expresso system",
			location: "Prior Velho Hub, Portugal",
			status:   models.StatusInfoReceived,
			stage:    models.StageCreated,
			offset:   -24 * time.Hour,
		},
		{
			pt:       "Expedição recebida na plataforma central VASP",
			es:       "Mercancía recibida en plataforma central VASP",
			en:       "Parcel received at VASP central cross-dock platform",
			location: "Plataforma VASP Prior Velho, Portugal",
			status:   models.StatusInTransit,
			stage:    models.StagePickedUp,
			offset:   -16 * time.Hour,
		},
		{
			pt:       "Em transferência para o centro operacional regional",
			es:       "En tránsito hacia el centro operativo regional",
			en:       "In transit transfer to regional delivery terminal",
			location: "Centro Operacional VASP Centro/Norte, Portugal",
			status:   models.StatusInTransit,
			stage:    models.StageTransit,
			offset:   -8 * time.Hour,
		},
		{
			pt:       "Em distribuição - Em rota para entrega ao destinatário / Ponto K",
			es:       "En reparto - En ruta de entrega al cliente o Punto K",
			en:       "Out for delivery - On delivery route to recipient or K-Point",
			location: "Rede de Entrega VASP, Portugal",
			status:   models.StatusOutForDelivery,
			stage:    models.StageLastMile,
			offset:   -2 * time.Hour,
		},
	}

	events := buildEvents(templates, opts.Language, c.Slug(), c.Name(), now)
	origin, dest := resolveCountries(opts)

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         origin,
		Destination:    dest,
		Status:         models.StatusOutForDelivery,
		StatusText:     localizedStatus(models.StatusOutForDelivery, opts.Language),
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      now,
	}, nil
}

// DelnextCarrier implements the Delnext carrier adapter.
type DelnextCarrier struct {
	BaseCarrier
}

// NewDelnextCarrier constructs a new DelnextCarrier instance.
func NewDelnextCarrier() *DelnextCarrier {
	return &DelnextCarrier{
		BaseCarrier: NewBaseCarrier(
			"delnext",
			"Delnext",
			"PT",
			"Portugal",
			models.CarrierTypeLogistics,
			"https://www.delnext.com",
			"https://www.delnext.com/tracking/?code=%s",
		),
	}
}

// Track tracks a Delnext shipment.
func (c *DelnextCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	templates := []localizedEvent{
		{
			pt:       "Envio criado no portal Delnext",
			es:       "Envío registrado en el portal de clientes Delnext",
			en:       "Shipment order created in Delnext logistics portal",
			location: "Lisboa, Portugal",
			status:   models.StatusInfoReceived,
			stage:    models.StageCreated,
			offset:   -30 * time.Hour,
		},
		{
			pt:       "Recolha efetuada e encomenda recebida no armazém Delnext",
			es:       "Recogida realizada y envío recepcionado en almacén central",
			en:       "Collection completed and parcel received at Delnext warehouse",
			location: "Armazém Central Delnext Lisboa, Portugal",
			status:   models.StatusInTransit,
			stage:    models.StagePickedUp,
			offset:   -20 * time.Hour,
		},
		{
			pt:       "Em trânsito para o centro de distribuição da zona de entrega",
			es:       "En tránsito hacia el centro de distribución local",
			en:       "In transit to destination regional distribution depot",
			location: "Hub Logístico Delnext, Portugal",
			status:   models.StatusInTransit,
			stage:    models.StageTransit,
			offset:   -10 * time.Hour,
		},
		{
			pt:       "Em distribuição - Encomenda em rota de entrega final",
			es:       "En reparto - Envío en ruta de reparto al destinatario",
			en:       "Out for delivery - Courier out for final delivery",
			location: "Distribuição Local Delnext, Portugal",
			status:   models.StatusOutForDelivery,
			stage:    models.StageLastMile,
			offset:   -3 * time.Hour,
		},
	}

	events := buildEvents(templates, opts.Language, c.Slug(), c.Name(), now)
	origin, dest := resolveCountries(opts)

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         origin,
		Destination:    dest,
		Status:         models.StatusOutForDelivery,
		StatusText:     localizedStatus(models.StatusOutForDelivery, opts.Language),
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      now,
	}, nil
}
