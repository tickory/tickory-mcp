package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleToolCallCreateRelaySourceBuildsTradingViewSetup(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/alert-sources" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tk_test_key" {
			t.Fatalf("expected bearer auth, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "tk_test_key" {
			t.Fatalf("expected api key header, got %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if got := body["name"]; got != "Momentum Strategy" {
			t.Fatalf("expected name Momentum Strategy, got %#v", got)
		}
		if got := body["source_type"]; got != relaySourceTypeTradingView {
			t.Fatalf("expected tradingview source_type, got %#v", got)
		}
		if got := body["is_active"]; got != true {
			t.Fatalf("expected is_active=true, got %#v", got)
		}

		writeJSONTest(t, w, http.StatusOK, map[string]any{
			"id":             "src_123",
			"name":           "Momentum Strategy",
			"source_type":    relaySourceTypeTradingView,
			"is_active":      true,
			"webhook_url":    "https://api.tickory.app/api/webhooks/tradingview/src_123",
			"webhook_secret": "tv_secret_abc",
			"created_at":     "2026-03-21T12:00:00Z",
		})
	}))
	defer upstream.Close()

	server := newReadyServerForTest(t, upstream)
	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"tools/call",
		"params":{
			"name":"tickory_create_relay_source",
			"arguments":{"name":"Momentum Strategy"}
		}
	}`)

	result := decodeToolResult(t, resp)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}

	payload := decodeStructured[CreateRelaySourceResult](t, result)
	if payload.Source.ID != "src_123" {
		t.Fatalf("expected src_123, got %+v", payload.Source)
	}
	if payload.Source.WebhookSecret != "tv_secret_abc" {
		t.Fatalf("expected webhook secret in create result, got %+v", payload.Source)
	}
	if payload.Setup.DocsURL != relayDocsURL {
		t.Fatalf("expected docs url %q, got %q", relayDocsURL, payload.Setup.DocsURL)
	}
	if !strings.Contains(payload.Setup.TradingViewPayloadTmpl, `"secret": "tv_secret_abc"`) {
		t.Fatalf("expected interpolated secret in payload template, got %q", payload.Setup.TradingViewPayloadTmpl)
	}
	if !strings.Contains(payload.Setup.TradingViewPayloadTmpl, `{{strategy.order.action}} {{ticker}}`) {
		t.Fatalf("expected TradingView placeholders in payload template, got %q", payload.Setup.TradingViewPayloadTmpl)
	}
}

func TestHandleToolCallListRelaySourcesFiltersTradingViewSourcesAndLegacyRoutes(t *testing.T) {
	routeCalls := make([]string, 0, 2)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/alert-sources":
			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"sources": []any{
					map[string]any{
						"id":          "src_active",
						"name":        "Active TV",
						"source_type": relaySourceTypeTradingView,
						"is_active":   true,
						"webhook_url": "https://api.tickory.app/api/webhooks/tradingview/src_active",
						"created_at":  "2026-03-21T12:00:00Z",
					},
					map[string]any{
						"id":          "src_inactive",
						"name":        "Inactive TV",
						"source_type": relaySourceTypeTradingView,
						"is_active":   false,
						"webhook_url": "https://api.tickory.app/api/webhooks/tradingview/src_inactive",
						"created_at":  "2026-03-21T12:05:00Z",
					},
					map[string]any{
						"id":          "src_webhook",
						"name":        "Generic Webhook",
						"source_type": "webhook",
						"is_active":   true,
						"webhook_url": "https://api.tickory.app/api/webhooks/tradingview/src_webhook",
						"created_at":  "2026-03-21T12:10:00Z",
					},
				},
			})
		case "/api/alert-sources/src_active/routes":
			routeCalls = append(routeCalls, r.URL.Path)
			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"routes": []any{
					map[string]any{
						"id":                  "route_direct",
						"source_id":           "src_active",
						"route_type":          relayRouteTypeDirect,
						"destination_summary": "telegram",
						"filter_expression":   `symbol == "BTCUSDT"`,
						"created_at":          "2026-03-21T12:15:00Z",
						"updated_at":          "2026-03-21T12:15:00Z",
					},
					map[string]any{
						"id":                  "route_legacy",
						"source_id":           "src_active",
						"route_type":          "scan_alert",
						"destination_summary": "email",
						"created_at":          "2026-03-21T12:16:00Z",
						"updated_at":          "2026-03-21T12:16:00Z",
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	server := newReadyServerForTest(t, upstream)
	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":2,
		"method":"tools/call",
		"params":{
			"name":"tickory_list_relay_sources",
			"arguments":{"include_inactive":false}
		}
	}`)

	result := decodeToolResult(t, resp)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}

	payload := decodeStructured[ListRelaySourcesResult](t, result)
	if payload.Count != 1 || len(payload.Sources) != 1 {
		t.Fatalf("expected one filtered relay source, got %+v", payload)
	}
	if payload.Sources[0].ID != "src_active" {
		t.Fatalf("expected src_active, got %+v", payload.Sources[0])
	}
	if len(payload.Sources[0].Routes) != 1 || payload.Sources[0].Routes[0].ID != "route_direct" {
		t.Fatalf("expected only direct route summaries, got %+v", payload.Sources[0].Routes)
	}
	if len(routeCalls) != 1 {
		t.Fatalf("expected exactly one route lookup, got %+v", routeCalls)
	}
}

