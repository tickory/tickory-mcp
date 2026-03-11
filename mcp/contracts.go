package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	contractVersion            = "v1"
	toolListScans              = "tickory_list_scans"
	toolGetScan                = "tickory_get_scan"
	toolCreateScan             = "tickory_create_scan"
	toolUpdateScan             = "tickory_update_scan"
	toolRunScan                = "tickory_run_scan"
	toolDescribeIndicators     = "tickory_describe_indicators"
	toolListAlertEvents        = "tickory_list_alert_events"
	toolGetAlertEvent          = "tickory_get_alert_event"
	toolExplainAlertEvent      = "tickory_explain_alert_event"
	defaultCreateScanTimeframe = "1m"
)

var validTimeframes = map[string]struct{}{
	"1m":  {},
	"5m":  {},
	"15m": {},
	"1h":  {},
}

// HardGatesInput mirrors the scan hard-gate contract exposed by the HTTP API.
type HardGatesInput struct {
	MinVolumeQuote   *float64 `json:"min_volume_quote,omitempty"`
	MaxVolumeQuote   *float64 `json:"max_volume_quote,omitempty"`
	MaxDailyMove     *float64 `json:"max_daily_move,omitempty"`
	MinDailyMove     *float64 `json:"min_daily_move,omitempty"`
	MinPrice         *float64 `json:"min_price,omitempty"`
	MaxPrice         *float64 `json:"max_price,omitempty"`
	RequireRSI14     bool     `json:"require_rsi_14,omitempty"`
	RequireMA50      bool     `json:"require_ma_50,omitempty"`
	AllowedExchanges []string `json:"allowed_exchanges,omitempty"`
	AllowedContracts []string `json:"allowed_contracts,omitempty"`
}

type ListScansArgs struct {
	IncludePublic             *bool `json:"include_public,omitempty"`
	IncludeNotificationStatus *bool `json:"include_notification_status,omitempty"`
}

func (a ListScansArgs) Validate() error {
	return nil
}

type GetScanArgs struct {
	ScanID string `json:"scan_id"`
}

func (a GetScanArgs) Validate() error {
	if strings.TrimSpace(a.ScanID) == "" {
		return fmt.Errorf("scan_id is required")
	}
	return nil
}

type CreateScanRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	CELExpression string          `json:"cel_expression"`
	Timeframe     string          `json:"timeframe,omitempty"`
	HardGates     *HardGatesInput `json:"hard_gates,omitempty"`
	BuilderConfig map[string]any  `json:"builder_config,omitempty"`
}

func (r CreateScanRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(r.CELExpression) == "" {
		return fmt.Errorf("cel_expression is required")
	}
	if strings.TrimSpace(r.Timeframe) == "" {
		return nil
	}
	return validateTimeframe(r.Timeframe)
}

type UpdateScanRequest struct {
	ScanID        string          `json:"scan_id"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	CELExpression string          `json:"cel_expression"`
	Timeframe     string          `json:"timeframe"`
	HardGates     *HardGatesInput `json:"hard_gates,omitempty"`
	BuilderConfig map[string]any  `json:"builder_config,omitempty"`
}

func (r UpdateScanRequest) Validate() error {
	if strings.TrimSpace(r.ScanID) == "" {
		return fmt.Errorf("scan_id is required")
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(r.CELExpression) == "" {
		return fmt.Errorf("cel_expression is required")
	}
	if strings.TrimSpace(r.Timeframe) == "" {
		return fmt.Errorf("timeframe is required")
	}
	return validateTimeframe(r.Timeframe)
}

func (r UpdateScanRequest) toUpstreamPayload() updateScanPayload {
	return updateScanPayload{
		Name:          r.Name,
		Description:   r.Description,
		CELExpression: r.CELExpression,
		Timeframe:     r.Timeframe,
		HardGates:     r.HardGates,
		BuilderConfig: r.BuilderConfig,
	}
}

type updateScanPayload struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	CELExpression string          `json:"cel_expression"`
	Timeframe     string          `json:"timeframe"`
	HardGates     *HardGatesInput `json:"hard_gates,omitempty"`
	BuilderConfig map[string]any  `json:"builder_config,omitempty"`
}

type RunScanRequest struct {
	ScanID  string   `json:"scan_id"`
	Symbols []string `json:"symbols,omitempty"`
}

func (r RunScanRequest) Validate() error {
	if strings.TrimSpace(r.ScanID) == "" {
		return fmt.Errorf("scan_id is required")
	}
	return nil
}

type ListAlertEventsArgs struct {
	Since  *string `json:"since,omitempty"`
	ScanID *string `json:"scan_id,omitempty"`
	Limit  *int    `json:"limit,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
}

