package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

var (
	yunExpressBaseURL    = "https://www.yuntrack.com"
	yunExpressHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)
)

// SetYunExpressBaseURLForTest overrides the YunExpress base URL in deterministic tests.
func SetYunExpressBaseURLForTest(baseURL string) func() {
	orig := yunExpressBaseURL
	yunExpressBaseURL = strings.TrimRight(baseURL, "/")
	return func() { yunExpressBaseURL = orig }
}

// SetYunExpressHTTPClientForTest overrides the HTTP client for YunExpress in deterministic tests.
func SetYunExpressHTTPClientForTest(client *http.Client) func() {
	orig := yunExpressHTTPClient
	yunExpressHTTPClient = client
	return func() { yunExpressHTTPClient = orig }
}

type YunExpressCarrier struct {
	BaseCarrier
}

func NewYunExpressCarrier() *YunExpressCarrier {
	bc := NewBaseCarrier(
		"yunexpress",
		"YunExpress",
		"CN",
		"China",
		models.CarrierTypeCrossBorder,
		"https://www.yunexpress.com",
		"https://www.yuntrack.com/parcelTracking?id=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &YunExpressCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that YunExpress implements real network tracking.
func (c *YunExpressCarrier) HasLiveTracking() bool {
	return true
}

type yunExpressEventItem struct {
	Date        string `json:"date"`
	Time        string `json:"time"`
	Status      string `json:"status"`
	Stage       string `json:"stage"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Carrier     string `json:"carrier"`
	CarrierSlug string `json:"carrierSlug"`
}

type yunExpressTrackItem struct {
	TrackingNumber         string                `json:"trackingNumber"`
	WaybillNumber          string                `json:"waybillNumber"`
	LastMileTrackingNumber string                `json:"lastMileTrackingNumber"`
	LastMileCarrier        string                `json:"lastMileCarrier"`
	LastMileCarrierSlug    string                `json:"lastMileCarrierSlug"`
	Status                 string                `json:"status"`
	StatusDescription      string                `json:"statusDescription"`
	OriginCountry          string                `json:"originCountry"`
	OriginCountryName      string                `json:"originCountryName"`
	DestinationCountry     string                `json:"destinationCountry"`
	DestinationCountryName string                `json:"destinationCountryName"`
	Events                 []yunExpressEventItem `json:"events"`
}

type yunExpressAPIResponse struct {
	Success bool            `json:"success"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Items   json.RawMessage `json:"items"`
}

var (
	yunHTMLTagRegex     = regexp.MustCompile(`<[^>]+>`)
	yunTRRegex          = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	yunTDRegex          = regexp.MustCompile(`(?is)<t[dh][^>]*>(.*?)</t[dh]>`)
	yunDatePatternRegex = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}(?:[T\s]\d{2}:\d{2}(?::\d{2})?)?\b|\b\d{2}/\d{2}/\d{4}(?:\s+\d{2}:\d{2}(?::\d{2})?)?\b`)
)

func parseYunExpressDate(dateStr, timeStr string) time.Time {
	cleanDate := strings.TrimSpace(dateStr)
	cleanTime := strings.TrimSpace(timeStr)
	combined := cleanDate
	if cleanTime != "" && !strings.Contains(cleanDate, ":") {
		combined = cleanDate + " " + cleanTime
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"2006-01-02",
		"02/01/2006",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, combined, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

func normalizeYunExpressStatus(status, desc string) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(status + " " + desc))

	switch {
	case strings.Contains(upper, "DELIVERED") || strings.Contains(upper, "SIGNED") || strings.Contains(upper, "ENTREGADO"):
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(upper, "OUT FOR DELIVERY") || strings.Contains(upper, "OUT_FOR_DELIVERY") || strings.Contains(upper, "DELIVERING") || strings.Contains(upper, "EN REPARTO"):
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(upper, "FAILED") || strings.Contains(upper, "UNDELIVERED") || strings.Contains(upper, "INCIDENCIA"):
		return models.StatusFailedAttempt, models.StageException
	case strings.Contains(upper, "EXCEPTION") || strings.Contains(upper, "HELD") || strings.Contains(upper, "RETURNED"):
		return models.StatusException, models.StageException
	case strings.Contains(upper, "CUSTOMS CLEARANCE COMPLETED") || strings.Contains(upper, "CUSTOMS CLEARED") || strings.Contains(upper, "IMPORT CUSTOMS"):
		return models.StatusInTransit, models.StageCustoms
	case strings.Contains(upper, "CUSTOMS") || strings.Contains(upper, "ADUAN"):
		return models.StatusInTransit, models.StageCustoms
	case strings.Contains(upper, "HANDED OVER TO LOCAL") || strings.Contains(upper, "HANDOVER") || strings.Contains(upper, "TRANSFERRED TO") || strings.Contains(upper, "DELIVERY PARTNER"):
		return models.StatusInTransit, models.StageHandover
	case strings.Contains(upper, "FLIGHT") || strings.Contains(upper, "AIRLINE") || strings.Contains(upper, "DEPARTED FROM AIRPORT") || strings.Contains(upper, "ARRIVED AT DESTINATION"):
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(upper, "SORTING") || strings.Contains(upper, "HUB") || strings.Contains(upper, "FACILITY") || strings.Contains(upper, "DEPARTED") || strings.Contains(upper, "IN TRANSIT") || strings.Contains(upper, "IN_TRANSIT"):
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(upper, "PICKED UP") || strings.Contains(upper, "ACCEPTED") || strings.Contains(upper, "RECEIVED AT FACILITY") || strings.Contains(upper, "RECOGIDO"):
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(upper, "SHIPMENT INFORMATION RECEIVED") || strings.Contains(upper, "INFO RECEIVED") || strings.Contains(upper, "INFO_RECEIVED") || strings.Contains(upper, "CREATED") || strings.Contains(upper, "REGISTRADO"):
		return models.StatusInfoReceived, models.StageCreated
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func isYunExpressBlocked(statusCode int, body string) bool {
	if statusCode == http.StatusForbidden {
		return true
	}
	upper := strings.ToUpper(body)
	return strings.Contains(upper, "RGV") ||
		strings.Contains(upper, "SEC_CHALLENGE") ||
		strings.Contains(upper, "ANTI_BOT") ||
		strings.Contains(upper, "ANTI-BOT") ||
		strings.Contains(upper, "CAPTCHA") ||
		strings.Contains(upper, "INCAPSULA") ||
		strings.Contains(upper, "CLOUDFLARE") ||
		strings.Contains(upper, "ACCESS DENIED") ||
		strings.Contains(upper, "WAF")
}

func isYunExpressNotFound(body string) bool {
	upper := strings.ToUpper(body)
	return strings.Contains(upper, "ORDER_NOT_FOUND") ||
		strings.Contains(upper, "PACKAGE_NOT_FOUND") ||
		strings.Contains(upper, "TRACKING_NUMBER_NOT_FOUND") ||
		strings.Contains(upper, "NOT_FOUND") ||
		strings.Contains(upper, "NOT FOUND") ||
		strings.Contains(upper, "NO TRACKING INFORMATION") ||
		strings.Contains(upper, "INEXISTENTE") ||
		strings.Contains(upper, "NO EXISTE")
}

func resolveYunExpressLastMileCarrier(lastMileCarrier, lastMileNum, destCode string) *models.CarrierInfo {
	clean := strings.ToLower(strings.TrimSpace(lastMileCarrier))
	cleanDest := strings.ToUpper(strings.TrimSpace(destCode))

	switch {
	case strings.Contains(clean, "correos express") || (clean == "" && strings.HasPrefix(lastMileNum, "323") && cleanDest == "ES"):
		return &models.CarrierInfo{
			Slug:        "correosexpress",
			Name:        "Correos Express",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeExpress,
			Website:     "https://www.correosexpress.com",
			TrackingURL: "https://www.correosexpress.com/web/correosexpress/consultanos?shippingNumber=%s",
		}
	case strings.Contains(clean, "correos") || (clean == "" && (strings.HasPrefix(lastMileNum, "PQ") || strings.HasPrefix(lastMileNum, "PK")) && (cleanDest == "ES" || cleanDest == "")):
		return &models.CarrierInfo{
			Slug:        "correos-spain",
			Name:        "Correos Espa?a",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypePostal,
			Website:     "https://www.correos.es",
			TrackingURL: "https://www.correos.es/es/es/herramientas/localizador/envios/detalle?tracking-number=%s",
		}
	case strings.Contains(clean, "ctt"):
		return &models.CarrierInfo{
			Slug:        "ctt-express",
			Name:        "CTT Express",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeExpress,
			Website:     "https://www.cttexpress.com",
			TrackingURL: "https://www.cttexpress.com/localizador-de-envios/?sc=%s",
		}
	case strings.Contains(clean, "gls"):
		return &models.CarrierInfo{
			Slug:        "asm-gls-spain",
			Name:        "GLS Spain",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeExpress,
			Website:     "https://www.gls-spain.es",
		}
	case strings.Contains(clean, "inpost") || strings.Contains(clean, "mondial"):
		return &models.CarrierInfo{
			Slug:        "inpost-es",
			Name:        "InPost Spain",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeLogistics,
			Website:     "https://inpost.es",
		}
	case strings.Contains(clean, "seur"):
		return &models.CarrierInfo{
			Slug:        "seur",
			Name:        "SEUR",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeExpress,
			Website:     "https://www.seur.com",
		}
	case strings.Contains(clean, "royal mail"):
		return &models.CarrierInfo{
			Slug:        "royalmail",
			Name:        "Royal Mail",
			Country:     "GB",
			CountryName: "United Kingdom",
			Type:        models.CarrierTypePostal,
			Website:     "https://www.royalmail.com",
		}
	case strings.Contains(clean, "usps"):
		return &models.CarrierInfo{
			Slug:        "usps",
			Name:        "USPS",
			Country:     "US",
			CountryName: "United States",
			Type:        models.CarrierTypePostal,
			Website:     "https://www.usps.com",
		}
	case strings.Contains(clean, "colissimo") || strings.Contains(clean, "la poste"):
		return &models.CarrierInfo{
			Slug:        "colissimo",
			Name:        "Colissimo",
			Country:     "FR",
			CountryName: "France",
			Type:        models.CarrierTypePostal,
			Website:     "https://www.laposte.fr",
		}
	case clean != "":
		slug := strings.ToLower(strings.ReplaceAll(clean, " ", "-"))
		return &models.CarrierInfo{
			Slug: slug,
			Name: lastMileCarrier,
		}
	default:
		return nil
	}
}

func (c *YunExpressCarrier) parseYunJSON(bodyBytes []byte) (*yunExpressTrackItem, error) {
	var resp yunExpressAPIResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		// Try direct array or object
		var single yunExpressTrackItem
		if errSingle := json.Unmarshal(bodyBytes, &single); errSingle == nil && (single.TrackingNumber != "" || len(single.Events) > 0) {
			return &single, nil
		}
		var list []yunExpressTrackItem
		if errList := json.Unmarshal(bodyBytes, &list); errList == nil && len(list) > 0 {
			return &list[0], nil
		}
		return nil, err
	}

	if (!resp.Success && resp.Code != "" && resp.Code != "200") || strings.Contains(strings.ToUpper(resp.Message), "NOT FOUND") {
		return nil, models.ErrShipmentNotFound
	}

	raw := resp.Data
	if len(raw) == 0 {
		raw = resp.Items
	}
	if len(raw) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	var items []yunExpressTrackItem
	if err := json.Unmarshal(raw, &items); err == nil && len(items) > 0 {
		return &items[0], nil
	}

	var item yunExpressTrackItem
	if err := json.Unmarshal(raw, &item); err == nil && (item.TrackingNumber != "" || len(item.Events) > 0) {
		return &item, nil
	}

	return nil, models.ErrScraperSchemaChanged
}

func (c *YunExpressCarrier) parseYunHTML(htmlContent string) (*yunExpressTrackItem, error) {
	res := &yunExpressTrackItem{}

	rows := yunTRRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, row := range rows {
		cells := yunTDRegex.FindAllStringSubmatch(row[1], -1)
		if len(cells) < 2 {
			continue
		}
		var cleaned []string
		for _, cell := range cells {
			txt := stripHTMLClean(cell[1])
			if txt != "" {
				cleaned = append(cleaned, txt)
			}
		}
		if len(cleaned) < 2 {
			continue
		}

		dateIdx := -1
		for i, col := range cleaned {
			if yunDatePatternRegex.MatchString(col) {
				dateIdx = i
				break
			}
		}
		if dateIdx == -1 {
			continue
		}

		dateVal := cleaned[dateIdx]
		var statusVal, locVal, descVal string
		rest := make([]string, 0, len(cleaned)-1)
		for i, col := range cleaned {
			if i != dateIdx {
				rest = append(rest, col)
			}
		}

		switch len(rest) {
		case 1:
			descVal = rest[0]
			statusVal = rest[0]
		case 2:
			statusVal = rest[0]
			descVal = rest[1]
		default:
			statusVal = rest[0]
			locVal = rest[1]
			descVal = rest[2]
		}

		res.Events = append(res.Events, yunExpressEventItem{
			Date:        dateVal,
			Status:      statusVal,
			Location:    locVal,
			Description: descVal,
		})
	}

	if len(res.Events) == 0 {
		return nil, models.ErrScraperSchemaChanged
	}
	return res, nil
}

func (c *YunExpressCarrier) fetchLive(ctx context.Context, trackingNumber string) (*yunExpressTrackItem, error) {
	targetURL := fmt.Sprintf("%s/api/tracker/track?nums=%s", yunExpressBaseURL, url.QueryEscape(trackingNumber))

	req, err := NewCarrierRequest(http.MethodGet, targetURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req.Header.Set("Accept", "application/json, text/html, application/xhtml+xml, */*")
	req = req.WithContext(ctx)

	resp, err := yunExpressHTTPClient.Do(req)
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
	if resp.StatusCode == http.StatusForbidden {
		return nil, models.ErrCarrierBlocked
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP status %d", models.ErrCarrierTemporarilyUnavailable, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	bodyStr := string(bodyBytes)
	if isYunExpressBlocked(resp.StatusCode, bodyStr) {
		return nil, models.ErrCarrierBlocked
	}
	if isYunExpressNotFound(bodyStr) {
		return nil, models.ErrShipmentNotFound
	}

	trimmed := strings.TrimSpace(bodyStr)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") || strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		return c.parseYunJSON(bodyBytes)
	}

	return c.parseYunHTML(bodyStr)
}

func (c *YunExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	destCountry := "Spain"
	destCode := "ES"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	yItem, err := c.fetchLive(ctx, cleanNum)
	if err == nil && yItem != nil {
		if yItem.DestinationCountry != "" {
			destCode = yItem.DestinationCountry
			destCountry = yItem.DestinationCountry
			if yItem.DestinationCountryName != "" {
				destCountry = yItem.DestinationCountryName
			}
		}

		lastMileNum := yItem.LastMileTrackingNumber
		lastMileCarrierInfo := resolveYunExpressLastMileCarrier(yItem.LastMileCarrier, lastMileNum, destCode)

		var events []models.TrackingEvent
		for _, item := range yItem.Events {
			rawDate := item.Date
			if rawDate == "" {
				rawDate = item.Time
			}
			evTime := parseYunExpressDate(rawDate, item.Time)

			status, stage := normalizeYunExpressStatus(item.Status, item.Description)
			if item.Stage != "" {
				stage = models.Stage(strings.ToLower(item.Stage))
			}

			loc := strings.TrimSpace(item.Location)

			evCarrierSlug := c.Slug()
			evCarrierName := c.Name()
			carrierIndex := 0

			if item.CarrierSlug != "" {
				evCarrierSlug = item.CarrierSlug
				if item.Carrier != "" {
					evCarrierName = item.Carrier
				}
				if !strings.EqualFold(evCarrierSlug, c.Slug()) {
					carrierIndex = 1
				}
			} else if lastMileCarrierInfo != nil && (status == models.StatusDelivered || status == models.StatusOutForDelivery || stage == models.StageLastMile || stage == models.StageDelivery) {
				evCarrierSlug = lastMileCarrierInfo.Slug
				evCarrierName = lastMileCarrierInfo.Name
				carrierIndex = 1
			}

			// If last mile number is mentioned in event description and wasn't extracted yet
			if lastMileNum == "" && strings.Contains(strings.ToLower(item.Description), "number") {
				match := regexp.MustCompile(`(?:number|no\.?|code)\s*[:\s]+([A-Za-z0-9]{8,30})`).FindStringSubmatch(item.Description)
				if len(match) > 1 {
					lastMileNum = match[1]
					if lastMileCarrierInfo == nil {
						lastMileCarrierInfo = resolveYunExpressLastMileCarrier("", lastMileNum, destCode)
					}
				}
			}

			events = append(events, models.TrackingEvent{
				Date:         evTime,
				RawDate:      rawDate,
				Status:       status,
				Stage:        stage,
				Description:  item.Description,
				Location:     loc,
				CarrierSlug:  evCarrierSlug,
				CarrierName:  evCarrierName,
				CarrierIndex: carrierIndex,
			})
		}

		sort.SliceStable(events, func(i, j int) bool {
			return events[i].Date.After(events[j].Date)
		})

		topStatus := yItem.Status
		topDesc := yItem.StatusDescription
		overallStatus, _ := normalizeYunExpressStatus(topStatus, topDesc)

		if len(events) > 0 {
			hasDelivered := false
			for _, ev := range events {
				if ev.Status == models.StatusDelivered {
					hasDelivered = true
					break
				}
			}
			if hasDelivered {
				overallStatus = models.StatusDelivered
			} else if topStatus == "" && topDesc == "" {
				overallStatus = events[0].Status
			}
		}

		statusText := topDesc
		if statusText == "" {
			statusText = topStatus
		}
		if statusText == "" {
			switch overallStatus {
			case models.StatusDelivered:
				statusText = "Delivered"
			case models.StatusOutForDelivery:
				statusText = "Out for delivery"
			case models.StatusFailedAttempt, models.StatusException:
				statusText = "Delivery exception"
			case models.StatusInfoReceived:
				statusText = "Shipment information received"
			default:
				statusText = "In transit"
			}
		}

		progress := 50
		switch overallStatus {
		case models.StatusDelivered:
			progress = 100
		case models.StatusOutForDelivery:
			progress = 85
		case models.StatusInfoReceived:
			progress = 15
		case models.StatusFailedAttempt, models.StatusException:
			progress = 75
		}

		var deliveredAt *time.Time
		for _, ev := range events {
			if ev.Status == models.StatusDelivered && !ev.Date.IsZero() {
				t := ev.Date
				deliveredAt = &t
				break
			}
		}

		var additionalCarriers []models.CarrierInfo
		if lastMileCarrierInfo != nil {
			additionalCarriers = append(additionalCarriers, *lastMileCarrierInfo)
		}

		daysInTransit := 1
		if len(events) >= 2 {
			newest := events[0].Date
			oldest := events[len(events)-1].Date
			if !newest.IsZero() && !oldest.IsZero() {
				diff := int(newest.Sub(oldest).Hours() / 24)
				if diff > 1 {
					daysInTransit = diff
				}
			}
		}

		originCountry := "China"
		originCode := "CN"
		if yItem.OriginCountry != "" {
			originCode = yItem.OriginCountry
			originCountry = yItem.OriginCountry
			if yItem.OriginCountryName != "" {
				originCountry = yItem.OriginCountryName
			}
		}

		return &models.Shipment{
			TrackingNumber:         cleanNum,
			Carrier:                c.Info(),
			LastMileTrackingNumber: lastMileNum,
			LastMileCarrier:        lastMileCarrierInfo,
			AdditionalCarriers:     additionalCarriers,
			Origin: models.CountryInfo{
				Name: originCountry,
				Code: originCode,
			},
			Destination: models.CountryInfo{
				Name: destCountry,
				Code: destCode,
			},
			Status:        overallStatus,
			StatusText:    statusText,
			Progress:      progress,
			DeliveredAt:   deliveredAt,
			DaysInTransit: daysInTransit,
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

func (c *YunExpressCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	destCountry := "Spain"
	destCode := "ES"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-7 * 24 * time.Hour),
			Description: "YunExpress shipment information received",
			Location:    "Shenzhen, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-6 * 24 * time.Hour),
			Description: "Package arrived at sorting facility",
			Location:    "Dongguan Hub, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-5 * 24 * time.Hour),
			Description: "Customs declaration completed at origin airport",
			Location:    "Hong Kong Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Departed on international flight",
			Location:    "Hong Kong Airport, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-40 * time.Hour),
			Description: "Arrived at destination country international airport",
			Location:    "Madrid-Barajas Airport, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Import customs clearance completed",
			Location:    "Madrid Customs Facility, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageCustoms,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "Transferred to destination carrier for local delivery",
			Location:    "Madrid Logistics Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageHandover,
		},
	}

	lastMileNum := "PK" + trackingNumber[max(0, len(trackingNumber)-10):] + "01"

	return &models.Shipment{
		TrackingNumber:         trackingNumber,
		Carrier:                c.Info(),
		LastMileTrackingNumber: lastMileNum,
		LastMileCarrier: &models.CarrierInfo{
			Slug:        "correos-spain",
			Name:        "Correos Espa?a",
			Country:     destCode,
			CountryName: destCountry,
			Type:        models.CarrierTypePostal,
			Website:     "https://www.correos.es",
			TrackingURL: "https://www.correos.es/es/es/herramientas/localizador/envios/detalle?tracking-number=%s",
		},
		Origin: models.CountryInfo{
			Name: "China",
			Code: "CN",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - Handed over to local carrier",
		Progress:      72,
		DaysInTransit: 7,
		Events:        events,
		UpdatedAt:     time.Now(),
	}
}
