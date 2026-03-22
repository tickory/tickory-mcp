package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	jsonRPCVersion        = "2.0"
	rpcCodeParseError     = -32700
	rpcCodeInvalidReq     = -32600
	rpcCodeMethodNotFound = -32601
	rpcCodeBadParams      = -32602
	rpcCodeInternal       = -32603
	rpcCodeNotReady       = -32002
)

var supportedProtocolVersions = []string{
	"2025-11-25",
	"2025-11-05",
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

type requestEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type responseEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type initializeParams struct {
	ProtocolVersion string              `json:"protocolVersion"`
	Capabilities    map[string]any      `json:"capabilities,omitempty"`
	ClientInfo      *implementationInfo `json:"clientInfo,omitempty"`
}

type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    serverCapabilities `json:"capabilities"`
	ServerInfo      implementationInfo `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

type implementationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type serverCapabilities struct {
	Tools toolsCapability `json:"tools"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ToolDefinition struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

type toolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Meta      json.RawMessage `json:"_meta,omitempty"`
}

type toolResult struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolHandler func(context.Context, json.RawMessage) (any, error)

type toolRegistration struct {
	definition ToolDefinition
	handler    ToolHandler
}

type ToolErrorResult struct {
	SchemaVersion string        `json:"schema_version"`
	Error         ToolErrorBody `json:"error"`
}

type ToolErrorBody struct {
	Type          string `json:"type"`
	MCPCode       string `json:"mcp_code"`
	Message       string `json:"message"`
	Retryable     bool   `json:"retryable"`
	HTTPStatus    int    `json:"http_status,omitempty"`
	UpstreamError string `json:"upstream_error,omitempty"`
	UpstreamCode  string `json:"upstream_code,omitempty"`
}

type invalidToolArgumentsError struct {
	err error
}

func (e *invalidToolArgumentsError) Error() string {
	if e == nil || e.err == nil {
		return "invalid tool arguments"
	}
	return e.err.Error()
}

func (e *invalidToolArgumentsError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Server exposes Tickory API functionality over MCP stdio.
type Server struct {
	client          *Client
	serverInfo      implementationInfo
	protocolVersion string
	negotiated      bool
	ready           bool
	tools           map[string]toolRegistration
	toolList        []ToolDefinition
}

// NewServer builds a Tickory MCP server with the v1 tool registry.
func NewServer(client *Client, version string) *Server {
	s := &Server{
		client: client,
		serverInfo: implementationInfo{
			Name:    "tickory-mcp",
			Version: strings.TrimSpace(version),
		},
		protocolVersion: supportedProtocolVersions[0],
		tools:           make(map[string]toolRegistration),
	}

	if s.serverInfo.Version == "" {
		s.serverInfo.Version = "dev"
	}

	for _, def := range toolDefinitions() {
		s.toolList = append(s.toolList, def)
		s.tools[def.Name] = toolRegistration{
			definition: def,
			handler:    s.toolHandler(def.Name),
		}
	}

	return s
}

// Serve runs the newline-delimited stdio transport for MCP.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	for {
		line, err := reader.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			response, handleErr := s.HandleMessage(ctx, bytes.TrimSpace(line))
			if handleErr != nil {
				return handleErr
			}
			if len(response) > 0 {
				if _, writeErr := out.Write(append(response, '\n')); writeErr != nil {
					return fmt.Errorf("write mcp response: %w", writeErr)
				}
			}
		}

		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read mcp message: %w", err)
		}
	}
}

// HandleMessage processes a single newline-delimited MCP message.
func (s *Server) HandleMessage(ctx context.Context, raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}

	if trimmed[0] == '[' {
		return s.handleBatchMessage(ctx, trimmed)
	}

	return s.handleSingleMessage(ctx, trimmed)
}

func (s *Server) handleBatchMessage(ctx context.Context, raw []byte) ([]byte, error) {
	var batch []json.RawMessage
	if err := json.Unmarshal(raw, &batch); err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			Error:   newRPCError(rpcCodeParseError, "parse error", nil),
		})
	}
	if len(batch) == 0 {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			Error:   newRPCError(rpcCodeInvalidReq, "invalid request", nil),
		})
	}

	responses := make([][]byte, 0, len(batch))
	for _, item := range batch {
		response, err := s.handleSingleMessage(ctx, item)
		if err != nil {
			return nil, err
		}
		if len(response) > 0 {
			responses = append(responses, response)
		}
	}

	if len(responses) == 0 {
		return nil, nil
	}

	return marshalBatchResponses(responses), nil
}

