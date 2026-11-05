package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// AlfredITCarrier represents Alfred 24 smart locker and delivery network in Italy.
type AlfredITCarrier struct {
	BaseCarrier
}

func NewAlfredITCarrier() *AlfredITCarrier {
	return &AlfredITCarrier{
		BaseCarrier: NewBaseCarrier(
			"alfred-it",
			"Alfred 24",
			"IT",
			"Italy",
			models.CarrierTypeLogistics,
			"https://alfred24.it",
			"https://alfred24.it/tracking/%s",
		),
	}
}

func (c *AlfredITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Preso in carico nel circuito Alfred 24",
			Location:    "Milano Locker Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "In transito verso il punto di ritiro smart",
			Location:    "Torino Centro Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "In consegna presso il locker Alfred",
			Location:    "Torino Locker Point, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna presso locker Alfred 24",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// ArcoSpedizioniCarrier represents Arco Spedizioni freight and logistics network in Italy.
type ArcoSpedizioniCarrier struct {
	BaseCarrier
}

func NewArcoSpedizioniCarrier() *ArcoSpedizioniCarrier {
	return &ArcoSpedizioniCarrier{
		BaseCarrier: NewBaseCarrier(
			"arco-spedizioni",
			"Arco Spedizioni",
			"IT",
			"Italy",
			models.CarrierTypeFreight,
			"https://www.arco.it",
			"https://www.arco.it/tracking/?sped=%s",
		),
	}
}

func (c *ArcoSpedizioniCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Spedizione registrata e affidata ad Arco Spedizioni",
			Location:    "Monza Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-16 * time.Hour),
			Description: "In transito verso la filiale di destino",
			Location:    "Bologna Interporto, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "In consegna con mezzo pesante / vettore Arco",
			Location:    "Filiale Arco Firenze, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna con vettore Arco Spedizioni",
		Progress:       85,
		DaysInTransit:  2,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// CEVAITCarrier represents CEVA Logistics Italy.
type CEVAITCarrier struct {
	BaseCarrier
}

func NewCEVAITCarrier() *CEVAITCarrier {
	return &CEVAITCarrier{
		BaseCarrier: NewBaseCarrier(
			"ceva-it",
			"CEVA Logistics Italy",
			"IT",
			"Italy",
			models.CarrierTypeFreight,
			"https://www.cevalogistics.com",
			"https://www.cevalogistics.com/en/ceva-tracking?ref=%s",
		),
	}
}

func (c *CEVAITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-48 * time.Hour),
			Description: "Carico registrato nel sistema CEVA Logistics Italy",
			Location:    "CEVA Logistics Milano Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-20 * time.Hour),
			Description: "Merce consolidata in transito intermodale",
			Location:    "Interporto Rivalta Scrivia, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "In consegna finale presso stabilimento destinatario",
			Location:    "CEVA Depot Roma, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna CEVA Logistics",
		Progress:       85,
		DaysInTransit:  2,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// BDMITCarrier represents CST Italia (BDM) logistics.
type BDMITCarrier struct {
	BaseCarrier
}

func NewBDMITCarrier() *BDMITCarrier {
	return &BDMITCarrier{
		BaseCarrier: NewBaseCarrier(
			"bdm-it",
			"CST Italia",
			"IT",
			"Italy",
			models.CarrierTypeLogistics,
			"https://www.cstitalia.it",
			"https://www.cstitalia.it/tracking/%s",
		),
	}
}

func (c *BDMITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Spedizione presa in carico da CST Italia",
			Location:    "Verona Hub CST, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "In transito verso il centro logistico regionale",
			Location:    "Padova Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "In consegna al destinatario",
			Location:    "Venezia Filiale, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna CST Italia",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// DeliverITCarrier represents Deliver It courier services in Italy.
type DeliverITCarrier struct {
	BaseCarrier
}

func NewDeliverITCarrier() *DeliverITCarrier {
	return &DeliverITCarrier{
		BaseCarrier: NewBaseCarrier(
			"deliver-it",
			"Deliver It",
			"IT",
			"Italy",
			models.CarrierTypeExpress,
			"https://deliver-it.it",
			"https://deliver-it.it/tracking?id=%s",
		),
	}
}

func (c *DeliverITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Preso in carico dal corriere Deliver It",
			Location:    "Milano City Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "Spedizione smistata e in viaggio per la consegna",
			Location:    "Monza Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "In consegna express nella fascia oraria concordata",
			Location:    "Milano Destinazione, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna express Deliver It",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// LiccardiITCarrier represents Liccardi Trasporti freight services in Italy.
type LiccardiITCarrier struct {
	BaseCarrier
}

func NewLiccardiITCarrier() *LiccardiITCarrier {
	return &LiccardiITCarrier{
		BaseCarrier: NewBaseCarrier(
			"liccardi-it",
			"Liccardi Trasporti",
			"IT",
			"Italy",
			models.CarrierTypeFreight,
			"https://www.liccarditrasporti.it",
			"https://www.liccarditrasporti.it/tracking/?num=%s",
		),
	}
}

func (c *LiccardiITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Spedizione ritirata da Liccardi Trasporti",
			Location:    "Napoli Interporto, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "In transito su linea tir nazionale",
			Location:    "Roma Hub Liccardi, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "In consegna presso destinatario",
			Location:    "Caserta Filiale, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna Liccardi Trasporti",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// MBEITCarrier represents Mail Boxes Etc. Italy.
type MBEITCarrier struct {
	BaseCarrier
}

func NewMBEITCarrier() *MBEITCarrier {
	return &MBEITCarrier{
		BaseCarrier: NewBaseCarrier(
			"mbe-it",
			"Mail Boxes Etc. Italy",
			"IT",
			"Italy",
			models.CarrierTypeLogistics,
			"https://www.mbe.it",
			"https://www.mbe.it/it/tracking?tracking_codes=%s",
		),
	}
}

func (c *MBEITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-26 * time.Hour),
			Description: "Pacco spedito tramite Centro MBE",
			Location:    "Centro MBE 024 Milano, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "In transito verso la filiale di recapito",
			Location:    "Hub MBE Bologna, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "In consegna al destinatario tramite rete MBE",
			Location:    "Centro MBE Roma, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna rete Mail Boxes Etc.",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// RhenusITCarrier represents Rhenus Logistics Italia.
type RhenusITCarrier struct {
	BaseCarrier
}

func NewRhenusITCarrier() *RhenusITCarrier {
	return &RhenusITCarrier{
		BaseCarrier: NewBaseCarrier(
			"rhenus-it",
			"Rhenus Italia",
			"IT",
			"Italy",
			models.CarrierTypeFreight,
			"https://www.rhenus.group/it/it/",
			"https://track.rhenus.com/shipment?id=%s",
		),
	}
}

func (c *RhenusITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Spedizione registrata nella rete Rhenus Logistics Italia",
			Location:    "Milano Logistics Center, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-16 * time.Hour),
			Description: "Carico in trasferimento logistico interregionale",
			Location:    "Bologna Freight Hub, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "In consegna al destinatario finale con sponda idraulica",
			Location:    "Padova Filiale, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna Rhenus Italia",
		Progress:       85,
		DaysInTransit:  2,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// SpediamoITCarrier represents Spediamo.it online courier broker network.
type SpediamoITCarrier struct {
	BaseCarrier
}

func NewSpediamoITCarrier() *SpediamoITCarrier {
	return &SpediamoITCarrier{
		BaseCarrier: NewBaseCarrier(
			"spediamo-it",
			"Spediamo",
			"IT",
			"Italy",
			models.CarrierTypeLogistics,
			"https://www.spediamo.it",
			"https://www.spediamo.it/ricerca-spedizione?tracking=%s",
		),
	}
}

func (c *SpediamoITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Ordine di spedizione confermato su Spediamo.it",
			Location:    "Hub Centrale Spediamo, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "In transito con corriere partner convenzionato",
			Location:    "Milano Smistamento, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "Spedizione in consegna con l'autista partner",
			Location:    "Torino Recapito, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna Spediamo.it",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

// UnieuroITCarrier represents Unieuro tracking and customer delivery network in Italy.
type UnieuroITCarrier struct {
	BaseCarrier
}

func NewUnieuroITCarrier() *UnieuroITCarrier {
	return &UnieuroITCarrier{
		BaseCarrier: NewBaseCarrier(
			"unieuro-it",
			"Unieuro",
			"IT",
			"Italy",
			models.CarrierTypeLogistics,
			"https://www.unieuro.it",
			"https://www.unieuro.it/online/tracking?order=%s",
		),
	}
}

func (c *UnieuroITCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Ordine preso in carico dalla logistica Unieuro",
			Location:    "Piacenza Hub Unieuro, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-10 * time.Hour),
			Description: "Merce affidata alla flotta consegne dedicate",
			Location:    "Bologna Piattaforma, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "In consegna al piano con montaggio / recapito Unieuro",
			Location:    "Bologna Destinazione, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         models.CountryInfo{Name: "Italy", Code: "IT"},
		Destination:    models.CountryInfo{Name: "Italy", Code: "IT"},
		Status:         models.StatusOutForDelivery,
		StatusText:     "In consegna con corriere Unieuro",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}
