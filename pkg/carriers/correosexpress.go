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

var correosExpressHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetCorreosExpressHTTPClientForTest overrides the HTTP client for Correos Express in deterministic tests.
func SetCorreosExpressHTTPClientForTest(client *http.Client) func() {
	orig := correosExpressHTTPClient
	correosExpressHTTPClient = client
	return func() { correosExpressHTTPClient = orig }
}

type CorreosExpressCarrier struct {
	BaseCarrier
}

func NewCorreosExpressCarrier() *CorreosExpressCarrier {
	bc := NewBaseCarrier(
		"correosexpress",
		"Correos Express",
		"ES",
		"Spain",
		models.CarrierTypeExpress,
		"https://www.correosexpress.com",
		"https://www.correosexpress.com/web/correosexpress/consultanos?shippingNumber=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI:     true,
		SupportsWebScraping: true,
	})
	return &CorreosExpressCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that Correos Express implements live network tracking.
func (c *CorreosExpressCarrier) HasLiveTracking() bool {
	return true
}

type correosExpressPosicion struct {
	Fecha   string `json:"fecha"`
	Hora    string `json:"hora"`
	Desc    string `json:"desc"`
	Descrip string `json:"descrip"`
	CodOfi  string `json:"codOfi"`
	NomOfi  string `json:"nomOfi"`
	Tipo    string `json:"tipo"`
}

type correosExpressResponse struct {
	CodigoError  string                   `json:"codigoError"`
	MensajeError string                   `json:"mensajeError"`
	CodEnvio     string                   `json:"codEnvio"`
	Referencia   string                   `json:"referencia"`
	Estado       string                   `json:"estado"`
	DescEstado   string                   `json:"descEstado"`
	FechaEntrega string                   `json:"fechaEntrega"`
	Origen       string                   `json:"origen"`
	Destino      string                   `json:"destino"`
	Posicion     []correosExpressPosicion `json:"posicion"`
}

func parseCorreosExpressDate(fechaStr, horaStr string) time.Time {
	cleanDate := strings.TrimSpace(fechaStr)
	cleanHora := strings.TrimSpace(horaStr)

	layouts := []string{
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"2006-01-02 15:04:05",
		"02/01/2006",
	}

	combined := cleanDate
	if cleanHora != "" {
		combined = cleanDate + " " + cleanHora
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, combined, time.UTC); err == nil {
			return t
		}
	}

	return time.Time{}
}

