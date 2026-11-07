package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"open-parcels/pkg/carriers"
	"open-parcels/pkg/engine"
)

func TestMCPServer(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	eng := engine.New()
	srv := NewServer(eng)

	ctx := context.Background()

	// 1. Initialize
	initReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	initResp := srv.handleRequest(ctx, initReq)
	if initResp == nil || initResp.Error != nil {
		t.Fatalf("expected initialize success, got error: %v", initResp.Error)
	}

	// 2. Tools List
	listReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	listResp := srv.handleRequest(ctx, listReq)
	if listResp == nil || listResp.Error != nil {
		t.Fatalf("expected tools/list success, got: %v", listResp.Error)
	}
	resMap := listResp.Result.(map[string]any)
	tools := resMap["tools"].([]Tool)
	if len(tools) < 4 {
		t.Errorf("expected at least 4 tools, got %d", len(tools))
	}

	// 3. Tools Call: track_shipment
	trackReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "track_shipment",
			"arguments": map[string]any{
				"tracking_number": "LP00561234567890",
				"language":        "es",
				"allow_simulated": true,
			},
		},
	}
	trackResp := srv.handleRequest(ctx, trackReq)
	if trackResp == nil || trackResp.Error != nil {
		t.Fatalf("expected track_shipment success, got: %v", trackResp.Error)
	}
	callRes := trackResp.Result.(ToolCallResult)
	if callRes.IsError || len(callRes.Content) == 0 {
		t.Fatalf("expected tool content, got error")
	}
	if !strings.Contains(callRes.Content[0].Text, "LP00561234567890") {
		t.Errorf("expected tracking number in tool output text")
	}

	// 4. Tools Call: detect_carrier
	detectReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "detect_carrier",
			"arguments": map[string]any{
				"tracking_number": "6900123456789012",
			},
		},
	}
	detectResp := srv.handleRequest(ctx, detectReq)
	if detectResp == nil || detectResp.Error != nil {
		t.Fatalf("expected detect_carrier success, got: %v", detectResp.Error)
	}
	detectCallRes, ok := detectResp.Result.(ToolCallResult)
	if !ok || detectCallRes.IsError || len(detectCallRes.Content) == 0 {
		t.Fatalf("expected valid detect_carrier ToolCallResult, got: %v", detectResp.Result)
	}
	if !strings.Contains(detectCallRes.Content[0].Text, "correosexpress") {
		t.Errorf("expected correosexpress in detect output text, got: %s", detectCallRes.Content[0].Text)
	}

	// 5. Tools Call: list_carriers
	listCarriersReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "list_carriers",
			"arguments": map[string]any{
				"query": "cainiao",
			},
		},
	}
	listCarriersResp := srv.handleRequest(ctx, listCarriersReq)
	if listCarriersResp == nil || listCarriersResp.Error != nil {
		t.Fatalf("expected list_carriers success, got: %v", listCarriersResp.Error)
	}
	listCallRes, ok := listCarriersResp.Result.(ToolCallResult)
	if !ok || listCallRes.IsError || len(listCallRes.Content) == 0 {
		t.Fatalf("expected valid list_carriers ToolCallResult, got: %v", listCarriersResp.Result)
	}
	if !strings.Contains(listCallRes.Content[0].Text, "cainiao") {
		t.Errorf("expected cainiao carrier in output text, got: %s", listCallRes.Content[0].Text)
	}

	// 6. Tools Call: estimate_delivery
	estReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "estimate_delivery",
			"arguments": map[string]any{
				"tracking_number": "LP00561234567890",
				"allow_simulated": true,
			},
		},
	}
	estResp := srv.handleRequest(ctx, estReq)
	if estResp == nil || estResp.Error != nil {
		t.Fatalf("expected estimate_delivery success, got: %v", estResp.Error)
	}
	estCallRes, ok := estResp.Result.(ToolCallResult)
	if !ok || estCallRes.IsError || len(estCallRes.Content) == 0 {
		t.Fatalf("expected valid estimate_delivery ToolCallResult, got: %v", estResp.Result)
	}
	if !strings.Contains(estCallRes.Content[0].Text, "days_in_transit") || !strings.Contains(estCallRes.Content[0].Text, "progress") {
		t.Errorf("expected days_in_transit and progress fields in estimate output text, got: %s", estCallRes.Content[0].Text)
	}

	// 7. Test HTTP Transport
	httpBody, _ := json.Marshal(initReq)
	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(httpBody)))
	httpRec := httptest.NewRecorder()
	srv.HandleHTTP(httpRec, httpReq)

	if httpRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for MCP HTTP, got %d", httpRec.Code)
	}
}
