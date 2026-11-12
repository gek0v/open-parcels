package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

var correosHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetCorreosHTTPClientForTest overrides the HTTP client for Correos España in deterministic tests.
func SetCorreosHTTPClientForTest(client *http.Client) func() {
	orig := correosHTTPClient
	correosHTTPClient = client
	return func() { correosHTTPClient = orig }
}

type CorreosSpainCarrier struct {
	BaseCarrier
}

func NewCorreosSpainCarrier() *CorreosSpainCarrier {
	bc := NewBaseCarrier(
		"correos-spain",
		"Correos España",
		"ES",
		"Spain",
		models.CarrierTypePostal,
		"https://www.correos.es",
		"https://www.correos.es/es/es/herramientas/localizador/envios/detalle?tracking-number=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &CorreosSpainCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that Correos España implements real network tracking.
func (c *CorreosSpainCarrier) HasLiveTracking() bool {
	return true
}

type correosEvent struct {
	EventDate   string `json:"eventDate"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Office      string `json:"office"`
}

type correosOriginDestination struct {
	Country     string `json:"country"`
	CountryName string `json:"countryName"`
	PostalCode  string `json:"postalCode"`
	Office      string `json:"office"`
}

type correosShipmentData struct {
	ShipmentCode      string                   `json:"shipmentCode"`
	Status            string                   `json:"status"`
	StatusDescription string                   `json:"statusDescription"`
	DeliveryDate      string                   `json:"deliveryDate"`
	Origin            correosOriginDestination `json:"origin"`
	Destination       correosOriginDestination `json:"destination"`
	PickupOffice      string                   `json:"pickupOffice"`
	DeadlineDate      string                   `json:"deadlineDate"`
	Events            []correosEvent           `json:"events"`
}

type correosResponse struct {
	Shipment *correosShipmentData `json:"shipment"`
	Error    string               `json:"error,omitempty"`
	Code     string               `json:"code,omitempty"`
}

func parseCorreosDate(dateStr string) time.Time {
	clean := strings.TrimSpace(dateStr)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02/01/2006",
	}

	for _, layout := range layouts {
		if t, err := ParseLocalTimeWithZone(layout, clean, "Europe/Madrid"); err == nil {
			return t
		}
	}
	return time.Time{}
}

func normalizeCorreosStatus(status, desc string) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(status + " " + desc))

	switch {
	case strings.Contains(upper, "ENTREGADO") || strings.Contains(upper, "DELIVERED"):
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(upper, "DISPONIBLE_OFICINA") || strings.Contains(upper, "EN LISTA") || strings.Contains(upper, "DISPONIBLE EN OFICINA") || strings.Contains(upper, "AVISO_LLEGADA") || strings.Contains(upper, "AVISO PARA RECOGIDA"):
		return models.StatusOutForDelivery, models.StageDelivery
	case strings.Contains(upper, "EN_REPARTO") || strings.Contains(upper, "EN REPARTO"):
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(upper, "ADMITIDO") || strings.Contains(upper, "RECOGIDO"):
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(upper, "TRANSITO") || strings.Contains(upper, "CLASIFICADO") || strings.Contains(upper, "CAMINO") || strings.Contains(upper, "CTA"):
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(upper, "PRE_REGISTRADO") || strings.Contains(upper, "PREREGISTRADO") || strings.Contains(upper, "INFORMADO"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(upper, "INCIDENCIA") || strings.Contains(upper, "AUSENTE") || strings.Contains(upper, "FALLIDO"):
		return models.StatusFailedAttempt, models.StageException
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func (c *CorreosSpainCarrier) fetchLive(ctx context.Context, trackingNumber, lang string) (*correosResponse, error) {
	apiURL := fmt.Sprintf("https://api1.correos.es/digital-services/searches-b2c/api/v1/shipments?codEnvio=%s&codIdioma=%s", trackingNumber, lang)
	req, err := NewCarrierRequest(http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req = req.WithContext(ctx)

	resp, err := correosHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, classifyScraperHTTPStatus(resp.StatusCode, bodyBytes, resp.Header, "Correos Espa?a")
	}

	if isWAFChallenge(resp.StatusCode, bodyBytes, resp.Header) {
		return nil, fmt.Errorf("%w: WAF or anti-bot challenge detected from Correos Espa?a", models.ErrCarrierBlocked)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		return nil, fmt.Errorf("%w: unexpected non-JSON or malformed response from Correos: %v", models.ErrScraperSchemaChanged, err)
	}
	if !hasAnyJSONKey(rawMap, "shipment", "error", "code") {
		return nil, fmt.Errorf("%w: unexpected JSON schema from Correos", models.ErrScraperSchemaChanged)
	}

	var cResp correosResponse
	if err := json.Unmarshal(bodyBytes, &cResp); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON from Correos: %v", models.ErrScraperSchemaChanged, err)
	}

	if cResp.Shipment == nil {
		return nil, models.ErrShipmentNotFound
	}
	if cResp.Shipment.ShipmentCode == "" && cResp.Shipment.Status == "" && cResp.Shipment.StatusDescription == "" && len(cResp.Shipment.Events) == 0 {
		return nil, fmt.Errorf("%w: empty or unrecognized shipment payload from Correos", models.ErrScraperSchemaChanged)
	}

	return &cResp, nil
}

func (c *CorreosSpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	lang := "ES"
	if strings.HasPrefix(strings.ToLower(opts.Language), "en") {
		lang = "EN"
	}

	cResp, err := c.fetchLive(ctx, cleanNum, lang)
	if err == nil && cResp != nil && cResp.Shipment != nil {
		shpData := cResp.Shipment

		var events []models.TrackingEvent
		for _, ev := range shpData.Events {
			evTime := parseCorreosDate(ev.EventDate)
			status, stage := normalizeCorreosStatus(ev.Status, ev.Description)

			location := strings.TrimSpace(ev.Location)
			if location == "" {
				location = strings.TrimSpace(ev.Office)
			}
			if location != "" && !strings.Contains(strings.ToLower(location), "spain") {
				location += ", Spain"
			}

			events = append(events, models.TrackingEvent{
				Date:         evTime,
				RawDate:      ev.EventDate,
				Status:       status,
				Stage:        stage,
				Description:  ev.Description,
				Location:     location,
				CarrierSlug:  c.Slug(),
				CarrierName:  c.Name(),
				CarrierIndex: 0,
			})
		}

		overallStatus, _ := normalizeCorreosStatus(shpData.Status, shpData.StatusDescription)
		for _, ev := range events {
			if ev.Status == models.StatusDelivered {
				overallStatus = models.StatusDelivered
				break
			}
		}

		statusText := shpData.StatusDescription
		if statusText == "" {
			statusText = shpData.Status
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

		// Handle pickup office and deadline date
		if shpData.PickupOffice != "" {
			deadlineInfo := ""
			if shpData.DeadlineDate != "" {
				deadlineInfo = fmt.Sprintf(" (Fecha límite de recogida: %s)", shpData.DeadlineDate)
			}
			statusText = fmt.Sprintf("Disponible en oficina: %s%s", shpData.PickupOffice, deadlineInfo)
		}

		progress := 50
		if overallStatus == models.StatusDelivered {
			progress = 100
		} else if overallStatus == models.StatusOutForDelivery {
			progress = 85
		}

		var deliveredAt *time.Time
		if shpData.DeliveryDate != "" {
			t := parseCorreosDate(shpData.DeliveryDate)
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

		origCountry := shpData.Origin.CountryName
		if origCountry == "" {
			origCountry = "Spain"
		}
		origCode := shpData.Origin.Country
		if origCode == "" {
			origCode = "ES"
		}

		destCountry := shpData.Destination.CountryName
		if destCountry == "" {
			destCountry = "Spain"
		}
		destCode := shpData.Destination.Country
		if destCode == "" {
			destCode = "ES"
		}

		return &models.Shipment{
			TrackingNumber: cleanNum,
			Carrier:        c.Info(),
			Origin: models.CountryInfo{
				Name: origCountry,
				Code: origCode,
			},
			Destination: models.CountryInfo{
				Name: destCountry,
				Code: destCode,
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

	// Fallback only if AllowSimulated is explicitly true
	if opts.AllowSimulated {
		return c.generateSimulatedShipment(cleanNum, opts), nil
	}

	if err != nil {
		return nil, err
	}
	return nil, models.ErrShipmentNotFound
}

func (c *CorreosSpainCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-48 * time.Hour),
			Description: "Pre-registrado en los sistemas de Correos",
			Location:    "Centro de Admisión, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Admitido en origen",
			Location:    "Madrid CTA, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "En tránsito - En camino al centro de distribución local",
			Location:    "Valencia CTA, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "En reparto - El cartero ha iniciado el reparto del envío",
			Location:    "Valencia Unidad de Reparto, Spain",
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
		DaysInTransit: 2,
		Events:        events,
		UpdatedAt:     time.Now(),
	}
}