func (a ListAlertEventsArgs) Validate() error {
	if a.Since != nil && strings.TrimSpace(*a.Since) != "" {
		if _, err := time.Parse(time.RFC3339, strings.TrimSpace(*a.Since)); err != nil {
			return fmt.Errorf("since must be RFC3339")
		}
	}
	if a.Limit != nil && (*a.Limit < 1 || *a.Limit > 100) {
		return fmt.Errorf("limit must be between 1 and 100")
	}
	return nil
}

type AlertEventIDArgs struct {
	EventID string `json:"event_id"`
}

func (a AlertEventIDArgs) Validate() error {
	if strings.TrimSpace(a.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if _, err := uuid.Parse(strings.TrimSpace(a.EventID)); err != nil {
		return fmt.Errorf("event_id must be a valid UUID")
	}
	return nil
}

type UpdateScanResponse struct {
	Status             string   `json:"status"`
	ValidationWarnings []string `json:"validation_warnings,omitempty"`
}

type ListScansResult struct {
	SchemaVersion string         `json:"schema_version"`
	Scans         []ScanResponse `json:"scans"`
	Count         int            `json:"count"`
}

type ScanResult struct {
	SchemaVersion string       `json:"schema_version"`
	Scan          ScanResponse `json:"scan"`
}

type UpdateScanResult struct {
	SchemaVersion      string   `json:"schema_version"`
	Status             string   `json:"status"`
	ValidationWarnings []string `json:"validation_warnings,omitempty"`
}

type RunScanResult struct {
	SchemaVersion string              `json:"schema_version"`
	Run           ExecuteScanResponse `json:"run"`
}

type ListAlertEventsResult struct {
	SchemaVersion  string               `json:"schema_version"`
	PayloadVersion string               `json:"payload_version"`
	Events         []AlertEventResponse `json:"events"`
	Count          int                  `json:"count"`
	NextCursor     *string              `json:"next_cursor,omitempty"`
}

type AlertEventResult struct {
	SchemaVersion  string             `json:"schema_version"`
	PayloadVersion string             `json:"payload_version"`
	Event          AlertEventResponse `json:"event"`
}

type ExplainAlertEventResult struct {
	SchemaVersion string `json:"schema_version"`
	AlertEventExplainResponse
}

func validateTimeframe(value string) error {
	if _, ok := validTimeframes[strings.TrimSpace(value)]; !ok {
		return fmt.Errorf("timeframe must be one of 1m, 5m, 15m, 1h")
	}
	return nil
}

func toolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        toolListScans,
			Description: "List scans visible to the API key owner.",
			InputSchema: schemaObject(map[string]any{
				"include_public":              boolSchema("Include public preset scans alongside user scans."),
				"include_notification_status": boolSchema("Include notification channel status for owned scans."),
			}),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"scans":          schemaArray(scanSchema()),
				"count":          integerSchema("Number of scans returned."),
			}, "schema_version", "scans", "count"),
		},
		{
			Name:        toolGetScan,
			Description: "Fetch one scan by ID.",
			InputSchema: schemaObject(map[string]any{
				"scan_id": stringSchema("Scan identifier."),
			}, "scan_id"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"scan":           scanSchema(),
			}, "schema_version", "scan"),
		},
		{
			Name:        toolCreateScan,
			Description: "Create a new user-owned scan.",
			InputSchema: schemaObject(map[string]any{
				"name":           stringSchema("Human-readable scan name."),
				"description":    stringSchema("Optional scan description."),
				"cel_expression": stringSchema("Tickory CEL expression."),
				"timeframe":      timeframeSchema("Optional timeframe. Defaults to 1m."),
				"hard_gates":     hardGatesSchema(),
				"builder_config": looseObjectSchema("Optional visual builder state."),
			}, "name", "cel_expression"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"scan":           scanSchema(),
			}, "schema_version", "scan"),
		},
		{
			Name:        toolUpdateScan,
			Description: "Replace an existing user-owned scan. Pass the full desired scan state, including timeframe.",
			InputSchema: schemaObject(map[string]any{
				"scan_id":        stringSchema("Scan identifier."),
				"name":           stringSchema("Human-readable scan name."),
				"description":    stringSchema("Optional scan description."),
				"cel_expression": stringSchema("Tickory CEL expression."),
				"timeframe":      timeframeSchema("Required timeframe for the updated scan."),
				"hard_gates":     hardGatesSchema(),
				"builder_config": looseObjectSchema("Optional visual builder state."),
			}, "scan_id", "name", "cel_expression", "timeframe"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version":      schemaVersionSchema(),
				"status":              stringSchema("Update result status."),
				"validation_warnings": schemaArray(stringSchema("Validation warning.")),
			}, "schema_version", "status"),
		},
		{
			Name:        toolRunScan,
			Description: "Trigger a scan run immediately.",
			InputSchema: schemaObject(map[string]any{
				"scan_id": stringSchema("Scan identifier."),
				"symbols": schemaArray(stringSchema("Optional symbol override.")),
			}, "scan_id"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"run": schemaObject(map[string]any{
					"run_id":     stringSchema("Scan run identifier."),
					"scan_id":    stringSchema("Scan identifier."),
					"started_at": dateTimeSchema("Run start time."),
					"status":     stringSchema("Run status."),
				}, "run_id", "scan_id", "started_at", "status"),
			}, "schema_version", "run"),
		},
		{
			Name:        toolDescribeIndicators,
			Description: "Describe the CEL variables, guards, ranges, and example expressions available for Tickory scan rules.",
			InputSchema: schemaObject(map[string]any{
				"contract_type": stringEnumSchema("Optional market filter. Defaults to perp so the full variable set is visible.", "spot", "perp"),
			}),
			OutputSchema: schemaObject(map[string]any{
				"schema_version": schemaVersionSchema(),
				"contract_type":  stringEnumSchema("Requested market filter.", "spot", "perp"),
				"categories":     schemaArray(indicatorCategorySchema()),
				"examples":       schemaArray(indicatorExampleSchema()),
				"notes":          schemaArray(stringSchema("Guidance for writing valid CEL expressions.")),
			}, "schema_version", "contract_type", "categories", "examples", "notes"),
		},
		{
			Name:        toolListAlertEvents,
			Description: "List alert events for the API key owner.",
			InputSchema: schemaObject(map[string]any{
				"since":   dateTimeSchema("Optional RFC3339 lower bound for created_at."),
				"scan_id": stringSchema("Optional scan filter."),
				"limit":   integerRangeSchema("Optional page size.", 1, 100),
				"cursor":  stringSchema("Opaque cursor from a previous list call."),
			}),
			OutputSchema: schemaObject(map[string]any{
				"schema_version":  schemaVersionSchema(),
				"payload_version": stringSchema("Upstream alert event payload version."),
				"events":          schemaArray(alertEventSchema()),
				"count":           integerSchema("Number of events returned."),
				"next_cursor":     nullableStringSchema("Opaque cursor for the next page."),
			}, "schema_version", "payload_version", "events", "count"),
		},
		{
			Name:        toolGetAlertEvent,
			Description: "Fetch one alert event by ID.",
			InputSchema: schemaObject(map[string]any{
				"event_id": uuidSchema("Alert event UUID."),
			}, "event_id"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version":  schemaVersionSchema(),
				"payload_version": stringSchema("Upstream alert event payload version."),
				"event":           alertEventSchema(),
			}, "schema_version", "payload_version", "event"),
		},
		{
			Name:        toolExplainAlertEvent,
			Description: "Explain why an alert event triggered or was suppressed.",
			InputSchema: schemaObject(map[string]any{
				"event_id": uuidSchema("Alert event UUID."),
			}, "event_id"),
			OutputSchema: schemaObject(map[string]any{
				"schema_version":  schemaVersionSchema(),
				"payload_version": stringSchema("Upstream explain payload version."),
				"alert_event_id":  uuidSchema("Alert event UUID."),
				"scan_id":         stringSchema("Scan identifier."),
				"run_id":          nullableStringSchema("Optional scan run identifier."),
				"event_type":      stringSchema("Event type."),
				"symbol":          nullableStringSchema("Optional symbol."),
				"timeframe":       stringSchema("Event timeframe."),
				"explanation":     explainBodySchema(),
			}, "schema_version", "payload_version", "alert_event_id", "scan_id", "event_type", "explanation"),
		},
	}
}

