package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"open-parcels/pkg/engine"
	"open-parcels/pkg/models"
	"open-parcels/pkg/ui"
)

// Version represents the build version, modifiable via ldflags during release compilation.
var Version = "v1.0.2"

// Execute runs the CLI application parsing os.Args.
func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "track":
		runTrack(args)
	case "detect":
		runDetect(args)
	case "carriers":
		runCarriers(args)
	case "serve", "server":
		runServe(args)
	case "mcp":
		runMCP(args)
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Printf("open-parcels %s (Go engine & CLI)\n", Version)
	default:
		// If the argument looks like a tracking number directly, track it!
		if !strings.HasPrefix(command, "-") {
			runTrack(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`%sOpen Parcels (open-parcels)%s - Universal Parcel Tracking Engine & CLI

Usage:
  open-parcels <command> [arguments]
  open-parcels <tracking_number> [--carrier <slug>] [--lang es|en]

Commands:
  track <number>      Track package with multi-carrier journey chaining & merging
  detect <number>     Auto-detect candidate carriers with confidence scoring
  carriers            List all supported carrier adapters and catalogs
  serve               Start HTTP REST API daemon & MCP server endpoint
  mcp                 Start Model Context Protocol (MCP) server over stdio

Flags for 'track':
  --carrier, -c       Explicit carrier slug hint (e.g. cainiao, correos-express, ups, usps)
  --lang, -l          Language for status & descriptions ('en' or 'es', default: en)
  --country           Destination country hint ISO-2 (e.g. ES, US, DE)
  --postal-code, -z   Destination postal code (required by carriers such as InPost)
  --mock, --allow-simulated Allow simulated/mock fallback tracking data for dev/testing
  --json              Output raw JSON response instead of visual terminal UI

Examples:
  open-parcels track LP00561234567890 --lang es
  open-parcels track 9400111206213785678901 --json
  open-parcels detect 6900123456789012
  open-parcels carriers --query express
  open-parcels serve --port 8080
  open-parcels mcp
`, ui.Bold+ui.Cyan, ui.Reset)
}

func runTrack(args []string) {
	fs := flag.NewFlagSet("track", flag.ExitOnError)
	carrier := fs.String("carrier", "", "Explicit carrier slug hint")
	fs.StringVar(carrier, "c", "", "Explicit carrier slug hint (shorthand)")
	lang := fs.String("lang", "en", "Language for status text (en, es)")
	fs.StringVar(lang, "l", "en", "Language for status text (shorthand)")
	country := fs.String("country", "", "Destination country code (ISO-2)")
	postalCode := fs.String("postal-code", "", "Destination postal code")
	fs.StringVar(postalCode, "z", "", "Destination postal code (shorthand)")
	mock := fs.Bool("mock", false, "Allow simulated tracking data")
	fs.BoolVar(mock, "allow-simulated", false, "Allow simulated tracking data (alias)")
	jsonOutput := fs.Bool("json", false, "Output JSON format")

	// Normalize arguments so flags can appear before or after the tracking number
	parsedArgs, trackingNumber := normalizeArgsWithPositional(args)

	if err := fs.Parse(parsedArgs); err != nil {
		os.Exit(1)
	}

	if trackingNumber == "" && len(fs.Args()) > 0 {
		trackingNumber = fs.Args()[0]
	}

	if trackingNumber == "" {
		if *jsonOutput {
			_ = ui.RenderJSON(os.Stdout, map[string]string{
				"error": "tracking number is required",
				"code":  "MISSING_NUMBER",
			})
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Error: tracking number is required. Usage: open-parcels track <tracking_number> [--carrier <slug>] [--postal-code <CP>] [--mock] [--json]")
		os.Exit(1)
	}

	eng := engine.New()
	shipment, err := eng.Track(context.Background(), trackingNumber, models.TrackOptions{
		CarrierHint:        *carrier,
		Language:           *lang,
		DestinationCountry: *country,
		PostalCode:         *postalCode,
		AllowSimulated:     *mock,
	})

	if err != nil {
		code := "NOT_FOUND"
		suggestion := "Check the tracking number or pass --mock to allow simulated test data."
		if errors.Is(err, models.ErrMissingPostalCode) || strings.Contains(strings.ToLower(err.Error()), "postal code") {
			code = "MISSING_POSTAL_CODE"
			suggestion = "This carrier requires a destination postal code. Pass --postal-code <CP> (or -z <CP>)."
		} else if errors.Is(err, models.ErrCarrierNotImplemented) || strings.Contains(strings.ToLower(err.Error()), "not implemented") {
			code = "NOT_IMPLEMENTED"
			suggestion = "Live API tracking is not implemented for this carrier yet. Use --mock for testing."
		} else if errors.Is(err, models.ErrShipmentNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			code = "NOT_FOUND"
			suggestion = "Shipment not found by carrier. Ensure the tracking number is correct or use --mock for testing."
		}

		if *jsonOutput {
			_ = ui.RenderJSON(os.Stdout, map[string]string{
				"error": err.Error(),
				"code":  code,
			})
			os.Exit(1)
		}

		ui.RenderErrorCard(os.Stderr, code, err, suggestion)
		os.Exit(1)
	}

	if *jsonOutput {
		if err := ui.RenderJSON(os.Stdout, shipment); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ui.RenderShipment(os.Stdout, shipment)
}

func runDetect(args []string) {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	country := fs.String("country", "", "Destination country hint")
	jsonOutput := fs.Bool("json", false, "Output JSON format")

	parsedArgs, trackingNumber := normalizeArgsWithPositional(args)

	if err := fs.Parse(parsedArgs); err != nil {
		os.Exit(1)
	}

	if trackingNumber == "" && len(fs.Args()) > 0 {
		trackingNumber = fs.Args()[0]
	}

	if trackingNumber == "" {
		fmt.Fprintln(os.Stderr, "Error: tracking number is required. Usage: open-parcels detect <tracking_number>")
		os.Exit(1)
	}

	eng := engine.New()
	candidates := eng.Detector().Detect(trackingNumber, *country)

	if *jsonOutput {
		if err := ui.RenderJSON(os.Stdout, candidates); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ui.RenderDetections(os.Stdout, trackingNumber, candidates)
}

func runCarriers(args []string) {
	fs := flag.NewFlagSet("carriers", flag.ExitOnError)
	query := fs.String("query", "", "Filter carriers by query")
	fs.StringVar(query, "q", "", "Filter carriers by query (shorthand)")
	jsonOutput := fs.Bool("json", false, "Output JSON format")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	eng := engine.New()
	list := eng.Registry().List(*query)

	if *jsonOutput {
		if err := ui.RenderJSON(os.Stdout, list); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ui.RenderCarriersTable(os.Stdout, list)
}

// normalizeArgsWithPositional separates positional non-flag arguments from flags.
func normalizeArgsWithPositional(args []string) ([]string, string) {
	var flags []string
	var positional string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if flag takes an argument
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Flags that take values
				flagName := strings.TrimLeft(arg, "-")
				if flagName == "c" || flagName == "carrier" || flagName == "l" || flagName == "lang" ||
					flagName == "country" || flagName == "p" || flagName == "port" || flagName == "q" || flagName == "query" ||
					flagName == "z" || flagName == "postal-code" {
					i++
					flags = append(flags, args[i])
				}
			}
		} else if positional == "" {
			positional = arg
		} else {
			flags = append(flags, arg)
		}
	}

	return flags, positional
}
