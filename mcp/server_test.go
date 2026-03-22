package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleMessageInitializeAndListTools(t *testing.T) {
	server := NewServer(&Client{}, "test-version")

	initResp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-06-18",
			"capabilities":{},
			"clientInfo":{"name":"test-client","version":"1.0.0"}
		}
	}`)

	if initResp.Error != nil {
		t.Fatalf("expected initialize success, got error: %+v", initResp.Error)
	}

	var initResult initializeResult
	if err := json.Unmarshal(initResp.Result, &initResult); err != nil {
		t.Fatalf("decode initialize result: %v", err)
	}
	if initResult.ProtocolVersion != "2025-06-18" {
		t.Fatalf("expected negotiated protocol 2025-06-18, got %s", initResult.ProtocolVersion)
	}
	if initResult.ServerInfo.Name != "tickory-mcp" {
		t.Fatalf("unexpected server info: %+v", initResult.ServerInfo)
	}

	mustHandleNotification(t, server, `{
		"jsonrpc":"2.0",
		"method":"notifications/initialized"
	}`)

	listResp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":2,
		"method":"tools/list"
	}`)
	if listResp.Error != nil {
		t.Fatalf("expected tools/list success, got error: %+v", listResp.Error)
	}

	var result toolsListResult
	if err := json.Unmarshal(listResp.Result, &result); err != nil {
		t.Fatalf("decode tools/list result: %v", err)
	}

	if len(result.Tools) != 18 {
		t.Fatalf("expected 18 tools, got %d", len(result.Tools))
	}

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
		if tool.InputSchema == nil {
			t.Fatalf("tool %s missing input schema", tool.Name)
		}
	}

	expected := []string{
		toolListScans,
		toolGetScan,
		toolCreateScan,
		toolUpdateScan,
		toolRunScan,
		toolRunAdHocScan,
		toolDescribeIndicators,
		toolListAlertEvents,
		toolGetAlertEvent,
		toolExplainAlertEvent,
		toolGetMarketData,
		toolListSymbols,
		toolCreateRelaySource,
		toolListRelaySources,
		toolAddRelayRoute,
		toolListRelayEvents,
		toolGetRelayTrace,
		toolReplayRelayEvent,
	}
	if strings.Join(names, ",") != strings.Join(expected, ",") {
		t.Fatalf("unexpected tool order: %v", names)
	}
}

func TestHandleMessageRequiresInitializedNotificationBeforeTools(t *testing.T) {
	server := NewServer(&Client{}, "test-version")

	initResp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{"protocolVersion":"2025-06-18","capabilities":{}}
	}`)
	if initResp.Error != nil {
		t.Fatalf("expected initialize success, got error: %+v", initResp.Error)
	}

	listResp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":2,
		"method":"tools/list"
	}`)
	if listResp.Error == nil {
		t.Fatal("expected not-ready error before notifications/initialized")
	}
	if listResp.Error.Code != rpcCodeNotReady {
		t.Fatalf("unexpected error before notifications/initialized: %+v", listResp.Error)
	}
}

func TestHandleMessageBatchRequests(t *testing.T) {
	server := NewServer(&Client{}, "test-version")

	respBytes, err := server.HandleMessage(context.Background(), []byte(`[
		{
			"jsonrpc":"2.0",
			"id":1,
			"method":"initialize",
			"params":{"protocolVersion":"2025-06-18","capabilities":{}}
		},
		{
			"jsonrpc":"2.0",
			"method":"notifications/initialized"
		},
		{
			"jsonrpc":"2.0",
			"id":2,
			"method":"tools/list"
		}
	]`))
	if err != nil {
		t.Fatalf("handle batch: %v", err)
	}

	var responses []rpcResponseForTest
	if err := json.Unmarshal(respBytes, &responses); err != nil {
		t.Fatalf("decode batch response: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 batch responses, got %d: %s", len(responses), string(respBytes))
	}
	if responses[0].Error != nil {
		t.Fatalf("unexpected initialize error in batch: %+v", responses[0].Error)
	}
	if responses[1].Error != nil {
		t.Fatalf("unexpected tools/list error in batch: %+v", responses[1].Error)
	}
}