func scanSchema() map[string]any {
	return schemaObject(map[string]any{
		"id":                  stringSchema("Scan identifier."),
		"name":                stringSchema("Human-readable scan name."),
		"description":         stringSchema("Scan description."),
		"expression":          stringSchema("CEL expression."),
		"timeframe":           timeframeSchema("Scan timeframe."),
		"visibility":          stringSchema("Visibility hint."),
		"market":              stringSchema("Market hint."),
		"explain":             stringSchema("Short scan summary."),
		"hard_gates":          hardGatesSchema(),
		"builder_config":      looseObjectSchema("Visual builder state."),
		"is_shared":           boolSchema("Whether the scan is shared."),
		"share_token":         nullableStringSchema("Optional share token."),
		"created_at":          dateTimeSchema("Creation timestamp."),
		"updated_at":          dateTimeSchema("Last update timestamp."),
		"validation_warnings": schemaArray(stringSchema("Validation warning.")),
		"notification_status": notificationStatusSchema(),
	}, "id", "name", "description", "expression", "timeframe", "is_shared", "created_at", "updated_at")
}

func hardGatesSchema() map[string]any {
	return schemaObject(map[string]any{
		"min_volume_quote":  nullableNumberSchema("Minimum quote volume."),
		"max_volume_quote":  nullableNumberSchema("Maximum quote volume."),
		"max_daily_move":    nullableNumberSchema("Maximum daily move percent."),
		"min_daily_move":    nullableNumberSchema("Minimum daily move percent."),
		"min_price":         nullableNumberSchema("Minimum price."),
		"max_price":         nullableNumberSchema("Maximum price."),
		"require_rsi_14":    boolSchema("Require RSI-14 data."),
		"require_ma_50":     boolSchema("Require MA-50 data."),
		"allowed_exchanges": schemaArray(stringSchema("Allowed exchange.")),
		"allowed_contracts": schemaArray(stringSchema("Allowed contract type.")),
	})
}

