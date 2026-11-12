package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

var seurHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetSEURHTTPClientForTest overrides the HTTP client for SEUR in deterministic tests.
func SetSEURHTTPClientForTest(client *http.Client) func() {
	orig := seurHTTPClient
	seurHTTPClient = client
	return func() { seurHTTPClient = orig }
}

type SEURCarrier struct {
	BaseCarrier
}

func NewSEURCarrier() *SEURCarrier {
	bc := NewBaseCarrier(
		"seur",
		"SEUR",
		"ES",
		"Spain",
		models.CarrierTypeExpress,
		"https://www.seur.com",
		"https://www.seur.com/livetracking/?segOnlineIdioma=es&segOnlineIdentificador=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		RequiresPostalCode:  true,
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &SEURCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that SEUR implements live network tracking.
func (c *SEURCarrier) HasLiveTracking() bool {
	return true
}

type seurEvento struct {
	Fecha       string `json:"fecha"`
	Estado      string `json:"estado"`
	Descripcion string `json:"descripcion"`
	Centro      string `json:"centro"`
}

type seurResponse struct {
	CodigoSeguimiento   string       `json:"codigoSeguimiento"`
	Estado              string       `json:"estado"`
	DescripcionEstado   string       `json:"descripcionEstado"`
	FechaEntrega        string       `json:"fechaEntrega"`
	Origen              string       `json:"origen"`
	Destino             string       `json:"destino"`
	CodigoPostalDestino string       `json:"codigoPostalDestino"`
	Eventos             []seurEvento `json:"eventos"`
}

func parseSEURDate(dateStr string) time.Time {
	clean := strings.TrimSpace(dateStr)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := ParseLocalTimeWithZone(layout, clean, "Europe/Madrid"); err == nil {
			return t
		}
	}
	return time.Time{}
}

func normalizeSEURStatus(estado, desc string) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(estado + " " + desc))

	switch {
	case strings.Contains(upper, "ENTREGADO") || strings.Contains(upper, "DELIVERED"):
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(upper, "REPARTO") || strings.Contains(upper, "CONDUCTOR"):
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(upper, "TRANSITO") || strings.Contains(upper, "PLATAFORMA") || strings.Contains(upper, "ARRASTRE") || strings.Contains(upper, "NAVE"):
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(upper, "REGISTRADO") || strings.Contains(upper, "INFORMADO") || strings.Contains(upper, "PRE_REGISTRADO"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(upper, "INCIDENCIA") || strings.Contains(upper, "AUSENTE") || strings.Contains(upper, "FALLIDO"):
		return models.StatusFailedAttempt, models.StageException
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func (c *SEURCarrier) fetchLive(ctx context.Context, trackingNumber, postalCode string) (*seurResponse, error) {
	apiURL := fmt.Sprintf("https://www.seur.com/livetracking/api/v1/shipments?identificador=%s&codigoPostal=%s",
		url.QueryEscape(trackingNumber), url.QueryEscape(postalCode))

	req, err := NewCarrierRequest(http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req = req.WithContext(ctx)

	resp, err := seurHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, models.ErrShipmentNotFound
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, models.ErrCarrierRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP status %d", models.ErrCarrierTemporarilyUnavailable, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	var sResp seurResponse
	if err := json.Unmarshal(bodyBytes, &sResp); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON from SEUR: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if sResp.Estado == "NO_ENCONTRADO" || (len(sResp.Eventos) == 0 && sResp.Estado == "") {
		return nil, models.ErrShipmentNotFound
	}

	return &sResp, nil
}

func (c *SEURCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	postalCode := strings.TrimSpace(opts.PostalCode)
	if postalCode == "" {
		if !opts.AllowSimulated {
			return nil, fmt.Errorf("%w: SEUR tracking requires destination postal code (--postal-code <CP>)", models.ErrPostalCodeRequired)
		}
		return c.generateSimulatedShipment(cleanNum, opts), nil
	}

	sResp, err := c.fetchLive(ctx, cleanNum, postalCode)
	if err == nil && sResp != nil {
		var events []models.TrackingEvent
		for _, ev := range sResp.Eventos {
			evTime := parseSEURDate(ev.Fecha)
			status, stage := normalizeSEURStatus(ev.Estado, ev.Descripcion)

			loc := strings.TrimSpace(ev.Centro)
			if loc != "" && !strings.Contains(strings.ToLower(loc), "spain") {
				loc += ", Spain"
			}

			// Deduplicate consecutive identical events within 15 minutes
			if len(events) > 0 {
				last := events[len(events)-1]
				if last.Status == status && last.Description == ev.Descripcion {
					diff := last.Date.Sub(evTime)
					if diff < 0 {
						diff = -diff
					}
					if diff < 15*time.Minute {
						continue
					}
				}
			}

			events = append(events, models.TrackingEvent{
				Date:         evTime,
				RawDate:      ev.Fecha,
				Status:       status,
				Stage:        stage,
				Description:  ev.Descripcion,
				Location:     loc,
				CarrierSlug:  c.Slug(),
				CarrierName:  c.Name(),
				CarrierIndex: 0,
			})
		}

		overallStatus, _ := normalizeSEURStatus(sResp.Estado, sResp.DescripcionEstado)
		for _, ev := range events {
			if ev.Status == models.StatusDelivered {
				overallStatus = models.StatusDelivered
				break
			}
		}

		statusText := sResp.DescripcionEstado
		if statusText == "" {
			statusText = sResp.Estado
		}
		if statusText == "" {
			if overallStatus == models.StatusDelivered {
				statusText = "Entregado"
			} else if overallStatus == models.StatusOutForDelivery {
				statusText = "En reparto"
			} else {
				statusText = "En tránsito"
			}
		}

		progress := 50
		if overallStatus == models.StatusDelivered {
			progress = 100
		} else if overallStatus == models.StatusOutForDelivery {
			progress = 85
		}

		var deliveredAt *time.Time
		if sResp.FechaEntrega != "" {
			t := parseSEURDate(sResp.FechaEntrega)
			if !t.IsZero() {
				deliveredAt = &t
			}
		}
		if deliveredAt == nil {
			for _, ev := range events {
				if ev.Status == models.StatusDelivered {
					t := ev.Date
					deliveredAt = &t
					break
				}
			}
		}

		origCity := sResp.Origen
		if origCity == "" {
			origCity = "Spain"
		}
		destCity := sResp.Destino
		if destCity == "" {
			destCity = "Spain"
		}

		return &models.Shipment{
			TrackingNumber: cleanNum,
			Carrier:        c.Info(),
			Origin: models.CountryInfo{
				Name: origCity,
				Code: "ES",
			},
			Destination: models.CountryInfo{
				Name: destCity,
				Code: "ES",
			},
			Status:        overallStatus,
			StatusText:    statusText,
			Progress:      progress,
			DeliveredAt:   deliveredAt,
			DaysInTransit: 1,
			Events:        events,
			UpdatedAt:     time.Now(),
		}, nil
	}

	if opts.AllowSimulated {
		return c.generateSimulatedShipment(cleanNum, opts), nil
	}

	if err != nil {
		return nil, err
	}
	return nil, models.ErrShipmentNotFound
}

func (c *SEURCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-28 * time.Hour),
			Description: "Datos del envío registrados por el remitente",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-16 * time.Hour),
			Description: "En tránsito - Envío en plataforma de clasificación",
			Location:    "Madrid Hub Central, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Su paquete está en reparto con el conductor",
			Location:    "Unidad de Reparto Local, Spain",
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
			Name: "Spain",
			Code: "ES",
		},
		Destination: models.CountryInfo{
			Name: "Spain",
			Code: "ES",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "En reparto / Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}
}
