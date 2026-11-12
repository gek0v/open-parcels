package mcp

import (
	"context"
	"encoding/json"
	"fmt"
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
	if len(tools) < 5 {
		t.Errorf("expected at least 5 tools, got %d", len(tools))
	}

	// 3. Tools Call: track_shipment (simulated: is_live_data = false)
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
				"postal_code":     "28001",
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
	if !strings.Contains(callRes.Content[0].Text, `"is_live_data": false`) {
		t.Errorf("expected is_live_data: false when allow_simulated is true, got: %s", callRes.Content[0].Text)
	}

	// 3b. Tools Call: track_shipment (live/fixture: is_live_data = true)
	trackLiveReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      31,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "track_shipment",
			"arguments": map[string]any{
				"tracking_number": "LP00561234567890",
				"language":        "es",
				"allow_simulated": false,
			},
		},
	}
	trackLiveResp := srv.handleRequest(ctx, trackLiveReq)
	if trackLiveResp == nil || trackLiveResp.Error != nil {
		t.Fatalf("expected track_shipment success with live fixture, got: %v", trackLiveResp.Error)
	}
	callLiveRes := trackLiveResp.Result.(ToolCallResult)
	if callLiveRes.IsError || len(callLiveRes.Content) == 0 {
		t.Fatalf("expected tool content for live track, got error: %s", callLiveRes.Content[0].Text)
	}
	if !strings.Contains(callLiveRes.Content[0].Text, `"is_live_data": true`) {
		t.Errorf("expected is_live_data: true for live track, got: %s", callLiveRes.Content[0].Text)
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

	// 6b. Tools Call: check_carrier_status (live_api, public_scraper, not_implemented)
	statusTests := []struct {
		slug         string
		expectedMode string
		expectedLive bool
	}{
		{"cainiao", "live_api", true},
		{"inpost", "public_scraper", true},
		{"tipsa", "not_implemented", false},
	}
	for _, tc := range statusTests {
		req := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      60,
			Method:  "tools/call",
			Params: map[string]any{
				"name": "check_carrier_status",
				"arguments": map[string]any{
					"carrier_slug": tc.slug,
				},
			},
		}
		resp := srv.handleRequest(ctx, req)
		if resp == nil || resp.Error != nil {
			t.Fatalf("expected check_carrier_status success for %s, got error: %v", tc.slug, resp.Error)
		}
		callRes := resp.Result.(ToolCallResult)
		if callRes.IsError || len(callRes.Content) == 0 {
			t.Fatalf("expected tool content for %s, got error", tc.slug)
		}
		if !strings.Contains(callRes.Content[0].Text, fmt.Sprintf(`"mode": "%s"`, tc.expectedMode)) {
			t.Errorf("carrier %s: expected mode %s, got output: %s", tc.slug, tc.expectedMode, callRes.Content[0].Text)
		}
		if !strings.Contains(callRes.Content[0].Text, fmt.Sprintf(`"is_live_adapter": %t`, tc.expectedLive)) {
			t.Errorf("carrier %s: expected is_live_adapter %t, got output: %s", tc.slug, tc.expectedLive, callRes.Content[0].Text)
		}
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

func TestMCPHTTPAuthentication(t *testing.T) {
	eng := engine.New()
	srv := NewServer(eng)

	initPayload, _ := json.Marshal(JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "ping",
	})

	// 1. Open mode without API keys configured -> 200 OK
	reqOpen := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
	recOpen := httptest.NewRecorder()
	srv.HandleHTTP(recOpen, reqOpen)
	if recOpen.Code != http.StatusOK {
		t.Fatalf("open mode: expected 200 OK, got %d", recOpen.Code)
	}

	// 2. Enable API keys
	srv.SetAPIKeys([]string{"mcp-secret-1", "mcp-secret-2"})

	// Missing credentials -> 401 Unauthorized
	reqNoAuth := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
	recNoAuth := httptest.NewRecorder()
	srv.HandleHTTP(recNoAuth, reqNoAuth)
	if recNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("no auth: expected 401 Unauthorized, got %d", recNoAuth.Code)
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recNoAuth.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("expected valid JSON error response, got err: %v", err)
	}
	if errResp.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code UNAUTHORIZED, got %s", errResp.Error.Code)
	}

	// Wrong Bearer token -> 401 Unauthorized
	reqBadBearer := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
	reqBadBearer.Header.Set("Authorization", "Bearer wrong-token")
	recBadBearer := httptest.NewRecorder()
	srv.HandleHTTP(recBadBearer, reqBadBearer)
	if recBadBearer.Code != http.StatusUnauthorized {
		t.Fatalf("bad bearer: expected 401 Unauthorized, got %d", recBadBearer.Code)
	}

	// Wrong X-API-Key -> 401 Unauthorized
	reqBadKey := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
	reqBadKey.Header.Set("X-API-Key", "wrong-key")
	recBadKey := httptest.NewRecorder()
	srv.HandleHTTP(recBadKey, reqBadKey)
	if recBadKey.Code != http.StatusUnauthorized {
		t.Fatalf("bad x-api-key: expected 401 Unauthorized, got %d", recBadKey.Code)
	}

	// Valid Bearer token -> 200 OK
	reqGoodBearer := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
	reqGoodBearer.Header.Set("Authorization", "Bearer mcp-secret-1")
	recGoodBearer := httptest.NewRecorder()
	srv.HandleHTTP(recGoodBearer, reqGoodBearer)
	if recGoodBearer.Code != http.StatusOK {
		t.Fatalf("good bearer: expected 200 OK, got %d: %s", recGoodBearer.Code, recGoodBearer.Body.String())
	}

	// Valid case-insensitive bearer -> 200 OK
	reqLowerBearer := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
	reqLowerBearer.Header.Set("Authorization", "bearer mcp-secret-1")
	recLowerBearer := httptest.NewRecorder()
	srv.HandleHTTP(recLowerBearer, reqLowerBearer)
	if recLowerBearer.Code != http.StatusOK {
		t.Fatalf("lowercase bearer: expected 200 OK, got %d", recLowerBearer.Code)
	}

	// Valid X-API-Key -> 200 OK
	reqGoodKey := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
	reqGoodKey.Header.Set("X-API-Key", "mcp-secret-2")
	recGoodKey := httptest.NewRecorder()
	srv.HandleHTTP(recGoodKey, reqGoodKey)
	if recGoodKey.Code != http.StatusOK {
		t.Fatalf("good x-api-key: expected 200 OK, got %d: %s", recGoodKey.Code, recGoodKey.Body.String())
	}
}

