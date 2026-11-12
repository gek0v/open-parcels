package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

var defaultInPostItalyJar, _ = cookiejar.New(nil)
var defaultInPostItalyHTTPClient = func() *http.Client {
	c := NewCarrierHTTPClient(DefaultRequestTimeout)
	c.Jar = defaultInPostItalyJar
	return c
}()
var inpostItalyHTTPClient = defaultInPostItalyHTTPClient

// SetInPostItalyHTTPClientForTest overrides the HTTP client for InPost Italy in tests.
func SetInPostItalyHTTPClientForTest(client *http.Client) func() {
	orig := inpostItalyHTTPClient
	inpostItalyHTTPClient = client
	return func() { inpostItalyHTTPClient = orig }
}

// InPostItalyCarrier represents InPost.it locker and delivery network.
type InPostItalyCarrier struct {
	BaseCarrier
}

func NewInPostItalyCarrier() *InPostItalyCarrier {
	bc := NewBaseCarrier(
		"inpost-it",
		"InPost.it",
		"IT",
		"Italy",
		models.CarrierTypeLogistics,
		"https://inpost.it",
		"https://inpost.it/trova-il-tuo-pacco?number=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &InPostItalyCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that InPost.it implements real network tracking.
func (c *InPostItalyCarrier) HasLiveTracking() bool {
	return true
}

func (c *InPostItalyCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	shipment, err := fetchInPostItalyLive(ctx, c, cleanNum, opts)
	if err == nil && shipment != nil {
		return shipment, nil
	}

	if opts.AllowSimulated || isSyntheticInPostItalyNumber(cleanNum) {
		return c.generateSimulatedShipment(cleanNum, opts), nil
	}

	if err != nil {
		return nil, err
	}
	return nil, models.ErrShipmentNotFound
}

func isSyntheticInPostItalyNumber(num string) bool {
	return strings.HasPrefix(num, "623456789") || strings.HasPrefix(num, "FAKE") || strings.HasPrefix(num, "TEST")
}

type shipxEventLocation struct {
	Type        string `json:"type"`
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
	LocationCode string `json:"locationCode"`
}

type shipxEvent struct {
	EventCode        string              `json:"eventCode"`
	Timestamp        string              `json:"timestamp"`
	Carrier          string              `json:"carrier"`
	EventTitle       string              `json:"eventTitle"`
	EventDescription string              `json:"eventDescription"`
	Location         *shipxEventLocation `json:"location"`
}

type shipxAddress struct {
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
	PostalCode  string `json:"postalCode"`
	Street      string `json:"street"`
	Province    string `json:"province"`
	Name        string `json:"name"`
}

type shipxTrackingResult struct {
	MainTrackingNumber string        `json:"mainTrackingNumber"`
	Events             []shipxEvent  `json:"events"`
	Origin             *shipxAddress `json:"origin"`
	Destination        *shipxAddress `json:"destination"`
}

func inpostCountryName(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "IT":
		return "Italy"
	case "ES":
		return "Spain"
	case "FR":
		return "France"
	case "PT":
		return "Portugal"
	case "PL":
		return "Poland"
	case "BE":
		return "Belgium"
	case "NL":
		return "Netherlands"
	case "LU":
		return "Luxembourg"
	case "DE":
		return "Germany"
	case "GB", "UK":
		return "United Kingdom"
	default:
		return strings.ToUpper(strings.TrimSpace(code))
	}
}

func fetchInPostItalyLive(ctx context.Context, carrier Carrier, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	client := inpostItalyHTTPClient
	landingURL := "https://inpost.it/en/find-parcel"

	visitLanding := func(c *http.Client) error {
		reqLanding, err := http.NewRequestWithContext(ctx, http.MethodGet, landingURL, nil)
		if err != nil {
			return err
		}
		reqLanding.Header.Set("User-Agent", ResolveUserAgent())
		reqLanding.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		reqLanding.Header.Set("Accept-Language", "it-IT,it;q=0.9,en;q=0.8")
		respLanding, err := c.Do(reqLanding)
		if err != nil {
			return fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
		}
		_, _ = io.Copy(io.Discard, respLanding.Body)
		respLanding.Body.Close()
		return nil
	}

	if client == defaultInPostItalyHTTPClient {
		if parsedLanding, err := url.Parse(landingURL); err == nil && client.Jar != nil && len(client.Jar.Cookies(parsedLanding)) == 0 {
			if err := visitLanding(client); err != nil {
				return nil, err
			}
		}
	}

	lang := "it"
	if strings.HasPrefix(strings.ToLower(opts.Language), "es") {
		lang = "es"
	} else if strings.HasPrefix(strings.ToLower(opts.Language), "en") {
		lang = "en"
	}

	apiURL := fmt.Sprintf(
		"https://inpost.it/shipx-proxy/?number=%s&returnTranslations=true&language=%s",
		url.QueryEscape(trackingNumber),
		url.QueryEscape(lang),
	)

	doAPIRequest := func(c *http.Client) (*http.Response, error) {
		reqAPI, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, err
		}
		reqAPI.Header.Set("User-Agent", ResolveUserAgent())
		reqAPI.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		reqAPI.Header.Set("Accept-Language", "it-IT,it;q=0.9,en;q=0.8")
		reqAPI.Header.Set("X-Requested-With", "XMLHttpRequest")
		reqAPI.Header.Set("Referer", landingURL)
		reqAPI.Header.Set("Sec-Fetch-Site", "same-origin")
		reqAPI.Header.Set("Sec-Fetch-Mode", "cors")
		reqAPI.Header.Set("Sec-Fetch-Dest", "empty")
		return c.Do(reqAPI)
	}

	respAPI, err := doAPIRequest(client)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if respAPI.StatusCode == http.StatusForbidden && client == defaultInPostItalyHTTPClient {
		_, _ = io.Copy(io.Discard, respAPI.Body)
		respAPI.Body.Close()
		// Refresh session cookie once on 403
		if freshJar, jarErr := cookiejar.New(nil); jarErr == nil {
			freshClient := &http.Client{
				Jar:       freshJar,
				Timeout:   client.Timeout,
				Transport: client.Transport,
			}
			if err := visitLanding(freshClient); err == nil {
				if retryResp, retryErr := doAPIRequest(freshClient); retryErr == nil {
					respAPI = retryResp
				}
			}
		}
	}
	defer respAPI.Body.Close()

	if respAPI.StatusCode == http.StatusNotFound {
		return nil, models.ErrShipmentNotFound
	}
	if respAPI.StatusCode == http.StatusTooManyRequests {
		return nil, models.ErrCarrierRateLimited
	}
	if respAPI.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP status %d", models.ErrCarrierTemporarilyUnavailable, respAPI.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(respAPI.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	var results []shipxTrackingResult
	if err := json.Unmarshal(bodyBytes, &results); err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	if len(results) == 0 || len(results[0].Events) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	res := results[0]
	var events []models.TrackingEvent
	var latestTitle string
	var latestDate time.Time

	for _, ev := range res.Events {
		evDate, err := parseInPostDate(ev.Timestamp)
		if err != nil {
			evDate = time.Now()
		}

		desc := ev.EventDescription
		if desc == "" {
			desc = ev.EventTitle
		}

		locStr := ""
		if ev.Location != nil {
			locStr = strings.TrimSpace(ev.Location.City)
			if ev.Location.CountryCode != "" {
				cName := inpostCountryName(ev.Location.CountryCode)
				if locStr != "" {
					locStr = fmt.Sprintf("%s, %s", locStr, cName)
				} else {
					locStr = cName
				}
			}
		}

		if ev.EventTitle != "" && (latestTitle == "" || evDate.After(latestDate)) {
			latestTitle = ev.EventTitle
			latestDate = evDate
		}

		status, stage := normalizeInPostEventStatus(ev.EventCode, ev.EventTitle, "", desc)

		events = append(events, models.TrackingEvent{
			Date:        evDate,
			RawDate:     ev.Timestamp,
			Status:      status,
			Stage:       stage,
			Description: desc,
			Location:    locStr,
			EventCode:   ev.EventCode,
			CarrierSlug: carrier.Slug(),
			CarrierName: carrier.Name(),
		})
	}

	if len(events) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Date.Before(events[j].Date)
	})

	latestEvent := events[len(events)-1]
	overallStatus := latestEvent.Status

	statusText := latestTitle
	if statusText == "" {
		statusText = latestEvent.Description
	}

	progress := inpostProgress(overallStatus)

	var deliveredAt *time.Time
	if overallStatus == models.StatusDelivered {
		for i := len(events) - 1; i >= 0; i-- {
			if events[i].Status == models.StatusDelivered {
				d := events[i].Date
				deliveredAt = &d
				break
			}
		}
	}

	daysInTransit := 0
	firstDate := events[0].Date
	if !firstDate.IsZero() && time.Since(firstDate) > 0 {
		daysInTransit = int(time.Since(firstDate).Hours() / 24)
	}

	// Origin
	origin := models.CountryInfo{Name: "Italy", Code: "IT"}
	if res.Origin != nil {
		cCode := strings.ToUpper(strings.TrimSpace(res.Origin.CountryCode))
		if cCode == "" {
			cCode = "IT"
		}
		cName := inpostCountryName(cCode)
		if res.Origin.City != "" {
			origin = models.CountryInfo{
				Name: fmt.Sprintf("%s, %s", strings.TrimSpace(res.Origin.City), cName),
				Code: cCode,
			}
		} else {
			origin = models.CountryInfo{Name: cName, Code: cCode}
		}
	} else {
		extracted := extractInPostOrigin(events)
		if extracted.Name != "" && extracted.Name != "Unknown" {
			origin = extracted
		}
	}

	// Destination
	destination := models.CountryInfo{Name: "Italy", Code: "IT"}
	if res.Destination != nil {
		cCode := strings.ToUpper(strings.TrimSpace(res.Destination.CountryCode))
		if cCode == "" {
			cCode = "IT"
		}
		cName := inpostCountryName(cCode)
		if res.Destination.City != "" {
			destination = models.CountryInfo{
				Name: fmt.Sprintf("%s, %s", strings.TrimSpace(res.Destination.City), cName),
				Code: cCode,
			}
		} else {
			destination = models.CountryInfo{Name: cName, Code: cCode}
		}
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        carrier.Info(),
		Status:         overallStatus,
		StatusText:     statusText,
		Progress:       progress,
		DeliveredAt:    deliveredAt,
		Origin:         origin,
		Destination:    destination,
		DaysInTransit:  daysInTransit,
		Events:         events,
		UpdatedAt:      time.Now(),
	}, nil
}

func (c *InPostItalyCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Spedizione registrata dal mittente per il punto di ritiro / locker",
			Location:    "Punto InPost Milano, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Pacco ritirato e in transito verso il centro logistico",
			Location:    "Hub InPost Bologna, Italy",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "In consegna presso il locker / Punto InPost di destinazione",
			Location:    "Locker InPost Roma, Italy",
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
			Name: "Italy",
			Code: "IT",
		},
		Destination: models.CountryInfo{
			Name: "Italy",
			Code: "IT",
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "In consegna presso Locker / Punto InPost",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}
}