func notificationStatusSchema() map[string]any {
	return nullableObjectSchemaWithValue(schemaObject(map[string]any{
		"active":           boolSchema("Whether notifications are active."),
		"channels":         schemaArray(notificationChannelSchema()),
		"deliveries_today": integerSchema("Deliveries sent today."),
		"max_per_day":      integerSchema("Daily delivery limit."),
	}, "active", "channels", "deliveries_today", "max_per_day"))
}

func notificationChannelSchema() map[string]any {
	return schemaObject(map[string]any{
		"type":      stringSchema("Channel type."),
		"status":    stringSchema("Channel delivery status."),
		"last_sent": nullableDateTimeSchema("Last delivery timestamp."),
		"error":     nullableStringSchema("Latest delivery error."),
	}, "type", "status")
}

func alertEventSchema() map[string]any {
	return schemaObject(map[string]any{
		"payload_version":          stringSchema("Alert payload version."),
		"alert_event_id":           stringSchema("Alert event identifier."),
		"event_type":               stringSchema("Event type."),
		"scan_id":                  stringSchema("Scan identifier."),
		"run_id":                   nullableStringSchema("Optional scan run identifier."),
		"symbol":                   nullableStringSchema("Optional symbol."),
		"timeframe":                stringSchema("Timeframe."),
		"created_at":               dateTimeSchema("Event creation timestamp."),
		"status":                   stringSchema("Event status."),
		"evidence":                 alertEventEvidenceSchema(),
		"delivery_summary":         schemaArray(alertEventDeliverySchema()),
		"suppression_reason_codes": schemaArray(stringSchema("Suppression reason code.")),
	}, "payload_version", "alert_event_id", "event_type", "scan_id", "created_at", "status", "evidence", "delivery_summary")
}

