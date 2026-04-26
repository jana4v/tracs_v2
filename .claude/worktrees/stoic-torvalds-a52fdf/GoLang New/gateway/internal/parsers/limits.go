package parsers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseLimitsPayload parses a raw JSON limits value into a []any slice.
// Accepted formats:
//
//   - JSON array:  [1, 10]  or  ["OFF", "ON"]
//   - CSV string:  "1,10"   or  "OFF,ON"
//   - Scalar:      42
func ParseLimitsPayload(raw json.RawMessage) ([]any, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("limits cannot be empty")
	}

	if strings.HasPrefix(trimmed, "[") {
		var arr []any
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("invalid limits array")
		}
		if len(arr) == 0 {
			return nil, fmt.Errorf("limits cannot be an empty array")
		}
		for i := range arr {
			if s, ok := arr[i].(string); ok {
				arr[i] = strings.TrimSpace(s)
			}
		}
		return arr, nil
	}

	var csv string
	if err := json.Unmarshal(raw, &csv); err == nil {
		parts := strings.Split(csv, ",")
		limits := make([]any, 0, len(parts))
		for _, part := range parts {
			token := strings.TrimSpace(part)
			if token == "" {
				continue
			}
			if v, err := strconv.ParseFloat(token, 64); err == nil {
				limits = append(limits, v)
			} else {
				limits = append(limits, token)
			}
		}
		if len(limits) == 0 {
			return nil, fmt.Errorf("limits cannot be empty")
		}
		return limits, nil
	}

	var single any
	if err := json.Unmarshal(raw, &single); err == nil {
		return []any{single}, nil
	}

	return nil, fmt.Errorf("invalid limits format; use \"1,10\", \"a,b,c\" or JSON array")
}

// ParseAvailableChains parses a raw JSON value into a slice of chain integers.
// Accepted formats: [1,2], "1,2", or a single integer.
func ParseAvailableChains(raw json.RawMessage) ([]int, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("value cannot be empty")
	}

	var intArray []int
	if err := json.Unmarshal(raw, &intArray); err == nil {
		if len(intArray) == 0 {
			return nil, fmt.Errorf("value cannot be an empty array")
		}
		return intArray, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var mixed []any
		if err := json.Unmarshal(raw, &mixed); err == nil {
			if len(mixed) == 0 {
				return nil, fmt.Errorf("value cannot be an empty array")
			}
			out := make([]int, 0, len(mixed))
			for _, m := range mixed {
				switch v := m.(type) {
				case float64:
					out = append(out, int(v))
				case string:
					n, err := strconv.Atoi(strings.TrimSpace(v))
					if err != nil {
						return nil, fmt.Errorf("value array must contain only integers")
					}
					out = append(out, n)
				default:
					return nil, fmt.Errorf("value array must contain only integers")
				}
			}
			return out, nil
		}
	}

	var csv string
	if err := json.Unmarshal(raw, &csv); err == nil {
		parts := strings.Split(csv, ",")
		out := make([]int, 0, len(parts))
		for _, part := range parts {
			t := strings.TrimSpace(part)
			if t == "" {
				continue
			}
			n, err := strconv.Atoi(t)
			if err != nil {
				return nil, fmt.Errorf("value must be integer list like \"1,2\" or [1,2]")
			}
			out = append(out, n)
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("value cannot be empty")
		}
		return out, nil
	}

	return nil, fmt.Errorf("invalid value format; use \"1,2\" or [1,2]")
}
