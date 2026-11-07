package carriers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

var fedexHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetFedExHTTPClientForTest overrides the HTTP client for FedEx in deterministic tests.
func SetFedExHTTPClientForTest(client *http.Client) func() {
	orig := fedexHTTPClient
	fedexHTTPClient = client
	return func() { fedexHTTPClient = orig }
}

type FedExCarrier struct {
	BaseCarrier
}

func NewFedExCarrier() *FedExCarrier {
	bc := NewBaseCarrier(
		"fedex",
		"FedEx",
		"US",
		"United States",
		models.CarrierTypeExpress,
		"https://www.fedex.com",
		"https://www.fedex.com/fedextrack/?trknbr=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		RequiresAuth:    true,
		SupportsLiveAPI: true,
	})
	return &FedExCarrier{
		BaseCarrier: bc,
	}
}

func (c *FedExCarrier) HasLiveTracking() bool {
	return true
}

type fedexScanLocation struct {
	City                string `json:"city"`
	StateOrProvinceCode string `json:"stateOrProvinceCode"`
	CountryCode         string `json:"countryCode"`
}

type fedexScanEvent struct {
	Date             string            `json:"date"`
	EventType        string            `json:"eventType"`
	EventDescription string            `json:"eventDescription"`
	ScanLocation     fedexScanLocation `json:"scanLocation"`
}

type fedexDateAndTime struct {
	Type     string `json:"type"`
	DateTime string `json:"dateTime"`
}

type fedexTrackResultItem struct {
	TrackingNumberInfo struct {
		TrackingNumber string `json:"trackingNumber"`
	} `json:"trackingNumberInfo"`
	LatestStatusDetail struct {
		Code           string `json:"code"`
		DerivedCode    string `json:"derivedCode"`
		StatusByLocale string `json:"statusByLocale"`
		Description    string `json:"description"`
	} `json:"latestStatusDetail"`
	DateAndTimes []fedexDateAndTime `json:"dateAndTimes"`
	ScanEvents   []fedexScanEvent   `json:"scanEvents"`
}

type fedexCompleteTrackResult struct {
	TrackingNumber string                 `json:"trackingNumber"`
	TrackResults   []fedexTrackResultItem `json:"trackResults"`
}

