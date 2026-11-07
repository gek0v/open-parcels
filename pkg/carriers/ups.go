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

var upsHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetUPSHTTPClientForTest overrides the HTTP client for UPS in deterministic tests.
func SetUPSHTTPClientForTest(client *http.Client) func() {
	orig := upsHTTPClient
	upsHTTPClient = client
	return func() { upsHTTPClient = orig }
}

type UPSCarrier struct {
	BaseCarrier
}

func NewUPSCarrier() *UPSCarrier {
	bc := NewBaseCarrier(
		"ups",
		"UPS",
		"US",
		"United States",
		models.CarrierTypeExpress,
		"https://www.ups.com",
		"https://www.ups.com/track?tracknum=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		RequiresAuth:    true,
		SupportsLiveAPI: true,
	})
	return &UPSCarrier{
		BaseCarrier: bc,
	}
}

func (c *UPSCarrier) HasLiveTracking() bool {
	return true
}

type upsActivityLocation struct {
	Address struct {
		City        string `json:"city"`
		CountryCode string `json:"country"`
	} `json:"address"`
}

type upsActivityStatus struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Code        string `json:"code"`
}

type upsActivity struct {
	Date     string              `json:"date"`
	Time     string              `json:"time"`
	Status   upsActivityStatus   `json:"status"`
	Location upsActivityLocation `json:"location"`
}

type upsCurrentStatus struct {
	Description string `json:"description"`
	Code        string `json:"code"`
}

type upsPackage struct {
	TrackingNumber string           `json:"trackingNumber"`
	CurrentStatus  upsCurrentStatus `json:"currentStatus"`
	Activity       []upsActivity    `json:"activity"`
}

type upsShipmentItem struct {
	InquiryNumber string       `json:"inquiryNumber"`
	Package       []upsPackage `json:"package"`
}

type upsTrackResponse struct {
	TrackResponse struct {
		Shipment []upsShipmentItem `json:"shipment"`
	} `json:"trackResponse"`
	Response struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"response"`
}

func parseUPSDate(dateStr, timeStr string) time.Time {
	cleanDate := strings.TrimSpace(dateStr)
	cleanTime := strings.TrimSpace(timeStr)

	combined := cleanDate
	if cleanTime != "" {
		combined = cleanDate + cleanTime
	}

	layouts := []string{
		"20060102150405",
		"20060102",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, combined, time.UTC); err == nil {
			return t
		}
	}

	return time.Time{}
}

func normalizeUPSStatus(code, desc string) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(code + " " + desc))

	switch {
	case strings.Contains(upper, "DELIVERED") || code == "FS" || code == "D":
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(upper, "OUT FOR DELIVERY") || code == "OT":
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(upper, "IN TRANSIT") || strings.Contains(upper, "DEPARTED") || strings.Contains(upper, "FACILITY") || code == "DP":
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(upper, "PICKED UP") || strings.Contains(upper, "ORIGIN SCAN") || code == "OR":
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(upper, "LABEL CREATED") || strings.Contains(upper, "SHIPPER CREATED") || code == "MP":
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(upper, "EXCEPTION") || strings.Contains(upper, "DELAY"):
		return models.StatusFailedAttempt, models.StageException
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func (c *UPSCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	creds := GetCarrierCredentials("ups")
	if !creds.HasAuth() {
		if !opts.AllowSimulated {
			return nil, fmt.Errorf("%w: UPS API requires OAuth2 credentials, set UPS_CLIENT_ID and UPS_CLIENT_SECRET (or OPEN_PARCELS_UPS_CLIENT_ID)", models.ErrCarrierNotImplemented)
		}
		return c.generateSimulatedShipment(cleanNum, opts), nil
	}

	// 1. Obtain OAuth2 Token
	tokenURL := "https://onlinetools.ups.com/security/v1/oauth/token"
	accessToken, err := globalOAuthCache.GetToken(ctx, "ups", tokenURL, creds.ClientID, creds.APISecret, upsHTTPClient, true)
	if err != nil {
		if opts.AllowSimulated {
			return c.generateSimulatedShipment(cleanNum, opts), nil
		}
		return nil, err
	}

	// 2. Query UPS Track API
	apiURL := fmt.Sprintf("https://onlinetools.ups.com/api/track/v1/details/%s", url.PathEscape(cleanNum))
	req, err := NewCarrierRequest(http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("transId", fmt.Sprintf("tx-%d", time.Now().UnixNano()))
	req.Header.Set("transactionSrc", "open-parcels")
	req = req.WithContext(ctx)

	resp, err := upsHTTPClient.Do(req)
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
		return nil, fmt.Errorf("%w: invalid UPS credentials or rejected token", models.ErrCarrierNotImplemented)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: UPS Track API returned HTTP %d", models.ErrCarrierTemporarilyUnavailable, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	var uResp upsTrackResponse
	if err := json.Unmarshal(bodyBytes, &uResp); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON from UPS: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if len(uResp.Response.Errors) > 0 {
		return nil, models.ErrShipmentNotFound
	}
	if len(uResp.TrackResponse.Shipment) == 0 || len(uResp.TrackResponse.Shipment[0].Package) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	pkg := uResp.TrackResponse.Shipment[0].Package[0]
	var events []models.TrackingEvent
	for _, act := range pkg.Activity {
		evTime := parseUPSDate(act.Date, act.Time)
		status, stage := normalizeUPSStatus(act.Status.Code, act.Status.Description)

		locParts := []string{}
		if act.Location.Address.City != "" {
			locParts = append(locParts, act.Location.Address.City)
		}
		if act.Location.Address.CountryCode != "" {
			locParts = append(locParts, act.Location.Address.CountryCode)
		}
		loc := strings.Join(locParts, ", ")

		events = append(events, models.TrackingEvent{
			Date:         evTime,
			RawDate:      fmt.Sprintf("%s %s", act.Date, act.Time),
			Status:       status,
			Stage:        stage,
			Description:  act.Status.Description,
			Location:     loc,
			CarrierSlug:  c.Slug(),
			CarrierName:  c.Name(),
			CarrierIndex: 0,
		})
	}

	overallStatus, _ := normalizeUPSStatus(pkg.CurrentStatus.Code, pkg.CurrentStatus.Description)
	for _, ev := range events {
		if ev.Status == models.StatusDelivered {
			overallStatus = models.StatusDelivered
			break
		}
	}

	statusText := pkg.CurrentStatus.Description
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
	for _, ev := range events {
		if ev.Status == models.StatusDelivered {
			t := ev.Date
			deliveredAt = &t
			break
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

func (c *UPSCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-48 * time.Hour),
			Description: "Shipper created a label, UPS has not received the package yet",
			Location:    "Louisville, KY, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Origin Scan / Picked up by UPS",
			Location:    "Worldport Hub, Louisville, KY, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Departed from Facility",
			Location:    "Philadelphia Hub, PA, United States",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Out For Delivery - Today by 7:00 PM",
			Location:    "Local Delivery Facility, United States",
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
		StatusText:     "Out For Delivery - Today by 7:00 PM",
		Progress:       85,
		DaysInTransit:  2,
		Events:         events,
		UpdatedAt:      time.Now(),
	}
}
