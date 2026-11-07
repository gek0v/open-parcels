package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"open-parcels/pkg/engine"
	"open-parcels/pkg/models"
)

// Server implements the Model Context Protocol (MCP) server.
type Server struct {
	engine *engine.Engine
	tools  []Tool
}

// NewServer creates a new MCP server.
func NewServer(eng *engine.Engine) *Server {
	s := &Server{
		engine: eng,
	}
	s.initTools()
	return s
}

func (s *Server) initTools() {
	s.tools = []Tool{
		{
			Name: "track_shipment",
			Description: "Track a shipment/parcel with automatic multi-carrier detection, international-to-local handover chaining (e.g. Cainiao/AliExpress to Correos Express), and unified chronological event merging.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"tracking_number": {
						Type:        "string",
						Description: "The parcel tracking number or barcode (e.g. Cainiao LP..., Correos Express, UPS, DHL, FedEx, USPS, YunExpress, etc.)",
					},
					"carrier_hint": {
						Type:        "string",
						Description: "Optional carrier slug hint (e.g. cainiao, correos-express, correos-spain, ups, usps, dhl-express, fedex, seur, ctt-express)",
					},
					"destination_country": {
						Type:        "string",
						Description: "Optional destination country code ISO-2 (e.g. ES, US, DE, FR, GB)",
					},
					"language": {
						Type:        "string",
						Description: "Language for status text and descriptions ('en' or 'es', default: 'en')",
					},
					"postal_code": {
						Type:        "string",
						Description: "Destination postal code (required by carriers such as InPost)",
					},
					"allow_simulated": {
						Type:        "boolean",
						Description: "Allow simulated/mock fallback tracking data for development or testing",
					},
				},
				Required: []string{"tracking_number"},
			},
		},
		{
			Name: "detect_carrier",
			Description: "Identify candidate delivery carriers for a tracking number with confidence scoring and regex/format validation.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"tracking_number": {
						Type:        "string",
						Description: "The parcel tracking number to analyze",
					},
					"destination_country": {
						Type:        "string",
						Description: "Optional destination country code hint (e.g. ES, US)",
					},
				},
				Required: []string{"tracking_number"},
			},
		},
		{
			Name: "list_carriers",
			Description: "List all supported logistics carriers, tracking slugs, countries, and service types.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "Optional query filter (carrier name, slug, or country)",
					},
				},
			},
		},
		{
			Name: "estimate_delivery",
			Description: "Estimate shipment delivery progress, transit time, and status based on tracked milestones.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"tracking_number": {
						Type:        "string",
						Description: "Tracking number to estimate",
					},
					"allow_simulated": {
						Type:        "boolean",
						Description: "Allow simulated/mock fallback tracking data for development or testing",
					},
				},
				Required: []string{"tracking_number"},
			},
		},
	}
}

// RunStdio starts the MCP server over standard input/output.
func (s *Server) RunStdio() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = []byte(strings.TrimSpace(string(line)))
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &JSONRPCError{
					Code:    -32700,
					Message: "Parse error",
				},
			}
			sendStdioResponse(resp)
			continue
		}

		resp := s.handleRequest(context.Background(), req)
		if resp != nil {
			sendStdioResponse(*resp)
		}
	}
}

func sendStdioResponse(resp JSONRPCResponse) {
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(os.Stdout, "%s\n", string(bytes))
}

// HandleHTTP handles JSON-RPC 2.0 requests over HTTP.
func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		})
		return
	}

	resp := s.handleRequest(r.Context(), req)
	w.Header().Set("Content-Type", "application/json")
	if resp == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleRequest(ctx context.Context, req JSONRPCRequest) *JSONRPCResponse {
	// Notifications (no ID)
	if req.ID == nil && req.Method == "notifications/initialized" {
		return nil
	}

	res := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		res.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "open-parcels",
				"version": "1.0.0",
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"instructions": "Open Parcels Universal Multi-Carrier Tracking Engine. Use track_shipment to track parcels, detect_carrier to identify carriers, and list_carriers to explore supported couriers.",
		}

	case "ping":
		res.Result = map[string]any{}

	case "tools/list":
		res.Result = map[string]any{
			"tools": s.tools,
		}

	case "tools/call":
		toolName, _ := req.Params["name"].(string)
		args, _ := req.Params["arguments"].(map[string]any)

		callResult := s.executeTool(ctx, toolName, args)
		res.Result = callResult

	default:
		res.Error = &JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method '%s' not found", req.Method),
		}
	}

	return res
}