func (s *Server) handleSingleMessage(ctx context.Context, raw []byte) ([]byte, error) {
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			Error:   newRPCError(rpcCodeParseError, "parse error", nil),
		})
	}

	if _, ok := payload.(map[string]any); !ok {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			Error:   newRPCError(rpcCodeInvalidReq, "invalid request", nil),
		})
	}

	var req requestEnvelope
	if err := json.Unmarshal(raw, &req); err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			Error:   newRPCError(rpcCodeInvalidReq, "invalid request", nil),
		})
	}

	if req.JSONRPC != jsonRPCVersion || strings.TrimSpace(req.Method) == "" {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   newRPCError(rpcCodeInvalidReq, "invalid request", nil),
		})
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		if s.negotiated {
			s.ready = true
		}
		return nil, nil
	case "ping":
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Result:  map[string]any{},
		})
	case "tools/list":
		if !s.ready {
			return marshalResponse(responseEnvelope{
				JSONRPC: jsonRPCVersion,
				ID:      req.ID,
				Error:   s.notReadyError(),
			})
		}
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Result: toolsListResult{
				Tools: s.toolList,
			},
		})
	case "tools/call":
		if !s.ready {
			return marshalResponse(responseEnvelope{
				JSONRPC: jsonRPCVersion,
				ID:      req.ID,
				Error:   s.notReadyError(),
			})
		}
		return s.handleToolCall(ctx, req)
	default:
		if len(req.ID) == 0 {
			return nil, nil
		}
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   newRPCError(rpcCodeMethodNotFound, "method not found", map[string]any{"method": req.Method}),
		})
	}
}

func (s *Server) handleInitialize(req requestEnvelope) ([]byte, error) {
	params := initializeParams{}
	if err := decodeStrict(req.Params, &params); err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   newRPCError(rpcCodeBadParams, err.Error(), nil),
		})
	}

	selectedVersion, err := negotiateProtocolVersion(params.ProtocolVersion)
	if err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   newRPCError(rpcCodeBadParams, err.Error(), map[string]any{"supported_versions": supportedProtocolVersions}),
		})
	}

	s.protocolVersion = selectedVersion
	s.negotiated = true
	s.ready = false

	return marshalResponse(responseEnvelope{
		JSONRPC: jsonRPCVersion,
		ID:      req.ID,
		Result: initializeResult{
			ProtocolVersion: selectedVersion,
			Capabilities: serverCapabilities{
				Tools: toolsCapability{ListChanged: false},
			},
			ServerInfo:   s.serverInfo,
			Instructions: "Use Tickory tools for scan management, alert event retrieval, and alert-event explanations. All tool payloads use schema_version v1.",
		},
	})
}

func (s *Server) handleToolCall(ctx context.Context, req requestEnvelope) ([]byte, error) {
	params := toolCallParams{}
	if err := decodeStrict(req.Params, &params); err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   newRPCError(rpcCodeBadParams, err.Error(), nil),
		})
	}
	if strings.TrimSpace(params.Name) == "" {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   newRPCError(rpcCodeBadParams, "tool name is required", nil),
		})
	}

	registration, ok := s.tools[params.Name]
	if !ok {
		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   newRPCError(rpcCodeMethodNotFound, "tool not found", map[string]any{"tool": params.Name}),
		})
	}

	payload, err := registration.handler(ctx, params.Arguments)
	if err != nil {
		var argErr *invalidToolArgumentsError
		if errors.As(err, &argErr) {
			return marshalResponse(responseEnvelope{
				JSONRPC: jsonRPCVersion,
				ID:      req.ID,
				Error:   newRPCError(rpcCodeBadParams, argErr.Error(), map[string]any{"tool": params.Name}),
			})
		}

		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return marshalResponse(responseEnvelope{
				JSONRPC: jsonRPCVersion,
				ID:      req.ID,
				Result:  toolResultFromError(apiErr),
			})
		}

		var transportErr *TransportError
		if errors.As(err, &transportErr) {
			return marshalResponse(responseEnvelope{
				JSONRPC: jsonRPCVersion,
				ID:      req.ID,
				Result:  toolResultFromError(transportErr),
			})
		}

		return marshalResponse(responseEnvelope{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error:   newRPCError(rpcCodeInternal, err.Error(), nil),
		})
	}

	return marshalResponse(responseEnvelope{
		JSONRPC: jsonRPCVersion,
		ID:      req.ID,
		Result: toolResult{
			Content: []toolContent{{
				Type: "text",
				Text: mustJSONText(payload),
			}},
			StructuredContent: payload,
		},
	})
}