func TestHandleToolCallIgnoresUnknownArguments(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeJSONTest(t, w, http.StatusOK, map[string]any{"id": "s1", "name": "Low RSI"})
	}))
	defer upstream.Close()

	client, err := NewClient(Config{BaseURL: upstream.URL, APIKey: "tk_test"})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(client, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"tools/call",
		"params":{
			"name":"tickory_create_scan",
			"arguments":{
				"name":"Low RSI",
				"cel_expression":"rsi_14 < 30",
				"_meta":{"progressToken":"abc"},
				"unknown_field":true
			}
		}
	}`)

	if resp.Error != nil {
		t.Fatalf("expected success with unknown fields, got error: %+v", resp.Error)
	}
}

func TestHandleToolCallListScansUsesAPIKeyAndQuery(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/crypto/scans" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "tk_test_key" {
			t.Fatalf("missing api key header: %+v", r.Header)
		}
		if got := r.URL.Query().Get("include_public"); got != "true" {
			t.Fatalf("expected include_public=true, got %q", got)
		}

		writeJSONTest(t, w, http.StatusOK, map[string]any{
			"scans": []any{map[string]any{
				"id":          "scan-1",
				"name":        "Low RSI",
				"description": "oversold bounce",
				"expression":  "rsi_14 < 30",
				"timeframe":   "1m",
				"is_shared":   false,
				"created_at":  "2026-03-10T12:00:00Z",
				"updated_at":  "2026-03-10T12:00:00Z",
			}},
			"count": 1,
		})
	}))
	defer upstream.Close()

	client, err := NewClient(Config{
		BaseURL:    upstream.URL,
		APIKey:     "tk_test_key",
		HTTPClient: upstream.Client(),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	server := NewServer(client, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":2,
		"method":"tools/call",
		"params":{
			"name":"tickory_list_scans",
			"arguments":{"include_public":true}
		}
	}`)

	if resp.Error != nil {
		t.Fatalf("expected tool success, got error: %+v", resp.Error)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success result, got error payload: %s", result.Content[0].Text)
	}
}

