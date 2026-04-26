package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// LimitViolation represents a detected limit violation for a mnemonic.
type LimitViolation struct {
	Mnemonic  string `json:"mnemonic"`
	Type      string `json:"type"`
	Value     string `json:"value"`
	Min       string `json:"min,omitempty"`
	Max       string `json:"max,omitempty"`
	Expected  string `json:"expected,omitempty"`
	Timestamp string `json:"timestamp"`
}

// Monitor performs limit checking against the TM_MAP data in Redis.
type Monitor struct {
	rdb    *redis.Client
	loader *MnemonicLoader
	logger *slog.Logger
}

// NewMonitor creates a new limit monitor.
func NewMonitor(rdb *redis.Client, loader *MnemonicLoader, logger *slog.Logger) *Monitor {
	return &Monitor{
		rdb:    rdb,
		loader: loader,
		logger: logger.With("component", "limit-monitor"),
	}
}

// Check reads TM_MAP from Redis and checks each mnemonic with enable_limit=true
// against its configured limits. Violations are written to TM_LIMIT_FAILURES_MAP,
// and resolved violations are cleared.
func (m *Monitor) Check(ctx context.Context) {
	// Read all telemetry values from TM_MAP
	tmData, err := m.rdb.HGetAll(ctx, models.TMMap).Result()
	if err != nil {
		m.logger.Error("failed to read TM_MAP", "error", err)
		return
	}

	if len(tmData) == 0 {
		return
	}

	// Read expected digital states map
	expectedStates, err := m.rdb.HGetAll(ctx, models.TMExpectedDigitalStatesMap).Result()
	if err != nil {
		m.logger.Warn("failed to read expected digital states", "error", err)
		expectedStates = make(map[string]string)
	}

	// Load active EXPECTED suppressions for this check cycle.
	// Mnemonics with active suppression entries (written by Julia before a SEND command)
	// are skipped entirely until the post-command master-frame window expires.
	suppressions, expiredEntries := LoadSuppressions(ctx, m.rdb, m.logger)

	mnemonics := m.loader.Get()
	now := time.Now().UTC().Format(time.RFC3339)

	for _, mnem := range mnemonics {
		id := string(mnem.ID)
		value, exists := tmData[id]
		if !exists {
			continue
		}

		var violation *LimitViolation

		if mnem.IsAnalog() {
			violation = AnalogCheck(value, mnem, suppressions)
		} else if mnem.IsBinary() {
			violation = DigitalCheck(value, mnem, expectedStates, suppressions)
		}

		if violation != nil {
			violation.Timestamp = now
			m.writeViolation(ctx, id, violation)
		} else {
			m.clearViolation(ctx, id)
		}
	}

	// Check whether expected outcomes were achieved for entries whose TTL just elapsed.
	// Must run before cleanup so the entry data (Operator/ExpectedValue) is still available.
	VerifyExpiredExpectations(ctx, m.logger, tmData, expiredEntries, m.writeViolation)

	// Remove any suppression entries whose TTL has elapsed (crash-safety cleanup).
	CleanupExpiredSuppressions(ctx, m.rdb, m.logger, expiredEntries)
}

// writeViolation writes a limit violation to TM_LIMIT_FAILURES_MAP.
func (m *Monitor) writeViolation(ctx context.Context, mnemonic string, v *LimitViolation) {
	data, err := json.Marshal(v)
	if err != nil {
		m.logger.Error("failed to marshal violation", "mnemonic", mnemonic, "error", err)
		return
	}

	if err := m.rdb.HSet(ctx, models.TMLimitFailuresMap, mnemonic, string(data)).Err(); err != nil {
		m.logger.Error("failed to write violation", "mnemonic", mnemonic, "error", err)
		return
	}

	m.logger.Debug("limit violation recorded",
		"mnemonic", mnemonic,
		"type", v.Type,
		"value", v.Value,
	)
}

// clearViolation removes a resolved violation from TM_LIMIT_FAILURES_MAP.
func (m *Monitor) clearViolation(ctx context.Context, mnemonic string) {
	// Only attempt delete; ignore if key doesn't exist
	deleted, err := m.rdb.HDel(ctx, models.TMLimitFailuresMap, mnemonic).Result()
	if err != nil {
		m.logger.Error("failed to clear violation", "mnemonic", mnemonic, "error", err)
		return
	}

	if deleted > 0 {
		m.logger.Debug("limit violation cleared", "mnemonic", mnemonic)
	}
}

// formatViolation is a helper that creates a LimitViolation with common fields.
func formatViolation(mnemonic string, violationType, value string) *LimitViolation {
	return &LimitViolation{
		Mnemonic: mnemonic,
		Type:     violationType,
		Value:    value,
	}
}

// FormatAnalogViolation creates an analog limit violation with range info.
func FormatAnalogViolation(mnemonic, value, min, max string) *LimitViolation {
	v := formatViolation(mnemonic, "ANALOG", value)
	v.Min = min
	v.Max = max
	return v
}

// FormatDigitalViolation creates a digital limit violation with expected state.
func FormatDigitalViolation(mnemonic, value, expected string) *LimitViolation {
	v := formatViolation(mnemonic, "DIGITAL", value)
	v.Expected = fmt.Sprintf("expected: %s", expected)
	return v
}
