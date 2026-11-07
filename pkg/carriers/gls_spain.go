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

var glsSpainHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetGLSSpainHTTPClientForTest overrides the HTTP client for GLS Spain in deterministic tests.
func SetGLSSpainHTTPClientForTest(client *http.Client) func() {
	orig := glsSpainHTTPClient
	glsSpainHTTPClient = client
	return func() { glsSpainHTTPClient = orig }
}

type GLSSpainCarrier struct {
	BaseCarrier
}

func NewGLSSpainCarrier() *GLSSpainCarrier {
	bc := NewBaseCarrier(
		"asm-gls-spain",
		"GLS Spain (ASM)",
		"ES",
		"Spain",
		models.CarrierTypeExpress,
		"https://www.gls-spain.es",
		"https://www.gls-spain.es/es/seguimiento-envios/?match=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		RequiresPostalCode:  true,
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &GLSSpainCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that GLS Spain implements live network tracking.
func (c *GLSSpainCarrier) HasLiveTracking() bool {
	return true
}

type glsSpainHistoryItem struct {
	Date        string `json:"date"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Location    string `json:"location"`
}

type glsSpainLocation struct {
	City       string `json:"city"`
	Country    string `json:"country"`
	PostalCode string `json:"postalCode"`
}

type glsSpainResponse struct {
	TrackingNumber    string                `json:"trackingNumber"`
	Status            string                `json:"status"`
	StatusDescription string                `json:"statusDescription"`
	DeliveryDate      string                `json:"deliveryDate"`
	Origin            glsSpainLocation      `json:"origin"`
	Destination       glsSpainLocation      `json:"destination"`
	History           []glsSpainHistoryItem `json:"history"`
}

func parseGLSSpainDate(dateStr string) time.Time {
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
		if t, err := time.ParseInLocation(layout, clean, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

func normalizeGLSSpainStatus(status, desc string) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(status + " " + desc))

	switch {
	case strings.Contains(upper, "DELIVERED") || strings.Contains(upper, "ENTREGADO"):
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(upper, "OUT_FOR_DELIVERY") || strings.Contains(upper, "EN REPARTO"):
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(upper, "IN_TRANSIT") || strings.Contains(upper, "TRANSITO") || strings.Contains(upper, "CENTRO"):
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(upper, "REGISTERED") || strings.Contains(upper, "SISTEMA") || strings.Contains(upper, "INFO_RECEIVED"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(upper, "FAILED") || strings.Contains(upper, "AUSENTE") || strings.Contains(upper, "FALLID") || strings.Contains(upper, "EXCEPTION"):
		return models.StatusFailedAttempt, models.StageException
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func (c *GLSSpainCarrier) fetchLive(ctx context.Context, trackingNumber, postalCode string) (*glsSpainResponse, error) {
	apiURL := fmt.Sprintf("https://www.gls-spain.es/api/tracking/v1/shipments?trackingNumber=%s&postalCode=%s",
		url.QueryEscape(trackingNumber), url.QueryEscape(postalCode))

	req, err := NewCarrierRequest(http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req = req.WithContext(ctx)

	resp, err := glsSpainHTTPClient.Do(req)
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

	var gResp glsSpainResponse
	if err := json.Unmarshal(bodyBytes, &gResp); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON from GLS Spain: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if gResp.Status == "NOT_FOUND" || (len(gResp.History) == 0 && gResp.Status == "") {
		return nil, models.ErrShipmentNotFound
	}

	return &gResp, nil
}

func (c *GLSSpainCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	postalCode := strings.TrimSpace(opts.PostalCode)
	if postalCode == "" {
		if !opts.AllowSimulated {
			return nil, fmt.Errorf("%w: GLS Spain tracking requires destination postal code (--postal-code <CP>)", models.ErrPostalCodeRequired)
		}
		return c.generateSimulatedShipment(cleanNum, opts), nil
	}

	gResp, err := c.fetchLive(ctx, cleanNum, postalCode)
	if err == nil && gResp != nil {
		var events []models.TrackingEvent
		for _, item := range gResp.History {
			evTime := parseGLSSpainDate(item.Date)
			status, stage := normalizeGLSSpainStatus(item.Status, item.Description)

			loc := strings.TrimSpace(item.Location)
			if loc != "" && !strings.Contains(strings.ToLower(loc), "spain") {
				loc += ", Spain"
			}

			// Deduplicate consecutive failed attempt / retry events within 15 minutes
			if len(events) > 0 {
				last := events[len(events)-1]
				if last.Status == status && last.Description == item.Description {
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
				RawDate:      item.Date,
				Status:       status,
				Stage:        stage,
				Description:  item.Description,
				Location:     loc,
				CarrierSlug:  c.Slug(),
				CarrierName:  c.Name(),
				CarrierIndex: 0,
			})
		}

		overallStatus, _ := normalizeGLSSpainStatus(gResp.Status, gResp.StatusDescription)
		for _, ev := range events {
			if ev.Status == models.StatusDelivered {
				overallStatus = models.StatusDelivered
				break
			}
		}

		statusText := gResp.StatusDescription
		if statusText == "" {
			statusText = gResp.Status
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
		if gResp.DeliveryDate != "" {
			t := parseGLSSpainDate(gResp.DeliveryDate)
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

		origCity := gResp.Origin.City
		if origCity == "" {
			origCity = "Spain"
		}
		destCity := gResp.Destination.City
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

func (c *GLSSpainCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Datos del paquete registrados en el sistema / Parcel registered in system",
			Location:    "Madrid Delegación ASM/GLS, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "El paquete ha salido del centro logístico / Parcel left sorting center",
			Location:    "San Fernando Hub, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto con el repartidor de GLS / Out for delivery with courier",
			Location:    "Madrid, Spain",
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
