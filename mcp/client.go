package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 15 * time.Second

// Config configures the Tickory API client used by the MCP server.
type Config struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// Client is a thin HTTP wrapper over the Tickory API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// APIError captures a non-2xx response from the upstream Tickory API.
type APIError struct {
	StatusCode    int    `json:"status_code"`
	UpstreamError string `json:"upstream_error,omitempty"`
	UpstreamCode  string `json:"upstream_code,omitempty"`
	Message       string `json:"message"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return http.StatusText(e.StatusCode)
}

// TransportError captures a network or transport-level failure when reaching Tickory.
type TransportError struct {
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *TransportError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "tickory transport error"
}

func (e *TransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewClient validates config and returns a reusable Tickory API client.
func NewClient(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	apiKey := strings.TrimSpace(cfg.APIKey)
	if baseURL == "" {
		return nil, errors.New("tickory api base url is required")
	}
	if apiKey == "" {
		return nil, errors.New("tickory api key is required")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	} else if httpClient.Timeout == 0 {
		httpClient.Timeout = timeout
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: httpClient,
	}, nil
}

func (c *Client) ListScans(ctx context.Context, args ListScansArgs) (*ListScansResponse, error) {
	query := url.Values{}
	if args.IncludePublic != nil {
		query.Set("include_public", strconv.FormatBool(*args.IncludePublic))
	}
	if args.IncludeNotificationStatus != nil {
		query.Set("include_notification_status", strconv.FormatBool(*args.IncludeNotificationStatus))
	}

	var resp ListScansResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/crypto/scans", query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetScan(ctx context.Context, scanID string) (*ScanResponse, error) {
	var resp ScanResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/crypto/scans/"+url.PathEscape(scanID), nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CreateScan(ctx context.Context, req CreateScanRequest) (*ScanResponse, error) {
	var resp ScanResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/crypto/scans", nil, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UpdateScan(ctx context.Context, req UpdateScanRequest) (*UpdateScanResponse, error) {
	var resp UpdateScanResponse
	if err := c.doJSON(ctx, http.MethodPut, "/api/crypto/scans/"+url.PathEscape(req.ScanID), nil, req.toUpstreamPayload(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) RunScan(ctx context.Context, req RunScanRequest) (*ExecuteScanResponse, error) {
	upstream := ExecuteScanRequest(req)

	var resp ExecuteScanResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/crypto/scans/execute", nil, upstream, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) DescribeIndicators(ctx context.Context, contractType string) (*CELVariableReferenceResponse, error) {
	query := url.Values{}
	if trimmed := strings.TrimSpace(contractType); trimmed != "" {
		query.Set("contract_type", trimmed)
	}

	var resp CELVariableReferenceResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/crypto/scans/variables", query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListAlertEvents(ctx context.Context, args ListAlertEventsArgs) (*ListAlertEventsResponse, error) {
	query := url.Values{}
	if args.Since != nil {
		query.Set("since", *args.Since)
	}
	if args.ScanID != nil {
		query.Set("scan_id", *args.ScanID)
	}
	if args.Limit != nil {
		query.Set("limit", strconv.Itoa(*args.Limit))
	}
	if args.Cursor != nil {
		query.Set("cursor", *args.Cursor)
	}

	var resp ListAlertEventsResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/crypto/alert-events", query, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetAlertEvent(ctx context.Context, eventID string) (*GetAlertEventResponse, error) {
	var resp GetAlertEventResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/crypto/alert-events/"+url.PathEscape(eventID), nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ExplainAlertEvent(ctx context.Context, eventID string) (*AlertEventExplainResponse, error) {
	var resp AlertEventExplainResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/crypto/alert-events/"+url.PathEscape(eventID)+"/explain", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, requestBody any, responseBody any) error {
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("build tickory request url: %w", err)
	}
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}

	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal tickory request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("create tickory request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "tickory-mcp/1.0")
	req.Header.Set("X-API-Key", c.apiKey)
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &TransportError{
			Message: "failed to reach Tickory API",
			Err:     err,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}

	if responseBody == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(responseBody); err != nil {
		return fmt.Errorf("decode tickory response: %w", err)
	}

	return nil
}

func decodeAPIError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return &TransportError{
			Message: "failed to read Tickory API error response",
			Err:     err,
		}
	}

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    http.StatusText(resp.StatusCode),
	}

	var upstream ErrorResponse
	if len(bytes.TrimSpace(body)) > 0 && json.Unmarshal(body, &upstream) == nil {
		apiErr.UpstreamError = strings.TrimSpace(upstream.Error)
		apiErr.UpstreamCode = strings.TrimSpace(upstream.Code)
		if strings.TrimSpace(upstream.Message) != "" {
			apiErr.Message = strings.TrimSpace(upstream.Message)
		} else if apiErr.UpstreamError != "" {
			apiErr.Message = apiErr.UpstreamError
		}
		return apiErr
	}

	if msg := strings.TrimSpace(string(body)); msg != "" {
		apiErr.Message = msg
	}
	return apiErr
}