func (s *Server) notReadyError() *rpcError {
	requiredMethod := "initialize"
	if s.negotiated {
		requiredMethod = "notifications/initialized"
	}

	return newRPCError(rpcCodeNotReady, "server not initialized", map[string]any{"required_method": requiredMethod})
}

func (s *Server) toolHandler(name string) ToolHandler {
	switch name {
	case toolListScans:
		return s.handleListScans
	case toolGetScan:
		return s.handleGetScan
	case toolCreateScan:
		return s.handleCreateScan
	case toolUpdateScan:
		return s.handleUpdateScan
	case toolRunScan:
		return s.handleRunScan
	case toolDescribeIndicators:
		return s.handleDescribeIndicators
	case toolListAlertEvents:
		return s.handleListAlertEvents
	case toolGetAlertEvent:
		return s.handleGetAlertEvent
	case toolExplainAlertEvent:
		return s.handleExplainAlertEvent
	case toolGetMarketData:
		return s.handleGetMarketData
	case toolListSymbols:
		return s.handleListSymbols
	case toolCreateRelaySource:
		return s.handleCreateRelaySource
	case toolListRelaySources:
		return s.handleListRelaySources
	case toolAddRelayRoute:
		return s.handleAddRelayRoute
	case toolListRelayEvents:
		return s.handleListRelayEvents
	case toolGetRelayTrace:
		return s.handleGetRelayTrace
	case toolReplayRelayEvent:
		return s.handleReplayRelayEvent
	default:
		return func(context.Context, json.RawMessage) (any, error) {
			return nil, fmt.Errorf("tool %s is not registered", name)
		}
	}
}

