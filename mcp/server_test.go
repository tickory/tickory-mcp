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

	if len(result.Tools) != 8 {
		t.Fatalf("expected 8 tools, got %d", len(result.Tools))
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
		toolListAlertEvents,
		toolGetAlertEvent,
		toolExplainAlertEvent,
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

func TestHandleToolCallRejectsUnknownArguments(t *testing.T) {
	server := NewServer(&Client{}, "test-version")
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
				"unknown_field":true
			}
		}
	}`)

	if resp.Error == nil {
		t.Fatal("expected invalid params error")
	}
	if resp.Error.Code != rpcCodeBadParams {
		t.Fatalf("expected invalid params code, got %+v", resp.Error)
	}
	if !strings.Contains(resp.Error.Message, "unknown field") {
		t.Fatalf("expected unknown field message, got %+v", resp.Error)
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
