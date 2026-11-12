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
	mrwBaseURL    = "https://www.mrw.es"
	mrwHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)
)

// SetMRWBaseURLForTest overrides the MRW base URL in deterministic tests.
func SetMRWBaseURLForTest(baseURL string) func() {
	orig := mrwBaseURL
	mrwBaseURL = strings.TrimRight(baseURL, "/")
	return func() { mrwBaseURL = orig }
}

// SetMRWHTTPClientForTest overrides the HTTP client for MRW in deterministic tests.
func SetMRWHTTPClientForTest(client *http.Client) func() {
	orig := mrwHTTPClient
	mrwHTTPClient = client
	return func() { mrwHTTPClient = orig }
}

type MRWCarrier struct {
	BaseCarrier
}

func NewMRWCarrier() *MRWCarrier {
	bc := NewBaseCarrier(
		"mrw-spain",
		"MRW",
		"ES",
		"Spain",
		models.CarrierTypeExpress,
		"https://www.mrw.es",
		"https://www.mrw.es/seguimiento_envios/mrw_resultados_consulta.asp?modo=nacional&envio=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &MRWCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that MRW implements real network tracking.
func (c *MRWCarrier) HasLiveTracking() bool {
	return true
}

type mrwEventItem struct {
	Fecha       string `json:"fecha"`
	Hora        string `json:"hora"`
	Date        string `json:"date"`
	Estado      string `json:"estado"`
	Status      string `json:"status"`
	Descripcion string `json:"descripcion"`
	Description string `json:"description"`
	Franquicia  string `json:"franquicia"`
	Ubicacion   string `json:"ubicacion"`
	Location    string `json:"location"`
}

type mrwAPIResponse struct {
	NumeroEnvio       string         `json:"numeroEnvio"`
	TrackingNumber    string         `json:"trackingNumber"`
	Estado            string         `json:"estado"`
	Status            string         `json:"status"`
	DescripcionEstado string         `json:"descripcionEstado"`
	StatusDescription string         `json:"statusDescription"`
	Origen            string         `json:"origen"`
	FranquiciaOrigen  string         `json:"franquiciaOrigen"`
	Destino           string         `json:"destino"`
	FranquiciaDestino string         `json:"franquiciaDestino"`
	FechaEntrega      string         `json:"fechaEntrega"`
	DeliveryDate      string         `json:"deliveryDate"`
	Eventos           []mrwEventItem `json:"eventos"`
	Events            []mrwEventItem `json:"events"`
	Error             string         `json:"error"`
	Codigo            string         `json:"codigo"`
}

var (
	mrwHTMLTagRegex     = regexp.MustCompile(`<[^>]+>`)
	mrwTRRegex          = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	mrwTDRegex          = regexp.MustCompile(`(?is)<t[dh][^>]*>(.*?)</t[dh]>`)
	mrwDatePatternRegex = regexp.MustCompile(`\b\d{2}/\d{2}/\d{4}(?:\s+\d{2}:\d{2}(?::\d{2})?)?\b`)
	mrwOriginRegex      = regexp.MustCompile(`(?is)(?:Franquicia\s*(?:de\s*)?[Oo]rigen|Origen)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
	mrwDestRegex        = regexp.MustCompile(`(?is)(?:Franquicia\s*(?:de\s*)?[Dd]estino|Destino)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
	mrwStatusRegex      = regexp.MustCompile(`(?is)(?:Estado\s*(?:actual)?|Situaci[o?]n)\s*[:<][^>]*>\s*(?:<[^>]+>\s*)*([^<\r\n]+)`)
)

func stripHTMLClean(s string) string {
	clean := mrwHTMLTagRegex.ReplaceAllString(s, " ")
	clean = html.UnescapeString(clean)
	return strings.Join(strings.Fields(clean), " ")
}

func parseMRWDate(dateStr, timeStr string) time.Time {
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

func normalizeMRWStatus(status, desc string) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(status + " " + desc))

	switch {
	case strings.Contains(upper, "ENTREGADO") || strings.Contains(upper, "DELIVERED") || strings.Contains(upper, "ENTREGA REALIZADA"):
		return models.StatusDelivered, models.StageDelivery
	case strings.Contains(upper, "EN REPARTO") || strings.Contains(upper, "OUT FOR DELIVERY") || strings.Contains(upper, "OUT_FOR_DELIVERY") || strings.Contains(upper, "FURGONETA DE REPARTO"):
		return models.StatusOutForDelivery, models.StageLastMile
	case strings.Contains(upper, "INCIDENCIA") || strings.Contains(upper, "AUSENTE") || strings.Contains(upper, "FALLID") || strings.Contains(upper, "DIRECCION INCORRECTA") || strings.Contains(upper, "FAILED"):
		return models.StatusFailedAttempt, models.StageException
	case strings.Contains(upper, "RETENIDO") || strings.Contains(upper, "DEVUELTO") || strings.Contains(upper, "EXCEPTION") || strings.Contains(upper, "CANCELADO"):
		return models.StatusException, models.StageException
	case strings.Contains(upper, "RECOGIDO") || strings.Contains(upper, "RECOGIDA") || strings.Contains(upper, "ADMITIDO") || strings.Contains(upper, "PICKED"):
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(upper, "DOCUMENTADO") || strings.Contains(upper, "GRABADO") || strings.Contains(upper, "PRE-REGISTRADO") || strings.Contains(upper, "INFORMADO") || strings.Contains(upper, "INFO_RECEIVED"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(upper, "TRANSITO") || strings.Contains(upper, "TR?NSITO") || strings.Contains(upper, "PLATAFORMA") || strings.Contains(upper, "HUB") || strings.Contains(upper, "FRANQUICIA") || strings.Contains(upper, "IN_TRANSIT"):
		return models.StatusInTransit, models.StageTransit
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func isMRWBlocked(statusCode int, body string) bool {
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

func isMRWNotFound(body string) bool {
	upper := strings.ToUpper(body)
	return strings.Contains(upper, "NO SE HA ENCONTRADO") ||
		strings.Contains(upper, "NO EXISTE NING?N ENV?O") ||
		strings.Contains(upper, "NO EXISTE NINGUN ENVIO") ||
		strings.Contains(upper, "ENV?O NO ENCONTRADO") ||
		strings.Contains(upper, "ENVIO NO ENCONTRADO") ||
		strings.Contains(upper, "SIN DATOS PARA EL ENV?O") ||
		strings.Contains(upper, "SIN DATOS PARA EL ENVIO") ||
		strings.Contains(upper, "ENV?O INEXISTENTE") ||
		strings.Contains(upper, "ENVIO INEXISTENTE") ||
		strings.Contains(upper, "ID=\"ERROR404\"") ||
		strings.Contains(upper, "NO EST? DISPONIBLE") ||
		strings.Contains(upper, "NOT_FOUND")
}

func (c *MRWCarrier) parseMRWJSON(bodyBytes []byte) (*mrwAPIResponse, error) {
	var resp mrwAPIResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, err
	}
	if strings.EqualFold(resp.Codigo, "NOT_FOUND") || strings.EqualFold(resp.Error, "NOT_FOUND") ||
		strings.Contains(strings.ToUpper(resp.Error), "NO ENCONTRADO") {
		return nil, models.ErrShipmentNotFound
	}
	if resp.NumeroEnvio == "" && resp.TrackingNumber == "" && resp.Estado == "" && resp.Status == "" &&
		len(resp.Eventos) == 0 && len(resp.Events) == 0 {
		return nil, models.ErrScraperSchemaChanged
	}
	return &resp, nil
}

func (c *MRWCarrier) parseMRWHTML(htmlContent string) (*mrwAPIResponse, error) {
	res := &mrwAPIResponse{}

	if m := mrwOriginRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Origen = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	if m := mrwDestRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Destino = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	if m := mrwStatusRegex.FindStringSubmatch(htmlContent); len(m) > 1 {
		res.Estado = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	rows := mrwTRRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, row := range rows {
		rowLower := strings.ToLower(row[1])
		if strings.Contains(rowLower, "cookie") || strings.Contains(rowLower, "consent") ||
			strings.Contains(rowLower, "analytics") {
			continue
		}
		cells := mrwTDRegex.FindAllStringSubmatch(row[1], -1)
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

		// Find cell matching date pattern
		dateIdx := -1
		for i, col := range cleaned {
			if mrwDatePatternRegex.MatchString(col) {
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

		res.Eventos = append(res.Eventos, mrwEventItem{
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

func (c *MRWCarrier) fetchLive(ctx context.Context, trackingNumber string) (*mrwAPIResponse, error) {
	targetURL := fmt.Sprintf("%s/seguimiento_envios/mrw_resultados_consulta.asp?modo=nacional&envio=%s",
		mrwBaseURL, url.QueryEscape(trackingNumber))

	req, err := NewCarrierRequest(http.MethodGet, targetURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req.Header.Set("Accept", "application/json, text/html, application/xhtml+xml, */*")
	req = req.WithContext(ctx)

	resp, err := mrwHTTPClient.Do(req)
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
	if isMRWBlocked(resp.StatusCode, bodyStr) {
		return nil, models.ErrCarrierBlocked
	}
	if isMRWNotFound(bodyStr) {
		return nil, models.ErrShipmentNotFound
	}

	trimmed := strings.TrimSpace(bodyStr)
	if strings.HasPrefix(trimmed, "{") || strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		return c.parseMRWJSON(bodyBytes)
	}

	return c.parseMRWHTML(bodyStr)
}

func (c *MRWCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	mResp, err := c.fetchLive(ctx, cleanNum)
	if err == nil && mResp != nil {
		rawEvents := mResp.Eventos
		if len(rawEvents) == 0 {
			rawEvents = mResp.Events
		}

		var events []models.TrackingEvent
		for _, item := range rawEvents {
			rawDate := item.Fecha
			if rawDate == "" {
				rawDate = item.Date
			}
			evTime := parseMRWDate(rawDate, item.Hora)
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

			status, stage := normalizeMRWStatus(evStatus, evDesc)

			loc := strings.TrimSpace(item.Ubicacion)
			if loc == "" {
				loc = strings.TrimSpace(item.Location)
			}
			if loc == "" {
				loc = strings.TrimSpace(item.Franquicia)
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

		// Sort newest first
		sort.SliceStable(events, func(i, j int) bool {
			return events[i].Date.After(events[j].Date)
		})

		topStatus := mResp.Estado
		if topStatus == "" {
			topStatus = mResp.Status
		}
		topDesc := mResp.DescripcionEstado
		if topDesc == "" {
			topDesc = mResp.StatusDescription
		}
		overallStatus, _ := normalizeMRWStatus(topStatus, topDesc)

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
		deliveryStr := mResp.FechaEntrega
		if deliveryStr == "" {
			deliveryStr = mResp.DeliveryDate
		}
		if deliveryStr != "" {
			t := parseMRWDate(deliveryStr, "")
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

		origCity := strings.TrimSpace(mResp.Origen)
		if origCity == "" {
			origCity = strings.TrimSpace(mResp.FranquiciaOrigen)
		}
		if origCity == "" {
			origCity = "Spain"
		}

		destCity := strings.TrimSpace(mResp.Destino)
		if destCity == "" {
			destCity = strings.TrimSpace(mResp.FranquiciaDestino)
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

func (c *MRWCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Env?o recogido en origen por franquicia MRW",
			Location:    "Franquicia Origen, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tr?nsito interurbano - Plataforma log?stica central",
			Location:    "Coslada Hub, Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "En reparto - Env?o en furgoneta de reparto para entrega hoy",
			Location:    "Franquicia Destino, Spain",
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
