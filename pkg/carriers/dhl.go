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

var dhlHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetDHLHTTPClientForTest overrides the HTTP client for DHL in deterministic tests.
func SetDHLHTTPClientForTest(client *http.Client) func() {
	orig := dhlHTTPClient
	dhlHTTPClient = client
	return func() { dhlHTTPClient = orig }
}

type DHLExpressCarrier struct {
	BaseCarrier
}

func NewDHLExpressCarrier() *DHLExpressCarrier {
	bc := NewBaseCarrier(
		"dhl-express",
		"DHL Express",
		"GLOBAL",
		"Global",
		models.CarrierTypeExpress,
		"https://www.dhl.com",
		"https://www.dhl.com/en/express/tracking.html?AWB=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		RequiresAuth:    true,
		SupportsLiveAPI: true,
	})
	return &DHLExpressCarrier{
		BaseCarrier: bc,
	}
}

func (c *DHLExpressCarrier) HasLiveTracking() bool {
	return true
}

func (c *DHLExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return trackDHLShared(ctx, c, trackingNumber, "express", opts, c.generateSimulatedShipment)
}

func (c *DHLExpressCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Shipment picked up",
			Location:    "Frankfurt, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Processed at Leipzig Hub, Germany",
			Location:    "Leipzig Hub, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "With delivery courier - Madrid, Spain",
			Location:    "Madrid Delivery Station, Spain",
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
			Name: "Germany",
			Code: "DE",
		},
		Destination: models.CountryInfo{
			Name: "Spain",
			Code: "ES",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "With delivery courier",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}
}

type DHLParcelCarrier struct {
	BaseCarrier
}

func NewDHLParcelCarrier() *DHLParcelCarrier {
	bc := NewBaseCarrier(
		"dhl-parcel",
		"DHL Parcel",
		"DE",
		"Germany",
		models.CarrierTypeLogistics,
		"https://www.dhl.de",
		"https://www.dhl.de/en/privatkunden/pakete-empfangen/verfolgen.html?piececode=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		RequiresAuth:    true,
		SupportsLiveAPI: true,
	})
	return &DHLParcelCarrier{
		BaseCarrier: bc,
	}
}

func (c *DHLParcelCarrier) HasLiveTracking() bool {
	return true
}

func (c *DHLParcelCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	service := "parcel-de"
	if len(trackingNumber) == 10 && isAllDigits(trackingNumber) {
		service = "express"
	}
	return trackDHLShared(ctx, c, trackingNumber, service, opts, c.generateSimulatedShipment)
}

func (c *DHLParcelCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "The instruction data for this shipment have been provided by the sender to DHL electronically",
			Location:    "Bonn, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "The shipment has been loaded onto the delivery vehicle",
			Location:    "Köln Paketzentrum, Germany",
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
			Name: "Germany",
			Code: "DE",
		},
		Destination: models.CountryInfo{
			Name: "Germany",
			Code: "DE",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}
}

type dhlAddress struct {
	AddressLocality string `json:"addressLocality"`
	CountryCode     string `json:"countryCode"`
}

type dhlOriginDestination struct {
	Address dhlAddress `json:"address"`
}

type dhlEventLocation struct {
	Address dhlAddress `json:"address"`
}

