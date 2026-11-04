// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package detector

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"open-parcels/pkg/models"
)

// PatternRule defines a single detection rule for a carrier.
type PatternRule struct {
	CarrierSlug    string
	CarrierName    string
	Country        string
	CountryName    string
	CarrierType    models.CarrierType
	Priority       int     // higher priority evaluated first when matching multiple
	ConfidenceBase float64 // default confidence for regex match
	Regex          *regexp.Regexp
	Validator      func(cleanNumber string) bool
	Description    string
}

// Detector is the carrier identification engine.
type Detector struct {
	rules []PatternRule
}

// New creates a new initialized Detector with all carrier rules.
func New() *Detector {
	d := &Detector{}
	d.initRules()
	return d
}

// Detect analyzes a tracking number and returns candidate carrier matches ranked by confidence.
func (d *Detector) Detect(rawNumber string, destCountryHint string) []models.DetectResult {
	clean := CleanTrackingNumber(rawNumber)
	if clean == "" {
		return nil
	}

	destUpper := strings.ToUpper(strings.TrimSpace(destCountryHint))
	var candidates []models.DetectResult
	seenCarrier := make(map[string]bool)

	for _, rule := range d.rules {
		if rule.Regex != nil && !rule.Regex.MatchString(clean) {
			continue
		}

		conf := rule.ConfidenceBase
		if rule.Validator != nil {
			if !rule.Validator(clean) {
				// If validator failed, don't completely discard if confidence is high, but penalize
				conf = maxFloat(0.3, conf-0.20)
			} else {
				conf = minFloat(1.0, conf+0.05)
			}
		}

		// Boost confidence if country matches destination hint
		if destUpper != "" && rule.Country != "" {
			if strings.EqualFold(rule.Country, destUpper) {
				conf = minFloat(1.0, conf+0.15)
			} else if rule.Country != "GLOBAL" && rule.Country != "CN" {
				// Slight penalty if carrier is country-specific and doesn't match hint
				conf = maxFloat(0.3, conf-0.10)
			}
		}

		// Avoid duplicate slugs in candidates
		if seenCarrier[rule.CarrierSlug] {
			continue
		}
		seenCarrier[rule.CarrierSlug] = true

		candidates = append(candidates, models.DetectResult{
			Carrier: models.CarrierInfo{
				Slug:        rule.CarrierSlug,
				Name:        rule.CarrierName,
				Country:     rule.Country,
				CountryName: rule.CountryName,
				Type:        rule.CarrierType,
			},
			Confidence:     conf,
			MatchedPattern: rule.Description,
			Reason:         "Matched pattern: " + rule.Description,
		})
	}

	// Check BIC container prefix mapping to propose specific ocean carrier candidate
	if len(clean) == 11 && (clean[3] == 'U' || clean[3] == 'J' || clean[3] == 'Z') {
		if slug, ok := LookupBIC(clean); ok && !seenCarrier[slug] {
			info, hasInfo := BICCarrierInfoMap[slug]
			if !hasInfo {
				info = models.CarrierInfo{
					Slug:        slug,
					Name:        slug,
					Country:     "GLOBAL",
					CountryName: "International",
					Type:        models.CarrierTypeOcean,
				}
			}
			conf := 0.95
			if validateISO6346Checksum(clean) {
				conf = 0.96
			}
			candidates = append(candidates, models.DetectResult{
				Carrier:        info,
				Confidence:     conf,
				MatchedPattern: "ISO 6346 BIC prefix " + clean[:4],
				Reason:         "Matched BIC prefix: " + clean[:4],
			})
			seenCarrier[slug] = true
		}
	}

	// Sort candidates by confidence descending
	sortCandidates(candidates)

	return candidates
}

// BestMatch returns the single highest confidence carrier match or nil.
func (d *Detector) BestMatch(rawNumber string, destCountryHint string) *models.DetectResult {
	matches := d.Detect(rawNumber, destCountryHint)
	if len(matches) == 0 {
		return nil
	}
	return &matches[0]
}