func TestHandleToolCallDescribeIndicatorsUsesAPIKeyAndBuildsCatalog(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/crypto/scans/variables" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "tk_test_key" {
			t.Fatalf("missing api key header: %+v", r.Header)
		}
		if got := r.URL.Query().Get("contract_type"); got != "perp" {
			t.Fatalf("expected contract_type=perp, got %q", got)
		}

		writeJSONTest(t, w, http.StatusOK, map[string]any{
			"categories": []string{"indicators", "safety", "metadata"},
			"variables": []any{
				map[string]any{
					"name":        "rsi_14",
					"type":        "double",
					"description": "14-period RSI (0-100)",
					"category":    "indicators",
					"requires":    "RSI-14 ready",
				},
				map[string]any{
					"name":        "funding_rate",
					"type":        "double",
					"description": "Perpetual funding rate",
					"category":    "indicators",
					"perp_only":   true,
				},
				map[string]any{
					"name":        "has_funding_rate",
					"type":        "bool",
					"description": "True if funding_rate is available (perp only)",
					"category":    "safety",
				},
				map[string]any{
					"name":        "symbol",
					"type":        "string",
					"description": "Trading pair symbol (e.g., BTCUSDT)",
					"category":    "metadata",
				},
			},
		})
	}))
	defer upstream.Close()

	client, err := NewClient(Config{
		BaseURL:    upstream.URL,
		APIKey:     "tk_test_key",
		HTTPClient: upstream.Client(),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	server := NewServer(client, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":4,
		"method":"tools/call",
		"params":{
			"name":"tickory_describe_indicators",
			"arguments":{"contract_type":"perp"}
		}
	}`)

	if resp.Error != nil {
		t.Fatalf("expected tool success, got error: %+v", resp.Error)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success result, got error payload: %s", result.Content[0].Text)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}

	var describeResult DescribeIndicatorsResult
	if err := json.Unmarshal(payload, &describeResult); err != nil {
		t.Fatalf("decode describe indicators result: %v", err)
	}

	if describeResult.ContractType != "perp" {
		t.Fatalf("expected contract type perp, got %q", describeResult.ContractType)
	}

	rsi14 := findDescribeIndicatorsVariable(t, describeResult, "rsi_14")
	if rsi14.Guard == nil || *rsi14.Guard != "has_rsi_14" {
		t.Fatalf("expected rsi_14 guard has_rsi_14, got %+v", rsi14.Guard)
	}

	fundingRate := findDescribeIndicatorsVariable(t, describeResult, "funding_rate")
	if !fundingRate.PerpOnly {
		t.Fatal("expected funding_rate to be marked perp_only")
	}

	if len(describeResult.Examples) == 0 {
		t.Fatal("expected CEL examples in describe indicators result")
	}
}

func TestHandleToolCallExplainMapsHTTPErrorsToDeterministicToolError(t *testing.T) {
	eventID := "11111111-1111-1111-1111-111111111111"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/crypto/alert-events/"+eventID+"/explain" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		writeJSONTest(t, w, http.StatusNotFound, map[string]any{
			"error":   "Not Found",
			"message": "alert event not found",
			"code":    "alert_event_not_found",
		})
	}))
	defer upstream.Close()

	client, err := NewClient(Config{
		BaseURL:    upstream.URL,
		APIKey:     "tk_test_key",
		HTTPClient: upstream.Client(),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	server := NewServer(client, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":3,
		"method":"tools/call",
		"params":{
			"name":"tickory_explain_alert_event",
			"arguments":{"event_id":"`+eventID+`"}
		}
	}`)

	if resp.Error != nil {
		t.Fatalf("expected tool error result, got rpc error: %+v", resp.Error)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error result, got success: %+v", result)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}

	var toolErr ToolErrorResult
	if err := json.Unmarshal(payload, &toolErr); err != nil {
		t.Fatalf("decode tool error payload: %v", err)
	}

	if toolErr.Error.MCPCode != "not_found" || toolErr.Error.HTTPStatus != http.StatusNotFound {
		t.Fatalf("unexpected tool error mapping: %+v", toolErr)
	}
	if toolErr.Error.UpstreamCode != "alert_event_not_found" {
		t.Fatalf("unexpected upstream code: %+v", toolErr)
	}
}

func TestHandleToolCallRunScanReturnsMatches(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/crypto/scans/execute" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		writeJSONTest(t, w, http.StatusOK, map[string]any{
			"run_id":     "run-1",
			"scan_id":    "scan-1",
			"started_at": "2026-03-13T12:00:00Z",
			"status":     "completed",
			"matches": []any{
				map[string]any{
					"symbol":        "ETHUSDT",
					"exchange":      "binance",
					"contract_type": "perp",
					"timestamp":     "2026-03-13T11:59:00Z",
					"price":         1850.25,
					"volume_quote":  250000.0,
					"rsi_14":        28.5,
					"ma_50":         1820.0,
					"ma_200":        1750.0,
					"score":         0.85,
					"chart_url":     "https://www.tradingview.com/chart/?symbol=BINANCE:ETHUSDT.P",
					"timeframe":     "1m",
				},
			},
		})
	}))
	defer upstream.Close()

	client, err := NewClient(Config{
		BaseURL:    upstream.URL,
		APIKey:     "tk_test_key",
		HTTPClient: upstream.Client(),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	server := NewServer(client, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":5,
		"method":"tools/call",
		"params":{
			"name":"tickory_run_scan",
			"arguments":{"scan_id":"scan-1"}
		}
	}`)

	if resp.Error != nil {
		t.Fatalf("expected tool success, got error: %+v", resp.Error)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success result, got error payload: %s", result.Content[0].Text)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}

	var runResult RunScanResult
	if err := json.Unmarshal(payload, &runResult); err != nil {
		t.Fatalf("decode run scan result: %v", err)
	}

	if runResult.Run.Status != "completed" {
		t.Fatalf("expected status completed, got %q", runResult.Run.Status)
	}
	if len(runResult.Run.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(runResult.Run.Matches))
	}

	match := runResult.Run.Matches[0]
	if match.Symbol != "ETHUSDT" {
		t.Fatalf("expected symbol ETHUSDT, got %q", match.Symbol)
	}
	if match.RSI14 != 28.5 {
		t.Fatalf("expected rsi_14 28.5, got %v", match.RSI14)
	}
	if match.Price != 1850.25 {
		t.Fatalf("expected price 1850.25, got %v", match.Price)
	}
}

func TestHandleToolCallGetMarketData(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/crypto/market-data" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbols"); got != "BTCUSDT,ETHUSDT" {
			t.Fatalf("expected symbols=BTCUSDT,ETHUSDT, got %q", got)
		}

		rsi := 45.2
		writeJSONTest(t, w, http.StatusOK, map[string]any{
			"market_data": []any{
				map[string]any{
					"symbol":         "BTCUSDT",
					"exchange":       "binance",
					"contract_type":  "perp",
					"timeframe":      "1m",
					"timestamp":      "2026-03-15T10:00:00Z",
					"price":          67500.50,
					"rsi_14":         rsi,
					"volume_quote":   500000.0,
					"daily_move_pct": 1.25,
				},
			},
			"count": 1,
		})
	}))
	defer upstream.Close()

	client, err := NewClient(Config{BaseURL: upstream.URL, APIKey: "tk_test_key", HTTPClient: upstream.Client()})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(client, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":10,
		"method":"tools/call",
		"params":{
			"name":"tickory_get_market_data",
			"arguments":{"symbols":["BTCUSDT","ETHUSDT"]}
		}
	}`)

	if resp.Error != nil {
		t.Fatalf("expected tool success, got error: %+v", resp.Error)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}

	var mdResult GetMarketDataResult
	if err := json.Unmarshal(payload, &mdResult); err != nil {
		t.Fatalf("decode market data result: %v", err)
	}
	if mdResult.Count != 1 {
		t.Fatalf("expected count 1, got %d", mdResult.Count)
	}
	if mdResult.MarketData[0].Symbol != "BTCUSDT" {
		t.Fatalf("expected BTCUSDT, got %q", mdResult.MarketData[0].Symbol)
	}
	if mdResult.MarketData[0].Price != 67500.50 {
		t.Fatalf("expected price 67500.50, got %v", mdResult.MarketData[0].Price)
	}
}