func (s *Server) executeTool(ctx context.Context, name string, args map[string]any) ToolCallResult {
	switch name {
	case "track_shipment":
		trackingNum, _ := args["tracking_number"].(string)
		if strings.TrimSpace(trackingNum) == "" {
			return errorResult("tracking_number argument is required")
		}

		carrierHint, _ := args["carrier_hint"].(string)
		destCountry, _ := args["destination_country"].(string)
		postalCode, _ := args["postal_code"].(string)
		allowSimulated, _ := args["allow_simulated"].(bool)
		lang, _ := args["language"].(string)
		if lang == "" {
			lang = "en"
		}

		shipment, err := s.engine.Track(ctx, trackingNum, models.TrackOptions{
			CarrierHint:        carrierHint,
			DestinationCountry: destCountry,
			PostalCode:         postalCode,
			AllowSimulated:     allowSimulated,
			Language:           lang,
		})
		if err != nil {
			return errorResult(fmt.Sprintf("Failed to track package: %v", err))
		}

		data, err := json.MarshalIndent(shipment, "", "  ")
		if err != nil {
			return errorResult("Failed to serialize tracking result")
		}

		return ToolCallResult{
			Content: []ToolCallContent{
				{
					Type: "text",
					Text: string(data),
				},
			},
		}

	case "detect_carrier":
		trackingNum, _ := args["tracking_number"].(string)
		if strings.TrimSpace(trackingNum) == "" {
			return errorResult("tracking_number argument is required")
		}

		destCountry, _ := args["destination_country"].(string)
		candidates := s.engine.Detector().Detect(trackingNum, destCountry)

		data, err := json.MarshalIndent(candidates, "", "  ")
		if err != nil {
			return errorResult("Failed to serialize detection result")
		}

		return ToolCallResult{
			Content: []ToolCallContent{
				{
					Type: "text",
					Text: string(data),
				},
			},
		}

	case "list_carriers":
		query, _ := args["query"].(string)
		list := s.engine.Registry().List(query)

		data, err := json.MarshalIndent(list, "", "  ")
		if err != nil {
			return errorResult("Failed to serialize carriers catalog")
		}

		return ToolCallResult{
			Content: []ToolCallContent{
				{
					Type: "text",
					Text: string(data),
				},
			},
		}

	case "estimate_delivery":
		trackingNum, _ := args["tracking_number"].(string)
		if strings.TrimSpace(trackingNum) == "" {
			return errorResult("tracking_number argument is required")
		}

		allowSimulated, _ := args["allow_simulated"].(bool)
		shipment, err := s.engine.Track(ctx, trackingNum, models.TrackOptions{AllowSimulated: allowSimulated})
		if err != nil {
			return errorResult(fmt.Sprintf("Failed to retrieve shipment for estimate: %v", err))
		}

		summary := map[string]any{
			"tracking_number": shipment.TrackingNumber,
			"carrier":         shipment.Carrier.Name,
			"status":          shipment.Status,
			"status_text":     shipment.StatusText,
			"progress":        fmt.Sprintf("%d%%", shipment.Progress),
			"days_in_transit": shipment.DaysInTransit,
			"latest_event":    nil,
		}
		if len(shipment.Events) > 0 {
			summary["latest_event"] = shipment.Events[0]
		}

		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return errorResult("Failed to serialize delivery estimate")
		}

		return ToolCallResult{
			Content: []ToolCallContent{
				{
					Type: "text",
					Text: string(data),
				},
			},
		}

	default:
		return errorResult(fmt.Sprintf("Unknown tool name '%s'", name))
	}
}

func errorResult(msg string) ToolCallResult {
	return ToolCallResult{
		IsError: true,
		Content: []ToolCallContent{
			{
				Type: "text",
				Text: msg,
			},
		},
	}
}