// CleanTrackingNumber removes spaces, dashes (unless AWB), and non-printable characters.
func CleanTrackingNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	// Check if it's an Air Waybill pattern: 3 digits, dash, 8 digits
	awbRegex := regexp.MustCompile(`^\d{3}-\d{8}$`)
	if awbRegex.MatchString(raw) {
		return raw
	}

	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(unicode.ToUpper(r))
		}
	}
	return sb.String()
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func sortCandidates(arr []models.DetectResult) {
	for i := 0; i < len(arr)-1; i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[j].Confidence > arr[i].Confidence {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
}

// ValidateISO6346Checksum calculates the ISO 6346 Modulo 11 check digit
// (omitting multiples of 11: 11, 22, 33) and verifies if it matches the 11th digit.
func ValidateISO6346Checksum(cleanNumber string) bool {
	return validateISO6346Checksum(cleanNumber)
}

func validateISO6346Checksum(clean string) bool {
	clean = strings.ToUpper(strings.TrimSpace(clean))
	if len(clean) != 11 {
		return false
	}
	// Chars 0-2: owner code (3 letters)
	for i := 0; i < 3; i++ {
		if clean[i] < 'A' || clean[i] > 'Z' {
			return false
		}
	}
	// Char 3: category identifier (U for freight containers, J for detachable equipment, Z for trailers)
	if clean[3] != 'U' && clean[3] != 'J' && clean[3] != 'Z' {
		return false
	}
	// Chars 4-9: serial number (6 digits)
	for i := 4; i < 10; i++ {
		if clean[i] < '0' || clean[i] > '9' {
			return false
		}
	}
	// Char 10: check digit (1 digit)
	if clean[10] < '0' || clean[10] > '9' {
		return false
	}

	sum := 0
	weights := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512}
	for i := 0; i < 4; i++ {
		sum += isoCharValue(clean[i]) * weights[i]
	}
	for i := 4; i < 10; i++ {
		sum += int(clean[i]-'0') * weights[i]
	}

	// Remainder modulo 11. If 10, the check digit is 0.
	check := (sum % 11) % 10
	expected := int(clean[10] - '0')
	return check == expected
}

// isoCharValue returns the ISO 6346 equivalent numerical value for letter A-Z,
// omitting multiples of 11 (11, 22, 33).
func isoCharValue(ch byte) int {
	if ch < 'A' || ch > 'Z' {
		return 0
	}
	val := int(ch - 'A') + 10
	if val >= 11 {
		val++
	}
	if val >= 22 {
		val++
	}
	if val >= 33 {
		val++
	}
	return val
}

// BICPrefixCarrierMap maps 4-letter container prefix codes to carrier slugs.
var BICPrefixCarrierMap = map[string]string{
	"MSKU": "maersk-line",
	"MAEU": "maersk-line",
	"MRKU": "maersk-line",
	"PONU": "maersk-line",
	"MSCU": "msc",
	"MEDU": "msc",
	"CMAU": "cma-cgm",
	"CGMU": "cma-cgm",
	"ANLU": "cma-cgm",
	"COSU": "cosco",
	"CBHU": "cosco",
	"CCLU": "cosco",
	"HLCU": "hapag-lloyd",
	"HLXU": "hapag-lloyd",
	"CSQU": "hapag-lloyd",
	"EGLV": "evergreen",
	"EMCU": "evergreen",
	"EISU": "evergreen",
	"EGSU": "evergreen",
	"UGXU": "evergreen",
	"ONEU": "one-line",
	"NYKU": "one-line",
	"MOLU": "one-line",
	"KKFU": "one-line",
	"KLXU": "one-line",
	"YMLU": "yang-ming",
	"HDMU": "hmm",
	"ZIMU": "zim",
	"DOLU": "dole-ocean-cargo",
}