func TestHandleToolCallRunAdHocScanUsesExecuteAdHocEndpoint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/crypto/scans/execute-ad-hoc" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if got := body["name"]; got != "Quick spot test" {
			t.Fatalf("expected name Quick spot test, got %#v", got)
		}
		if got := body["expression"]; got != "has_rsi_14 && rsi_14 < 30" {
			t.Fatalf("expected expression, got %#v", got)
		}
		if symbols, ok := body["symbols"].([]any); !ok || len(symbols) != 2 || symbols[0] != "BTCUSDT" || symbols[1] != "ETHUSDT" {
			t.Fatalf("expected symbols [BTCUSDT ETHUSDT], got %#v", body["symbols"])
		}

		hardGates, ok := body["hard_gates"].(map[string]any)
		if !ok {
			t.Fatalf("expected hard_gates object, got %#v", body["hard_gates"])
		}
		if got := hardGates["require_rsi_14"]; got != true {
			t.Fatalf("expected require_rsi_14=true, got %#v", got)
		}

		builderConfig, ok := body["builder_config"].(map[string]any)
		if !ok || builderConfig["source"] != "unit-test" {
			t.Fatalf("expected builder_config source=unit-test, got %#v", body["builder_config"])
		}

		writeJSONTest(t, w, http.StatusOK, map[string]any{
			"run_id":     "run-adhoc-1",
			"scan_id":    "",
			"started_at": "2026-03-22T09:00:00Z",
			"status":     "completed",
			"matches": []any{
				map[string]any{
					"symbol":        "BTCUSDT",
					"exchange":      "binance",
					"contract_type": "spot",
					"timestamp":     "2026-03-22T08:59:00Z",
					"price":         51000.0,
					"volume_quote":  1500000.0,
					"rsi_14":        25.0,
					"ma_50":         50000.0,
					"timeframe":     "1m",
				},
			},
		})
	}))
	defer upstream.Close()

	client, err := NewClient(Config{
		BaseURL:    upstream.URL,
		APIKey:     "tk_test_key",
		HTTPClient: upstream.Client(),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	server := NewServer(client, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":11,
		"method":"tools/call",
		"params":{
			"name":"tickory_run_ad_hoc_scan",
			"arguments":{
				"name":"Quick spot test",
				"expression":"has_rsi_14 && rsi_14 < 30",
				"symbols":["BTCUSDT","ETHUSDT"],
				"hard_gates":{"require_rsi_14":true},
				"builder_config":{"source":"unit-test"}
			}
		}
	}`)

	if resp.Error != nil {
		t.Fatalf("expected tool success, got error: %+v", resp.Error)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success result, got error payload: %s", result.Content[0].Text)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}

	var runResult RunAdHocScanResult
	if err := json.Unmarshal(payload, &runResult); err != nil {
		t.Fatalf("decode ad hoc run result: %v", err)
	}

	if runResult.Run.RunID != "run-adhoc-1" {
		t.Fatalf("expected run-adhoc-1, got %q", runResult.Run.RunID)
	}
	if runResult.Run.ScanID != "" {
		t.Fatalf("expected empty scan_id for ad hoc run, got %q", runResult.Run.ScanID)
	}
	if len(runResult.Run.Matches) != 1 || runResult.Run.Matches[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected matches: %#v", runResult.Run.Matches)
	}
}

func TestHandleToolCallRunAdHocScanValidation(t *testing.T) {
	server := NewServer(&Client{}, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":12,
		"method":"tools/call",
		"params":{
			"name":"tickory_run_ad_hoc_scan",
			"arguments":{"symbols":["BTCUSDT"]}
		}
	}`)

	if resp.Error == nil {
		t.Fatal("expected validation error, got success")
	}
	if resp.Error.Code != rpcCodeBadParams {
		t.Fatalf("expected bad params, got %+v", resp.Error)
	}
	if !strings.Contains(resp.Error.Message, "expression is required") {
		t.Fatalf("expected expression validation error, got %+v", resp.Error)
	}
}