type dhlStatus struct {
	Timestamp   string `json:"timestamp"`
	StatusCode  string `json:"statusCode"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type dhlEventItem struct {
	Timestamp   string           `json:"timestamp"`
	StatusCode  string           `json:"statusCode"`
	Status      string           `json:"status"`
	Description string           `json:"description"`
	Location    dhlEventLocation `json:"location"`
}

type dhlShipmentItem struct {
	ID          string                `json:"id"`
	Service     string                `json:"service"`
	Origin      dhlOriginDestination `json:"origin"`
	Destination dhlOriginDestination `json:"destination"`
	Status      dhlStatus             `json:"status"`
	Events      []dhlEventItem        `json:"events"`
}

type dhlTrackResponse struct {
	Shipments []dhlShipmentItem `json:"shipments"`
	Title     string            `json:"title,omitempty"`
	Status    int               `json:"status,omitempty"`
	Detail    string            `json:"detail,omitempty"`
}

func parseDHLDate(dateStr string) time.Time {
	clean := strings.TrimSpace(dateStr)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"02/01/2006 15:04:05",
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

func normalizeDHLStatus(statusCode, status, desc string) (models.Status, models.Stage) {
	combined := strings.ToUpper(strings.TrimSpace(statusCode + " " + status + " " + desc))

	switch {
	case strings.Contains(combined, "DELIVERED"):
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(combined, "OUT_FOR_DELIVERY") || strings.Contains(combined, "DELIVERY VEHICLE") || strings.Contains(combined, "DELIVERY COURIER"):
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(combined, "TRANSIT") || strings.Contains(combined, "SORTING") || strings.Contains(combined, "HUB") || strings.Contains(combined, "FACILITY"):
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(combined, "PICKED_UP") || strings.Contains(combined, "COLLECTED") || strings.Contains(combined, "ADMITTED"):
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(combined, "PRE-TRANSIT") || strings.Contains(combined, "INSTRUCTION") || strings.Contains(combined, "ELECTRONIC"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(combined, "FAILED") || strings.Contains(combined, "EXCEPTION") || strings.Contains(combined, "RETURNED"):
		return models.StatusFailedAttempt, models.StageException
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func trackDHLShared(
	ctx context.Context,
	carrier Carrier,
	trackingNumber string,
	serviceHint string,
	opts models.TrackOptions,
	simulatedFn func(string, models.TrackOptions) *models.Shipment,
) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	creds := GetCarrierCredentials("dhl")
	if !creds.HasAuth() {
		if !opts.AllowSimulated {
			return nil, fmt.Errorf("%w: DHL API requires authentication, set DHL_API_KEY or OPEN_PARCELS_DHL_KEY", models.ErrCarrierNotImplemented)
		}
		if simulatedFn != nil {
			return simulatedFn(cleanNum, opts), nil
		}
		return nil, models.ErrShipmentNotFound
	}

	lang := "en"
	if strings.HasPrefix(strings.ToLower(opts.Language), "de") {
		lang = "de"
	} else if strings.HasPrefix(strings.ToLower(opts.Language), "es") {
		lang = "es"
	}

	service := serviceHint
	if len(cleanNum) == 10 && isAllDigits(cleanNum) {
		service = "express"
	}

	apiURL := fmt.Sprintf("https://api-eu.dhl.com/track/shipments?trackingNumber=%s&service=%s&language=%s",
		url.QueryEscape(cleanNum), url.QueryEscape(service), url.QueryEscape(lang))

	req, err := NewCarrierRequest(http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req.Header.Set("DHL-API-Key", creds.APIKey)
	req = req.WithContext(ctx)

	resp, err := dhlHTTPClient.Do(req)
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
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: invalid DHL API key or authentication rejected", models.ErrCarrierNotImplemented)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP status %d", models.ErrCarrierTemporarilyUnavailable, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	var dResp dhlTrackResponse
	if err := json.Unmarshal(bodyBytes, &dResp); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON from DHL: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if len(dResp.Shipments) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	item := dResp.Shipments[0]
	var events []models.TrackingEvent
	for _, ev := range item.Events {
		evTime := parseDHLDate(ev.Timestamp)
		status, stage := normalizeDHLStatus(ev.StatusCode, ev.Status, ev.Description)

		locParts := []string{}
		if ev.Location.Address.AddressLocality != "" {
			locParts = append(locParts, ev.Location.Address.AddressLocality)
		}
		if ev.Location.Address.CountryCode != "" {
			locParts = append(locParts, ev.Location.Address.CountryCode)
		}
		loc := strings.Join(locParts, ", ")

		events = append(events, models.TrackingEvent{
			Date:         evTime,
			RawDate:      ev.Timestamp,
			Status:       status,
			Stage:        stage,
			Description:  ev.Description,
			Location:     loc,
			CarrierSlug:  carrier.Slug(),
			CarrierName:  carrier.Name(),
			CarrierIndex: 0,
		})
	}

	overallStatus, _ := normalizeDHLStatus(item.Status.StatusCode, item.Status.Status, item.Status.Description)
	for _, ev := range events {
		if ev.Status == models.StatusDelivered {
			overallStatus = models.StatusDelivered
			break
		}
	}

	statusText := item.Status.Description
	if statusText == "" {
		statusText = item.Status.Status
	}
	if statusText == "" {
		if overallStatus == models.StatusDelivered {
			statusText = "Delivered"
		} else if overallStatus == models.StatusOutForDelivery {
			statusText = "Out for delivery"
		} else {
			statusText = "In transit"
		}
	}

	progress := 50
	if overallStatus == models.StatusDelivered {
		progress = 100
	} else if overallStatus == models.StatusOutForDelivery {
		progress = 85
	}

	var deliveredAt *time.Time
	if overallStatus == models.StatusDelivered && item.Status.Timestamp != "" {
		t := parseDHLDate(item.Status.Timestamp)
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

	origCity := item.Origin.Address.AddressLocality
	if origCity == "" {
		origCity = "Germany"
	}
	destCity := item.Destination.Address.AddressLocality
	if destCity == "" {
		destCity = "Destination"
	}

	return &models.Shipment{
		TrackingNumber: cleanNum,
		Carrier:        carrier.Info(),
		Origin: models.CountryInfo{
			Name: origCity,
			Code: item.Origin.Address.CountryCode,
		},
		Destination: models.CountryInfo{
			Name: destCity,
			Code: item.Destination.Address.CountryCode,
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