func alertEventEvidenceSchema() map[string]any {
	return schemaObject(map[string]any{
		"indicator_values": looseObjectSchema("Matched indicator values."),
		"hard_gate_summary": schemaObject(map[string]any{
			"configured": looseObjectSchema("Configured hard-gate values."),
			"result":     stringSchema("Hard-gate result."),
		}, "configured", "result"),
		"cel_summary": schemaObject(map[string]any{
			"expression": stringSchema("CEL expression."),
			"result":     stringSchema("CEL result."),
		}, "result"),
		"data_freshness": schemaObject(map[string]any{
			"source_timestamp": nullableDateTimeSchema("Source candle timestamp."),
			"event_created_at": dateTimeSchema("Event creation timestamp."),
			"lag_seconds":      nullableIntegerSchema("Lag between source and event."),
		}, "event_created_at"),
	}, "indicator_values", "hard_gate_summary", "cel_summary", "data_freshness")
}

func alertEventDeliverySchema() map[string]any {
	return schemaObject(map[string]any{
		"channel":       stringSchema("Delivery channel."),
		"status":        stringSchema("Delivery status."),
		"attempts":      integerSchema("Delivery attempts."),
		"error_message": nullableStringSchema("Latest delivery error."),
		"sent_at":       nullableDateTimeSchema("Latest sent timestamp."),
	}, "channel", "status", "attempts")
}

func explainBodySchema() map[string]any {
	return schemaObject(map[string]any{
		"gates_passed":             schemaArray(explainGateSchema()),
		"gates_failed":             schemaArray(explainGateSchema()),
		"cel_evaluation":           explainCELEvaluationSchema(),
		"top_contributing_metrics": schemaArray(explainMetricSchema()),
		"timestamps":               explainTimestampsSchema(),
		"suppression":              nullableObjectSchemaWithValue(explainSuppressionSchema()),
	}, "gates_passed", "gates_failed", "cel_evaluation", "top_contributing_metrics", "timestamps")
}

func explainGateSchema() map[string]any {
	return schemaObject(map[string]any{
		"gate":             stringSchema("Gate name."),
		"reason_code":      stringSchema("Machine-friendly reason code."),
		"summary":          stringSchema("Human-readable summary."),
		"configured_value": anySchema("Configured threshold."),
		"observed_value":   anySchema("Observed runtime value."),
		"count":            nullableIntegerSchema("Optional symbol count."),
	}, "gate", "reason_code", "summary")
}

func explainCELEvaluationSchema() map[string]any {
	return schemaObject(map[string]any{
		"expression":  stringSchema("CEL expression."),
		"result":      stringSchema("CEL evaluation result."),
		"reason_code": stringSchema("Machine-friendly reason code."),
		"summary":     stringSchema("Human-readable summary."),
	}, "result", "reason_code", "summary")
}

func explainMetricSchema() map[string]any {
	return schemaObject(map[string]any{
		"rank":        integerSchema("Metric rank."),
		"name":        stringSchema("Metric name."),
		"value":       anySchema("Metric value."),
		"source":      stringSchema("Metric source."),
		"captured_at": nullableDateTimeSchema("Metric capture time."),
	}, "rank", "name", "value", "source")
}

func explainTimestampsSchema() map[string]any {
	return schemaObject(map[string]any{
		"source_timestamp": nullableDateTimeSchema("Source candle timestamp."),
		"event_created_at": nullableDateTimeSchema("Alert event creation time."),
		"run_started_at":   nullableDateTimeSchema("Scan run start time."),
		"reset_at":         nullableDateTimeSchema("Suppression window reset time."),
		"lag_seconds":      nullableIntegerSchema("Lag between source and event."),
	})
}

