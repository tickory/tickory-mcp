package mcp

import "time"

// ScanResponse represents a scan returned by the Tickory API.
type ScanResponse struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	Expression         string                 `json:"expression"`
	Timeframe          string                 `json:"timeframe"`
	Visibility         string                 `json:"visibility,omitempty"`
	Market             string                 `json:"market,omitempty"`
	Explain            string                 `json:"explain,omitempty"`
	HardGates          *HardGatesResponse     `json:"hard_gates,omitempty"`
	BuilderConfig      map[string]interface{} `json:"builder_config,omitempty"`
	IsShared           bool                   `json:"is_shared"`
	ShareToken         *string                `json:"share_token,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	ValidationWarnings []string               `json:"validation_warnings,omitempty"`
	NotificationStatus *NotificationStatus    `json:"notification_status,omitempty"`
}

// HardGatesResponse represents the hard gate configuration of a scan.
type HardGatesResponse struct {
	MinVolumeQuote *float64 `json:"min_volume_quote,omitempty"`
	MaxVolumeQuote *float64 `json:"max_volume_quote,omitempty"`
	MaxDailyMove   *float64 `json:"max_daily_move,omitempty"`
	MinDailyMove   *float64 `json:"min_daily_move,omitempty"`
	MinPrice       *float64 `json:"min_price,omitempty"`
	MaxPrice       *float64 `json:"max_price,omitempty"`
	RequireRSI14   bool     `json:"require_rsi_14"`
	RequireMA50    bool     `json:"require_ma_50"`
	AllowedExchanges []string `json:"allowed_exchanges,omitempty"`
	AllowedContracts []string `json:"allowed_contracts,omitempty"`
}

// NotificationStatus represents the notification status for a scan.
type NotificationStatus struct {
	Active          bool                      `json:"active"`
	Channels        []NotificationChannelInfo `json:"channels"`
	DeliveriesToday int                       `json:"deliveries_today"`
	MaxPerDay       int                       `json:"max_per_day"`
}

// NotificationChannelInfo represents a single notification channel.
type NotificationChannelInfo struct {
	Type     string     `json:"type"`
	Status   string     `json:"status"`
	LastSent *time.Time `json:"last_sent,omitempty"`
	Error    *string    `json:"error,omitempty"`
}

// ListScansResponse is the collection response for scans.
type ListScansResponse struct {
	Scans []ScanResponse `json:"scans"`
	Count int            `json:"count"`
}

// ExecuteScanRequest represents a scan execution request.
type ExecuteScanRequest struct {
	ScanID  string   `json:"scan_id"`
	Symbols []string `json:"symbols,omitempty"`
}

// ExecuteScanResponse represents an immediate scan execution response.
type ExecuteScanResponse struct {
	RunID     string    `json:"run_id"`
	ScanID    string    `json:"scan_id"`
	StartedAt time.Time `json:"started_at"`
	Status    string    `json:"status"`
}

// AlertEventResponse represents a single alert event.
type AlertEventResponse struct {
	PayloadVersion         string                       `json:"payload_version"`
	AlertEventID           string                       `json:"alert_event_id"`
	EventType              string                       `json:"event_type"`
	ScanID                 string                       `json:"scan_id"`
	RunID                  *string                      `json:"run_id,omitempty"`
	Symbol                 *string                      `json:"symbol,omitempty"`
	Timeframe              string                       `json:"timeframe,omitempty"`
	CreatedAt              string                       `json:"created_at"`
	Status                 string                       `json:"status"`
	Evidence               AlertEventEvidenceResponse   `json:"evidence"`
	DeliverySummary        []AlertEventDeliveryResponse `json:"delivery_summary"`
	SuppressionReasonCodes []string                     `json:"suppression_reason_codes,omitempty"`
}

// AlertEventEvidenceResponse explains why an event exists.
type AlertEventEvidenceResponse struct {
	IndicatorValues map[string]interface{}          `json:"indicator_values"`
	HardGateSummary AlertEventHardGateSummary       `json:"hard_gate_summary"`
	CELSummary      AlertEventCELSummary            `json:"cel_summary"`
	DataFreshness   AlertEventDataFreshnessResponse `json:"data_freshness"`
}

// AlertEventHardGateSummary describes the configured hard gates for the event.
type AlertEventHardGateSummary struct {
	Configured map[string]interface{} `json:"configured"`
	Result     string                 `json:"result"`
}

// AlertEventCELSummary captures CEL evaluation context.
type AlertEventCELSummary struct {
	Expression string `json:"expression,omitempty"`
	Result     string `json:"result"`
}

// AlertEventDataFreshnessResponse captures source timestamp lag at emission time.
type AlertEventDataFreshnessResponse struct {
	SourceTimestamp *string `json:"source_timestamp,omitempty"`
	EventCreatedAt string  `json:"event_created_at"`
	LagSeconds     *int64  `json:"lag_seconds,omitempty"`
}

// AlertEventDeliveryResponse is the latest delivery status per channel.
type AlertEventDeliveryResponse struct {
	Channel      string  `json:"channel"`
	Status       string  `json:"status"`
	Attempts     int     `json:"attempts"`
	ErrorMessage *string `json:"error_message,omitempty"`
	SentAt       *string `json:"sent_at,omitempty"`
}

// ListAlertEventsResponse is the collection response for the alert-events feed.
type ListAlertEventsResponse struct {
	PayloadVersion string               `json:"payload_version"`
	Events         []AlertEventResponse `json:"events"`
	Count          int                  `json:"count"`
	NextCursor     *string              `json:"next_cursor,omitempty"`
}

// GetAlertEventResponse is the detail response for one alert event.
type GetAlertEventResponse struct {
	PayloadVersion string             `json:"payload_version"`
	Event          AlertEventResponse `json:"event"`
}

// ExplainGateResult describes a single gate evaluation.
type ExplainGateResult struct {
	Gate            string      `json:"gate"`
	ReasonCode      string      `json:"reason_code"`
	Summary         string      `json:"summary"`
	ConfiguredValue interface{} `json:"configured_value,omitempty"`
	ObservedValue   interface{} `json:"observed_value,omitempty"`
	Count           *int        `json:"count,omitempty"`
}

// ExplainCELEvaluation describes the CEL outcome that contributed to the result.
type ExplainCELEvaluation struct {
	Expression string `json:"expression,omitempty"`
	Result     string `json:"result"`
	ReasonCode string `json:"reason_code"`
	Summary    string `json:"summary"`
}

// ExplainMetric captures a ranked metric that contributed to the result.
type ExplainMetric struct {
	Rank       int         `json:"rank"`
	Name       string      `json:"name"`
	Value      interface{} `json:"value"`
	Source     string      `json:"source"`
	CapturedAt *string     `json:"captured_at,omitempty"`
}

// ExplainTimestamps contains the relevant timestamps for an explanation.
type ExplainTimestamps struct {
	SourceTimestamp *string `json:"source_timestamp,omitempty"`
	EventCreatedAt *string `json:"event_created_at,omitempty"`
	RunStartedAt   *string `json:"run_started_at,omitempty"`
	ResetAt        *string `json:"reset_at,omitempty"`
	LagSeconds     *int64  `json:"lag_seconds,omitempty"`
}

// ExplainSuppressionContext describes why a scan match may not have emitted a normal alert.
type ExplainSuppressionContext struct {
	Suppressed      bool     `json:"suppressed"`
	ReasonCodes     []string `json:"reason_codes"`
	SuppressedCount *int     `json:"suppressed_count,omitempty"`
	DailyLimit      *int     `json:"daily_limit,omitempty"`
	SentToday       *int     `json:"sent_today,omitempty"`
	ResetAt         *string  `json:"reset_at,omitempty"`
}

// ExplainResponseBody is shared by alert-event and no-match explanation endpoints.
type ExplainResponseBody struct {
	GatesPassed           []ExplainGateResult        `json:"gates_passed"`
	GatesFailed           []ExplainGateResult        `json:"gates_failed"`
	CELEvaluation         ExplainCELEvaluation       `json:"cel_evaluation"`
	TopContributingMetric []ExplainMetric            `json:"top_contributing_metrics"`
	Timestamps            ExplainTimestamps          `json:"timestamps"`
	Suppression           *ExplainSuppressionContext `json:"suppression,omitempty"`
}

// AlertEventExplainResponse is returned by GET /api/crypto/alert-events/{id}/explain.
type AlertEventExplainResponse struct {
	PayloadVersion string              `json:"payload_version"`
	AlertEventID   string              `json:"alert_event_id"`
	ScanID         string              `json:"scan_id"`
	RunID          *string             `json:"run_id,omitempty"`
	EventType      string              `json:"event_type"`
	Symbol         *string             `json:"symbol,omitempty"`
	Timeframe      string              `json:"timeframe,omitempty"`
	Explanation    ExplainResponseBody `json:"explanation"`
}

// ErrorResponse represents an error response from the Tickory API.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
