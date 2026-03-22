package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const (
	toolCreateRelaySource = "tickory_create_relay_source"
	toolListRelaySources  = "tickory_list_relay_sources"
	toolAddRelayRoute     = "tickory_add_relay_route"
	toolListRelayEvents   = "tickory_list_relay_events"
	toolGetRelayTrace     = "tickory_get_relay_trace"
	toolReplayRelayEvent  = "tickory_replay_relay_event"

	relaySourceTypeTradingView = "tradingview"
	relayRouteTypeDirect       = "direct"
	relayReplayQueuedStatus    = "queued"
	relayDocsURL               = "https://tickory.app/docs/user-guide/tradingview-alert-sources"
)

// CreateRelaySourceArgs configures a new TradingView relay source.
type CreateRelaySourceArgs struct {
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active,omitempty"`
}

func (a CreateRelaySourceArgs) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// ListRelaySourcesArgs configures relay source listing.
type ListRelaySourcesArgs struct {
	IncludeRoutes   *bool `json:"include_routes,omitempty"`
	IncludeInactive *bool `json:"include_inactive,omitempty"`
}

func (a ListRelaySourcesArgs) Validate() error {
	return nil
}

// AddRelayRouteArgs configures one direct relay destination.
type AddRelayRouteArgs struct {
	SourceID         string  `json:"source_id"`
	DestinationType  string  `json:"destination_type"`
	TelegramChatID   string  `json:"telegram_chat_id,omitempty"`
	WebhookURL       string  `json:"webhook_url,omitempty"`
	DeliveryEmail    string  `json:"delivery_email,omitempty"`
	FilterExpression *string `json:"filter_expression,omitempty"`
}

func (a AddRelayRouteArgs) Validate() error {
	if strings.TrimSpace(a.SourceID) == "" {
		return fmt.Errorf("source_id is required")
	}

	switch strings.TrimSpace(a.DestinationType) {
	case "telegram":
		if strings.TrimSpace(a.TelegramChatID) == "" {
			return fmt.Errorf("telegram_chat_id is required when destination_type=telegram")
		}
	case "webhook", "discord":
		if strings.TrimSpace(a.WebhookURL) == "" {
			return fmt.Errorf("webhook_url is required when destination_type=%s", strings.TrimSpace(a.DestinationType))
		}
	case "email":
		if strings.TrimSpace(a.DeliveryEmail) == "" {
			return fmt.Errorf("delivery_email is required when destination_type=email")
		}
	default:
		return fmt.Errorf("destination_type must be one of telegram, webhook, discord, email")
	}

	return nil
}

// ListRelayEventsArgs scopes recent relay events for one source.
type ListRelayEventsArgs struct {
	SourceID string `json:"source_id"`
	Limit    *int   `json:"limit,omitempty"`
}

func (a ListRelayEventsArgs) Validate() error {
	if strings.TrimSpace(a.SourceID) == "" {
		return fmt.Errorf("source_id is required")
	}
	if a.Limit != nil && (*a.Limit < 1 || *a.Limit > 100) {
		return fmt.Errorf("limit must be between 1 and 100")
	}
	return nil
}

// GetRelayTraceArgs fetches one source event trace.
type GetRelayTraceArgs struct {
	SourceID      string `json:"source_id"`
	SourceEventID string `json:"source_event_id"`
}

func (a GetRelayTraceArgs) Validate() error {
	if strings.TrimSpace(a.SourceID) == "" {
		return fmt.Errorf("source_id is required")
	}
	if strings.TrimSpace(a.SourceEventID) == "" {
		return fmt.Errorf("source_event_id is required")
	}
	if _, err := uuid.Parse(strings.TrimSpace(a.SourceEventID)); err != nil {
		return fmt.Errorf("source_event_id must be a valid UUID")
	}
	return nil
}

// ReplayRelayEventArgs replays one route from a relay event trace.
type ReplayRelayEventArgs struct {
	SourceID      string `json:"source_id"`
	SourceEventID string `json:"source_event_id"`
	RouteID       string `json:"route_id"`
}

func (a ReplayRelayEventArgs) Validate() error {
	if strings.TrimSpace(a.SourceID) == "" {
		return fmt.Errorf("source_id is required")
	}
	if strings.TrimSpace(a.RouteID) == "" {
		return fmt.Errorf("route_id is required")
	}
	if strings.TrimSpace(a.SourceEventID) == "" {
		return fmt.Errorf("source_event_id is required")
	}
	if _, err := uuid.Parse(strings.TrimSpace(a.SourceEventID)); err != nil {
		return fmt.Errorf("source_event_id must be a valid UUID")
	}
	return nil
}

