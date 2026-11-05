package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericLATAMCarrier models a Latin American postal or courier service.
type GenericLATAMCarrier struct {
	BaseCarrier
}

func NewGenericLATAMCarrier(slug, name string) *GenericLATAMCarrier {
	return &GenericLATAMCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			"LATAM",
			"Latin America",
			models.CarrierTypeExpress,
			"",
			"",
		),
	}
}

func (c *GenericLATAMCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Latin America"
	destCode := "LATAM"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Envío recibido / Datos registrados por " + c.Name(),
			Location:    "Centro Operativo de Origen",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "En tránsito - Envío clasificado en centro logístico central",
			Location:    "Planta de Distribución Principal",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "En reparto - Envío en poder del mensajero para entrega hoy",
			Location:    destCountry + " Sucursal de Destino",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Latin America",
			Code: "LATAM",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "En reparto / Out for delivery",
		Progress:      85,
		DaysInTransit: 4,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerLATAMCatalog(r *Registry) {
	latamList := []struct {
		slug string
		name string
	}{
		{"andreani", "Andreani"},
		{"correo-argentino", "Argentine Post"},
		{"correios-brazil", "Brazil Post"},
		{"caribou", "Caribou"},
		{"chilexpress-cl", "Chilexpress"},
		{"correios-cape-verde", "Correios de Cabo Verde"},
		{"correo-argentino-domestic", "Correo Argentino Domestic"},
		{"uruguay-post", "Correo Uruguayo"},
		{"correos-cuba", "Correos Cuba"},
		{"correos-guatemala", "Correos de Guatemala"},
		{"correos-el-salvador", "Correos El Salvador"},
		{"correos-panama", "Correos Panama"},
		{"cpost-international", "CPost International Curacao"},
		{"estafeta", "Estafeta"},
		{"expresstime-co", "ExpresstimeCO"},
		{"fastship-co", "Fastship"},
		{"ivoy-mx", "Ivoy"},
		{"mexico-post", "Mexico Post"},
		{"paquetexpress", "Paquetexpress"},
		{"peru-post", "Peru Post"},
		{"procarrier", "PRO Carrier"},
		{"redpack-mexico", "Redpack Mexico"},
		{"sendex-mx", "Sendex"},
		{"shopee-express-br", "Shopee Express Brazil"},
		{"starken", "Starken"},
	}

	for _, c := range latamList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericLATAMCarrier(c.slug, c.name))
		}
	}
}
