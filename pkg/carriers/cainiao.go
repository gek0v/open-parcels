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

var cainiaoHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)

// SetCainiaoHTTPClientForTest overrides the HTTP client for Cainiao in deterministic tests.
func SetCainiaoHTTPClientForTest(client *http.Client) func() {
	orig := cainiaoHTTPClient
	cainiaoHTTPClient = client
	return func() { cainiaoHTTPClient = orig }
}

type CainiaoCarrier struct {
	BaseCarrier
}

func NewCainiaoCarrier() *CainiaoCarrier {
	bc := NewBaseCarrier(
		"cainiao",
		"Cainiao Global",
		"CN",
		"China",
		models.CarrierTypeCrossBorder,
		"https://global.cainiao.com",
		"https://global.cainiao.com/detail.htm?mailNoList=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		SupportsLiveAPI: true,
	})
	return &CainiaoCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that Cainiao implements real API connectivity.
func (c *CainiaoCarrier) HasLiveTracking() bool {
	return true
}

type cainiaoDetailItem struct {
	Desc         string `json:"desc"`
	Time         any    `json:"time"`
	TimeStr      string `json:"timeStr"`
	Date         string `json:"date"`
	StanderdDesc string `json:"standerdDesc"`
	DescTitle    string `json:"descTitle"`
	ActionCode   string `json:"actionCode"`
	TimeZone     string `json:"timeZone"`
	Group        *struct {
		NodeCode string `json:"nodeCode"`
		NodeDesc string `json:"nodeDesc"`
	} `json:"group"`
}

type cainiaoSectionItem struct {
	SectionName string              `json:"sectionName"`
	StatusDesc  string              `json:"statusDesc"`
	DetailList  []cainiaoDetailItem `json:"detailList"`
}

type cainiaoItem struct {
	MailNo        string               `json:"mailNo"`
	OriginCountry string               `json:"originCountry"`
	DestCountry   string               `json:"destCountry"`
	DestMailNo    string               `json:"destMailNo"`
	DestCp        string               `json:"destCp"` // Destination courier partner
	Status        string               `json:"status"`
	StatusDesc    string               `json:"statusDesc"`
	DetailList    []cainiaoDetailItem  `json:"detailList"`
	SectionList   []cainiaoSectionItem `json:"sectionList"`
	DaysNumber    string               `json:"daysNumber"`
}

type cainiaoResponse struct {
	Success   bool          `json:"success"`
	ErrorCode string        `json:"errorCode"`
	ErrorMsg  string        `json:"errorMsg"`
	Module    []cainiaoItem `json:"module"`
	Data      []cainiaoItem `json:"data"`
}

func parseCainiaoTime(det cainiaoDetailItem) (time.Time, string) {
	if det.TimeStr != "" {
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, det.TimeStr); err == nil {
				return t, det.TimeStr
			}
		}
	}

	if det.Time != nil {
		switch v := det.Time.(type) {
		case float64:
			if v > 0 {
				t := time.UnixMilli(int64(v))
				return t, t.Format("2006-01-02 15:04:05")
			}
		case string:
			if v != "" {
				for _, layout := range []string{
					"2006-01-02 15:04:05",
					"2006-01-02 15:04",
					time.RFC3339,
				} {
					if t, err := time.Parse(layout, v); err == nil {
						return t, v
					}
				}
			}
		}
	}

	if det.Date != "" {
		if t, err := time.Parse("2006-01-02", det.Date); err == nil {
			return t, det.Date
		}
	}

	return time.Time{}, ""
}