// RelaySourceAPIResponse mirrors the upstream alert-source response.
type RelaySourceAPIResponse struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id,omitempty"`
	Name          string  `json:"name"`
	SourceType    string  `json:"source_type"`
	WebhookSecret *string `json:"webhook_secret,omitempty"`
	IsActive      bool    `json:"is_active"`
	WebhookURL    string  `json:"webhook_url"`
	CreatedAt     string  `json:"created_at"`
}

// ListRelaySourcesAPIResponse mirrors GET /api/alert-sources.
type ListRelaySourcesAPIResponse struct {
	Sources []RelaySourceAPIResponse `json:"sources"`
}

// RelayRouteAPIResponse mirrors the upstream alert-route response.
type RelayRouteAPIResponse struct {
	ID                 string         `json:"id"`
	SourceID           string         `json:"source_id"`
	RouteType          string         `json:"route_type"`
	AlertConfigID      string         `json:"alert_config_id,omitempty"`
	DeliveryEmail      string         `json:"delivery_email,omitempty"`
	WebhookURL         *string        `json:"webhook_url,omitempty"`
	TelegramChatID     *string        `json:"telegram_chat_id,omitempty"`
	Destinations       map[string]any `json:"destinations,omitempty"`
	ScanID             string         `json:"scan_id,omitempty"`
	ScanName           string         `json:"scan_name,omitempty"`
	DestinationSummary string         `json:"destination_summary,omitempty"`
	FilterExpression   *string        `json:"filter_expression,omitempty"`
	CreatedAt          string         `json:"created_at"`
	UpdatedAt          string         `json:"updated_at"`
}

// ListRelayRoutesAPIResponse mirrors GET /api/alert-sources/{id}/routes.
type ListRelayRoutesAPIResponse struct {
	Routes []RelayRouteAPIResponse `json:"routes"`
}

// RelayEventRouteAPIResponse mirrors the route-trace shape inside relay events.
type RelayEventRouteAPIResponse struct {
	RouteID          string                       `json:"route_id"`
	FilterStatus     string                       `json:"filter_status"`
	FilterReason     *string                      `json:"filter_reason,omitempty"`
	EnqueueStatus    string                       `json:"enqueue_status"`
	EnqueueError     *string                      `json:"enqueue_error,omitempty"`
	DuplicateCount   int                          `json:"duplicate_count"`
	AlertEventID     *string                      `json:"alert_event_id,omitempty"`
	AlertEventStatus *string                      `json:"alert_event_status,omitempty"`
	AlertEventError  *string                      `json:"alert_event_error,omitempty"`
	ReceiptID        *string                      `json:"receipt_id,omitempty"`
	DeliverySummary  []AlertEventDeliveryResponse `json:"delivery_summary"`
}

// RelayEventAPIResponse mirrors the upstream relay event payload.
type RelayEventAPIResponse struct {
	SourceEventID   string                       `json:"source_event_id"`
	SourceEventKey  string                       `json:"source_event_key,omitempty"`
	Status          string                       `json:"status"`
	SourceType      string                       `json:"source_type,omitempty"`
	Symbol          *string                      `json:"symbol,omitempty"`
	EventTimestamp  *string                      `json:"event_timestamp,omitempty"`
	AcceptedAt      string                       `json:"accepted_at"`
	FirstReceivedAt string                       `json:"first_received_at,omitempty"`
	LastReceivedAt  string                       `json:"last_received_at,omitempty"`
	DuplicateCount  int                          `json:"duplicate_count"`
	PayloadSummary  map[string]any               `json:"payload_summary,omitempty"`
	Payload         map[string]any               `json:"payload,omitempty"`
	Routes          []RelayEventRouteAPIResponse `json:"routes"`
}

// ListRelayEventsAPIResponse mirrors GET /api/alert-sources/{id}/events.
type ListRelayEventsAPIResponse struct {
	Events []RelayEventAPIResponse `json:"events"`
	Count  int                     `json:"count"`
}

// GetRelayTraceAPIResponse mirrors GET /api/alert-sources/{id}/events/{eventId}.
type GetRelayTraceAPIResponse struct {
	Event RelayEventAPIResponse `json:"event"`
}

// ReplayRelayEventAPIResponse mirrors the upstream replay response.
type ReplayRelayEventAPIResponse struct {
	Status        string `json:"status"`
	SourceEventID string `json:"source_event_id"`
	RouteID       string `json:"route_id"`
	AlertEventID  string `json:"alert_event_id"`
	ReplayedAt    string `json:"replayed_at"`
}

// RelaySourceCreated is returned by tickory_create_relay_source.
type RelaySourceCreated struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SourceType    string `json:"source_type"`
	IsActive      bool   `json:"is_active"`
	WebhookURL    string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret"`
	CreatedAt     string `json:"created_at"`
}

