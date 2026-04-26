package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// VerifyExpiredExpectations checks each just-expired SuppressionEntry that declared
// an ExpectedValue. If the current TM value does not satisfy the declared expectation,
// an EXPECTED_VIOLATION is written to TM_LIMIT_FAILURES_MAP.
//
// This is called in Monitor.Check() after LoadSuppressions returns the expired slice
// and before CleanupExpiredSuppressions removes those entries from Redis.
func VerifyExpiredExpectations(
	ctx            context.Context,
	logger         *slog.Logger,
	tmData         map[string]string,
	expired        []cleanupItem,
	writeViolation func(context.Context, string, *LimitViolation),
) {
	now := time.Now().UTC().Format(time.RFC3339)

	for _, item := range expired {
		if item.Entry.ExpectedValue == "" {
			// No expected outcome was declared — nothing to verify.
			continue
		}

		current, ok := tmData[item.Mnemonic]
		if !ok {
			// Mnemonic not present in TM_MAP; cannot verify — skip silently.
			continue
		}

		if !meetsExpectation(current, item.Entry.Operator, item.Entry.ExpectedValue) {
			logger.Warn("EXPECTED outcome not met after suppression window",
				"mnemonic", item.Mnemonic,
				"operator", item.Entry.Operator,
				"expected", item.Entry.ExpectedValue,
				"actual", current,
				"request_id", item.Entry.RequestID,
				"procedure_id", item.Entry.ProcedureID,
			)
			writeViolation(ctx, item.Mnemonic, &LimitViolation{
				Mnemonic:  item.Mnemonic,
				Type:      "EXPECTED_VIOLATION",
				Value:     current,
				Expected:  fmt.Sprintf("%s %s", item.Entry.Operator, item.Entry.ExpectedValue),
				Timestamp: now,
			})
		} else {
			logger.Debug("EXPECTED outcome verified after suppression window",
				"mnemonic", item.Mnemonic,
				"operator", item.Entry.Operator,
				"expected", item.Entry.ExpectedValue,
				"actual", current,
			)
		}
	}
}

// meetsExpectation returns true when current satisfies (operator expected).
// String equality/inequality is compared as-is; all other operators require
// both values to be parseable as float64.
func meetsExpectation(current, operator, expected string) bool {
	switch operator {
	case "==":
		return current == expected
	case "!=":
		return current != expected
	}

	a, errA := strconv.ParseFloat(current, 64)
	b, errB := strconv.ParseFloat(expected, 64)
	if errA != nil || errB != nil {
		return false
	}

	switch operator {
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case "<":
		return a < b
	}

	return false
}