func normalizeCainiaoEvent(det cainiaoDetailItem) (models.Status, models.Stage) {
	action := strings.ToUpper(strings.TrimSpace(det.ActionCode))
	desc := strings.ToLower(det.Desc + " " + det.StanderdDesc)

	switch {
	case action == "GTMS_SIGNED" || strings.Contains(desc, "delivered") || strings.Contains(desc, "entregado"):
		return models.StatusDelivered, models.StageDelivery
	case action == "GTMS_DELIVERING" || strings.Contains(desc, "out for delivery") || strings.Contains(desc, "en reparto") || strings.Contains(desc, "en proceso de entrega"):
		return models.StatusOutForDelivery, models.StageLastMile
	case action == "GTMS_ACCEPT" || strings.Contains(desc, "local delivery company") || strings.Contains(desc, "operador logístico"):
		return models.StatusInTransit, models.StageHandover
	case strings.Contains(desc, "customs") || strings.Contains(desc, "clearance") || strings.Contains(desc, "aduan") || strings.Contains(desc, "import"):
		return models.StatusInTransit, models.StageCustoms
	case strings.Contains(desc, "airline") || strings.Contains(desc, "departed") || strings.Contains(desc, "transit") || strings.Contains(desc, "flight") || strings.Contains(desc, "arrived at destination"):
		return models.StatusInTransit, models.StageTransit
	case strings.Contains(desc, "sorting") || strings.Contains(desc, "outbound") || strings.Contains(desc, "inbound") || strings.Contains(desc, "warehouse"):
		return models.StatusInTransit, models.StagePickedUp
	case strings.Contains(desc, "order received") || strings.Contains(desc, "electronic") || strings.Contains(desc, "pre-registered") || strings.Contains(desc, "recibida"):
		return models.StatusInfoReceived, models.StageCreated
	case strings.Contains(desc, "failed") || strings.Contains(desc, "fallido") || strings.Contains(desc, "exception") || strings.Contains(desc, "incidencia"):
		return models.StatusFailedAttempt, models.StageException
	default:
		return models.StatusInTransit, models.StageTransit
	}
}

func normalizeCainiaoShipmentStatus(status string, events []models.TrackingEvent) (models.Status, models.Stage) {
	upper := strings.ToUpper(strings.TrimSpace(status))
	switch upper {
	case "DELIVERED", "SIGNED":
		return models.StatusDelivered, models.StageDelivery
	case "DELIVERING", "OUT_FOR_DELIVERY":
		return models.StatusOutForDelivery, models.StageLastMile
	case "EXCEPTION", "FAILED":
		return models.StatusException, models.StageException
	case "WAITING_FOR_DELIVERY", "TRANSIT":
		return models.StatusInTransit, models.StageTransit
	}

	for _, ev := range events {
		if ev.Status == models.StatusDelivered {
			return models.StatusDelivered, models.StageDelivery
		}
	}
	for _, ev := range events {
		if ev.Status == models.StatusOutForDelivery {
			return models.StatusOutForDelivery, models.StageLastMile
		}
	}

	return models.StatusInTransit, models.StageTransit
}

func isCainiaoAntiBot(statusCode int, body string) bool {
	if statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests {
		return true
	}
	bodyUpper := strings.ToUpper(body)
	return strings.Contains(bodyUpper, "RGV587_FLAG") ||
		strings.Contains(bodyUpper, "SEC_CHALLENGE") ||
		strings.Contains(bodyUpper, "PUNISH_ACTION") ||
		strings.Contains(bodyUpper, "ANTI_BOT") ||
		strings.Contains(bodyUpper, "CAPTCHA") ||
		strings.Contains(bodyUpper, "DENIED")
}

func isCainiaoNotFound(cResp *cainiaoResponse) bool {
	errCode := strings.ToUpper(strings.TrimSpace(cResp.ErrorCode))
	errMsg := strings.ToUpper(strings.TrimSpace(cResp.ErrorMsg))
	return errCode == "ORDER_NOT_FOUND" ||
		errCode == "PACKAGE_NOT_FOUND" ||
		errCode == "NOT_FOUND" ||
		errCode == "MAIL_NO_NOT_FOUND" ||
		errCode == "PARAM_ERROR" ||
		strings.Contains(errMsg, "NOT FOUND") ||
		strings.Contains(errMsg, "NOT EXIST") ||
		strings.Contains(errMsg, "INEXISTENTE") ||
		strings.Contains(errMsg, "NO EXISTE")
}

func resolveCainiaoLastMileCarrier(destCp, trackingNumber, destCode string) *models.CarrierInfo {
	cleanCp := strings.ToLower(strings.TrimSpace(destCp))

	switch {
	case strings.Contains(cleanCp, "correos express") || (destCp == "" && strings.HasPrefix(trackingNumber, "323") && destCode == "ES"):
		return &models.CarrierInfo{
			Slug:        "correosexpress",
			Name:        "Correos Express",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeExpress,
			Website:     "https://www.correosexpress.com",
			TrackingURL: "https://www.correosexpress.com/web/correosexpress/consultanos?shippingNumber=%s",
		}
	case strings.Contains(cleanCp, "correos"):
		return &models.CarrierInfo{
			Slug:        "correos-spain",
			Name:        "Correos España",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypePostal,
			Website:     "https://www.correos.es",
			TrackingURL: "https://www.correos.es/es/es/herramientas/localizador/envios/detalle?tracking-number=%s",
		}
	case strings.Contains(cleanCp, "ctt"):
		return &models.CarrierInfo{
			Slug:        "ctt-express",
			Name:        "CTT Express",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeExpress,
			Website:     "https://www.cttexpress.com",
		}
	case strings.Contains(cleanCp, "gls"):
		return &models.CarrierInfo{
			Slug:        "asm-gls-spain",
			Name:        "GLS Spain",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeExpress,
			Website:     "https://www.gls-spain.es",
		}
	case strings.Contains(cleanCp, "inpost") || strings.Contains(cleanCp, "mondial"):
		return &models.CarrierInfo{
			Slug:        "inpost-es",
			Name:        "InPost Spain",
			Country:     "ES",
			CountryName: "Spain",
			Type:        models.CarrierTypeLogistics,
			Website:     "https://inpost.es",
		}
	case destCp != "":
		slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(destCp), " ", "-"))
		return &models.CarrierInfo{
			Slug: slug,
			Name: destCp,
		}
	default:
		return nil
	}
}