func TestHandleToolCallAddRelayRouteMapsDiscordDestination(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/alert-sources/src_123":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for source lookup: %s", r.Method)
			}
			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"id":          "src_123",
				"name":        "TV Source",
				"source_type": relaySourceTypeTradingView,
				"is_active":   true,
				"webhook_url": "https://api.tickory.app/api/webhooks/tradingview/src_123",
				"created_at":  "2026-03-21T12:19:00Z",
			})
		case "/api/alert-routes":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if got := body["source_id"]; got != "src_123" {
				t.Fatalf("expected source_id src_123, got %#v", got)
			}
			destinations, ok := body["destinations"].(map[string]any)
			if !ok {
				t.Fatalf("expected destinations object, got %#v", body["destinations"])
			}
			discord, ok := destinations["discord"].(map[string]any)
			if !ok {
				t.Fatalf("expected discord destination, got %#v", destinations["discord"])
			}
			if got := discord["webhook_url"]; got != "https://discord.com/api/webhooks/123" {
				t.Fatalf("expected discord webhook url, got %#v", got)
			}
			if got := body["filter_expression"]; got != `payload_summary.kind == "entry"` {
				t.Fatalf("expected filter expression, got %#v", got)
			}

			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"id":                  "route_123",
				"source_id":           "src_123",
				"route_type":          relayRouteTypeDirect,
				"destination_summary": "discord",
				"filter_expression":   `payload_summary.kind == "entry"`,
				"created_at":          "2026-03-21T12:20:00Z",
				"updated_at":          "2026-03-21T12:20:00Z",
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	server := newReadyServerForTest(t, upstream)
	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":3,
		"method":"tools/call",
		"params":{
			"name":"tickory_add_relay_route",
			"arguments":{
				"source_id":"src_123",
				"destination_type":"discord",
				"webhook_url":"https://discord.com/api/webhooks/123",
				"filter_expression":"payload_summary.kind == \"entry\""
			}
		}
	}`)

	result := decodeToolResult(t, resp)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}

	payload := decodeStructured[AddRelayRouteResult](t, result)
	if payload.Route.ID != "route_123" || payload.Route.RouteType != relayRouteTypeDirect {
		t.Fatalf("unexpected route payload: %+v", payload.Route)
	}
	if payload.Route.DestinationSummary != "discord" {
		t.Fatalf("expected discord destination summary, got %+v", payload.Route)
	}
}

func TestHandleToolCallListRelayEventsUsesSourceAndLimit(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/alert-sources/src_123":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for source lookup: %s", r.Method)
			}
			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"id":          "src_123",
				"name":        "TV Source",
				"source_type": relaySourceTypeTradingView,
				"is_active":   true,
				"webhook_url": "https://api.tickory.app/api/webhooks/tradingview/src_123",
				"created_at":  "2026-03-21T12:29:00Z",
			})
		case "/api/alert-sources/src_123/events":
			if got := r.URL.Query().Get("limit"); got != "5" {
				t.Fatalf("expected limit=5, got %q", got)
			}

			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"events": []any{
					map[string]any{
						"source_event_id":  "11111111-1111-1111-1111-111111111111",
						"source_event_key": "tv-1",
						"status":           "accepted",
						"symbol":           "BTCUSDT",
						"accepted_at":      "2026-03-21T12:30:00Z",
						"duplicate_count":  1,
						"payload_summary":  map[string]any{"kind": "entry"},
						"routes": []any{
							map[string]any{
								"route_id":           "route_123",
								"filter_status":      "matched",
								"enqueue_status":     "enqueued",
								"duplicate_count":    0,
								"alert_event_status": "pending",
							},
						},
					},
				},
				"count": 1,
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	server := newReadyServerForTest(t, upstream)
	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":4,
		"method":"tools/call",
		"params":{
			"name":"tickory_list_relay_events",
			"arguments":{"source_id":"src_123","limit":5}
		}
	}`)

	result := decodeToolResult(t, resp)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}

	payload := decodeStructured[ListRelayEventsResult](t, result)
	if payload.Count != 1 || len(payload.Events) != 1 {
		t.Fatalf("expected one relay event, got %+v", payload)
	}
	if payload.Events[0].SourceEventID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected event payload: %+v", payload.Events[0])
	}
	if len(payload.Events[0].Routes) != 1 || payload.Events[0].Routes[0].RouteID != "route_123" {
		t.Fatalf("unexpected route summary: %+v", payload.Events[0].Routes)
	}
}

