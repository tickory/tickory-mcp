package mcp

import "testing"

func TestDescribeIndicatorsArgsValidate(t *testing.T) {
	tests := []struct {
		name    string
		args    DescribeIndicatorsArgs
		wantErr bool
	}{
		{name: "default contract type", args: DescribeIndicatorsArgs{}},
		{name: "spot", args: DescribeIndicatorsArgs{ContractType: "spot"}},
		{name: "perp", args: DescribeIndicatorsArgs{ContractType: "perp"}},
		{name: "invalid", args: DescribeIndicatorsArgs{ContractType: "options"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestBuildDescribeIndicatorsResultGroupsVariablesAndAddsGuards(t *testing.T) {
	resp := &CELVariableReferenceResponse{
		Categories: []string{"price", "indicators", "derived", "safety"},
		Variables: []CELVariableInfo{
			{Name: "rsi_14", Type: "double", Description: "14-period RSI (0-100)", Category: "indicators", Requires: "RSI-14 ready"},
			{Name: "close_to_ma50", Type: "double", Description: "Ratio of close to MA-50", Category: "derived", Requires: "MA-50 ready"},
			{Name: "funding_rate", Type: "double", Description: "Perpetual funding rate", Category: "indicators", PerpOnly: true},
			{Name: "has_funding_rate", Type: "bool", Description: "True if funding_rate is available (perp only)", Category: "safety"},
		},
	}

	result := buildDescribeIndicatorsResult("perp", resp)
	if result.SchemaVersion != contractVersion {
		t.Fatalf("expected schema version %q, got %q", contractVersion, result.SchemaVersion)
	}
	if result.ContractType != "perp" {
		t.Fatalf("expected contract type perp, got %q", result.ContractType)
	}

	rsi14 := findDescribeIndicatorsVariable(t, result, "rsi_14")
	if rsi14.Guard == nil || *rsi14.Guard != "has_rsi_14" {
		t.Fatalf("expected rsi_14 guard has_rsi_14, got %+v", rsi14.Guard)
	}
	if rsi14.ValidRange != "0 to 100" {
		t.Fatalf("unexpected rsi_14 valid range: %q", rsi14.ValidRange)
	}
	if rsi14.Requires == nil || *rsi14.Requires != "RSI-14 ready" {
		t.Fatalf("unexpected rsi_14 requires: %+v", rsi14.Requires)
	}

	closeToMA50 := findDescribeIndicatorsVariable(t, result, "close_to_ma50")
	if closeToMA50.Guard == nil || *closeToMA50.Guard != "has_ma_50" {
		t.Fatalf("expected close_to_ma50 guard has_ma_50, got %+v", closeToMA50.Guard)
	}
	if closeToMA50.ValidRange != "positive ratio; 1.0 means exactly on the moving average" {
		t.Fatalf("unexpected close_to_ma50 valid range: %q", closeToMA50.ValidRange)
	}

	fundingRate := findDescribeIndicatorsVariable(t, result, "funding_rate")
	if fundingRate.Guard == nil || *fundingRate.Guard != "has_funding_rate" {
		t.Fatalf("expected funding_rate guard has_funding_rate, got %+v", fundingRate.Guard)
	}
	if !fundingRate.PerpOnly {
		t.Fatal("expected funding_rate to be marked perp_only")
	}
	if fundingRate.ValidRange != "signed decimal" {
		t.Fatalf("unexpected funding_rate valid range: %q", fundingRate.ValidRange)
	}

	hasFundingRate := findDescribeIndicatorsVariable(t, result, "has_funding_rate")
	if hasFundingRate.Guard != nil {
		t.Fatalf("expected no guard for has_funding_rate, got %+v", hasFundingRate.Guard)
	}
	if hasFundingRate.ValidRange != "true or false" {
		t.Fatalf("unexpected has_funding_rate valid range: %q", hasFundingRate.ValidRange)
	}

	if len(result.Examples) == 0 {
		t.Fatal("expected example expressions")
	}
	if len(result.Notes) == 0 {
		t.Fatal("expected guidance notes")
	}
}

func TestBuildDescribeIndicatorsResultFiltersPerpExamplesForSpot(t *testing.T) {
	result := buildDescribeIndicatorsResult("spot", &CELVariableReferenceResponse{})

	for _, example := range result.Examples {
		if example.ContractType == "perp" {
			t.Fatalf("did not expect perp-only example in spot output: %+v", example)
		}
	}
}

func findDescribeIndicatorsVariable(t *testing.T, result DescribeIndicatorsResult, name string) DescribeIndicatorsVariable {
	t.Helper()

	for _, category := range result.Categories {
		for _, variable := range category.Variables {
			if variable.Name == name {
				return variable
			}
		}
	}

	t.Fatalf("variable %q not found", name)
	return DescribeIndicatorsVariable{}
}