func (s *Server) handleListScans(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[ListScansArgs](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.ListScans(ctx, args)
	if err != nil {
		return nil, err
	}

	return ListScansResult{
		SchemaVersion: contractVersion,
		Scans:         resp.Scans,
		Count:         resp.Count,
	}, nil
}

func (s *Server) handleGetScan(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[GetScanArgs](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.GetScan(ctx, args.ScanID)
	if err != nil {
		return nil, err
	}

	return ScanResult{
		SchemaVersion: contractVersion,
		Scan:          *resp,
	}, nil
}

func (s *Server) handleCreateScan(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[CreateScanRequest](raw)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.Timeframe) == "" {
		args.Timeframe = defaultCreateScanTimeframe
	}

	resp, err := s.client.CreateScan(ctx, args)
	if err != nil {
		return nil, err
	}

	return ScanResult{
		SchemaVersion: contractVersion,
		Scan:          *resp,
	}, nil
}

func (s *Server) handleUpdateScan(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[UpdateScanRequest](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.UpdateScan(ctx, args)
	if err != nil {
		return nil, err
	}

	return UpdateScanResult{
		SchemaVersion:      contractVersion,
		Status:             resp.Status,
		ValidationWarnings: resp.ValidationWarnings,
	}, nil
}

func (s *Server) handleRunScan(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[RunScanRequest](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.RunScan(ctx, args)
	if err != nil {
		return nil, err
	}

	return RunScanResult{
		SchemaVersion: contractVersion,
		Run:           *resp,
	}, nil
}

func (s *Server) handleDescribeIndicators(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[DescribeIndicatorsArgs](raw)
	if err != nil {
		return nil, err
	}
	args.ContractType = normalizeDescribeIndicatorsContractType(args.ContractType)
	if args.ContractType == "" {
		args.ContractType = defaultDescribeIndicatorsContractType
	}

	resp, err := s.client.DescribeIndicators(ctx, args.ContractType)
	if err != nil {
		return nil, err
	}

	return buildDescribeIndicatorsResult(args.ContractType, resp), nil
}

func (s *Server) handleListAlertEvents(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[ListAlertEventsArgs](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.ListAlertEvents(ctx, args)
	if err != nil {
		return nil, err
	}

	return ListAlertEventsResult{
		SchemaVersion:  contractVersion,
		PayloadVersion: resp.PayloadVersion,
		Events:         resp.Events,
		Count:          resp.Count,
		NextCursor:     resp.NextCursor,
	}, nil
}

func (s *Server) handleGetAlertEvent(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[AlertEventIDArgs](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.GetAlertEvent(ctx, args.EventID)
	if err != nil {
		return nil, err
	}

	return AlertEventResult{
		SchemaVersion:  contractVersion,
		PayloadVersion: resp.PayloadVersion,
		Event:          resp.Event,
	}, nil
}

func (s *Server) handleExplainAlertEvent(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[AlertEventIDArgs](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.ExplainAlertEvent(ctx, args.EventID)
	if err != nil {
		return nil, err
	}

	return ExplainAlertEventResult{
		SchemaVersion:             contractVersion,
		AlertEventExplainResponse: *resp,
	}, nil
}

func (s *Server) handleGetMarketData(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[GetMarketDataArgs](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.GetMarketData(ctx, args.Symbols)
	if err != nil {
		return nil, err
	}

	return GetMarketDataResult{
		SchemaVersion: contractVersion,
		MarketData:    resp.MarketData,
		Count:         resp.Count,
	}, nil
}

func (s *Server) handleListSymbols(ctx context.Context, raw json.RawMessage) (any, error) {
	args, err := decodeArguments[ListSymbolsArgs](raw)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.ListSymbols(ctx, args)
	if err != nil {
		return nil, err
	}

	return ListSymbolsResult{
		SchemaVersion: contractVersion,
		Symbols:       resp.Symbols,
		Count:         resp.Count,
	}, nil
}

func decodeArguments[T interface{ Validate() error }](raw json.RawMessage) (T, error) {
	var args T
	payload := bytes.TrimSpace(raw)
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	if err := json.Unmarshal(payload, &args); err != nil {
		return args, &invalidToolArgumentsError{err: fmt.Errorf("invalid tool arguments: %w", err)}
	}
	if err := args.Validate(); err != nil {
		return args, &invalidToolArgumentsError{err: fmt.Errorf("invalid tool arguments: %w", err)}
	}
	return args, nil
}

func decodeStrict(raw []byte, target any) error {
	payload := bytes.TrimSpace(raw)
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}

func toolResultFromError(err error) toolResult {
	body := ToolErrorResult{
		SchemaVersion: contractVersion,
		Error: ToolErrorBody{
			Type:      "tickory_api_error",
			MCPCode:   "internal_error",
			Message:   err.Error(),
			Retryable: false,
		},
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		body.Error.MCPCode, body.Error.Retryable = mapHTTPStatus(apiErr.StatusCode)
		body.Error.HTTPStatus = apiErr.StatusCode
		body.Error.Message = apiErr.Message
		body.Error.UpstreamError = apiErr.UpstreamError
		body.Error.UpstreamCode = apiErr.UpstreamCode
	}

	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		body.Error.MCPCode = "upstream_error"
		body.Error.Message = transportErr.Error()
		body.Error.Retryable = true
	}

	return toolResult{
		Content: []toolContent{{
			Type: "text",
			Text: mustJSONText(body),
		}},
		StructuredContent: body,
		IsError:           true,
	}
}

func mapHTTPStatus(status int) (string, bool) {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request", false
	case http.StatusUnauthorized:
		return "unauthorized", false
	case http.StatusForbidden:
		return "forbidden", false
	case http.StatusNotFound:
		return "not_found", false
	case http.StatusConflict:
		return "conflict", false
	case http.StatusTooManyRequests:
		return "rate_limited", true
	default:
		if status >= 500 {
			return "upstream_error", true
		}
		return "upstream_http_error", false
	}
}

func negotiateProtocolVersion(requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", fmt.Errorf("protocolVersion is required")
	}

	for _, candidate := range supportedProtocolVersions {
		if requested == candidate {
			return requested, nil
		}
	}

	return "", fmt.Errorf("unsupported protocolVersion %q", requested)
}

func newRPCError(code int, message string, data any) *rpcError {
	return &rpcError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func marshalResponse(resp responseEnvelope) ([]byte, error) {
	return json.Marshal(resp)
}

func marshalBatchResponses(responses [][]byte) []byte {
	return append(append([]byte{'['}, bytes.Join(responses, []byte{','})...), ']')
}

func mustJSONText(value any) string {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\"schema_version\":\"%s\",\"error\":{\"message\":\"failed to marshal tool response\"}}", contractVersion)
	}
	return string(payload)
}
