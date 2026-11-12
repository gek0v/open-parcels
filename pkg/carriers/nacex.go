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
	nacexBaseURL    = "https://www.nacex.com"
	nacexHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)
)

// SetNacexBaseURLForTest overrides the Nacex base URL in deterministic tests.
func SetNacexBaseURLForTest(baseURL string) func() {
	orig := nacexBaseURL
	nacexBaseURL = strings.TrimRight(baseURL, "/")
	return func() { nacexBaseURL = orig }
}

// SetNacexHTTPClientForTest overrides the HTTP client for Nacex in deterministic tests.
func SetNacexHTTPClientForTest(client *http.Client) func() {
	orig := nacexHTTPClient
	nacexHTTPClient = client
	return func() { nacexHTTPClient = orig }
}

type NacexCarrier struct {
	BaseCarrier
}

func NewNacexCarrier() *NacexCarrier {
	bc := NewBaseCarrier(
		"nacex-spain",
		"Nacex",
		"ES",
		"Spain",
		models.CarrierTypeExpress,
		"https://www.nacex.com",
		"https://www.nacex.com/seguimientoDetalle.do?ag_rec=&albaran_rec=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &NacexCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that Nacex implements real network tracking.
func (c *NacexCarrier) HasLiveTracking() bool {
	return true
}

type nacexEventItem struct {
	Fecha       string `json:"fecha"`
	Hora        string `json:"hora"`
	Date        string `json:"date"`
	Estado      string `json:"estado"`
	Status      string `json:"status"`
	Descripcion string `json:"descripcion"`
	Description string `json:"description"`
	Agencia     string `json:"agencia"`
	Location    string `json:"location"`
	Ubicacion   string `json:"ubicacion"`
}

type nacexAPIResponse struct {
	Albaran           string           `json:"albaran"`
	TrackingNumber    string           `json:"trackingNumber"`
	AgenciaOrigen     string           `json:"agenciaOrigen"`
	AgenciaDestino    string           `json:"agenciaDestino"`
	Origen            string           `json:"origen"`
	Destino           string           `json:"destino"`
	Estado            string           `json:"estado"`
	Status            string           `json:"status"`
	DescripcionEstado string           `json:"descripcionEstado"`
	StatusDescription string           `json:"statusDescription"`
	FechaEntrega      string           `json:"fechaEntrega"`
	DeliveryDate      string           `json:"deliveryDate"`
	Eventos           []nacexEventItem `json:"eventos"`
	Events            []nacexEventItem `json:"events"`
	Error             string           `json:"error"`
	Codigo            string           `json:"codigo"`
}

var (
	nacexHTMLTagRegex     = regexp.MustCompile(`<[^>]+>`)
	nacexTRRegex          = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	nacexTDRegex          = regexp.MustCompile(`(?is)<t[dh][^>]*>(.*?)</t[dh]>`)
	nacexDatePatternRegex = regexp.MustCompile(`\b\d{2}/\d{2}/\d{4}(?:\s+\d{2}:\d{2}(?::\d{2})?)?\b`)
	nacexOriginRegex      = regexp.MustCompile(`(?is)(?:Agencia\s*(?:de\s*)?[Oo]rigen|Origen)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
	nacexDestRegex        = regexp.MustCompile(`(?is)(?:Agencia\s*(?:de\s*)?[Dd]estino|Destino)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
	nacexStatusRegex      = regexp.MustCompile(`(?is)(?:Estado\s*(?:actual)?|Situaci[o?]n)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
)

func parseNacexDate(dateStr, timeStr string) time.Time {
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
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"2006-01-02 15:04:05",
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

func normalizeNacexStatus(status, desc string) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(status + " " + desc))

	switch {
	case strings.Contains(upper, "ENTREGADO") || strings.Contains(upper, "DELIVERED") || strings.Contains(upper, "ENTREGA OK") || strings.Contains(upper, "ENTREGADO EN DESTINO"):
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(upper, "EN REPARTO") || strings.Contains(upper, "OUT FOR DELIVERY") || strings.Contains(upper, "OUT_FOR_DELIVERY") || strings.Contains(upper, "EN RUTA") || strings.Contains(upper, "REPARTO"):
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(upper, "INCIDENCIA") || strings.Contains(upper, "AUSENTE") || strings.Contains(upper, "FALLID") || strings.Contains(upper, "DIRECCION INCORRECTA") || strings.Contains(upper, "FAILED"):
		return models.StatusFailedAttempt, models.StageException
	case strings.Contains(upper, "RETENIDO") || strings.Contains(upper, "DEVUELTO") || strings.Contains(upper, "EXCEPTION") || strings.Contains(upper, "CANCELADO"):
		return models.StatusException, models.StageException
	case strings.Contains(upper, "RECOGIDO") || strings.Contains(upper, "RECOGIDA") || strings.Contains(upper, "ADMITIDO") || strings.Contains(upper, "PICKED"):
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(upper, "DOCUMENTADO") || strings.Contains(upper, "GRABADO") || strings.Contains(upper, "PRE-REGISTRADO") || strings.Contains(upper, "INFORMADO") || strings.Contains(upper, "INFO_RECEIVED"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(upper, "TRANSITO") || strings.Contains(upper, "TR?NSITO") || strings.Contains(upper, "PLATAFORMA") || strings.Contains(upper, "HUB") || strings.Contains(upper, "AGENCIA") || strings.Contains(upper, "IN_TRANSIT"):
		return models.StatusInTransit, models.StageTransit
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func isNacexBlocked(statusCode int, body string) bool {
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
		strings.Contains(upper, "ACCESO NO AUTORIZADO") ||
		strings.Contains(upper, "WAF")
}

func isNacexNotFound(body string) bool {
	upper := strings.ToUpper(body)
	return strings.Contains(upper, "NO SE HA ENCONTRADO") ||
		strings.Contains(upper, "NO EXISTE EL ALBAR?N") ||
		strings.Contains(upper, "NO EXISTE EL ALBARAN") ||
		strings.Contains(upper, "ALBAR?N NO ENCONTRADO") ||
		strings.Contains(upper, "ALBARAN NO ENCONTRADO") ||
		strings.Contains(upper, "SIN DATOS PARA EL ALBAR?N") ||
		strings.Contains(upper, "SIN DATOS PARA EL ALBARAN") ||
		strings.Contains(upper, "ALBAR?N INEXISTENTE") ||
		strings.Contains(upper, "ALBARAN INEXISTENTE") ||
		strings.Contains(upper, "FORMULARIO DE SEGUIMIENTO") ||
		strings.Contains(upper, "NOT_FOUND")
}

func (c *NacexCarrier) parseNacexJSON(bodyBytes []byte) (*nacexAPIResponse, error) {
	var resp nacexAPIResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Codigo, "NOT_FOUND") || strings.EqualFold(resp.Error, "NOT_FOUND") ||
		strings.Contains(strings.ToUpper(resp.Error), "NO ENCONTRADO") {
		return nil, models.ErrShipmentNotFound
	}
	if resp.Albaran == "" && resp.TrackingNumber == "" && resp.Estado == "" && resp.Status == "" &&
		len(resp.Eventos) == 0 && len(resp.Events) == 0 {
		return nil, models.ErrScraperSchemaChanged
	}
	return &resp, nil
}

func (c *NacexCarrier) parseNacexHTML(htmlContent string) (*nacexAPIResponse, error) {
	res := &nacexAPIResponse{}

	if m := nacexOriginRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Origen = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	if m := nacexDestRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Destino = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	if m := nacexStatusRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Estado = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	rows := nacexTRRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, row := range rows {
		rowLower := strings.ToLower(row[1])
		if strings.Contains(rowLower, "cookie") || strings.Contains(rowLower, "consent") ||
			strings.Contains(rowLower, "zendesk") || strings.Contains(rowLower, "analytics") ||
			strings.Contains(rowLower, "duraci") {
			continue
		}
		cells := nacexTDRegex.FindAllStringSubmatch(row[1], -1)
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
			if nacexDatePatternRegex.MatchString(col) {
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

		res.Eventos = append(res.Eventos, nacexEventItem{
			Fecha:       dateVal,
			Estado:      statusVal,
			Agencia:     locVal,
			Descripcion: descVal,
		})
	}

	if len(res.Eventos) == 0 && res.Estado == "" {
		return nil, models.ErrScraperSchemaChanged
	}
	return res, nil
}

func (c *NacexCarrier) fetchLive(ctx context.Context, trackingNumber string) (*nacexAPIResponse, error) {
	targetURL := fmt.Sprintf("%s/seguimientoDetalle.do?ag_rec=&albaran_rec=%s",
		nacexBaseURL, url.QueryEscape(trackingNumber))

	req, err := NewCarrierRequest(http.MethodGet, targetURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req.Header.Set("Accept", "application/json, text/html, application/xhtml+xml, */*")
	req = req.WithContext(ctx)

	resp, err := nacexHTTPClient.Do(req)
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
	if isNacexBlocked(resp.StatusCode, bodyStr) {
		return nil, models.ErrCarrierBlocked
	}
	if isNacexNotFound(bodyStr) {
		return nil, models.ErrShipmentNotFound
	}

	trimmed := strings.TrimSpace(bodyStr)
	if strings.HasPrefix(trimmed, "{") || strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		return c.parseNacexJSON(bodyBytes)
	}

	return c.parseNacexHTML(bodyStr)
}

func (c *NacexCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	nResp, err := c.fetchLive(ctx, cleanNum)
	if err == nil && nResp != nil {
		rawEvents := nResp.Eventos
		if len(rawEvents) == 0 {
			rawEvents = nResp.Events
		}

		var events []models.TrackingEvent
		for _, item := range rawEvents {
			rawDate := item.Fecha
			if rawDate == "" {
				rawDate = item.Date
			}
			evTime := parseNacexDate(rawDate, item.Hora)
			if item.Hora != "" && !strings.Contains(rawDate, ":") {
				rawDate = rawDate + " " + item.Hora
			}

			evStatus := item.Estado
			if evStatus == "" {
				evStatus = item.Status
			}
			evDesc := item.Descripcion
			if evDesc == "" {
				evDesc = item.Description
			}
			if evDesc == "" {
				evDesc = evStatus
			}

			status, stage := normalizeNacexStatus(evStatus, evDesc)

			loc := strings.TrimSpace(item.Agencia)
			if loc == "" {
				loc = strings.TrimSpace(item.Location)
			}
			if loc == "" {
				loc = strings.TrimSpace(item.Ubicacion)
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

		topStatus := nResp.Estado
		if topStatus == "" {
			topStatus = nResp.Status
		}
		topDesc := nResp.DescripcionEstado
		if topDesc == "" {
			topDesc = nResp.StatusDescription
		}
		overallStatus, _ := normalizeNacexStatus(topStatus, topDesc)

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
				statusText = "Documentado"
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
		deliveryStr := nResp.FechaEntrega
		if deliveryStr == "" {
			deliveryStr = nResp.DeliveryDate
		}
		if deliveryStr != "" {
			t := parseNacexDate(deliveryStr, "")
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

		origCity := strings.TrimSpace(nResp.Origen)
		if origCity == "" {
			origCity = strings.TrimSpace(nResp.AgenciaOrigen)
		}
		if origCity == "" {
			origCity = "Spain"
		}

		destCity := strings.TrimSpace(nResp.Destino)
		if destCity == "" {
			destCity = strings.TrimSpace(nResp.AgenciaDestino)
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

func (c *NacexCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-22 * time.Hour),
			Description: "Env?o recogido en origen / Parcel picked up at origin",
			Location:    "Agencia Origen Nacex, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tr?nsito - Clasificado en plataforma central / Processed at central platform",
			Location:    "Coslada Hub Nacex, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Paquete en ruta de entrega / Out for delivery",
			Location:    "Agencia Destino Nacex, Spain",
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