func TestHandleToolCallGetMarketDataValidation(t *testing.T) {
	server := NewServer(&Client{}, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":11,
		"method":"tools/call",
		"params":{
			"name":"tickory_get_market_data",
			"arguments":{}
		}
	}`)

	if resp.Error == nil {
		t.Fatal("expected validation error for empty symbols")
	}
	if resp.Error.Code != rpcCodeBadParams {
		t.Fatalf("expected bad params error, got %d", resp.Error.Code)
	}
}

func TestHandleToolCallListSymbols(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/crypto/symbols" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("contract_type"); got != "perp" {
			t.Fatalf("expected contract_type=perp, got %q", got)
		}
		if got := r.URL.Query().Get("q"); got != "BTC" {
			t.Fatalf("expected q=BTC, got %q", got)
		}

		writeJSONTest(t, w, http.StatusOK, map[string]any{
			"symbols": []any{
				map[string]any{
					"symbol":        "BTCUSDT",
					"exchange":      "binance",
					"contract_type": "perp",
					"available":     true,
				},
			},
			"count": 1,
		})
	}))
	defer upstream.Close()

	client, err := NewClient(Config{BaseURL: upstream.URL, APIKey: "tk_test_key", HTTPClient: upstream.Client()})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(client, "test-version")
	server.negotiated = true
	server.ready = true

	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":12,
		"method":"tools/call",
		"params":{
			"name":"tickory_list_symbols",
			"arguments":{"contract_type":"perp","q":"BTC"}
		}
	}`)

	if resp.Error != nil {
		t.Fatalf("expected tool success, got error: %+v", resp.Error)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}

	var symResult ListSymbolsResult
	if err := json.Unmarshal(payload, &symResult); err != nil {
		t.Fatalf("decode list symbols result: %v", err)
	}
	if symResult.Count != 1 {
		t.Fatalf("expected count 1, got %d", symResult.Count)
	}
	if symResult.Symbols[0].Symbol != "BTCUSDT" {
		t.Fatalf("expected BTCUSDT, got %q", symResult.Symbols[0].Symbol)
	}
	if !symResult.Symbols[0].Available {
		t.Fatal("expected symbol to be available")
	}
}

type rpcResponseForTest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func mustHandleMessage(t *testing.T, server *Server, raw string) rpcResponseForTest {
	t.Helper()

	respBytes, err := server.HandleMessage(context.Background(), []byte(raw))
	if err != nil {
		t.Fatalf("handle message: %v", err)
	}

	var resp rpcResponseForTest
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("decode rpc response: %v", err)
	}

	return resp
}

func mustHandleNotification(t *testing.T, server *Server, raw string) {
	t.Helper()

	respBytes, err := server.HandleMessage(context.Background(), []byte(raw))
	if err != nil {
		t.Fatalf("handle notification: %v", err)
	}
	if len(respBytes) != 0 {
		t.Fatalf("expected no response for notification, got %s", string(respBytes))
	}
}

func writeJSONTest(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode test response: %v", err)
	}
}