func TestHandleToolCallGetRelayTraceReturnsPayload(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/alert-sources/src_123":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for source lookup: %s", r.Method)
			}
			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"id":          "src_123",
				"name":        "TV Source",
				"source_type": relaySourceTypeTradingView,
				"is_active":   true,
				"webhook_url": "https://api.tickory.app/api/webhooks/tradingview/src_123",
				"created_at":  "2026-03-21T12:29:00Z",
			})
		case "/api/alert-sources/src_123/events/11111111-1111-1111-1111-111111111111":
			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"event": map[string]any{
					"source_event_id":   "11111111-1111-1111-1111-111111111111",
					"source_event_key":  "tv-1",
					"status":            "accepted",
					"source_type":       relaySourceTypeTradingView,
					"symbol":            "BTCUSDT",
					"accepted_at":       "2026-03-21T12:30:00Z",
					"first_received_at": "2026-03-21T12:30:00Z",
					"last_received_at":  "2026-03-21T12:31:00Z",
					"duplicate_count":   1,
					"payload_summary":   map[string]any{"kind": "entry"},
					"payload":           map[string]any{"ticker": "BTCUSDT", "secret": "redacted"},
					"routes": []any{
						map[string]any{
							"route_id":           "route_123",
							"filter_status":      "matched",
							"enqueue_status":     "enqueued",
							"duplicate_count":    0,
							"alert_event_id":     "22222222-2222-2222-2222-222222222222",
							"alert_event_status": "failed",
							"receipt_id":         "receipt_123",
							"delivery_summary": []any{
								map[string]any{
									"channel":       "discord",
									"status":        "failed",
									"attempts":      3,
									"error_message": "timeout",
								},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	server := newReadyServerForTest(t, upstream)
	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":5,
		"method":"tools/call",
		"params":{
			"name":"tickory_get_relay_trace",
			"arguments":{
				"source_id":"src_123",
				"source_event_id":"11111111-1111-1111-1111-111111111111"
			}
		}
	}`)

	result := decodeToolResult(t, resp)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}

	payload := decodeStructured[GetRelayTraceResult](t, result)
	if payload.Event.Payload["ticker"] != "BTCUSDT" {
		t.Fatalf("expected raw payload, got %+v", payload.Event.Payload)
	}
	if len(payload.Event.Routes) != 1 || payload.Event.Routes[0].ReceiptID == nil || *payload.Event.Routes[0].ReceiptID != "receipt_123" {
		t.Fatalf("unexpected route trace: %+v", payload.Event.Routes)
	}
}

func TestHandleToolCallReplayRelayEventReturnsQueuedStatus(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/alert-sources/src_123":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for source lookup: %s", r.Method)
			}
			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"id":          "src_123",
				"name":        "TV Source",
				"source_type": relaySourceTypeTradingView,
				"is_active":   true,
				"webhook_url": "https://api.tickory.app/api/webhooks/tradingview/src_123",
				"created_at":  "2026-03-21T12:39:00Z",
			})
		case "/api/alert-sources/src_123/events/11111111-1111-1111-1111-111111111111/replay":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if got := body["route_id"]; got != "route_123" {
				t.Fatalf("expected route_id route_123, got %#v", got)
			}

			writeJSONTest(t, w, http.StatusOK, map[string]any{
				"status":          "replayed",
				"source_event_id": "11111111-1111-1111-1111-111111111111",
				"route_id":        "route_123",
				"alert_event_id":  "22222222-2222-2222-2222-222222222222",
				"replayed_at":     "2026-03-21T12:40:00Z",
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	server := newReadyServerForTest(t, upstream)
	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":6,
		"method":"tools/call",
		"params":{
			"name":"tickory_replay_relay_event",
			"arguments":{
				"source_id":"src_123",
				"source_event_id":"11111111-1111-1111-1111-111111111111",
				"route_id":"route_123"
			}
		}
	}`)

	result := decodeToolResult(t, resp)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}

	payload := decodeStructured[ReplayRelayEventResult](t, result)
	if payload.Status != relayReplayQueuedStatus {
		t.Fatalf("expected queued status, got %+v", payload)
	}
	if payload.AlertEventID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("unexpected replay payload: %+v", payload)
	}
}

