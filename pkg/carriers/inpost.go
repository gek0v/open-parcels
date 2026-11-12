package carriers

import (
	"context"
	"encoding/json"
	"errors"
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

var inpostHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
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

	if errors.Is(err, models.ErrPostalCodeRequired) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")

	resp, err := inpostHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, models.ErrShipmentNotFound
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	htmlContent := string(bodyBytes)

	if strings.Contains(htmlContent, "Paquete no encontrado") || strings.Contains(htmlContent, "paquete no encontrado") {
		return nil, models.ErrShipmentNotFound
	}

	propsJSON, err := extractInPostProps(htmlContent)
	if err != nil {
		return nil, err
	}

	var rawData map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal InPost props: %w", err)
	}

	statusVal, ok := unwrapAstroInt(rawData["trackingDetailsStatus"])
	if !ok || statusVal != 200 {
		return nil, models.ErrShipmentNotFound
	}

	tdMap := unwrapAstroMap(rawData["trackingDetails"])
	if tdMap == nil {
		return nil, models.ErrShipmentNotFound
	}

	milestonesRaw := tdMap["milestones"]
	milestonesList := unwrapAstroList(milestonesRaw)
	if len(milestonesList) == 0 {
		return nil, models.ErrShipmentNotFound
	}

	var events []models.TrackingEvent
	var latestMilestoneName string

	for _, mItem := range milestonesList {
		mMap := unwrapAstroMap(mItem)
		if mMap == nil {
			continue
		}

		mType := unwrapAstroString(mMap["type"])
		mName := unwrapAstroString(mMap["name"])
		mDateStr := unwrapAstroString(mMap["date"])

		if mName != "" {
			latestMilestoneName = mName
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

				evStatus, evStage := normalizeInPostStatus(evName, mType, evDesc)

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

	propsStart := strings.LastIndex(htmlContent[:idx], `props="`)
	if propsStart == -1 {
		return "", fmt.Errorf("props attribute start not found")
	}
	propsStart += len(`props="`)

	propsEnd := strings.Index(htmlContent[propsStart:], `"`)
	if propsEnd == -1 {
		return "", fmt.Errorf("props attribute end not found")
	}

	rawProps := htmlContent[propsStart : propsStart+propsEnd]
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
	combined := strings.ToLower(name + " " + typeStr + " " + desc)

	switch {
	case strings.Contains(combined, "entregado") ||
		strings.Contains(combined, "delivered") ||
		strings.Contains(combined, "recogido por el destinatario") ||
		strings.Contains(combined, "recogido por el cliente") ||
		strings.Contains(combined, "retiré par le destinataire"):
		return models.StatusDelivered, models.StageDelivery

	case strings.Contains(combined, "disponible") ||
		strings.Contains(combined, "punto pack") ||
		strings.Contains(combined, "locker") ||
		strings.Contains(combined, "reparto") ||
		strings.Contains(combined, "out_for_delivery") ||
		strings.Contains(combined, "available_for_pickup") ||
		strings.Contains(combined, "ready_for_pickup"):
		return models.StatusOutForDelivery, models.StageLastMile

	case strings.Contains(combined, "será recogido") ||
		strings.Contains(combined, "sera recogido") ||
		strings.Contains(combined, "listo para ser recogido") ||
		strings.Contains(combined, "preparación") ||
		strings.Contains(combined, "preparacion") ||
		strings.Contains(combined, "registrado informáticamente") ||
		strings.Contains(combined, "registrado informaticamente") ||
		strings.Contains(combined, "info received") ||
		strings.Contains(combined, "cre.") ||
		strings.Contains(combined, "created"):
		return models.StatusInfoReceived, models.StageCreated

	case strings.Contains(combined, "tránsito") ||
		strings.Contains(combined, "transito") ||
		strings.Contains(combined, "in_transit") ||
		strings.Contains(combined, "en ruta") ||
		strings.Contains(combined, "en camino") ||
		strings.Contains(combined, "clasificación") ||
		strings.Contains(combined, "clasificacion") ||
		strings.Contains(combined, "hub") ||
		strings.Contains(combined, "depot") ||
		strings.Contains(combined, "sorting") ||
		strings.Contains(combined, "recogido por el mensajero") ||
		strings.Contains(combined, "recogido por el repartidor") ||
		strings.Contains(combined, "centro logístico") ||
		strings.Contains(combined, "centro logistico"):
		return models.StatusInTransit, models.StageTransit

	case strings.Contains(combined, "incidencia") ||
		strings.Contains(combined, "fallido") ||
		strings.Contains(combined, "exception") ||
		strings.Contains(combined, "devuelto") ||
		strings.Contains(combined, "error"):
		return models.StatusException, models.StageException

	case strings.Contains(combined, "preparación") ||
		strings.Contains(combined, "preparacion") ||
		strings.Contains(combined, "at_the_origin") ||
		strings.Contains(combined, "registrado informáticamente") ||
		strings.Contains(combined, "registrado informaticamente") ||
		strings.Contains(combined, "será recogido") ||
		strings.Contains(combined, "sera recogido") ||
		strings.Contains(combined, "listo para ser recogido") ||
		strings.Contains(combined, "info received") ||
		strings.Contains(combined, "cre.") ||
		strings.Contains(combined, "created"):
		return models.StatusInfoReceived, models.StageCreated

	default:
		return models.StatusInfoReceived, models.StageCreated
	}
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
