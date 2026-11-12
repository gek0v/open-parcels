package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

var defaultInPostHTTPClient = NewCarrierHTTPClient(DefaultRequestTimeout)
var inpostHTTPClient = defaultInPostHTTPClient

// SetInPostHTTPClientForTest overrides the HTTP client for InPost in deterministic tests.
func SetInPostHTTPClientForTest(client *http.Client) func() {
	orig := inpostHTTPClient
	inpostHTTPClient = client
	return func() { inpostHTTPClient = orig }
}

type InPostCarrier struct {
	BaseCarrier
}

func NewInPostCarrier() *InPostCarrier {
	bc := NewBaseCarrier(
		"inpost",
		"InPost / Mondial Relay",
		"GLOBAL",
		"Global",
		models.CarrierTypeLogistics,
		"https://inpost.es",
		"https://www.inpost.es/seguimiento-del-envio?parcelNumber=%s",
	)
	bc.SetCapabilities(CarrierCapabilities{
		RequiresPostalCode:  true,
		SupportsWebScraping: true,
	})
	return &InPostCarrier{
		BaseCarrier: bc,
	}
}

// HasLiveTracking signals that InPost implements real network scraping/API.
func (c *InPostCarrier) HasLiveTracking() bool {
	return true
}

func (c *InPostCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return trackInPostShared(ctx, c, trackingNumber, opts, c.generateSimulatedShipment)
}