func TestMCPEnvAPIKeys(t *testing.T) {
	t.Setenv("OPEN_PARCELS_API_KEY", "env-single-key")
	t.Setenv("OPEN_PARCELS_API_KEYS", "env-multi-1, env-multi-2")

	srv := NewServer(engine.New())
	initPayload, _ := json.Marshal(JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "ping",
	})

	for _, key := range []string{"env-single-key", "env-multi-1", "env-multi-2"} {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
		req.Header.Set("X-API-Key", key)
		rec := httptest.NewRecorder()
		srv.HandleHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for env key %q, got %d", key, rec.Code)
		}
	}

	reqNoAuth := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(initPayload)))
	recNoAuth := httptest.NewRecorder()
	srv.HandleHTTP(recNoAuth, reqNoAuth)
	if recNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized without key when env set, got %d", recNoAuth.Code)
	}
}

func TestMCPAuthMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Open mode
	openHandler := AuthMiddleware(nil, inner)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	openHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 in open mode middleware, got %d", rec.Code)
	}

	// Protected mode
	protectedHandler := AuthMiddleware([]string{"mw-secret"}, inner)
	recFail := httptest.NewRecorder()
	protectedHandler.ServeHTTP(recFail, req)
	if recFail.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 in protected middleware, got %d", recFail.Code)
	}

	reqOk := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqOk.Header.Set("Authorization", "Bearer mw-secret")
	recOk := httptest.NewRecorder()
	protectedHandler.ServeHTTP(recOk, reqOk)
	if recOk.Code != http.StatusOK {
		t.Errorf("expected 200 with valid key in protected middleware, got %d", recOk.Code)
	}
}