func explainSuppressionSchema() map[string]any {
	return schemaObject(map[string]any{
		"suppressed":       boolSchema("Whether the result was suppressed."),
		"reason_codes":     schemaArray(stringSchema("Suppression reason code.")),
		"suppressed_count": nullableIntegerSchema("Suppressed event count."),
		"daily_limit":      nullableIntegerSchema("Daily delivery limit."),
		"sent_today":       nullableIntegerSchema("Deliveries sent today."),
		"reset_at":         nullableDateTimeSchema("Suppression reset timestamp."),
	}, "suppressed", "reason_codes")
}

func indicatorCategorySchema() map[string]any {
	return schemaObject(map[string]any{
		"name":        stringSchema("Category name."),
		"description": stringSchema("Category description."),
		"variables":   schemaArray(indicatorVariableSchema()),
	}, "name", "description", "variables")
}

func indicatorVariableSchema() map[string]any {
	return schemaObject(map[string]any{
		"name":        stringSchema("CEL variable name."),
		"type":        stringSchema("CEL value type."),
		"description": stringSchema("What the variable represents."),
		"valid_range": stringSchema("Practical range or shape of the value."),
		"guard":       nullableStringSchema("Recommended has_* guard, if any."),
		"requires":    nullableStringSchema("Data-readiness requirement surfaced by Tickory."),
		"perp_only":   boolSchema("Whether the variable only exists for perpetual markets."),
	}, "name", "type", "description", "valid_range", "guard", "requires", "perp_only")
}

func indicatorExampleSchema() map[string]any {
	return schemaObject(map[string]any{
		"name":          stringSchema("Example name."),
		"description":   stringSchema("Why the example is useful."),
		"expression":    stringSchema("CEL expression example."),
		"contract_type": stringEnumSchema("Market context where the example applies.", "both", "spot", "perp"),
	}, "name", "description", "expression", "contract_type")
}

func schemaObject(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func schemaArray(item map[string]any) map[string]any {
	return map[string]any{
		"type":  "array",
		"items": item,
	}
}

func stringSchema(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func stringEnumSchema(description string, values ...string) map[string]any {
	schema := stringSchema(description)
	schema["enum"] = values
	return schema
}

func uuidSchema(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"format":      "uuid",
		"description": description,
	}
}

func timeframeSchema(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"enum":        []string{"1m", "5m", "15m", "1h"},
		"description": description,
	}
}

func boolSchema(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}

func integerSchema(description string) map[string]any {
	return map[string]any{
		"type":        "integer",
		"description": description,
	}
}

func integerRangeSchema(description string, min, max int) map[string]any {
	schema := integerSchema(description)
	schema["minimum"] = min
	schema["maximum"] = max
	return schema
}

func dateTimeSchema(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"format":      "date-time",
		"description": description,
	}
}

func anySchema(description string) map[string]any {
	return map[string]any{
		"description": description,
	}
}

func looseObjectSchema(description string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"description":          description,
		"additionalProperties": true,
	}
}

func nullableStringSchema(description string) map[string]any {
	return map[string]any{
		"type":        []string{"string", "null"},
		"description": description,
	}
}

func nullableDateTimeSchema(description string) map[string]any {
	return map[string]any{
		"type":        []string{"string", "null"},
		"format":      "date-time",
		"description": description,
	}
}

func nullableNumberSchema(description string) map[string]any {
	return map[string]any{
		"type":        []string{"number", "null"},
		"description": description,
	}
}

func nullableIntegerSchema(description string) map[string]any {
	return map[string]any{
		"type":        []string{"integer", "null"},
		"description": description,
	}
}

func nullableObjectSchemaWithValue(schema map[string]any) map[string]any {
	return map[string]any{
		"anyOf": []any{
			schema,
			map[string]any{"type": "null"},
		},
	}
}

func schemaVersionSchema() map[string]any {
	return map[string]any{
		"type":        "string",
		"const":       contractVersion,
		"description": "Tickory MCP tool schema version.",
	}
}