func (c *CainiaoCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	lang := "en-US"
	if strings.HasPrefix(strings.ToLower(opts.Language), "es") {
		lang = "es-ES"
	} else if opts.Language != "" {
		lang = strings.ToLower(opts.Language)
	}

	apiURL := fmt.Sprintf("https://global.cainiao.com/global/detail.json?mailNos=%s&lang=%s", trackingNumber, lang)
	req, err := NewCarrierRequest(http.MethodGet, apiURL)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req = req.WithContext(ctx)

	resp, err := cainiaoHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: %w: Cainiao returned HTTP 429", models.ErrCarrierRateLimited, models.ErrCarrierBlocked)
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: %w: Cainiao returned HTTP 403", models.ErrCarrierBlocked, models.ErrCarrierTemporarilyUnavailable)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, models.ErrShipmentNotFound
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	bodyStr := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		return nil, classifyScraperHTTPStatus(resp.StatusCode, bodyBytes, resp.Header, "Cainiao")
	}

	if isCainiaoAntiBot(resp.StatusCode, bodyStr) || isWAFChallenge(resp.StatusCode, bodyBytes, resp.Header) {
		return nil, fmt.Errorf("%w: %w: Cainiao anti-bot challenge detected", models.ErrCarrierBlocked, models.ErrCarrierTemporarilyUnavailable)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		return nil, fmt.Errorf("%w: %w: malformed JSON from Cainiao: %v", models.ErrScraperSchemaChanged, models.ErrCarrierTemporarilyUnavailable, err)
	}
	if !hasAnyJSONKey(rawMap, "module", "data", "success", "errorCode", "errorMsg") {
		return nil, fmt.Errorf("%w: unexpected JSON schema from Cainiao", models.ErrScraperSchemaChanged)
	}

	var cResp cainiaoResponse
	if err := json.Unmarshal(bodyBytes, &cResp); err != nil {
		return nil, fmt.Errorf("%w: %w: malformed JSON from Cainiao: %v", models.ErrScraperSchemaChanged, models.ErrCarrierTemporarilyUnavailable, err)
	}

	if isCainiaoNotFound(&cResp) {
		return nil, models.ErrShipmentNotFound
	}
	if strings.Contains(strings.ToUpper(cResp.ErrorCode), "RGV") || strings.Contains(strings.ToUpper(cResp.ErrorCode), "CHALLENGE") {
		return nil, fmt.Errorf("%w: %w: Cainiao challenge error code %s", models.ErrCarrierBlocked, models.ErrCarrierTemporarilyUnavailable, cResp.ErrorCode)
	}

	var items []cainiaoItem
	if len(cResp.Module) > 0 {
		items = cResp.Module
	} else if len(cResp.Data) > 0 {
		items = cResp.Data
	}

	if len(items) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	item := items[0]
	var rawDetails []cainiaoDetailItem
	if len(item.DetailList) > 0 {
		rawDetails = item.DetailList
	} else {
		for _, sec := range item.SectionList {
			rawDetails = append(rawDetails, sec.DetailList...)
		}
	}

	if len(rawDetails) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	originCode := "CN"
	if strings.EqualFold(item.OriginCountry, "España") || strings.EqualFold(item.OriginCountry, "Spain") || strings.EqualFold(item.OriginCountry, "ES") {
		originCode = "ES"
	} else if len(item.OriginCountry) == 2 {
		originCode = strings.ToUpper(item.OriginCountry)
	}

	destCode := "ES"
	if strings.EqualFold(item.DestCountry, "China") || strings.EqualFold(item.DestCountry, "CN") {
		destCode = "CN"
	} else if strings.EqualFold(item.DestCountry, "España") || strings.EqualFold(item.DestCountry, "Spain") || strings.EqualFold(item.DestCountry, "ES") {
		destCode = "ES"
	} else if len(item.DestCountry) == 2 {
		destCode = strings.ToUpper(item.DestCountry)
	} else if opts.DestinationCountry != "" {
		destCode = strings.ToUpper(opts.DestinationCountry)
	}

	originName := item.OriginCountry
	if originName == "" {
		originName = "China"
	}
	destName := item.DestCountry
	if destName == "" {
		destName = "Spain"
	}

	isSpainShipment := strings.EqualFold(destCode, "ES") || strings.HasPrefix(trackingNumber, "323") || strings.EqualFold(item.DestCountry, "Spain") || strings.EqualFold(item.DestCountry, "España")

	var events []models.TrackingEvent
	for _, det := range rawDetails {
		evTime, rawDate := parseCainiaoTime(det)
		status, stage := normalizeCainiaoEvent(det)

		desc := det.Desc
		if strings.HasPrefix(strings.ToLower(opts.Language), "es") && det.StanderdDesc != "" {
			desc = det.StanderdDesc
		}
		if desc == "" {
			desc = det.StanderdDesc
		}
		if desc == "" {
			desc = "Tracking update"
		}

		carrierSlug := c.Slug()
		carrierName := c.Name()
		carrierIdx := 0

		actionCode := strings.ToUpper(strings.TrimSpace(det.ActionCode))
		combinedDesc := strings.ToLower(det.Desc + " " + det.StanderdDesc)

		isLocalDelivery := isSpainShipment && (stage == models.StageHandover ||
			stage == models.StageLastMile ||
			stage == models.StageDelivery ||
			status == models.StatusDelivered ||
			status == models.StatusOutForDelivery ||
			strings.Contains(combinedDesc, "local delivery company") ||
			strings.Contains(combinedDesc, "local  delivery company") ||
			strings.Contains(combinedDesc, "operador logístico") ||
			strings.Contains(combinedDesc, "en reparto") ||
			strings.Contains(combinedDesc, "en proceso de entrega") ||
			strings.Contains(combinedDesc, "entregado") ||
			strings.Contains(combinedDesc, "package delivered") ||
			actionCode == "GTMS_ACCEPT" ||
			actionCode == "GTMS_DELIVERING" ||
			actionCode == "GTMS_SIGNED")

		if isLocalDelivery {
			carrierSlug = "correosexpress"
			carrierName = "Correos Express"
			carrierIdx = 1
		}

		events = append(events, models.TrackingEvent{
			Date:         evTime,
			RawDate:      rawDate,
			Status:       status,
			Stage:        stage,
			Description:  desc,
			CarrierSlug:  carrierSlug,
			CarrierName:  carrierName,
			CarrierIndex: carrierIdx,
		})
	}

	overallStatus, _ := normalizeCainiaoShipmentStatus(item.Status, events)
	statusText := item.StatusDesc
	if statusText == "" {
		if overallStatus == models.StatusDelivered {
			if strings.HasPrefix(strings.ToLower(opts.Language), "es") {
				statusText = "Entregado"
			} else {
				statusText = "Delivered"
			}
		} else if overallStatus == models.StatusOutForDelivery {
			if strings.HasPrefix(strings.ToLower(opts.Language), "es") {
				statusText = "En reparto"
			} else {
				statusText = "Out for delivery"
			}
		} else {
			if strings.HasPrefix(strings.ToLower(opts.Language), "es") {
				statusText = "En tránsito"
			} else {
				statusText = "In transit"
			}
		}
	}

	progress := 60
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

	shipment := &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Status:         overallStatus,
		StatusText:     statusText,
		Progress:       progress,
		DeliveredAt:    deliveredAt,
		Origin: models.CountryInfo{
			Name: originName,
			Code: originCode,
		},
		Destination: models.CountryInfo{
			Name: destName,
			Code: destCode,
		},
		LastMileTrackingNumber: item.DestMailNo,
		Events:                 events,
		UpdatedAt:              time.Now(),
	}

	lastMileCarrier := resolveCainiaoLastMileCarrier(item.DestCp, trackingNumber, destCode)
	if lastMileCarrier != nil {
		shipment.LastMileCarrier = lastMileCarrier
		if shipment.LastMileTrackingNumber == "" && strings.HasPrefix(trackingNumber, "323") {
			shipment.LastMileTrackingNumber = trackingNumber
		}
		shipment.AdditionalCarriers = []models.CarrierInfo{*lastMileCarrier}
	}

	return shipment, nil
}
