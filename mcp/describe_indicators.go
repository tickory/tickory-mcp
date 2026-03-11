package mcp

import (
	"fmt"
	"strings"
)

const defaultDescribeIndicatorsContractType = "perp"

// DescribeIndicatorsArgs configures CEL-variable discovery for the MCP tool.
type DescribeIndicatorsArgs struct {
	ContractType string `json:"contract_type,omitempty"`
}

func (a DescribeIndicatorsArgs) Validate() error {
	switch normalizeDescribeIndicatorsContractType(a.ContractType) {
	case "", "spot", "perp":
		return nil
	default:
		return fmt.Errorf("contract_type must be one of spot, perp")
	}
}

// CELVariableInfo mirrors the upstream variable reference payload.
type CELVariableInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Category    string `json:"category"`
	PerpOnly    bool   `json:"perp_only,omitempty"`
	Requires    string `json:"requires,omitempty"`
}

// CELVariableReferenceResponse is returned by GET /api/crypto/scans/variables.
type CELVariableReferenceResponse struct {
	Variables  []CELVariableInfo `json:"variables"`
	Categories []string          `json:"categories"`
}

// DescribeIndicatorsResult is returned by tickory_describe_indicators.
type DescribeIndicatorsResult struct {
	SchemaVersion string                       `json:"schema_version"`
	ContractType  string                       `json:"contract_type"`
	Categories    []DescribeIndicatorsCategory `json:"categories"`
	Examples      []DescribeIndicatorsExample  `json:"examples"`
	Notes         []string                     `json:"notes"`
}

// DescribeIndicatorsCategory groups CEL variables by topic.
type DescribeIndicatorsCategory struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Variables   []DescribeIndicatorsVariable `json:"variables"`
}

// DescribeIndicatorsVariable describes a single CEL variable.
type DescribeIndicatorsVariable struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	ValidRange  string  `json:"valid_range"`
	Guard       *string `json:"guard"`
	Requires    *string `json:"requires"`
	PerpOnly    bool    `json:"perp_only"`
}

// DescribeIndicatorsExample is a ready-to-use CEL expression pattern.
type DescribeIndicatorsExample struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Expression   string `json:"expression"`
	ContractType string `json:"contract_type"`
}

var recommendedGuards = map[string]string{
	"funding_rate":       "has_funding_rate",
	"mark_premium":       "has_mark_premium",
	"rsi_14":             "has_rsi_14",
	"rsi_14_normalized":  "has_rsi_14",
	"rsi_20":             "has_rsi_20",
	"ma_20":              "has_ma_20",
	"ma_50":              "has_ma_50",
	"ma_200":             "has_ma_200",
	"above_ma50":         "has_ma_50",
	"below_ma50":         "has_ma_50",
	"close_to_ma50":      "has_ma_50",
	"above_ma200":        "has_ma_200",
	"below_ma200":        "has_ma_200",
	"close_to_ma200":     "has_ma_200",
	"prev_close":         "has_prev_close",
	"prev_rsi_14":        "has_prev_rsi_14",
	"prev_rsi_20":        "has_prev_rsi_20",
	"prev_ma_50":         "has_prev_ma_50",
	"prev_ma_200":        "has_prev_ma_200",
	"prev_volume_quote":  "has_prev_volume_quote",
	"prev_volume_base":   "has_prev_volume_base",
	"delta_close":        "has_prev_close",
	"delta_rsi_14":       "has_delta_rsi_14",
	"delta_volume_quote": "has_delta_volume_quote",
}

func buildDescribeIndicatorsResult(contractType string, resp *CELVariableReferenceResponse) DescribeIndicatorsResult {
	effectiveContractType := normalizeDescribeIndicatorsContractType(contractType)
	if effectiveContractType == "" {
		effectiveContractType = defaultDescribeIndicatorsContractType
	}

	order := append([]string{}, resp.Categories...)
	categoriesByName := make(map[string]*DescribeIndicatorsCategory, len(order))
	for _, name := range order {
		categoriesByName[name] = &DescribeIndicatorsCategory{
			Name:        name,
			Description: categoryDescription(name),
		}
	}

	for _, variable := range resp.Variables {
		categoryName := strings.TrimSpace(variable.Category)
		if categoryName == "" {
			categoryName = "other"
		}

		category, ok := categoriesByName[categoryName]
		if !ok {
			order = append(order, categoryName)
			category = &DescribeIndicatorsCategory{
				Name:        categoryName,
				Description: categoryDescription(categoryName),
			}
			categoriesByName[categoryName] = category
		}

		category.Variables = append(category.Variables, DescribeIndicatorsVariable{
			Name:        variable.Name,
			Type:        variable.Type,
			Description: variable.Description,
			ValidRange:  validRangeForVariable(variable),
			Guard:       stringPointer(recommendedGuards[variable.Name]),
			Requires:    stringPointer(variable.Requires),
			PerpOnly:    variable.PerpOnly,
		})
	}

	categories := make([]DescribeIndicatorsCategory, 0, len(order))
	for _, name := range order {
		category := categoriesByName[name]
		if category == nil || len(category.Variables) == 0 {
			continue
		}
		categories = append(categories, *category)
	}

	return DescribeIndicatorsResult{
		SchemaVersion: contractVersion,
		ContractType:  effectiveContractType,
		Categories:    categories,
		Examples:      describeIndicatorsExamples(effectiveContractType),
		Notes:         describeIndicatorsNotes(effectiveContractType),
	}
}

