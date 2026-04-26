package parsers_test

import (
	"encoding/json"
	"testing"

	"github.com/mainframe/tm-system/gateway/internal/parsers"
)

// ── ParseLimitsPayload ────────────────────────────────────────────────────────

func TestParseLimitsPayload_JSONArray(t *testing.T) {
	raw := json.RawMessage(`[1, 100]`)
	got, err := parsers.ParseLimitsPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(got))
	}
}

func TestParseLimitsPayload_JSONStringArray(t *testing.T) {
	raw := json.RawMessage(`["OFF", "ON"]`)
	got, err := parsers.ParseLimitsPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "OFF" || got[1] != "ON" {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestParseLimitsPayload_CSVString(t *testing.T) {
	raw := json.RawMessage(`"1,100"`)
	got, err := parsers.ParseLimitsPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 elements, got %d: %v", len(got), got)
	}
	// Values should have been parsed as float64
	if _, ok := got[0].(float64); !ok {
		t.Fatalf("expected float64, got %T", got[0])
	}
}

func TestParseLimitsPayload_CSVStringLabels(t *testing.T) {
	raw := json.RawMessage(`"OFF,ON,STANDBY"`)
	got, err := parsers.ParseLimitsPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 elements, got %d: %v", len(got), got)
	}
}

func TestParseLimitsPayload_SingleScalar(t *testing.T) {
	raw := json.RawMessage(`42`)
	got, err := parsers.ParseLimitsPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 element, got %d", len(got))
	}
}

func TestParseLimitsPayload_EmptyArray_Error(t *testing.T) {
	raw := json.RawMessage(`[]`)
	_, err := parsers.ParseLimitsPayload(raw)
	if err == nil {
		t.Fatal("expected error for empty array")
	}
}

func TestParseLimitsPayload_EmptyString_Error(t *testing.T) {
	raw := json.RawMessage(`""`)
	_, err := parsers.ParseLimitsPayload(raw)
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParseLimitsPayload_Empty_Error(t *testing.T) {
	raw := json.RawMessage(``)
	_, err := parsers.ParseLimitsPayload(raw)
	if err == nil {
		t.Fatal("expected error for nil raw")
	}
}

// ── ParseAvailableChains ──────────────────────────────────────────────────────

func TestParseAvailableChains_IntArray(t *testing.T) {
	raw := json.RawMessage(`[1, 2]`)
	got, err := parsers.ParseAvailableChains(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestParseAvailableChains_CSVString(t *testing.T) {
	raw := json.RawMessage(`"1,2,3"`)
	got, err := parsers.ParseAvailableChains(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3, got %v", got)
	}
}

func TestParseAvailableChains_MixedArray_FloatsCoerced(t *testing.T) {
	// JSON numbers are float64; they should be coerced to int
	raw := json.RawMessage(`[1.0, 2.0]`)
	got, err := parsers.ParseAvailableChains(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestParseAvailableChains_EmptyArray_Error(t *testing.T) {
	raw := json.RawMessage(`[]`)
	_, err := parsers.ParseAvailableChains(raw)
	if err == nil {
		t.Fatal("expected error for empty array")
	}
}

func TestParseAvailableChains_NonInteger_Error(t *testing.T) {
	raw := json.RawMessage(`"1,abc"`)
	_, err := parsers.ParseAvailableChains(raw)
	if err == nil {
		t.Fatal("expected error for non-integer in CSV")
	}
}

func TestParseAvailableChains_Empty_Error(t *testing.T) {
	raw := json.RawMessage(``)
	_, err := parsers.ParseAvailableChains(raw)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