// BICCarrierInfoMap provides full carrier info for ocean carriers.
var BICCarrierInfoMap = map[string]models.CarrierInfo{
	"maersk-line": {
		Slug:        "maersk-line",
		Name:        "Maersk Line",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.maersk.com",
	},
	"msc": {
		Slug:        "msc",
		Name:        "MSC",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.msc.com",
	},
	"cma-cgm": {
		Slug:        "cma-cgm",
		Name:        "CMA CGM",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.cma-cgm.com",
	},
	"cosco": {
		Slug:        "cosco",
		Name:        "COSCO Shipping",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://lines.coscoshipping.com",
	},
	"hapag-lloyd": {
		Slug:        "hapag-lloyd",
		Name:        "Hapag Lloyd",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.hapag-lloyd.com",
	},
	"evergreen": {
		Slug:        "evergreen",
		Name:        "Evergreen",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.evergreen-line.com",
	},
	"one-line": {
		Slug:        "one-line",
		Name:        "ONE Ocean Network Express",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.one-line.com",
	},
	"yang-ming": {
		Slug:        "yang-ming",
		Name:        "Yang Ming",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.yangming.com",
	},
	"hmm": {
		Slug:        "hmm",
		Name:        "HMM",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.hmm21.com",
	},
	"zim": {
		Slug:        "zim",
		Name:        "ZIM",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.zim.com",
	},
	"dole-ocean-cargo": {
		Slug:        "dole-ocean-cargo",
		Name:        "Dole Ocean Cargo",
		Country:     "GLOBAL",
		CountryName: "International",
		Type:        models.CarrierTypeOcean,
		Website:     "https://www.doleocean.com",
	},
}

// LookupBICPrefix returns the ocean carrier slug for a given 4-letter BIC prefix.
func LookupBICPrefix(prefix string) (string, bool) {
	slug, ok := BICPrefixCarrierMap[strings.ToUpper(strings.TrimSpace(prefix))]
	return slug, ok
}

// LookupBIC returns the ocean carrier slug for a container tracking number or prefix.
func LookupBIC(containerOrPrefix string) (string, bool) {
	clean := strings.ToUpper(strings.TrimSpace(containerOrPrefix))
	if len(clean) >= 4 {
		return LookupBICPrefix(clean[:4])
	}
	return "", false
}

// Checksum validation helpers
func validateUPS(trackingNumber string) bool {
	// Standard UPS: 1Z + 6 chars (account) + 2 chars (service code) + 7 chars (package) + 1 digit (check)
	if len(trackingNumber) != 18 || !strings.HasPrefix(trackingNumber, "1Z") {
		return false
	}
	// Verify check digit using UPS alphanumeric mapping
	payload := trackingNumber[2:17]
	checkDigit, err := strconv.Atoi(string(trackingNumber[17]))
	if err != nil {
		return false
	}

	total := 0
	for i, r := range payload {
		var val int
		if unicode.IsDigit(r) {
			val = int(r - '0')
		} else if unicode.IsUpper(r) {
			val = (int(r-'A') + 2) % 10
		} else {
			return false
		}
		if i%2 == 1 {
			val *= 2
		}
		total += val
	}
	calcCheck := (10 - (total % 10)) % 10
	return calcCheck == checkDigit
}

func validateAWB(raw string) bool {
	// 3 digits + dash + 7 serial digits + 1 check digit (Mod 7)
	parts := strings.Split(raw, "-")
	if len(parts) != 2 || len(parts[0]) != 3 || len(parts[1]) != 8 {
		return false
	}
	serial, err1 := strconv.Atoi(parts[1][:7])
	check, err2 := strconv.Atoi(string(parts[1][7]))
	if err1 != nil || err2 != nil {
		return false
	}
	return (serial % 7) == check
}

func validateContainer(clean string) bool {
	return validateISO6346Checksum(clean)
}

func validateUPUS10(clean string) bool {
	// 2 letters + 8 digits + 1 check digit + 2 country letters
	if len(clean) != 13 {
		return false
	}
	weights := []int{8, 6, 4, 2, 3, 5, 9, 7}
	sum := 0
	for i := 0; i < 8; i++ {
		digit := int(clean[2+i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		sum += digit * weights[i]
	}
	rem := sum % 11
	check := 11 - rem
	if check == 10 {
		check = 0
	} else if check == 11 {
		check = 5
	}
	expectedCheck := int(clean[10] - '0')
	return check == expectedCheck
}
