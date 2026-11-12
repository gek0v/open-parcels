package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
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
	cttExpressBaseURL    = "https://www.cttexpress.com"
	cttExpressHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)
)

// SetCTTExpressBaseURLForTest overrides the CTT Express base URL in deterministic tests.
func SetCTTExpressBaseURLForTest(baseURL string) func() {
	orig := cttExpressBaseURL
	cttExpressBaseURL = strings.TrimRight(baseURL, "/")
	return func() { cttExpressBaseURL = orig }
}

// SetCTTExpressHTTPClientForTest overrides the HTTP client for CTT Express in deterministic tests.
func SetCTTExpressHTTPClientForTest(client *http.Client) func() {
	orig := cttExpressHTTPClient
	cttExpressHTTPClient = client
	return func() { cttExpressHTTPClient = orig }
}

type CTTExpressCarrier struct {
	BaseCarrier
}

func NewCTTExpressCarrier() *CTTExpressCarrier {
	bc := NewBaseCarrier(
		"ctt-express",
		"CTT Express",
		"ES",
		"Spain",
		models.CarrierTypeExpress,
		"https://www.cttexpress.com",
		"https://www.cttexpress.com/localizador-de-envios/?sc=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &CTTExpressCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that CTT Express implements real network tracking.
func (c *CTTExpressCarrier) HasLiveTracking() bool {
	return true
}

type cttEventItem struct {
	Fecha       string `json:"fecha"`
	Hora        string `json:"hora"`
	Date        string `json:"date"`
	Estado      string `json:"estado"`
	Status      string `json:"status"`
	Descripcion string `json:"descripcion"`
	Description string `json:"description"`
	Ubicacion   string `json:"ubicacion"`
	Location    string `json:"location"`
	Centro      string `json:"centro"`
}

type cttAPIResponse struct {
	Codigo            string         `json:"codigo"`
	ShipmentCode      string         `json:"shipmentCode"`
	TrackingNumber    string         `json:"trackingNumber"`
	Estado            string         `json:"estado"`
	Status            string         `json:"status"`
	DescripcionEstado string         `json:"descripcionEstado"`
	StatusDescription string         `json:"statusDescription"`
	Origen            any            `json:"origen"`
	Destino           any            `json:"destino"`
	Origin            any            `json:"origin"`
	Destination       any            `json:"destination"`
	FechaEntrega      string         `json:"fechaEntrega"`
	DeliveryDate      string         `json:"deliveryDate"`
	Eventos           []cttEventItem `json:"eventos"`
	Events            []cttEventItem `json:"events"`
	Error             string         `json:"error"`
	Message           string         `json:"message"`
}

var (
	cttHTMLTagRegex     = regexp.MustCompile(`<[^>]+>`)
	cttTRRegex          = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	cttTDRegex          = regexp.MustCompile(`(?is)<t[dh][^>]*>(.*?)</t[dh]>`)
	cttDatePatternRegex = regexp.MustCompile(`\b\d{2}/\d{2}/\d{4}(?:\s+\d{2}:\d{2}(?::\d{2})?)?\b|\b\d{4}-\d{2}-\d{2}(?:[T\s]\d{2}:\d{2}(?::\d{2})?)?\b`)
	cttOriginRegex      = regexp.MustCompile(`(?is)(?:Origen|Origin)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
	cttDestRegex        = regexp.MustCompile(`(?is)(?:Destino|Destination)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
	cttStatusRegex      = regexp.MustCompile(`(?is)(?:Estado\s*(?:actual)?|Situaci[o?]n|Status)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
)

func parseCTTDate(dateStr, timeStr string) time.Time {
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
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"2006-01-02 15:04",
		"02/01/2006",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, combined, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

func normalizeCTTStatus(status, desc string) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(status + " " + desc))

	switch {
	case strings.Contains(upper, "DELIVERED") || strings.Contains(upper, "ENTREGADO") || strings.Contains(upper, "ENTREGA REALIZADA"):
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(upper, "OUT_FOR_DELIVERY") || strings.Contains(upper, "OUT FOR DELIVERY") || strings.Contains(upper, "EN REPARTO") || strings.Contains(upper, "EN RUTA"):
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(upper, "INCIDENCE") || strings.Contains(upper, "INCIDENCIA") || strings.Contains(upper, "AUSENTE") || strings.Contains(upper, "FALLID") || strings.Contains(upper, "FAILED") || strings.Contains(upper, "DIRECCION"):
		return models.StatusFailedAttempt, models.StageException
	case strings.Contains(upper, "RETENIDO") || strings.Contains(upper, "DEVUELTO") || strings.Contains(upper, "EXCEPTION"):
		return models.StatusException, models.StageException
	case strings.Contains(upper, "PICKED") || strings.Contains(upper, "RECOGIDO") || strings.Contains(upper, "ADMITIDO"):
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(upper, "PRE_REGISTERED") || strings.Contains(upper, "PREREGISTRADO") || strings.Contains(upper, "PRE-REGISTRADO") || strings.Contains(upper, "INFORMADO") || strings.Contains(upper, "INFO_RECEIVED"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(upper, "IN_TRANSIT") || strings.Contains(upper, "IN TRANSIT") || strings.Contains(upper, "TRANSITO") || strings.Contains(upper, "TR?NSITO") || strings.Contains(upper, "PLATAFORMA") || strings.Contains(upper, "CENTRO") || strings.Contains(upper, "DISTRIBUCION"):
		return models.StatusInTransit, models.StageTransit
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func isCTTBlocked(statusCode int, body string) bool {
	if statusCode == http.StatusForbidden {
		return true
	}
	upper := strings.ToUpper(body)
	return strings.Contains(upper, "INCAPSULA") ||
		strings.Contains(upper, "CLOUDFLARE") ||
		strings.Contains(upper, "CAPTCHA") ||
		strings.Contains(upper, "SEC_CHALLENGE") ||
		strings.Contains(upper, "ACCESS DENIED") ||
		strings.Contains(upper, "ACCESO DENEGADO") ||
		strings.Contains(upper, "WAF")
}

func isCTTNotFound(body string) bool {
	upper := strings.ToUpper(body)
	return strings.Contains(upper, "NO SE HA ENCONTRADO") ||
		strings.Contains(upper, "ENV?O NO ENCONTRADO") ||
		strings.Contains(upper, "ENVIO NO ENCONTRADO") ||
		strings.Contains(upper, "C?DIGO NO ENCONTRADO") ||
		strings.Contains(upper, "CODIGO NO ENCONTRADO") ||
		strings.Contains(upper, "SIN DATOS") ||
		strings.Contains(upper, "SIN RESULTADOS") ||
		strings.Contains(upper, "NOT_FOUND") ||
		strings.Contains(upper, "SHIPMENT NOT FOUND")
}

func extractCityFromField(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if name, ok := v["name"].(string); ok && name != "" {
			return strings.TrimSpace(name)
		}
		if city, ok := v["city"].(string); ok && city != "" {
			return strings.TrimSpace(city)
		}
		if location, ok := v["location"].(string); ok && location != "" {
			return strings.TrimSpace(location)
		}
	}
	return ""
}

func (c *CTTExpressCarrier) parseCTTJSON(bodyBytes []byte) (*cttAPIResponse, error) {
	var resp cttAPIResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Error, "NOT_FOUND") || strings.Contains(strings.ToUpper(resp.Message), "NOT FOUND") ||
		strings.Contains(strings.ToUpper(resp.Message), "NO ENCONTRADO") {
		return nil, models.ErrShipmentNotFound
	}
	if resp.Codigo == "" && resp.ShipmentCode == "" && resp.TrackingNumber == "" && resp.Estado == "" && resp.Status == "" &&
		len(resp.Eventos) == 0 && len(resp.Events) == 0 {
		return nil, models.ErrScraperSchemaChanged
	}
	return &resp, nil
}

func (c *CTTExpressCarrier) parseCTTHTML(htmlContent string) (*cttAPIResponse, error) {
	res := &cttAPIResponse{}

	if m := cttOriginRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Origen = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	if m := cttDestRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Destino = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	if m := cttStatusRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Estado = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	rows := cttTRRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, row := range rows {
		rowLower := strings.ToLower(row[1])
		if strings.Contains(rowLower, "cookie") || strings.Contains(rowLower, "consent") ||
			strings.Contains(rowLower, "analytics") {
			continue
		}
		cells := cttTDRegex.FindAllStringSubmatch(row[1], -1)
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
			if cttDatePatternRegex.MatchString(col) {
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

		res.Eventos = append(res.Eventos, cttEventItem{
			Fecha:       dateVal,
			Estado:      statusVal,
			Ubicacion:   locVal,
			Descripcion: descVal,
		})
	}

	if len(res.Eventos) == 0 && res.Estado == "" {
		return nil, models.ErrScraperSchemaChanged
	}
	return res, nil
}

func (c *CTTExpressCarrier) fetchLive(ctx context.Context, trackingNumber string) (*cttAPIResponse, error) {
	targetURL := fmt.Sprintf("%s/api/v1/tracking/%s", cttExpressBaseURL, url.PathEscape(trackingNumber))

	req, err := NewCarrierRequest(http.MethodGet, targetURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req.Header.Set("Accept", "application/json, text/html, application/xhtml+xml, */*")
	req = req.WithContext(ctx)

	resp, err := cttExpressHTTPClient.Do(req)
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
	if isCTTBlocked(resp.StatusCode, bodyStr) {
		return nil, models.ErrCarrierBlocked
	}
	if isCTTNotFound(bodyStr) {
		return nil, models.ErrShipmentNotFound
	}

	trimmed := strings.TrimSpace(bodyStr)
	if strings.HasPrefix(trimmed, "{") || strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		return c.parseCTTJSON(bodyBytes)
	}

	return c.parseCTTHTML(bodyStr)
}

func (c *CTTExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	cResp, err := c.fetchLive(ctx, cleanNum)
	if err == nil && cResp != nil {
		rawEvents := cResp.Eventos
		if len(rawEvents) == 0 {
			rawEvents = cResp.Events
		}

		var events []models.TrackingEvent
		for _, item := range rawEvents {
			rawDate := item.Date
			if rawDate == "" {
				rawDate = item.Fecha
			}
			evTime := parseCTTDate(rawDate, item.Hora)
			if item.Hora != "" && !strings.Contains(rawDate, ":") {
				rawDate = rawDate + " " + item.Hora
			}

			evStatus := item.Status
			if evStatus == "" {
				evStatus = item.Estado
			}
			evDesc := item.Description
			if evDesc == "" {
				evDesc = item.Descripcion
			}
			if evDesc == "" {
				evDesc = evStatus
			}

			status, stage := normalizeCTTStatus(evStatus, evDesc)

			loc := strings.TrimSpace(item.Location)
			if loc == "" {
				loc = strings.TrimSpace(item.Ubicacion)
			}
			if loc == "" {
				loc = strings.TrimSpace(item.Centro)
			}
			if loc != "" && !strings.Contains(strings.ToLower(loc), "spain") && !strings.Contains(strings.ToLower(loc), "espa?a") {
				loc += ", Spain"
			}

			events = append(events, models.TrackingEvent{
				Date:         evTime,
				RawDate:      rawDate,
				Status:       status,
				Stage:        stage,
				Description:  evDesc,
				Location:     loc,
				CarrierSlug:  c.Slug(),
				CarrierName:  c.Name(),
				CarrierIndex: 0,
			})
		}

		sort.SliceStable(events, func(i, j int) bool {
			return events[i].Date.After(events[j].Date)
		})

		topStatus := cResp.Status
		if topStatus == "" {
			topStatus = cResp.Estado
		}
		topDesc := cResp.StatusDescription
		if topDesc == "" {
			topDesc = cResp.DescripcionEstado
		}
		overallStatus, _ := normalizeCTTStatus(topStatus, topDesc)

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
				statusText = "Entregado"
			case models.StatusOutForDelivery:
				statusText = "En reparto"
			case models.StatusFailedAttempt, models.StatusException:
				statusText = "Incidencia"
			case models.StatusInfoReceived:
				statusText = "Pre-registrado"
			default:
				statusText = "En tr?nsito"
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
		deliveryStr := cResp.DeliveryDate
		if deliveryStr == "" {
			deliveryStr = cResp.FechaEntrega
		}
		if deliveryStr != "" {
			t := parseCTTDate(deliveryStr, "")
			if !t.IsZero() {
				deliveredAt = &t
			}
		}
		if deliveredAt == nil {
			for _, ev := range events {
				if ev.Status == models.StatusDelivered && !ev.Date.IsZero() {
					t := ev.Date
					deliveredAt = &t
					break
				}
			}
		}

		origCity := extractCityFromField(cResp.Origin)
		if origCity == "" {
			origCity = extractCityFromField(cResp.Origen)
		}
		if origCity == "" {
			origCity = "Spain"
		}

		destCity := extractCityFromField(cResp.Destination)
		if destCity == "" {
			destCity = extractCityFromField(cResp.Destino)
		}
		if destCity == "" {
			destCity = "Spain"
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

func (c *CTTExpressCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Env?o pre-registrado telem?ticamente",
			Location:    "Madrid Hub, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-14 * time.Hour),
			Description: "En tr?nsito - Recepcionado en plataforma de distribuci?n",
			Location:    "Plataforma Central, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto - El env?o se encuentra en ruta de entrega",
			Location:    "Centro de Distribuci?n Local, Spain",
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