func (c *InPostCarrier) generateSimulatedShipment(trackingNumber string, opts models.TrackOptions) *models.Shipment {
	now := time.Now()
	events := []models.TrackingEvent{
		{
			Date:        now.Add(-30 * time.Hour),
			Description: "Parcel registered by sender for locker drop-off",
			Location:    "Point Relais / Locker",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Parcel collected from locker and in transit to sorting depot",
			Location:    "Regional Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Parcel is being delivered to destination locker / Punto Pack",
			Location:    "Destination Locker, Spain",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin:         extractInPostOrigin(events),
		Destination:    resolveInPostDestination(c, opts.PostalCode, opts),
		Status:         models.StatusOutForDelivery,
		StatusText:     "Out for delivery to Locker / Pickup Point",
		Progress:       85,
		DaysInTransit:  1,
		Events:         events,
		UpdatedAt:      time.Now(),
	}
}

func trackInPostShared(
	ctx context.Context,
	carrier Carrier,
	trackingNumber string,
	opts models.TrackOptions,
	simulatedFn func(string, models.TrackOptions) *models.Shipment,
) (*models.Shipment, error) {
	postalCode := strings.TrimSpace(opts.PostalCode)
	if postalCode == "" {
		if !opts.AllowSimulated {
			return nil, fmt.Errorf("%w: InPost tracking requires destination postal code (--postal-code <CP>)", models.ErrPostalCodeRequired)
		}
		if simulatedFn != nil {
			return simulatedFn(trackingNumber, opts), nil
		}
		return nil, models.ErrShipmentNotFound
	}

	shipment, err := fetchInPostLive(ctx, carrier, trackingNumber, postalCode, opts)
	if err == nil && shipment != nil {
		return shipment, nil
	}

	if opts.AllowSimulated && simulatedFn != nil {
		return simulatedFn(trackingNumber, opts), nil
	}

	if err != nil {
		return nil, err
	}

	return nil, models.ErrShipmentNotFound
}

func fetchInPostLive(ctx context.Context, carrier Carrier, trackingNumber, postalCode string, opts models.TrackOptions) (*models.Shipment, error) {
	targetURL := fmt.Sprintf(
		"https://www.inpost.es/seguimiento-del-envio?parcelNumber=%s&zipCode=%s",
		url.QueryEscape(trackingNumber),
		url.QueryEscape(postalCode),
	)

	req, err := NewCarrierRequest(http.MethodGet, targetURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", ResolveUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")

	resp, err := inpostHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", models.ErrCarrierTemporarilyUnavailable, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, classifyScraperHTTPStatus(resp.StatusCode, bodyBytes, resp.Header, "InPost")
	}

	if isWAFChallenge(resp.StatusCode, bodyBytes, resp.Header) {
		return nil, fmt.Errorf("%w: WAF or anti-bot challenge detected in InPost response", models.ErrCarrierBlocked)
	}

	htmlContent := string(bodyBytes)

	if strings.Contains(htmlContent, "Paquete no encontrado") || strings.Contains(htmlContent, "paquete no encontrado") {
		return nil, models.ErrShipmentNotFound
	}

	propsJSON, err := extractInPostProps(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrScraperSchemaChanged, err)
	}

	var rawData map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &rawData); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal InPost props: %v", models.ErrScraperSchemaChanged, err)
	}

	statusVal, ok := unwrapAstroInt(rawData["trackingDetailsStatus"])
	if !ok {
		return nil, fmt.Errorf("%w: missing or invalid trackingDetailsStatus in InPost props", models.ErrScraperSchemaChanged)
	}
	if statusVal == http.StatusForbidden {
		return nil, fmt.Errorf("%w: InPost trackingDetailsStatus 403", models.ErrCarrierBlocked)
	}
	if statusVal == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: %w: InPost trackingDetailsStatus 429", models.ErrCarrierRateLimited, models.ErrCarrierBlocked)
	}
	if statusVal == http.StatusConflict {
		return nil, models.ErrPostalCodeRequired
	}
	if statusVal != http.StatusOK {
		return nil, models.ErrShipmentNotFound
	}

	tdMap := unwrapAstroMap(rawData["trackingDetails"])
	if tdMap == nil {
		return nil, fmt.Errorf("%w: missing or invalid trackingDetails in InPost props", models.ErrScraperSchemaChanged)
	}

	milestonesRaw, hasMilestones := tdMap["milestones"]
	if !hasMilestones {
		return nil, fmt.Errorf("%w: missing milestones field in InPost trackingDetails", models.ErrScraperSchemaChanged)
	}
	milestonesList := unwrapAstroList(milestonesRaw)
	if milestonesList == nil {
		return nil, fmt.Errorf("%w: invalid milestones structure in InPost trackingDetails", models.ErrScraperSchemaChanged)
	}
	if len(milestonesList) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	var events []models.TrackingEvent
	var latestMilestoneName string
	var latestMilestoneDate time.Time

	for _, mItem := range milestonesList {
		mMap := unwrapAstroMap(mItem)
		if mMap == nil {
			continue
		}

		mType := unwrapAstroString(mMap["type"])
		mName := unwrapAstroString(mMap["name"])
		mDateStr := unwrapAstroString(mMap["date"])

		if mName != "" {
			if mDate, err := parseInPostDate(mDateStr); err == nil {
				if latestMilestoneName == "" || mDate.After(latestMilestoneDate) {
					latestMilestoneName = mName
					latestMilestoneDate = mDate
				}
			} else if latestMilestoneName == "" {
				latestMilestoneName = mName
			}
		}

		eventsRaw := mMap["events"]
		eventsList := unwrapAstroList(eventsRaw)

		mLocation := unwrapAstroString(mMap["location"])
		if mLocation == "" {
			mLocation = unwrapAstroString(mMap["place"])
		}
		if mLocation == "" {
			mLocation = unwrapAstroString(mMap["city"])
		}

		if len(eventsList) > 0 {
			for _, evItem := range eventsList {
				evMap := unwrapAstroMap(evItem)
				if evMap == nil {
					continue
				}

				evCode := unwrapAstroString(evMap["code"])
				evName := unwrapAstroString(evMap["name"])
				evDateStr := unwrapAstroString(evMap["date"])
				evDesc := unwrapAstroString(evMap["description"])

				evLocation := unwrapAstroString(evMap["location"])
				if evLocation == "" {
					evLocation = unwrapAstroString(evMap["place"])
				}
				if evLocation == "" {
					evLocation = unwrapAstroString(evMap["city"])
				}
				if evLocation == "" {
					evLocation = unwrapAstroString(evMap["agency"])
				}
				if evLocation == "" {
					evLocation = unwrapAstroString(evMap["depot"])
				}
				if evLocation == "" {
					evLocation = mLocation
				}

				if evDesc == "" {
					evDesc = evName
				}
				if evDesc == "" {
					evDesc = mName
				}
				if evDateStr == "" {
					evDateStr = mDateStr
				}

				evDate, err := parseInPostDate(evDateStr)
				if err != nil {
					evDate = time.Now()
				}

				evStatus, evStage := normalizeInPostEventStatus(evCode, evName, mType, evDesc)

				events = append(events, models.TrackingEvent{
					Date:        evDate,
					RawDate:     evDateStr,
					Status:      evStatus,
					Stage:       evStage,
					Description: evDesc,
					Location:    evLocation,
					EventCode:   evCode,
					CarrierSlug: carrier.Slug(),
					CarrierName: carrier.Name(),
				})
			}
		} else {
			evDate, err := parseInPostDate(mDateStr)
			if err != nil {
				evDate = time.Now()
			}
			evStatus, evStage := normalizeInPostStatus(mName, mType, "")

			events = append(events, models.TrackingEvent{
				Date:        evDate,
				RawDate:     mDateStr,
				Status:      evStatus,
				Stage:       evStage,
				Description: mName,
				Location:    mLocation,
				CarrierSlug: carrier.Slug(),
				CarrierName: carrier.Name(),
			})
		}
	}

	if len(events) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Date.Before(events[j].Date)
	})

	latestEvent := events[len(events)-1]
	overallStatus := latestEvent.Status

	statusText := latestMilestoneName
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

	origin := extractInPostOrigin(events)
	destination := resolveInPostDestination(carrier, postalCode, opts)
	if destCountry := unwrapAstroString(rawData["country"]); destCountry != "" && (destination.Code == "" || destination.Name == "Unknown") {
		if strings.EqualFold(destCountry, "ES") {
			destination = models.CountryInfo{Name: "Spain", Code: "ES"}
		} else {
			destination = models.CountryInfo{Name: destCountry, Code: strings.ToUpper(destCountry)}
		}
	}

	// Enrich origin, destination city, and event locations via InPost ShipX IT proxy when running live
	if origin.Name == "Unknown" && (inpostHTTPClient == defaultInPostHTTPClient || inpostItalyHTTPClient != defaultInPostItalyHTTPClient) {
		enrichCtx, enrichCancel := context.WithTimeout(ctx, 3*time.Second)
		defer enrichCancel()
		if itShipment, err := fetchInPostItalyLive(enrichCtx, carrier, trackingNumber, opts); err == nil && itShipment != nil {
			if itShipment.Origin.Name != "" && itShipment.Origin.Name != "Unknown" {
				origin = itShipment.Origin
			}
			if itShipment.Destination.Name != "" && itShipment.Destination.Name != "Unknown" && itShipment.Destination.Code == destination.Code {
				destination = itShipment.Destination
			}
			for i := range events {
				if events[i].Location != "" {
					continue
				}
				for _, itEv := range itShipment.Events {
					if itEv.Location != "" && ((events[i].EventCode != "" && itEv.EventCode == events[i].EventCode) || itEv.Date.Equal(events[i].Date)) {
						events[i].Location = itEv.Location
						break
					}
				}
			}
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

func extractInPostProps(htmlContent string) (string, error) {
	needle := "trackingDetailsStatus"
	idx := strings.Index(htmlContent, needle)
	if idx == -1 {
		return "", fmt.Errorf("trackingDetailsStatus not found in html")
	}

	prefix := htmlContent[:idx]
	propsDouble := strings.LastIndex(prefix, `props="`)
	propsSingle := strings.LastIndex(prefix, `props='`)

	var rawProps string
	if propsDouble != -1 && (propsSingle == -1 || propsDouble > propsSingle) {
		start := propsDouble + len(`props="`)
		end := strings.Index(htmlContent[start:], `"`)
		if end == -1 {
			return "", fmt.Errorf("props attribute end not found")
		}
		rawProps = htmlContent[start : start+end]
	} else if propsSingle != -1 {
		start := propsSingle + len(`props='`)
		end := strings.Index(htmlContent[start:], `'`)
		if end == -1 {
			return "", fmt.Errorf("props attribute end not found")
		}
		rawProps = htmlContent[start : start+end]
	} else {
		return "", fmt.Errorf("props attribute start not found")
	}

	return html.UnescapeString(rawProps), nil
}

func unwrapAstroVal(v any) any {
	arr, ok := v.([]any)
	if !ok || len(arr) < 2 {
		return nil
	}
	return arr[1]
}

func unwrapAstroString(v any) string {
	val := unwrapAstroVal(v)
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func unwrapAstroInt(v any) (int, bool) {
	val := unwrapAstroVal(v)
	if f, ok := val.(float64); ok {
		return int(f), true
	}
	return 0, false
}

func unwrapAstroMap(v any) map[string]any {
	val := unwrapAstroVal(v)
	if m, ok := val.(map[string]any); ok {
		return m
	}
	return nil
}

func unwrapAstroList(v any) []any {
	val := unwrapAstroVal(v)
	if list, ok := val.([]any); ok {
		return list
	}
	return nil
}

func parseInPostDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := ParseLocalTimeWithZone(f, dateStr, "Europe/Madrid"); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date %q", dateStr)
}

func normalizeInPostStatus(name, typeStr, desc string) (models.Status, models.Stage) {
	return normalizeInPostEventStatus("", name, typeStr, desc)
}

func normalizeInPostEventStatus(code, name, typeStr, desc string) (models.Status, models.Stage) {
	upperCode := strings.ToUpper(strings.TrimSpace(code))
	upperType := strings.ToUpper(strings.TrimSpace(typeStr))
	lower := strings.ToLower(strings.TrimSpace(name + " " + desc))

	// 1. Official InPost event code mapping when available
	if upperCode != "" {
		switch {
		case upperCode == "EOL.9004":
			return models.StatusException, models.StageException
		case strings.HasPrefix(upperCode, "EOL."):
			return models.StatusDelivered, models.StageDelivery
		case strings.HasPrefix(upperCode, "RTS."):
			return models.StatusException, models.StageException
		case strings.HasPrefix(upperCode, "LMD.9"):
			return models.StatusFailedAttempt, models.StageException
		case strings.HasPrefix(upperCode, "LMD."):
			if upperType == "DELIVERED" || isInPostDeliveredText(lower) {
				return models.StatusDelivered, models.StageDelivery
			}
			return models.StatusOutForDelivery, models.StageLastMile
		case strings.HasPrefix(upperCode, "MMD."):
			return models.StatusInTransit, models.StageTransit
		case upperCode == "FMD.1001":
			return models.StatusInfoReceived, models.StageCreated
		case upperCode == "FMD.9001":
			return models.StatusException, models.StageException
		case strings.HasPrefix(upperCode, "FMD."):
			return models.StatusInTransit, models.StagePickedUp
		case strings.HasPrefix(upperCode, "CRE."), strings.HasPrefix(upperCode, "FUL."):
			return models.StatusInfoReceived, models.StageCreated
		}
	}

	// 2. Official InPost milestone type enum mapping
	switch upperType {
	case "DELIVERED":
		return models.StatusDelivered, models.StageDelivery
	case "AWAITING_COLLECTION", "IN_TRANSIT_LAST_MILE":
		return models.StatusOutForDelivery, models.StageLastMile
	case "RETURN":
		return models.StatusException, models.StageException
	case "NOT_DELIVERED":
		return models.StatusFailedAttempt, models.StageException
	case "IN_TRANSIT":
		if isInPostCourierPickupText(lower) {
			return models.StatusInTransit, models.StagePickedUp
		}
		return models.StatusInTransit, models.StageTransit
	case "AT_THE_ORIGIN":
		if isInPostCourierPickupText(lower) {
			return models.StatusInTransit, models.StagePickedUp
		}
		return models.StatusInfoReceived, models.StageCreated
	}

	// 3. Keyword fallback when milestone type/code are absent
	switch {
	case isInPostDeliveredText(lower):
		return models.StatusDelivered, models.StageDelivery

	case isInPostCourierPickupText(lower):
		return models.StatusInTransit, models.StagePickedUp

	case isInPostOutForDeliveryText(lower):
		return models.StatusOutForDelivery, models.StageLastMile

	case isInPostTransitText(lower):
		return models.StatusInTransit, models.StageTransit

	case isInPostExceptionText(lower):
		return models.StatusException, models.StageException

	case isInPostOriginText(lower):
		return models.StatusInfoReceived, models.StageCreated

	default:
		return models.StatusInfoReceived, models.StageCreated
	}
}

func isInPostCourierPickupText(s string) bool {
	if strings.Contains(s, "listo para") ||
		strings.Contains(s, "será recogido") ||
		strings.Contains(s, "sera recogido") {
		return false
	}
	return strings.Contains(s, "recogido por el mensajero") ||
		strings.Contains(s, "recogido por mensajero") ||
		strings.Contains(s, "recogido por el repartidor") ||
		strings.Contains(s, "collected by the courier") ||
		strings.Contains(s, "collected by courier")
}

func isInPostDeliveredText(s string) bool {
	if strings.Contains(s, "no entregado") ||
		strings.Contains(s, "not delivered") ||
		strings.Contains(s, "undelivered") ||
		strings.Contains(s, "listo para") ||
		strings.Contains(s, "será") ||
		strings.Contains(s, "sera") ||
		strings.Contains(s, "mensajero") ||
		strings.Contains(s, "courier") {
		return false
	}
	return strings.Contains(s, "entregado") ||
		strings.Contains(s, "delivered") ||
		strings.Contains(s, "recogido por el cliente") ||
		strings.Contains(s, "recogido por el destinatario") ||
		strings.Contains(s, "recogido por cliente") ||
		strings.Contains(s, "recogido por destinatario") ||
		strings.Contains(s, "collected by customer") ||
		strings.Contains(s, "collected by recipient")
}

func isInPostOutForDeliveryText(s string) bool {
	return strings.Contains(s, "disponible") ||
		strings.Contains(s, "punto pack") ||
		strings.Contains(s, "locker") ||
		strings.Contains(s, "reparto") ||
		strings.Contains(s, "out_for_delivery") ||
		strings.Contains(s, "awaiting_collection") ||
		strings.Contains(s, "available_for_pickup") ||
		strings.Contains(s, "ready_for_pickup") ||
		strings.Contains(s, "ready for pickup")
}

func isInPostTransitText(s string) bool {
	return strings.Contains(s, "tránsito") ||
		strings.Contains(s, "transito") ||
		strings.Contains(s, "in_transit") ||
		strings.Contains(s, "en ruta") ||
		strings.Contains(s, "en camino") ||
		strings.Contains(s, "clasificación") ||
		strings.Contains(s, "centro logístico") ||
		strings.Contains(s, "centro logistico") ||
		strings.Contains(s, "logistics centre") ||
		strings.Contains(s, "hub") ||
		strings.Contains(s, "depot") ||
		strings.Contains(s, "sorting")
}

func isInPostExceptionText(s string) bool {
	return strings.Contains(s, "incidencia") ||
		strings.Contains(s, "fallido") ||
		strings.Contains(s, "exception") ||
		strings.Contains(s, "devuelto") ||
		strings.Contains(s, "return") ||
		strings.Contains(s, "error")
}

func isInPostOriginText(s string) bool {
	return strings.Contains(s, "preparación") ||
		strings.Contains(s, "preparacion") ||
		strings.Contains(s, "at_the_origin") ||
		strings.Contains(s, "registrado informáticamente") ||
		strings.Contains(s, "registrado informaticamente") ||
		strings.Contains(s, "info received") ||
		strings.Contains(s, "shipment created") ||
		strings.Contains(s, "cre.") ||
		strings.Contains(s, "created") ||
		strings.Contains(s, "listo para ser recogido") ||
		strings.Contains(s, "será recogido")
}

func inpostProgress(status models.Status) int {
	switch status {
	case models.StatusDelivered:
		return 100
	case models.StatusOutForDelivery:
		return 85
	case models.StatusInTransit:
		return 50
	case models.StatusInfoReceived:
		return 25
	case models.StatusException:
		return 40
	default:
		return 20
	}
}

// extractInPostOrigin inspects events to determine origin if an explicit location exists.
// If no explicit origin location is found or recognizable, it returns Name: "Unknown", Code: "".
func extractInPostOrigin(events []models.TrackingEvent) models.CountryInfo {
	// Look through non-delivery events for an explicit origin location.
	// In InPost, events are chronologically ordered. The earliest event with an explicit location
	// represents the parcel's origin.
	for _, ev := range events {
		if ev.Stage == models.StageLastMile || ev.Stage == models.StageDelivery ||
			ev.Status == models.StatusOutForDelivery || ev.Status == models.StatusDelivered {
			continue
		}
		loc := strings.TrimSpace(ev.Location)
		if loc != "" {
			if country, ok := detectCountryFromLocation(loc); ok {
				return country
			}
		}
	}

	return models.CountryInfo{
		Name: "Unknown",
		Code: "",
	}
}

// resolveInPostDestination determines destination country info based on carrier and postal code.
func resolveInPostDestination(carrier Carrier, postalCode string, opts models.TrackOptions) models.CountryInfo {
	postalCode = strings.TrimSpace(postalCode)
	destCountry := strings.ToUpper(strings.TrimSpace(opts.DestinationCountry))

	if destCountry == "ES" || strings.EqualFold(destCountry, "Spain") || strings.EqualFold(destCountry, "España") {
		return models.CountryInfo{Name: "Spain", Code: "ES"}
	}

	if isSpanishPostalCode(postalCode) || carrier.Country() == "ES" || strings.Contains(carrier.Website(), ".es") {
		return models.CountryInfo{
			Name: "Spain",
			Code: "ES",
		}
	}

	if destCountry != "" {
		return models.CountryInfo{
			Name: destCountry,
			Code: destCountry,
		}
	}

	return models.CountryInfo{
		Name: "Unknown",
		Code: "",
	}
}

func isSpanishPostalCode(postalCode string) bool {
	postalCode = strings.TrimSpace(postalCode)
	if len(postalCode) != 5 {
		return false
	}
	for _, r := range postalCode {
		if r < '0' || r > '9' {
			return false
		}
	}
	prefix := postalCode[:2]
	return prefix >= "01" && prefix <= "52"
}

func detectCountryFromLocation(loc string) (models.CountryInfo, bool) {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return models.CountryInfo{}, false
	}

	lower := strings.ToLower(loc)

	// France
	if containsAny(lower, "france", "francia", "français", "francais", "paris", "lille", "lyon", "marseille", "bordeaux", "toulouse", "strasbourg", "nantes") || matchExplicitCountryCode(loc, "FR") {
		return models.CountryInfo{Name: "France", Code: "FR"}, true
	}

	// Poland
	if containsAny(lower, "poland", "polonia", "polska", "warsaw", "warszawa", "krakow", "kraków", "lodz", "łódź", "wroclaw", "wrocław", "poznan", "poznań", "gdansk", "gdańsk") || matchExplicitCountryCode(loc, "PL") {
		return models.CountryInfo{Name: "Poland", Code: "PL"}, true
	}

	// Spain
	if containsAny(lower, "spain", "españa", "espana", "espanya", "madrid", "barcelona", "valencia", "sevilla", "zaragoza", "málaga", "malaga", "bilbao", "valladolid") || matchExplicitCountryCode(loc, "ES") {
		return models.CountryInfo{Name: "Spain", Code: "ES"}, true
	}

	// Italy
	if containsAny(lower, "italy", "italia", "rome", "roma", "milan", "milano", "naples", "napoli", "turin", "torino", "bologna", "firenze") || matchExplicitCountryCode(loc, "IT") {
		return models.CountryInfo{Name: "Italy", Code: "IT"}, true
	}

	// Portugal
	if containsAny(lower, "portugal", "lisbon", "lisboa", "porto", "braga", "coimbra", "faro") || matchExplicitCountryCode(loc, "PT") {
		return models.CountryInfo{Name: "Portugal", Code: "PT"}, true
	}

	// Belgium
	if containsAny(lower, "belgium", "belgique", "bélgica", "belgica", "belgië", "belgie", "brussels", "bruxelles", "antwerp", "anvers", "gent", "liege") || matchExplicitCountryCode(loc, "BE") {
		return models.CountryInfo{Name: "Belgium", Code: "BE"}, true
	}

	// Netherlands
	if containsAny(lower, "netherlands", "países bajos", "paises bajos", "nederland", "amsterdam", "rotterdam", "utrecht", "the hague") || matchExplicitCountryCode(loc, "NL") {
		return models.CountryInfo{Name: "Netherlands", Code: "NL"}, true
	}

	// Luxembourg
	if containsAny(lower, "luxembourg", "luxemburgo") || matchExplicitCountryCode(loc, "LU") {
		return models.CountryInfo{Name: "Luxembourg", Code: "LU"}, true
	}

	// Germany
	if containsAny(lower, "germany", "allemagne", "alemania", "deutschland", "berlin", "munich", "münchen", "frankfurt", "hamburg", "köln", "cologne") || matchExplicitCountryCode(loc, "DE") {
		return models.CountryInfo{Name: "Germany", Code: "DE"}, true
	}

	// United Kingdom
	if containsAny(lower, "united kingdom", "reino unido", "london", "manchester") || matchExplicitCountryCode(loc, "GB") || matchExplicitCountryCode(loc, "UK") {
		return models.CountryInfo{Name: "United Kingdom", Code: "GB"}, true
	}

	// Austria
	if containsAny(lower, "austria", "österreich", "vienna", "wien") || matchExplicitCountryCode(loc, "AT") {
		return models.CountryInfo{Name: "Austria", Code: "AT"}, true
	}

	// Switzerland
	if containsAny(lower, "switzerland", "suisse", "schweiz", "svizzera", "suiza", "zurich", "geneva") || matchExplicitCountryCode(loc, "CH") {
		return models.CountryInfo{Name: "Switzerland", Code: "CH"}, true
	}

	// Czech Republic
	if containsAny(lower, "czech", "czechia", "chequia", "prague", "praha") || matchExplicitCountryCode(loc, "CZ") {
		return models.CountryInfo{Name: "Czech Republic", Code: "CZ"}, true
	}

	// Slovakia
	if containsAny(lower, "slovakia", "eslovaquia", "bratislava") || matchExplicitCountryCode(loc, "SK") {
		return models.CountryInfo{Name: "Slovakia", Code: "SK"}, true
	}

	// Hungary
	if containsAny(lower, "hungary", "hungría", "hungria", "budapest") || matchExplicitCountryCode(loc, "HU") {
		return models.CountryInfo{Name: "Hungary", Code: "HU"}, true
	}

	// Romania
	if containsAny(lower, "romania", "rumania", "bucharest", "bucuresti") || matchExplicitCountryCode(loc, "RO") {
		return models.CountryInfo{Name: "Romania", Code: "RO"}, true
	}

	// Ireland
	if containsAny(lower, "ireland", "irlanda", "dublin") || matchExplicitCountryCode(loc, "IE") {
		return models.CountryInfo{Name: "Ireland", Code: "IE"}, true
	}

	return models.CountryInfo{}, false
}

func matchExplicitCountryCode(loc, code string) bool {
	if strings.EqualFold(strings.TrimSpace(loc), code) {
		return true
	}
	patterns := []string{
		"(" + code + ")",
		"[" + code + "]",
		", " + code,
		"-" + code,
		" " + code,
	}
	for _, p := range patterns {
		if strings.Contains(loc, p) {
			return true
		}
	}
	return false
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