type fedexTrackResponse struct {
	Output struct {
		CompleteTrackResults []fedexCompleteTrackResult `json:"completeTrackResults"`
	} `json:"output"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func parseFedExDate(dateStr string) time.Time {
	clean := strings.TrimSpace(dateStr)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
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

func normalizeFedExStatus(code, desc string) (models.Status, models.Stage) {
	combined := strings.ToUpper(strings.TrimSpace(code + " " + desc))

	switch {
	case strings.Contains(combined, "DELIVERED") || code == "DL":
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(combined, "ON VEHICLE") || strings.Contains(combined, "OUT FOR DELIVERY") || code == "OD":
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(combined, "IN TRANSIT") || strings.Contains(combined, "DEPARTED") || strings.Contains(combined, "ARRIVED") || code == "IT" || code == "AR" || code == "DP":
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(combined, "PICKED UP") || code == "PU":
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(combined, "INFORMATION SENT") || strings.Contains(combined, "LABEL CREATED") || code == "OC":
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(combined, "EXCEPTION") || strings.Contains(combined, "DELAY"):
		return models.StatusFailedAttempt, models.StageException
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func (c *FedExCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	creds := GetCarrierCredentials("fedex")
	if !creds.HasAuth() {
		if !opts.AllowSimulated {
			return nil, fmt.Errorf("%w: FedEx API requires OAuth2 credentials, set FEDEX_CLIENT_ID and FEDEX_CLIENT_SECRET (or OPEN_PARCELS_FEDEX_CLIENT_ID)", models.ErrCarrierNotImplemented)
		}
		return c.generateSimulatedShipment(cleanNum, opts), nil
	}

	// 1. Obtain OAuth2 Token
	tokenURL := "https://apis.fedex.com/oauth/token"
	accessToken, err := globalOAuthCache.GetToken(ctx, "fedex", tokenURL, creds.ClientID, creds.APISecret, fedexHTTPClient, false)
	if err != nil {
		if opts.AllowSimulated {
			return c.generateSimulatedShipment(cleanNum, opts), nil
		}
		return nil, err
	}

	// 2. Query FedEx Track API
	apiURL := "https://apis.fedex.com/track/v1/trackingnumbers"
	payload := map[string]any{
		"includeDetailedScans": true,
		"trackingInfo": []any{
			map[string]any{
				"trackingNumberInfo": map[string]any{
					"trackingNumber": cleanNum,
				},
			},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", ResolveUserAgent())
	req.Header.Set("Accept", "application/json")

	resp, err := fedexHTTPClient.Do(req)
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
		return nil, fmt.Errorf("%w: invalid FedEx credentials or rejected token", models.ErrCarrierNotImplemented)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: FedEx Track API returned HTTP %d", models.ErrCarrierTemporarilyUnavailable, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	var fResp fedexTrackResponse
	if err := json.Unmarshal(bodyBytes, &fResp); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON from FedEx: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if len(fResp.Errors) > 0 {
		return nil, models.ErrShipmentNotFound
	}
	if len(fResp.Output.CompleteTrackResults) == 0 || len(fResp.Output.CompleteTrackResults[0].TrackResults) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	trackRes := fResp.Output.CompleteTrackResults[0].TrackResults[0]
	var events []models.TrackingEvent
	for _, sc := range trackRes.ScanEvents {
		evTime := parseFedExDate(sc.Date)
		status, stage := normalizeFedExStatus(sc.EventType, sc.EventDescription)

		locParts := []string{}
		if sc.ScanLocation.City != "" {
			locParts = append(locParts, sc.ScanLocation.City)
		}
		if sc.ScanLocation.StateOrProvinceCode != "" {
			locParts = append(locParts, sc.ScanLocation.StateOrProvinceCode)
		}
		if sc.ScanLocation.CountryCode != "" {
			locParts = append(locParts, sc.ScanLocation.CountryCode)
		}
		loc := strings.Join(locParts, ", ")

		events = append(events, models.TrackingEvent{
			Date:         evTime,
			RawDate:      sc.Date,
			Status:       status,
			Stage:        stage,
			Description:  sc.EventDescription,
			Location:     loc,
			CarrierSlug:  c.Slug(),
			CarrierName:  c.Name(),
			CarrierIndex: 0,
		})
	}

	overallStatus, _ := normalizeFedExStatus(trackRes.LatestStatusDetail.Code, trackRes.LatestStatusDetail.Description)
	for _, ev := range events {
		if ev.Status == models.StatusDelivered {
			overallStatus = models.StatusDelivered
			break
		}
	}

	statusText := trackRes.LatestStatusDetail.Description
	if statusText == "" {
		statusText = trackRes.LatestStatusDetail.StatusByLocale
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
	for _, dt := range trackRes.DateAndTimes {
		if dt.Type == "ACTUAL_DELIVERY" {
			t := parseFedExDate(dt.DateTime)
			if !t.IsZero() {
				deliveredAt = &t
			}
			break
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

	return &models.Shipment{
		TrackingNumber: cleanNum,
		Carrier:        c.Info(),
		Status:         overallStatus,
		StatusText:     statusText,
		Progress:       progress,
		DeliveredAt:    deliveredAt,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

func (c *FedExCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Shipment information sent to FedEx",
			Location:    "Memphis, TN, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Picked up and left FedEx origin facility",
			Location:    "Memphis World Hub, TN, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "On FedEx vehicle for delivery",
			Location:    "Destination Facility, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Status:         models.StatusOutForDelivery,
		StatusText:     "On FedEx vehicle for delivery",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}
}