func normalizeDescribeIndicatorsContractType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func categoryDescription(name string) string {
	switch name {
	case "price":
		return "Current-candle price fields."
	case "indicators":
		return "Technical indicator values and perp-specific pricing signals."
	case "volume":
		return "Per-candle volume fields."
	case "derived":
		return "Derived ratios, percent moves, and trend booleans."
	case "previous":
		return "Previous-candle values and deltas."
	case "metadata":
		return "Symbol and market metadata."
	case "safety":
		return "has_* flags for guarding optional metrics."
	default:
		return "Additional CEL variables."
	}
}

func validRangeForVariable(variable CELVariableInfo) string {
	switch variable.Name {
	case "rsi_14", "rsi_20", "prev_rsi_14", "prev_rsi_20":
		return "0 to 100"
	case "rsi_14_normalized":
		return "0.0 to 1.0"
	case "daily_move_pct":
		return "signed percentage"
	case "delta_rsi_14":
		return "-100 to 100"
	case "delta_close", "delta_volume_quote":
		return "signed numeric delta"
	case "close_to_ma50", "close_to_ma200":
		return "positive ratio; 1.0 means exactly on the moving average"
	case "funding_rate", "mark_premium":
		return "signed decimal"
	case "symbol":
		return `pair symbol, e.g. "BTCUSDT"`
	case "exchange":
		return `exchange slug, e.g. "binance"`
	case "contract_type":
		return `"spot" or "perp"`
	}

	switch variable.Type {
	case "bool":
		return "true or false"
	case "string":
		return "non-empty string"
	default:
		return "non-negative number"
	}
}

func describeIndicatorsExamples(contractType string) []DescribeIndicatorsExample {
	all := []DescribeIndicatorsExample{
		{
			Name:         "oversold_bounce",
			Description:  "RSI is oversold while price holds above the 50-period moving average.",
			Expression:   "has_rsi_14 && has_ma_50 && rsi_14 < 30 && close > ma_50",
			ContractType: "both",
		},
		{
			Name:         "volume_breakout",
			Description:  "A strong daily move backed by meaningful quote volume.",
			Expression:   "has_volume_quote && volume_quote > 100000 && daily_move_pct > 5",
			ContractType: "both",
		},
		{
			Name:         "trend_pullback",
			Description:  "Bullish trend with price staying close to the 50-period moving average.",
			Expression:   "has_ma_50 && has_ma_200 && bullish_trend && close_to_ma50 >= 0.99 && close_to_ma50 <= 1.01",
			ContractType: "both",
		},
		{
			Name:         "previous_candle_reversal",
			Description:  "RSI recovers after an oversold previous candle.",
			Expression:   "has_prev_rsi_14 && has_rsi_14 && prev_rsi_14 < 30 && rsi_14 >= 30",
			ContractType: "both",
		},
		{
			Name:         "positive_funding_bias",
			Description:  "Perpetual contracts with positive funding and available funding data.",
			Expression:   "contract_type == \"perp\" && has_funding_rate && funding_rate > 0.001",
			ContractType: "perp",
		},
	}

	if contractType != "spot" {
		return all
	}

	filtered := make([]DescribeIndicatorsExample, 0, len(all))
	for _, example := range all {
		if example.ContractType == "perp" {
			continue
		}
		filtered = append(filtered, example)
	}
	return filtered
}

func describeIndicatorsNotes(contractType string) []string {
	notes := []string{
		"Check the matching has_* flag before comparing optional metrics such as RSI, moving averages, previous-candle values, deltas, and perp signals.",
		"Expressions must evaluate to a boolean and should avoid banned helpers like matches(), split(), and join().",
	}

	if contractType == "perp" {
		notes = append(notes, `Use contract_type == "perp"`+" when you want to pin an expression to perpetual markets.")
	}

	return notes
}

func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
