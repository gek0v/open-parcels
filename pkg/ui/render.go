package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"open-parcels/pkg/models"
)

// ANSI Color Codes
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Italic  = "\033[3m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	BgGreen = "\033[42m"
	BgCyan  = "\033[46m"
	BgBlue  = "\033[44m"
	BgRed   = "\033[41m"
)

// RenderShipment writes a formatted visual tracking card to out.
func RenderShipment(out io.Writer, s *models.Shipment) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s%s📦 OPEN PARCELS %s%s - Universal Parcel Tracking Engine%s\n", Bold, Cyan, Reset, Dim, Reset)
	fmt.Fprintln(out, strings.Repeat("─", 68))

	// Tracking Number & Status Badge
	statusColor, statusIcon := getStatusStyle(s.Status)
	fmt.Fprintf(out, " %sTracking Number:%s %s%s%s   %s%s %s %s%s\n",
		Dim, Reset, Bold, s.TrackingNumber, Reset,
		statusColor, Bold, statusIcon, strings.ToUpper(s.StatusText), Reset)

	// Carriers
	carrierStr := fmt.Sprintf("%s (%s)", s.Carrier.Name, s.Carrier.Slug)
	if len(s.AdditionalCarriers) > 0 {
		var adds []string
		for _, ac := range s.AdditionalCarriers {
			adds = append(adds, fmt.Sprintf("%s (%s)", ac.Name, ac.Slug))
		}
		carrierStr += fmt.Sprintf(" ➔ Last-Mile: %s", strings.Join(adds, ", "))
	}
	fmt.Fprintf(out, " %sCarriers:%s        %s%s%s\n", Dim, Reset, Cyan, carrierStr, Reset)

	// Route
	origin := s.Origin.Name
	if origin == "" {
		origin = "Unknown"
	}
	dest := s.Destination.Name
	if dest == "" {
		dest = "Unknown"
	}
	fmt.Fprintf(out, " %sJourney:%s         %s%s%s ➔ %s%s%s\n",
		Dim, Reset, Yellow, origin, Reset, Green, dest, Reset)

	// Progress Bar
	progressBar := renderProgressBar(s.Progress, 26)
	fmt.Fprintf(out, " %sProgress:%s        %s %s%d%%%s (In Transit: %s%d days%s)\n",
		Dim, Reset, progressBar, Bold, s.Progress, Reset, Bold, s.DaysInTransit, Reset)

	if s.LastMileTrackingNumber != "" && s.LastMileTrackingNumber != s.TrackingNumber {
		fmt.Fprintf(out, " %sLast-Mile Ref:%s   %s%s%s\n", Dim, Reset, Magenta, s.LastMileTrackingNumber, Reset)
	}

	fmt.Fprintln(out, strings.Repeat("─", 68))
	fmt.Fprintf(out, " %s%sTIMELINE & MILESTONES%s\n", Bold, White, Reset)
	fmt.Fprintln(out, strings.Repeat("─", 68))

	if len(s.Events) == 0 {
		fmt.Fprintf(out, "   %sNo tracking events recorded yet.%s\n", Dim, Reset)
	} else {
		for i, ev := range s.Events {
			dateStr := ev.Date.Format("2006-01-02 15:04")
			evIcon := getEventIcon(ev)
			isLatest := i == 0

			prefix := "├─"
			if i == len(s.Events)-1 {
				prefix = "└─"
			}

			carrierTag := fmt.Sprintf("[%s]", ev.CarrierName)
			if ev.CarrierIndex > 0 {
				carrierTag = fmt.Sprintf("%s[%s]%s", Magenta, ev.CarrierName, Reset)
			} else {
				carrierTag = fmt.Sprintf("%s[%s]%s", Cyan, ev.CarrierName, Reset)
			}

			if isLatest {
				fmt.Fprintf(out, " %s %s %s%s%s  %s  %s%s%s\n",
					prefix, evIcon, Bold, dateStr, Reset, carrierTag, Bold, ev.Description, Reset)
			} else {
				fmt.Fprintf(out, " %s %s %s%s%s  %s  %s\n",
					prefix, evIcon, Dim, dateStr, Reset, carrierTag, ev.Description)
			}

			if ev.Location != "" {
				locPrefix := "│   "
				if i == len(s.Events)-1 {
					locPrefix = "    "
				}
				fmt.Fprintf(out, " %s     %s📍 %s%s\n", locPrefix, Dim, ev.Location, Reset)
			}
		}
	}

	// Flight legs if available
	if len(s.FlightLegs) > 0 {
		fmt.Fprintln(out, strings.Repeat("─", 68))
		fmt.Fprintf(out, " %s%sFLIGHT LEGS & AIR CARGO SCHEDULE%s\n", Bold, White, Reset)
		fmt.Fprintln(out, strings.Repeat("─", 68))
		for _, leg := range s.FlightLegs {
			fmt.Fprintf(out, "  ✈ Flight %s%s%s: %s ➔ %s (%s)\n",
				Bold, leg.FlightNumber, Reset, leg.Origin, leg.Destination, leg.Status)
			if leg.ActualDeparture != "" {
				fmt.Fprintf(out, "    %sActual Departure: %s%s\n", Dim, leg.ActualDeparture, Reset)
			}
		}
	}

	fmt.Fprintln(out, strings.Repeat("─", 68))
	fmt.Fprintf(out, " %sLast Updated: %s%s\n", Dim, s.UpdatedAt.Format("2006-01-02 15:04:05 MST"), Reset)
	fmt.Fprintln(out)
}

