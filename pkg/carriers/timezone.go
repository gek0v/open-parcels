package carriers

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

var countryToZone = map[string]string{
	"CN": "Asia/Shanghai",
	"ES": "Europe/Madrid",
	"IT": "Europe/Rome",
	"PT": "Europe/Lisbon",
	"FR": "Europe/Paris",
	"DE": "Europe/Berlin",
	"GB": "Europe/London",
	"UK": "Europe/London",
	"PL": "Europe/Warsaw",
	"NL": "Europe/Amsterdam",
	"BE": "Europe/Brussels",
	"US": "America/New_York",
	"JP": "Asia/Tokyo",
	"HK": "Asia/Hong_Kong",
	"TW": "Asia/Taipei",
	"KR": "Asia/Seoul",
	"SG": "Asia/Singapore",
	"AU": "Australia/Sydney",
	"BR": "America/Sao_Paulo",
	"MX": "America/Mexico_City",
}

var carrierToZone = map[string]string{
	"cainiao":         "Asia/Shanghai",
	"yanwen":          "Asia/Shanghai",
	"yunexpress":      "Asia/Shanghai",
	"sunyou":          "Asia/Shanghai",
	"fourpx":          "Asia/Shanghai",
	"four-px":         "Asia/Shanghai",
	"chinapost":       "Asia/Shanghai",
	"sfexpress":       "Asia/Shanghai",
	"deppon":          "Asia/Shanghai",
	"yto":             "Asia/Shanghai",
	"cne":             "Asia/Shanghai",
	"correos":         "Europe/Madrid",
	"correosexpress":  "Europe/Madrid",
	"correos-express": "Europe/Madrid",
	"correos-spain":   "Europe/Madrid",
	"seur":            "Europe/Madrid",
	"mrw":             "Europe/Madrid",
	"nacex":           "Europe/Madrid",
	"tipsa":           "Europe/Madrid",
	"zeleris":         "Europe/Madrid",
	"paack":           "Europe/Madrid",
	"envialia":        "Europe/Madrid",
	"redur":           "Europe/Madrid",
	"ontime":          "Europe/Madrid",
	"asigna":          "Europe/Madrid",
	"activos24":       "Europe/Madrid",
	"wepickup":        "Europe/Madrid",
	"mbe_es":          "Europe/Madrid",
	"asendia_es":      "Europe/Madrid",
	"dhl_parcel_es":   "Europe/Madrid",
	"gls_spain":       "Europe/Madrid",
	"inpost-es":       "Europe/Madrid",
	"inpost":          "Europe/Madrid",
	"brt":             "Europe/Rome",
	"poste_italiane":  "Europe/Rome",
	"gls_italy":       "Europe/Rome",
	"tnt_italy":       "Europe/Rome",
	"sailpost":        "Europe/Rome",
	"amazon_italy":    "Europe/Rome",
	"inpost_italy":    "Europe/Rome",
	"ctt":             "Europe/Lisbon",
	"ctt-express":     "Europe/Lisbon",
	"delnext":         "Europe/Lisbon",
	"portugal_post":   "Europe/Lisbon",
	"dpd_portugal":    "Europe/Lisbon",
	"royalmail":       "Europe/London",
	"colissimo":       "Europe/Paris",
	"usps":            "America/New_York",
	"fedex":           "America/New_York",
	"ups":             "America/New_York",
	"dhl":             "Europe/Berlin",
	"dhl-parcel":      "Europe/Berlin",
}

// CarrierZone returns the IANA timezone name associated with a carrier slug.
func CarrierZone(slug string) string {
	clean := strings.ToLower(strings.TrimSpace(slug))
	if z, ok := carrierToZone[clean]; ok {
		return z
	}
	return "UTC"
}

// ResolveCountryZone returns the IANA timezone name associated with a country code or name.
func ResolveCountryZone(country string) string {
	c := strings.ToUpper(strings.TrimSpace(country))
	if z, ok := countryToZone[c]; ok {
		return z
	}
	switch strings.ToLower(country) {
	case "china", "prc":
		return "Asia/Shanghai"
	case "spain", "españa", "espana":
		return "Europe/Madrid"
	case "italy", "italia":
		return "Europe/Rome"
	case "portugal":
		return "Europe/Lisbon"
	case "france", "francia":
		return "Europe/Paris"
	case "germany", "alemania", "deutschland":
		return "Europe/Berlin"
	case "united kingdom", "reino unido":
		return "Europe/London"
	case "poland", "polonia":
		return "Europe/Warsaw"
	case "netherlands", "holanda":
		return "Europe/Amsterdam"
	case "belgium", "bélgica", "belgica":
		return "Europe/Brussels"
	case "united states", "usa", "eeuu", "estados unidos":
		return "America/New_York"
	case "japan", "japón", "japon":
		return "Asia/Tokyo"
	}
	return "UTC"
}

