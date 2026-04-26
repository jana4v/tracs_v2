package internal

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// RuleEngine implements storage rules per SRS Section 13.4:
// - BINARY: store only on value change
// - ANALOG: store when abs(current - last_stored) > tolerance
// - First sample after data break: always store
// - Last sample before data break: always store
// - Check enable_storage flag per mnemonic
// - Check global TM_STORAGE_ENABLE from TM_SOFTWARE_CFG_MAP
type RuleEngine struct {
	cache  *Cache
	rdb    *redis.Client
	logger *slog.Logger
}

// NewRuleEngine creates a new storage rule engine.
func NewRuleEngine(cache *Cache, rdb *redis.Client, logger *slog.Logger) *RuleEngine {
	return &RuleEngine{
		cache:  cache,
		rdb:    rdb,
		logger: logger.With("component", "rule-engine"),
	}
}

// IsGlobalStorageEnabled checks TM_SOFTWARE_CFG_MAP for the TM_STORAGE_ENABLE flag.
func (r *RuleEngine) IsGlobalStorageEnabled(ctx context.Context) bool {
	candidates := []struct {
		mapKey   string
		fieldKey string
	}{
		{models.SoftwareCfgMap, models.CfgInfluxLoggingEnable},
		{models.SoftwareCfgMap, models.CfgTMStorageEnable},
		{models.TMSoftwareCfgMap, models.CfgInfluxLoggingEnable},
		{models.TMSoftwareCfgMap, models.CfgTMStorageEnable},
	}

	for _, candidate := range candidates {
		val, err := r.rdb.HGet(ctx, candidate.mapKey, candidate.fieldKey).Result()
		if err != nil {
			if err != redis.Nil {
				r.logger.Warn("failed to read storage enable flag", "map", candidate.mapKey, "field", candidate.fieldKey, "error", err)
			}
			continue
		}
		return parseBoolLike(val)
	}

	return true
}

func parseBoolLike(raw string) bool {
	val := strings.ToLower(strings.TrimSpace(raw))
	return val == "1" || val == "true" || val == "yes" || val == "on" || val == "enabled"
}

// ShouldStore evaluates the storage rules for a given mnemonic and value.
// Returns true if the value should be written to InfluxDB.
func (r *RuleEngine) ShouldStore(mnem models.TmMnemonic, value string, now time.Time) bool {
	if !mnem.EnableStorage {
		return false
	}

	entry, exists := r.cache.Get(string(mnem.ID))

	// First sample after data break: always store
	if !exists {
		return true
	}

	// If previous entry was a data break, this is the first sample after break: always store
	if entry.IsBreak {
		return true
	}

	if mnem.IsBinary() {
		// BINARY: store only on value change
		return value != entry.Value
	}

	if mnem.IsAnalog() {
		// ANALOG: store when abs(current - last_stored) > tolerance
		currentVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return true // store if unparseable
		}

		lastVal, err := strconv.ParseFloat(entry.Value, 64)
		if err != nil {
			return true // store if last was unparseable
		}

		return math.Abs(currentVal-lastVal) > float64(mnem.Tolerance)
	}

	// Unknown type — store
	return true
}

// MarkDataBreak marks a mnemonic as having experienced a data break.
// The last sample before break should have already been stored.
// The next sample will be treated as first-after-break and always stored.
func (r *RuleEngine) MarkDataBreak(mnemonic string, ts time.Time) {
	entry, exists := r.cache.Get(mnemonic)
	if exists {
		r.cache.Update(mnemonic, entry.Value, ts, true)
	}
}