// RelaySourceSetup describes how to configure TradingView after source creation.
type RelaySourceSetup struct {
	Mode                   string   `json:"mode"`
	SecretTransport        string   `json:"secret_transport"`
	DocsURL                string   `json:"docs_url"`
	TradingViewPayloadTmpl string   `json:"tradingview_payload_template"`
	Notes                  []string `json:"notes,omitempty"`
}

// CreateRelaySourceResult is returned by tickory_create_relay_source.
type CreateRelaySourceResult struct {
	SchemaVersion string             `json:"schema_version"`
	Source        RelaySourceCreated `json:"source"`
	Setup         RelaySourceSetup   `json:"setup"`
}

// RelayRouteSummary is the MCP-safe direct-route view.
type RelayRouteSummary struct {
	ID                 string  `json:"id"`
	RouteType          string  `json:"route_type"`
	DestinationSummary string  `json:"destination_summary"`
	FilterExpression   *string `json:"filter_expression,omitempty"`
}

// RelaySourceSummary is the MCP-safe relay-source view.
type RelaySourceSummary struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	SourceType string              `json:"source_type"`
	IsActive   bool                `json:"is_active"`
	WebhookURL string              `json:"webhook_url"`
	CreatedAt  string              `json:"created_at"`
	Routes     []RelayRouteSummary `json:"routes,omitempty"`
}

// ListRelaySourcesResult is returned by tickory_list_relay_sources.
type ListRelaySourcesResult struct {
	SchemaVersion string               `json:"schema_version"`
	Sources       []RelaySourceSummary `json:"sources"`
	Count         int                  `json:"count"`
}