func getStatusStyle(s models.Status) (string, string) {
	switch s {
	case models.StatusDelivered:
		return Green, "✓"
	case models.StatusOutForDelivery:
		return Cyan, "🚚"
	case models.StatusInTransit:
		return Blue, "✈"
	case models.StatusInfoReceived:
		return Yellow, "📦"
	case models.StatusFailedAttempt:
		return Red, "⚠"
	case models.StatusException:
		return Red, "✖"
	default:
		return Gray, "⏳"
	}
}

func getEventIcon(ev models.TrackingEvent) string {
	switch ev.Status {
	case models.StatusDelivered:
		return Green + "✓" + Reset
	case models.StatusOutForDelivery:
		return Cyan + "🚚" + Reset
	case models.StatusFailedAttempt:
		return Red + "⚠" + Reset
	case models.StatusException:
		return Red + "✖" + Reset
	case models.StatusInfoReceived:
		return Yellow + "📋" + Reset
	}

	switch ev.Stage {
	case models.StageCustoms:
		return Magenta + "🛃" + Reset
	case models.StageHandover:
		return Cyan + "🔄" + Reset
	case models.StageTransit:
		return Blue + "✈" + Reset
	case models.StagePickedUp:
		return Yellow + "📦" + Reset
	default:
		return Gray + "•" + Reset
	}
}

func renderProgressBar(pct, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := (pct * width) / 100
	empty := width - filled

	bar := Green + strings.Repeat("█", filled) + Reset + Gray + strings.Repeat("░", empty) + Reset
	return "[" + bar + "]"
}

// RenderJSON writes raw JSON to stdout.
func RenderJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// RenderDetections formats candidate carrier detections.
func RenderDetections(out io.Writer, num string, candidates []models.DetectResult) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s%s🔍 CARRIER AUTO-DETECTION RESULTS%s\n", Bold, Cyan, Reset)
	fmt.Fprintf(out, " %sTracking Number:%s %s%s%s\n", Dim, Reset, Bold, num, Reset)
	fmt.Fprintln(out, strings.Repeat("─", 68))

	if len(candidates) == 0 {
		fmt.Fprintf(out, " %sNo recognized carrier patterns matched.%s\n\n", Yellow, Reset)
		return
	}

	for i, c := range candidates {
		pct := int(c.Confidence * 100)
		color := Green
		if pct < 85 {
			color = Yellow
		}
		if pct < 70 {
			color = Gray
		}

		fmt.Fprintf(out, " #%d  %s%3d%%%s  %s%-22s%s  %s%-16s%s  %s(%s)%s\n",
			i+1, color, pct, Reset,
			Bold, c.Carrier.Name, Reset,
			Cyan, c.Carrier.Slug, Reset,
			Dim, c.MatchedPattern, Reset)
	}
	fmt.Fprintln(out, strings.Repeat("─", 68))
	fmt.Fprintln(out)
}

// RenderErrorCard writes a visually polished error card in ANSI mode to out.
func RenderErrorCard(out io.Writer, code string, err error, suggestion string) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s%s✖ TRACKING ERROR [%s]%s\n", Bold, Red, code, Reset)
	fmt.Fprintln(out, strings.Repeat("─", 68))
	fmt.Fprintf(out, " %sMessage:%s    %s%s%s\n", Dim, Reset, Bold, err.Error(), Reset)
	if suggestion != "" {
		fmt.Fprintf(out, " %sSuggestion:%s %s💡 %s%s\n", Dim, Reset, Yellow, suggestion, Reset)
	}
	fmt.Fprintln(out, strings.Repeat("─", 68))
	fmt.Fprintln(out)
}
func RenderCarriersTable(out io.Writer, list []models.CarrierInfo) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s%s📋 SUPPORTED CARRIERS CATALOG (%d)%s\n", Bold, Cyan, len(list), Reset)
	fmt.Fprintln(out, strings.Repeat("─", 68))
	fmt.Fprintf(out, " %s%-20s %-22s %-8s %-12s%s\n", Bold, "SLUG", "NAME", "COUNTRY", "TYPE", Reset)
	fmt.Fprintln(out, strings.Repeat("─", 68))

	for _, c := range list {
		country := c.Country
		if country == "" {
			country = "GLOBAL"
		}
		fmt.Fprintf(out, " %-20s %-22s %-8s %-12s\n", c.Slug, c.Name, country, string(c.Type))
	}
	fmt.Fprintln(out, strings.Repeat("─", 68))
	fmt.Fprintln(out)
}