func TestHandleToolCallReplayRelayEventSurfacesConflict(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONTest(t, w, http.StatusConflict, map[string]any{
			"error": "route is not replayable in its current state",
		})
	}))
	defer upstream.Close()

	server := newReadyServerForTest(t, upstream)
	resp := mustHandleMessage(t, server, `{
		"jsonrpc":"2.0",
		"id":7,
		"method":"tools/call",
		"params":{
			"name":"tickory_replay_relay_event",
			"arguments":{
				"source_id":"src_123",
				"source_event_id":"11111111-1111-1111-1111-111111111111",
				"route_id":"route_123"
			}
		}
	}`)

	result := decodeToolResult(t, resp)
	if !result.IsError {
		t.Fatalf("expected error result, got %+v", result)
	}

	payload := decodeStructured[ToolErrorResult](t, result)
	if payload.Error.MCPCode != "conflict" {
		t.Fatalf("expected conflict mcp_code, got %+v", payload.Error)
	}
	if payload.Error.Message != "route is not replayable in its current state" {
		t.Fatalf("unexpected error payload: %+v", payload.Error)
	}
}

func TestRelayToolsRejectNonTradingViewSources(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantPath   string
		wantMethod string
	}{
		{
			name:       "add route",
			wantMethod: http.MethodGet,
			wantPath:   "/api/alert-sources/src_non_tv",
			raw: `{
				"jsonrpc":"2.0",
				"id":8,
				"method":"tools/call",
				"params":{
					"name":"tickory_add_relay_route",
					"arguments":{
						"source_id":"src_non_tv",
						"destination_type":"email",
						"delivery_email":"relay@example.com"
					}
				}
			}`,
		},
		{
			name:       "list events",
			wantMethod: http.MethodGet,
			wantPath:   "/api/alert-sources/src_non_tv",
			raw: `{
				"jsonrpc":"2.0",
				"id":9,
				"method":"tools/call",
				"params":{
					"name":"tickory_list_relay_events",
					"arguments":{"source_id":"src_non_tv","limit":5}
				}
			}`,
		},
		{
			name:       "get trace",
			wantMethod: http.MethodGet,
			wantPath:   "/api/alert-sources/src_non_tv",
			raw: `{
				"jsonrpc":"2.0",
				"id":10,
				"method":"tools/call",
				"params":{
					"name":"tickory_get_relay_trace",
					"arguments":{
						"source_id":"src_non_tv",
						"source_event_id":"11111111-1111-1111-1111-111111111111"
					}
				}
			}`,
		},
		{
			name:       "replay event",
			wantMethod: http.MethodGet,
			wantPath:   "/api/alert-sources/src_non_tv",
			raw: `{
				"jsonrpc":"2.0",
				"id":11,
				"method":"tools/call",
				"params":{
					"name":"tickory_replay_relay_event",
					"arguments":{
						"source_id":"src_non_tv",
						"source_event_id":"11111111-1111-1111-1111-111111111111",
						"route_id":"route_123"
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.wantMethod {
					t.Fatalf("unexpected method: %s", r.Method)
				}
				if r.URL.Path != tt.wantPath {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				writeJSONTest(t, w, http.StatusOK, map[string]any{
					"id":          "src_non_tv",
					"name":        "Webhook Source",
					"source_type": "webhook",
					"is_active":   true,
					"webhook_url": "https://api.tickory.app/api/webhooks/tradingview/src_non_tv",
					"created_at":  "2026-03-21T12:50:00Z",
				})
			}))
			defer upstream.Close()

			server := newReadyServerForTest(t, upstream)
			resp := mustHandleMessage(t, server, tt.raw)
			result := decodeToolResult(t, resp)
			if !result.IsError {
				t.Fatalf("expected error result, got %+v", result)
			}

			payload := decodeStructured[ToolErrorResult](t, result)
			if payload.Error.MCPCode != "invalid_request" {
				t.Fatalf("expected invalid_request, got %+v", payload.Error)
			}
			if payload.Error.Message != "source_id must reference a tradingview relay source" {
				t.Fatalf("unexpected error payload: %+v", payload.Error)
			}
		})
	}
}

func newReadyServerForTest(t *testing.T, upstream *httptest.Server) *Server {
	t.Helper()

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
	return server
}

func decodeToolResult(t *testing.T, resp rpcResponseForTest) toolResult {
	t.Helper()

	if resp.Error != nil {
		t.Fatalf("expected tool result, got rpc error: %+v", resp.Error)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	return result
}

func decodeStructured[T any](t *testing.T, result toolResult) T {
	t.Helper()

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}

	var decoded T
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
	return decoded
}