// RelayRouteResult is returned by tickory_add_relay_route.
type RelayRouteResult struct {
	ID                 string  `json:"id"`
	SourceID           string  `json:"source_id"`
	RouteType          string  `json:"route_type"`
	DestinationSummary string  `json:"destination_summary"`
	FilterExpression   *string `json:"filter_expression,omitempty"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// AddRelayRouteResult is returned by tickory_add_relay_route.
type AddRelayRouteResult struct {
	SchemaVersion string           `json:"schema_version"`
	Route         RelayRouteResult `json:"route"`
}

// RelayEventRouteSummary is the route summary inside tickory_list_relay_events.
type RelayEventRouteSummary struct {
	RouteID          string  `json:"route_id"`
	FilterStatus     string  `json:"filter_status"`
	EnqueueStatus    string  `json:"enqueue_status"`
	DuplicateCount   int     `json:"duplicate_count"`
	AlertEventStatus *string `json:"alert_event_status,omitempty"`
}

// RelayEventSummary is returned by tickory_list_relay_events.
type RelayEventSummary struct {
	SourceEventID  string                   `json:"source_event_id"`
	SourceEventKey string                   `json:"source_event_key,omitempty"`
	Status         string                   `json:"status"`
	Symbol         *string                  `json:"symbol,omitempty"`
	EventTimestamp *string                  `json:"event_timestamp,omitempty"`
	AcceptedAt     string                   `json:"accepted_at"`
	DuplicateCount int                      `json:"duplicate_count"`
	PayloadSummary map[string]any           `json:"payload_summary"`
	Routes         []RelayEventRouteSummary `json:"routes"`
}

// ListRelayEventsResult is returned by tickory_list_relay_events.
type ListRelayEventsResult struct {
	SchemaVersion string              `json:"schema_version"`
	Events        []RelayEventSummary `json:"events"`
	Count         int                 `json:"count"`
}

// RelayTraceRoute is the full per-route trace inside tickory_get_relay_trace.
type RelayTraceRoute struct {
	RouteID          string                       `json:"route_id"`
	FilterStatus     string                       `json:"filter_status"`
	FilterReason     *string                      `json:"filter_reason,omitempty"`
	EnqueueStatus    string                       `json:"enqueue_status"`
	EnqueueError     *string                      `json:"enqueue_error,omitempty"`
	DuplicateCount   int                          `json:"duplicate_count"`
	AlertEventID     *string                      `json:"alert_event_id,omitempty"`
	AlertEventStatus *string                      `json:"alert_event_status,omitempty"`
	AlertEventError  *string                      `json:"alert_event_error,omitempty"`
	ReceiptID        *string                      `json:"receipt_id,omitempty"`
	DeliverySummary  []AlertEventDeliveryResponse `json:"delivery_summary"`
}

// RelayTraceEvent is returned by tickory_get_relay_trace.
type RelayTraceEvent struct {
	SourceEventID   string            `json:"source_event_id"`
	SourceEventKey  string            `json:"source_event_key,omitempty"`
	Status          string            `json:"status"`
	SourceType      string            `json:"source_type"`
	Symbol          *string           `json:"symbol,omitempty"`
	EventTimestamp  *string           `json:"event_timestamp,omitempty"`
	AcceptedAt      string            `json:"accepted_at"`
	FirstReceivedAt string            `json:"first_received_at"`
	LastReceivedAt  string            `json:"last_received_at"`
	DuplicateCount  int               `json:"duplicate_count"`
	PayloadSummary  map[string]any    `json:"payload_summary"`
	Payload         map[string]any    `json:"payload"`
	Routes          []RelayTraceRoute `json:"routes"`
}

// GetRelayTraceResult is returned by tickory_get_relay_trace.
type GetRelayTraceResult struct {
	SchemaVersion string          `json:"schema_version"`
	Event         RelayTraceEvent `json:"event"`
}

// ReplayRelayEventResult is returned by tickory_replay_relay_event.
type ReplayRelayEventResult struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	SourceEventID string `json:"source_event_id"`
	RouteID       string `json:"route_id"`
	AlertEventID  string `json:"alert_event_id"`
	ReplayedAt    string `json:"replayed_at"`
}

func relayToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        toolCreateRelaySource,
			Description: "Create a TradingView relay source and return the webhook URL, source secret, and TradingView-ready payload template.",
			InputSchema: schemaObject(map[string]any{
				"name":      stringSchema("Human-readable relay source name, usually one TradingView strategy or alert family."),
				"is_active": boolSchema("Optional active flag. Defaults to true."),
			}, "name"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"source":         relaySourceCreatedSchema(),
				"setup":          relaySourceSetupSchema(),
			}, "schema_version", "source", "setup"),
		},
		{
			Name:        toolListRelaySources,
			Description: "List TradingView relay sources, optionally including direct-route summaries. Non-TradingView sources and legacy scan-backed routes are omitted.",
			InputSchema: schemaObject(map[string]any{
				"include_routes":   boolSchema("Whether to include direct-route summaries. Defaults to true."),
				"include_inactive": boolSchema("Whether to include inactive relay sources. Defaults to true."),
			}),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"sources":        schemaArray(relaySourceSummarySchema()),
				"count":          integerSchema("Number of relay sources returned."),
			}, "schema_version", "sources", "count"),
		},
		{
			Name:        toolAddRelayRoute,
			Description: "Add one direct relay destination to a TradingView source for telegram, webhook, discord, or email delivery.",
			InputSchema: schemaObject(map[string]any{
				"source_id":         stringSchema("TradingView relay source identifier."),
				"destination_type":  stringEnumSchema("Direct destination type.", "telegram", "webhook", "discord", "email"),
				"telegram_chat_id":  stringSchema("Telegram chat identifier. Required when destination_type=telegram."),
				"webhook_url":       stringSchema("Destination webhook URL. Required when destination_type=webhook or destination_type=discord."),
				"delivery_email":    stringSchema("Destination email address. Required when destination_type=email."),
				"filter_expression": stringSchema("Optional CEL filter expression evaluated against the inbound payload."),
			}, "source_id", "destination_type"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"route":          relayRouteResultSchema(),
			}, "schema_version", "route"),
		},
		{
			Name:        toolListRelayEvents,
			Description: "List recent inbound relay events for one TradingView source with per-route summary status.",
			InputSchema: schemaObject(map[string]any{
				"source_id": stringSchema("TradingView relay source identifier."),
				"limit":     integerRangeSchema("Optional event limit.", 1, 100),
			}, "source_id"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"events":         schemaArray(relayEventSummarySchema()),
				"count":          integerSchema("Number of relay events returned."),
			}, "schema_version", "events", "count"),
		},
		{
			Name:        toolGetRelayTrace,
			Description: "Fetch the full lifecycle trace for one inbound TradingView relay event, including the stored payload and per-route outcomes.",
			InputSchema: schemaObject(map[string]any{
				"source_id":       stringSchema("TradingView relay source identifier."),
				"source_event_id": uuidSchema("Relay source event UUID."),
			}, "source_id", "source_event_id"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"event":          relayTraceEventSchema(),
			}, "schema_version", "event"),
		},
		{
			Name:        toolReplayRelayEvent,
			Description: "Replay one failed route from a TradingView relay event trace when the downstream alert event is still replayable.",
			InputSchema: schemaObject(map[string]any{
				"source_id":       stringSchema("TradingView relay source identifier."),
				"source_event_id": uuidSchema("Relay source event UUID."),
				"route_id":        stringSchema("Route identifier to replay."),
			}, "source_id", "source_event_id", "route_id"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version":  schemaVersionSchema(),
				"status":          stringEnumSchema("Replay queueing status.", relayReplayQueuedStatus),
				"source_event_id": uuidSchema("Relay source event UUID."),
				"route_id":        stringSchema("Route identifier."),
				"alert_event_id":  uuidSchema("Downstream alert event UUID."),
				"replayed_at":     dateTimeSchema("Replay request timestamp."),
			}, "schema_version", "status", "source_event_id", "route_id", "alert_event_id", "replayed_at"),
		},
	}
}

func relaySourceCreatedSchema() map[string]any {
	return schemaObject(map[string]any{
		"id":             stringSchema("Relay source identifier."),
		"name":           stringSchema("Relay source name."),
		"source_type":    stringEnumSchema("Relay source type.", relaySourceTypeTradingView),
		"is_active":      boolSchema("Whether the relay source is active."),
		"webhook_url":    stringSchema("TradingView webhook URL."),
		"webhook_secret": stringSchema("Raw TradingView webhook secret, returned only on create."),
		"created_at":     dateTimeSchema("Creation timestamp."),
	}, "id", "name", "source_type", "is_active", "webhook_url", "webhook_secret", "created_at")
}

func relaySourceSetupSchema() map[string]any {
	return schemaObject(map[string]any{
		"mode":                         stringEnumSchema("Relay setup mode.", "native_tradingview_webhook"),
		"secret_transport":             stringEnumSchema("How TradingView must send the relay secret.", "json_body.secret"),
		"docs_url":                     stringSchema("TradingView relay setup docs."),
		"tradingview_payload_template": stringSchema("JSON payload template ready to paste into TradingView."),
		"notes":                        schemaArray(stringSchema("Important setup note.")),
	}, "mode", "secret_transport", "docs_url", "tradingview_payload_template")
}

func relaySourceSummarySchema() map[string]any {
	return schemaObject(map[string]any{
		"id":          stringSchema("Relay source identifier."),
		"name":        stringSchema("Relay source name."),
		"source_type": stringEnumSchema("Relay source type.", relaySourceTypeTradingView),
		"is_active":   boolSchema("Whether the relay source is active."),
		"webhook_url": stringSchema("TradingView webhook URL."),
		"created_at":  dateTimeSchema("Creation timestamp."),
		"routes":      schemaArray(relayRouteSummarySchema()),
	}, "id", "name", "source_type", "is_active", "webhook_url", "created_at")
}

func relayRouteSummarySchema() map[string]any {
	return schemaObject(map[string]any{
		"id":                  stringSchema("Route identifier."),
		"route_type":          stringEnumSchema("Route type.", relayRouteTypeDirect),
		"destination_summary": stringSchema("Human-readable destination summary."),
		"filter_expression":   nullableStringSchema("Optional CEL filter expression."),
	}, "id", "route_type", "destination_summary")
}

func relayRouteResultSchema() map[string]any {
	return schemaObject(map[string]any{
		"id":                  stringSchema("Route identifier."),
		"source_id":           stringSchema("Relay source identifier."),
		"route_type":          stringEnumSchema("Route type.", relayRouteTypeDirect),
		"destination_summary": stringSchema("Human-readable destination summary."),
		"filter_expression":   nullableStringSchema("Optional CEL filter expression."),
		"created_at":          dateTimeSchema("Creation timestamp."),
		"updated_at":          dateTimeSchema("Last update timestamp."),
	}, "id", "source_id", "route_type", "destination_summary", "created_at", "updated_at")
}

func relayEventRouteSummarySchema() map[string]any {
	return schemaObject(map[string]any{
		"route_id":           stringSchema("Route identifier."),
		"filter_status":      stringSchema("Filter evaluation status."),
		"enqueue_status":     stringSchema("Downstream enqueue status."),
		"duplicate_count":    integerSchema("Duplicate count for this route trace."),
		"alert_event_status": nullableStringSchema("Optional downstream alert event status."),
	}, "route_id", "filter_status", "enqueue_status", "duplicate_count")
}

func relayEventSummarySchema() map[string]any {
	return schemaObject(map[string]any{
		"source_event_id":  uuidSchema("Relay source event UUID."),
		"source_event_key": stringSchema("Optional upstream idempotency key."),
		"status":           stringSchema("Overall relay event status."),
		"symbol":           nullableStringSchema("Optional symbol."),
		"event_timestamp":  nullableDateTimeSchema("Optional source event timestamp."),
		"accepted_at":      dateTimeSchema("When Tickory accepted the relay event."),
		"duplicate_count":  integerSchema("Duplicate count for the source event."),
		"payload_summary":  looseObjectSchema("High-level relay payload summary."),
		"routes":           schemaArray(relayEventRouteSummarySchema()),
	}, "source_event_id", "status", "accepted_at", "duplicate_count", "payload_summary", "routes")
}

func relayTraceRouteSchema() map[string]any {
	return schemaObject(map[string]any{
		"route_id":           stringSchema("Route identifier."),
		"filter_status":      stringSchema("Filter evaluation status."),
		"filter_reason":      nullableStringSchema("Optional filter skip or error reason."),
		"enqueue_status":     stringSchema("Downstream enqueue status."),
		"enqueue_error":      nullableStringSchema("Optional enqueue error."),
		"duplicate_count":    integerSchema("Duplicate count for this route trace."),
		"alert_event_id":     nullableStringSchema("Optional downstream alert event UUID."),
		"alert_event_status": nullableStringSchema("Optional downstream alert event status."),
		"alert_event_error":  nullableStringSchema("Optional downstream alert event error."),
		"receipt_id":         nullableStringSchema("Optional receipt identifier."),
		"delivery_summary":   schemaArray(alertEventDeliverySchema()),
	}, "route_id", "filter_status", "enqueue_status", "duplicate_count", "delivery_summary")
}

func relayTraceEventSchema() map[string]any {
	return schemaObject(map[string]any{
		"source_event_id":   uuidSchema("Relay source event UUID."),
		"source_event_key":  stringSchema("Optional upstream idempotency key."),
		"status":            stringSchema("Overall relay event status."),
		"source_type":       stringSchema("Relay source type."),
		"symbol":            nullableStringSchema("Optional symbol."),
		"event_timestamp":   nullableDateTimeSchema("Optional source event timestamp."),
		"accepted_at":       dateTimeSchema("When Tickory accepted the relay event."),
		"first_received_at": dateTimeSchema("When Tickory first received the source event."),
		"last_received_at":  dateTimeSchema("When Tickory last received the source event."),
		"duplicate_count":   integerSchema("Duplicate count for the source event."),
		"payload_summary":   looseObjectSchema("High-level relay payload summary."),
		"payload":           looseObjectSchema("Stored raw relay payload."),
		"routes":            schemaArray(relayTraceRouteSchema()),
	}, "source_event_id", "status", "source_type", "accepted_at", "first_received_at", "last_received_at", "duplicate_count", "payload_summary", "payload", "routes")
}

func (c *Client) CreateRelaySource(ctx context.Context, args CreateRelaySourceArgs) (*RelaySourceAPIResponse, error) {
	payload := map[string]any{
		"name":        strings.TrimSpace(args.Name),
		"source_type": relaySourceTypeTradingView,
		"is_active":   true,
	}
	if args.IsActive != nil {
		payload["is_active"] = *args.IsActive
	}

	var resp RelaySourceAPIResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/alert-sources", nil, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListRelaySources(ctx context.Context) (*ListRelaySourcesAPIResponse, error) {
	var resp ListRelaySourcesAPIResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/alert-sources", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListRelayRoutes(ctx context.Context, sourceID string) (*ListRelayRoutesAPIResponse, error) {
	var resp ListRelayRoutesAPIResponse
	path := "/api/alert-sources/" + url.PathEscape(strings.TrimSpace(sourceID)) + "/routes"
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetRelaySource(ctx context.Context, sourceID string) (*RelaySourceAPIResponse, error) {
	var resp RelaySourceAPIResponse
	path := "/api/alert-sources/" + url.PathEscape(strings.TrimSpace(sourceID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) AddRelayRoute(ctx context.Context, args AddRelayRouteArgs) (*RelayRouteAPIResponse, error) {
	payload := map[string]any{
		"source_id": strings.TrimSpace(args.SourceID),
	}
	if args.FilterExpression != nil && strings.TrimSpace(*args.FilterExpression) != "" {
		payload["filter_expression"] = strings.TrimSpace(*args.FilterExpression)
	}

	switch strings.TrimSpace(args.DestinationType) {
	case "telegram":
		payload["telegram_chat_id"] = strings.TrimSpace(args.TelegramChatID)
	case "webhook":
		payload["webhook_url"] = strings.TrimSpace(args.WebhookURL)
	case "discord":
		payload["destinations"] = map[string]any{
			"discord": map[string]any{
				"webhook_url": strings.TrimSpace(args.WebhookURL),
			},
		}
	case "email":
		payload["delivery_email"] = strings.TrimSpace(args.DeliveryEmail)
	}

	var resp RelayRouteAPIResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/alert-routes", nil, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListRelayEvents(ctx context.Context, args ListRelayEventsArgs) (*ListRelayEventsAPIResponse, error) {
	query := url.Values{}
	if args.Limit != nil {
		query.Set("limit", fmt.Sprintf("%d", *args.Limit))
	}

	var resp ListRelayEventsAPIResponse
	path := "/api/alert-sources/" + url.PathEscape(strings.TrimSpace(args.SourceID)) + "/events"
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetRelayTrace(ctx context.Context, args GetRelayTraceArgs) (*GetRelayTraceAPIResponse, error) {
	var resp GetRelayTraceAPIResponse
	path := "/api/alert-sources/" + url.PathEscape(strings.TrimSpace(args.SourceID)) + "/events/" + url.PathEscape(strings.TrimSpace(args.SourceEventID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ReplayRelayEvent(ctx context.Context, args ReplayRelayEventArgs) (*ReplayRelayEventAPIResponse, error) {
	payload := map[string]any{
		"route_id": strings.TrimSpace(args.RouteID),
	}

	var resp ReplayRelayEventAPIResponse
	path := "/api/alert-sources/" + url.PathEscape(strings.TrimSpace(args.SourceID)) + "/events/" + url.PathEscape(strings.TrimSpace(args.SourceEventID)) + "/replay"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Server) handleCreateRelaySource(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[CreateRelaySourceArgs](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.CreateRelaySource(ctx, args)
	if err != nil {
		return nil, err
	}

	secret := ""
	if resp.WebhookSecret != nil {
		secret = strings.TrimSpace(*resp.WebhookSecret)
	}
	if secret == "" {
		return nil, fmt.Errorf("tickory relay source response did not include webhook_secret")
	}

	return CreateRelaySourceResult{
		SchemaVersion: contractVersion,
		Source: RelaySourceCreated{
			ID:            resp.ID,
			Name:          resp.Name,
			SourceType:    resp.SourceType,
			IsActive:      resp.IsActive,
			WebhookURL:    resp.WebhookURL,
			WebhookSecret: secret,
			CreatedAt:     resp.CreatedAt,
		},
		Setup: RelaySourceSetup{
			Mode:                   "native_tradingview_webhook",
			SecretTransport:        "json_body.secret",
			DocsURL:                relayDocsURL,
			TradingViewPayloadTmpl: buildTradingViewPayloadTemplate(secret),
			Notes: []string{
				"Paste the webhook_url into the TradingView webhook field.",
				"Native TradingView alerts cannot set custom headers.",
				"Put the source secret inside the JSON body under secret.",
			},
		},
	}, nil
}

func (s *Server) handleListRelaySources(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[ListRelaySourcesArgs](raw)
	if err != nil {
		return nil, err
	}

	includeRoutes := true
	if args.IncludeRoutes != nil {
		includeRoutes = *args.IncludeRoutes
	}
	includeInactive := true
	if args.IncludeInactive != nil {
		includeInactive = *args.IncludeInactive
	}

	resp, err := s.client.ListRelaySources(ctx)
	if err != nil {
		return nil, err
	}

	sources := make([]RelaySourceSummary, 0, len(resp.Sources))
	for _, source := range resp.Sources {
		if source.SourceType != relaySourceTypeTradingView {
			continue
		}
		if !includeInactive && !source.IsActive {
			continue
		}

		summary := RelaySourceSummary{
			ID:         source.ID,
			Name:       source.Name,
			SourceType: source.SourceType,
			IsActive:   source.IsActive,
			WebhookURL: source.WebhookURL,
			CreatedAt:  source.CreatedAt,
		}

		if includeRoutes {
			routesResp, err := s.client.ListRelayRoutes(ctx, source.ID)
			if err != nil {
				return nil, err
			}
			summary.Routes = filterDirectRelayRoutes(routesResp.Routes)
		}

		sources = append(sources, summary)
	}

	return ListRelaySourcesResult{
		SchemaVersion: contractVersion,
		Sources:       sources,
		Count:         len(sources),
	}, nil
}

func (s *Server) handleAddRelayRoute(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[AddRelayRouteArgs](raw)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireTradingViewRelaySource(ctx, args.SourceID); err != nil {
		return nil, err
	}

	resp, err := s.client.AddRelayRoute(ctx, args)
	if err != nil {
		return nil, err
	}

	return AddRelayRouteResult{
		SchemaVersion: contractVersion,
		Route: RelayRouteResult{
			ID:                 resp.ID,
			SourceID:           resp.SourceID,
			RouteType:          resp.RouteType,
			DestinationSummary: resp.DestinationSummary,
			FilterExpression:   resp.FilterExpression,
			CreatedAt:          resp.CreatedAt,
			UpdatedAt:          resp.UpdatedAt,
		},
	}, nil
}

func (s *Server) handleListRelayEvents(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[ListRelayEventsArgs](raw)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireTradingViewRelaySource(ctx, args.SourceID); err != nil {
		return nil, err
	}

	resp, err := s.client.ListRelayEvents(ctx, args)
	if err != nil {
		return nil, err
	}

	events := make([]RelayEventSummary, 0, len(resp.Events))
	for _, event := range resp.Events {
		events = append(events, relayEventSummaryFromAPI(event))
	}

	return ListRelayEventsResult{
		SchemaVersion: contractVersion,
		Events:        events,
		Count:         resp.Count,
	}, nil
}

func (s *Server) handleGetRelayTrace(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[GetRelayTraceArgs](raw)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireTradingViewRelaySource(ctx, args.SourceID); err != nil {
		return nil, err
	}

	resp, err := s.client.GetRelayTrace(ctx, args)
	if err != nil {
		return nil, err
	}

	return GetRelayTraceResult{
		SchemaVersion: contractVersion,
		Event:         relayTraceEventFromAPI(resp.Event),
	}, nil
}

func (s *Server) handleReplayRelayEvent(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[ReplayRelayEventArgs](raw)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireTradingViewRelaySource(ctx, args.SourceID); err != nil {
		return nil, err
	}

	resp, err := s.client.ReplayRelayEvent(ctx, args)
	if err != nil {
		return nil, err
	}

	sourceEventID := strings.TrimSpace(resp.SourceEventID)
	if sourceEventID == "" {
		sourceEventID = strings.TrimSpace(args.SourceEventID)
	}
	routeID := strings.TrimSpace(resp.RouteID)
	if routeID == "" {
		routeID = strings.TrimSpace(args.RouteID)
	}

	return ReplayRelayEventResult{
		SchemaVersion: contractVersion,
		Status:        relayReplayQueuedStatus,
		SourceEventID: sourceEventID,
		RouteID:       routeID,
		AlertEventID:  resp.AlertEventID,
		ReplayedAt:    resp.ReplayedAt,
	}, nil
}

func filterDirectRelayRoutes(routes []RelayRouteAPIResponse) []RelayRouteSummary {
	filtered := make([]RelayRouteSummary, 0, len(routes))
	for _, route := range routes {
		if route.RouteType != relayRouteTypeDirect {
			continue
		}
		filtered = append(filtered, RelayRouteSummary{
			ID:                 route.ID,
			RouteType:          route.RouteType,
			DestinationSummary: route.DestinationSummary,
			FilterExpression:   route.FilterExpression,
		})
	}
	return filtered
}

func relayEventSummaryFromAPI(event RelayEventAPIResponse) RelayEventSummary {
	routes := make([]RelayEventRouteSummary, 0, len(event.Routes))
	for _, route := range event.Routes {
		routes = append(routes, RelayEventRouteSummary{
			RouteID:          route.RouteID,
			FilterStatus:     route.FilterStatus,
			EnqueueStatus:    route.EnqueueStatus,
			DuplicateCount:   route.DuplicateCount,
			AlertEventStatus: route.AlertEventStatus,
		})
	}

	return RelayEventSummary{
		SourceEventID:  event.SourceEventID,
		SourceEventKey: event.SourceEventKey,
		Status:         event.Status,
		Symbol:         event.Symbol,
		EventTimestamp: event.EventTimestamp,
		AcceptedAt:     event.AcceptedAt,
		DuplicateCount: event.DuplicateCount,
		PayloadSummary: ensureMap(event.PayloadSummary),
		Routes:         routes,
	}
}

func relayTraceEventFromAPI(event RelayEventAPIResponse) RelayTraceEvent {
	routes := make([]RelayTraceRoute, 0, len(event.Routes))
	for _, route := range event.Routes {
		routes = append(routes, RelayTraceRoute{
			RouteID:          route.RouteID,
			FilterStatus:     route.FilterStatus,
			FilterReason:     route.FilterReason,
			EnqueueStatus:    route.EnqueueStatus,
			EnqueueError:     route.EnqueueError,
			DuplicateCount:   route.DuplicateCount,
			AlertEventID:     route.AlertEventID,
			AlertEventStatus: route.AlertEventStatus,
			AlertEventError:  route.AlertEventError,
			ReceiptID:        route.ReceiptID,
			DeliverySummary:  route.DeliverySummary,
		})
	}

	return RelayTraceEvent{
		SourceEventID:   event.SourceEventID,
		SourceEventKey:  event.SourceEventKey,
		Status:          event.Status,
		SourceType:      event.SourceType,
		Symbol:          event.Symbol,
		EventTimestamp:  event.EventTimestamp,
		AcceptedAt:      event.AcceptedAt,
		FirstReceivedAt: event.FirstReceivedAt,
		LastReceivedAt:  event.LastReceivedAt,
		DuplicateCount:  event.DuplicateCount,
		PayloadSummary:  ensureMap(event.PayloadSummary),
		Payload:         ensureMap(event.Payload),
		Routes:          routes,
	}
}

func ensureMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func buildTradingViewPayloadTemplate(secret string) string {
	return fmt.Sprintf("{\n  \"ticker\": \"{{ticker}}\",\n  \"close\": {{close}},\n  \"volume\": {{volume}},\n  \"time\": \"{{timenow}}\",\n  \"alert_message\": \"{{strategy.order.action}} {{ticker}}\",\n  \"secret\": %q\n}", secret)
}

func (s *Server) requireTradingViewRelaySource(ctx context.Context, sourceID string) (*RelaySourceAPIResponse, error) {
	source, err := s.client.GetRelaySource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if source.SourceType != relaySourceTypeTradingView {
		return nil, &APIError{
			StatusCode: http.StatusBadRequest,
			Message:    "source_id must reference a tradingview relay source",
		}
	}
	return source, nil
}