// ResolveLocation maps a timezone name, country code, country name, or carrier slug to a *time.Location.
func ResolveLocation(zoneOrCountry string) *time.Location {
	clean := strings.TrimSpace(zoneOrCountry)
	if clean == "" || strings.EqualFold(clean, "UTC") || strings.EqualFold(clean, "GMT") {
		return time.UTC
	}

	// Known abbreviations
	switch strings.ToUpper(clean) {
	case "CST":
		clean = "Asia/Shanghai"
	case "CET", "CEST":
		clean = "Europe/Madrid"
	case "WET", "WEST":
		clean = "Europe/Lisbon"
	case "JST":
		clean = "Asia/Tokyo"
	case "BST":
		clean = "Europe/London"
	}

	// Try loading as timezone name directly
	if loc, err := time.LoadLocation(clean); err == nil {
		return loc
	}

	// Try resolving country
	if zone := ResolveCountryZone(clean); zone != "UTC" {
		if loc, err := time.LoadLocation(zone); err == nil {
			return loc
		}
	}

	// Try resolving carrier slug
	if zone := CarrierZone(clean); zone != "UTC" {
		if loc, err := time.LoadLocation(zone); err == nil {
			return loc
		}
	}

	// Try parsing explicit offset strings like GMT+8, UTC+1, +08:00
	if loc := parseOffsetLocation(clean); loc != nil {
		return loc
	}

	return time.UTC
}

func parseOffsetLocation(s string) *time.Location {
	upper := strings.ToUpper(strings.TrimSpace(s))
	upper = strings.TrimPrefix(upper, "GMT")
	upper = strings.TrimPrefix(upper, "UTC")
	upper = strings.TrimSpace(upper)
	if upper == "" {
		return nil
	}
	sign := 1
	if strings.HasPrefix(upper, "+") {
		upper = upper[1:]
	} else if strings.HasPrefix(upper, "-") {
		sign = -1
		upper = upper[1:]
	} else {
		return nil
	}
	var hours, mins int
	if strings.Contains(upper, ":") {
		if _, err := fmt.Sscanf(upper, "%d:%d", &hours, &mins); err != nil {
			return nil
		}
	} else if len(upper) == 4 {
		if _, err := fmt.Sscanf(upper, "%02d%02d", &hours, &mins); err != nil {
			return nil
		}
	} else {
		if _, err := fmt.Sscanf(upper, "%d", &hours); err != nil {
			return nil
		}
	}
	offsetSeconds := sign * (hours*3600 + mins*60)
	return time.FixedZone(strings.TrimSpace(s), offsetSeconds)
}

// ParseLocalTimeWithZone parses a date string. If the string contains an explicit timezone
// indicator (e.g. RFC3339 offset like "+08:00" or "Z"), it parses preserving the offset
// and converts to UTC. Otherwise, it interprets the timestamp in the specified location
// and converts the resulting instant to UTC.
func ParseLocalTimeWithZone(layout string, value string, zoneOrCountry string) (time.Time, error) {
	cleanVal := strings.TrimSpace(value)
	if cleanVal == "" {
		return time.Time{}, fmt.Errorf("empty date value")
	}

	loc := ResolveLocation(zoneOrCountry)

	// If layout includes timezone indicator, parse directly
	if strings.Contains(layout, "Z07") || strings.Contains(layout, "-07") || strings.Contains(layout, "MST") {
		if t, err := time.Parse(layout, cleanVal); err == nil {
			return t.UTC(), nil
		}
	}

	// Parse in specified location
	t, err := time.ParseInLocation(layout, cleanVal, loc)
	if err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, err
}

// CommonLayouts lists the standard date/time layouts found in parcel logistics.
var CommonLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04:05",
	"02/01/2006 15:04:05",
	"02/01/2006 15:04",
	"20060102150405",
	"20060102",
	"2006-01-02",
	"02/01/2006",
}

// ParseLocalTime attempts to parse a date string using common logistics formats
// in the context of the specified timezone or country, returning UTC.
func ParseLocalTime(value string, zoneOrCountry string, customLayouts ...string) (time.Time, error) {
	cleanVal := strings.TrimSpace(value)
	if cleanVal == "" {
		return time.Time{}, fmt.Errorf("empty date value")
	}

	layouts := CommonLayouts
	if len(customLayouts) > 0 {
		layouts = append(customLayouts, CommonLayouts...)
	}

	for _, layout := range layouts {
		if t, err := ParseLocalTimeWithZone(layout, cleanVal, zoneOrCountry); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date string %q in zone %q", value, zoneOrCountry)
}

// NormalizeEventTime converts a timestamp representing local clock time in zoneOrCountry
// into its exact UTC instant. If zoneOrCountry is UTC or empty, it returns t.UTC().
func NormalizeEventTime(t time.Time, zoneOrCountry string) time.Time {
	if t.IsZero() {
		return t
	}
	loc := ResolveLocation(zoneOrCountry)
	if loc == time.UTC {
		return t.UTC()
	}

	localTime := time.Date(
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
		loc,
	)
	return localTime.UTC()
}