func normalizeCorreosExpressStatus(desc, tipo string) (models.Status, models.Stage) {
	upperDesc := strings.ToUpper(strings.TrimSpace(desc))
	upperTipo := strings.ToUpper(strings.TrimSpace(tipo))

	switch {
	case upperTipo == "D" || strings.Contains(upperDesc, "ENTREGADO") || strings.Contains(upperDesc, "DELIVERED"):
		return models.StatusDelivered, models.StageDelivery
	case upperTipo == "R" || strings.Contains(upperDesc, "REPARTO") || strings.Contains(upperDesc, "OUT FOR DELIVERY"):
		return models.StatusOutForDelivery, models.StageLastMile
	case upperTipo == "T" || strings.Contains(upperDesc, "TRANSITO") || strings.Contains(upperDesc, "NAVE") || strings.Contains(upperDesc, "CLASIFICADO"):
		return models.StatusInTransit, models.StageTransit
	case upperTipo == "I" || strings.Contains(upperDesc, "SIN RECIBIR") || strings.Contains(upperDesc, "INFORMADO") || strings.Contains(upperDesc, "REGISTRADO"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(upperDesc, "INCIDENCIA") || strings.Contains(upperDesc, "AUSENTE") || strings.Contains(upperDesc, "FALLIDO"):
		return models.StatusFailedAttempt, models.StageException
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func (c *CorreosExpressCarrier) fetchLive(ctx context.Context, trackingNumber string) (*correosExpressResponse, error) {
	apiURL := fmt.Sprintf("https://www.correosexpress.com/ws-cws-portal/rest/seguimientoEnvio/segEnvioWeb?codEnvio=%s", trackingNumber)
	req, err := NewCarrierRequest(http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req = req.WithContext(ctx)

	resp, err := correosExpressHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, classifyScraperHTTPStatus(resp.StatusCode, bodyBytes, resp.Header, "Correos Express")
	}

	if isWAFChallenge(resp.StatusCode, bodyBytes, resp.Header) {
		return nil, fmt.Errorf("%w: WAF or anti-bot challenge detected from Correos Express", models.ErrCarrierBlocked)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		return nil, fmt.Errorf("%w: unexpected non-JSON or malformed response from Correos Express: %v", models.ErrScraperSchemaChanged, err)
	}
	if !hasAnyJSONKey(rawMap, "codigoError", "codEnvio", "estado", "posicion") {
		return nil, fmt.Errorf("%w: unexpected JSON schema from Correos Express", models.ErrScraperSchemaChanged)
	}

	var ceResp correosExpressResponse
	if err := json.Unmarshal(bodyBytes, &ceResp); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON from Correos Express: %v", models.ErrScraperSchemaChanged, err)
	}

	if ceResp.CodigoError != "" && ceResp.CodigoError != "0" {
		return nil, models.ErrShipmentNotFound
	}
	if len(ceResp.Posicion) == 0 && ceResp.Estado == "" {
		return nil, models.ErrShipmentNotFound
	}

	return &ceResp, nil
}

func (c *CorreosExpressCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	cleanNum := strings.TrimSpace(trackingNumber)
	if cleanNum == "" {
		return nil, models.ErrShipmentNotFound
	}

	// 1. Attempt live tracking against Correos Express public REST API
	ceResp, err := c.fetchLive(ctx, cleanNum)
	if err == nil && ceResp != nil {
		var events []models.TrackingEvent
		for _, pos := range ceResp.Posicion {
			evTime := parseCorreosExpressDate(pos.Fecha, pos.Hora)
			status, stage := normalizeCorreosExpressStatus(pos.Desc+" "+pos.Descrip, pos.Tipo)

			location := strings.TrimSpace(pos.NomOfi)
			if location == "" {
				location = strings.TrimSpace(pos.CodOfi)
			}
			if location != "" && !strings.Contains(strings.ToLower(location), "spain") {
				location += ", Spain"
			}

			desc := pos.Descrip
			if desc == "" {
				desc = pos.Desc
			}

			events = append(events, models.TrackingEvent{
				Date:         evTime,
				RawDate:      fmt.Sprintf("%s %s", pos.Fecha, pos.Hora),
				Status:       status,
				Stage:        stage,
				Description:  desc,
				Location:     location,
				CarrierSlug:  c.Slug(),
				CarrierName:  c.Name(),
				CarrierIndex: 0,
			})
		}

		overallStatus, _ := normalizeCorreosExpressStatus(ceResp.Estado+" "+ceResp.DescEstado, "")
		for _, ev := range events {
			if ev.Status == models.StatusDelivered {
				overallStatus = models.StatusDelivered
				break
			}
		}

		statusText := ceResp.DescEstado
		if statusText == "" {
			statusText = ceResp.Estado
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

		progress := 60
		if overallStatus == models.StatusDelivered {
			progress = 100
		} else if overallStatus == models.StatusOutForDelivery {
			progress = 85
		}

		var deliveredAt *time.Time
		if ceResp.FechaEntrega != "" {
			t := parseCorreosExpressDate(ceResp.FechaEntrega, "")
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

		originName := ceResp.Origen
		if originName == "" {
			originName = "Spain"
		}
		destName := ceResp.Destino
		if destName == "" {
			destName = "Spain"
		}

		return &models.Shipment{
			TrackingNumber: cleanNum,
			Carrier:        c.Info(),
			Origin: models.CountryInfo{
				Name: originName,
				Code: "ES",
			},
			Destination: models.CountryInfo{
				Name: destName,
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

	// 2. Cross-border chaining: for 323... AliExpress parcels, check Cainiao handover if CE direct failed
	if strings.HasPrefix(cleanNum, "323") && len(cleanNum) == 23 {
		if opts.AdditionalFields == nil || opts.AdditionalFields["skip_cainiao_query"] != "true" {
			cOpts := opts
			if cOpts.AdditionalFields == nil {
				cOpts.AdditionalFields = make(map[string]string)
			}
			cOpts.AdditionalFields["skip_cainiao_query"] = "true"
			cainiao := NewCainiaoCarrier()
			cShipment, cErr := cainiao.Track(ctx, cleanNum, cOpts)
			if cErr == nil && cShipment != nil && len(cShipment.Events) > 0 {
				var ceEvents []models.TrackingEvent
				for _, ev := range cShipment.Events {
					evCopy := ev
					if evCopy.CarrierIndex == 1 || strings.EqualFold(evCopy.CarrierSlug, "correosexpress") {
						evCopy.CarrierSlug = c.Slug()
						evCopy.CarrierName = c.Name()
						evCopy.CarrierIndex = 0
						ceEvents = append(ceEvents, evCopy)
					}
				}
				if len(ceEvents) == 0 {
					ceEvents = cShipment.Events
				}

				return &models.Shipment{
					TrackingNumber:     cleanNum,
					Carrier:            c.Info(),
					AdditionalCarriers: []models.CarrierInfo{cainiao.Info()},
					Origin:             cShipment.Origin,
					Destination:        cShipment.Destination,
					Status:             cShipment.Status,
					StatusText:         cShipment.StatusText,
					Progress:           cShipment.Progress,
					DeliveredAt:        cShipment.DeliveredAt,
					DaysInTransit:      cShipment.DaysInTransit,
					Events:             ceEvents,
					UpdatedAt:          time.Now(),
				}, nil
			}
		}
	}

	// 3. Fallback only if AllowSimulated is explicitly true
	if opts.AllowSimulated {
		return c.generateSimulatedShipment(cleanNum, opts), nil
	}

	// If live tracking returned not found or failed, do not generate fake mocks
	if err != nil {
		return nil, err
	}
	return nil, models.ErrShipmentNotFound
}

func (c *CorreosExpressCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Sin recibir - Envío informado electrónicamente",
			Location:    "Madrid, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-12 * time.Hour),
			Description: "En tránsito - Envío clasificado en nave central",
			Location:    "Coslada (Madrid), Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-6 * time.Hour),
			Description: "En tránsito - Llegada a delegación de destino",
			Location:    "Barcelona Delegación, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-2 * time.Hour),
			Description: "En reparto - Envío asignado al repartidor para entrega",
			Location:    "Barcelona, Spain",
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
		StatusText:    "En reparto",
		Progress:      85,
		DaysInTransit: 1,
		Events:        events,
		UpdatedAt:     time.Now(),
	}
}
