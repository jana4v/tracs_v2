package internal

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// ModeBHandler implements Time Delta Mode (SRS 6.3.2).
// On mismatch, it waits for CHAIN_COMPARE_DELAY_SECONDS, then re-reads
// both values. If still mismatched after the delay, the mismatch is confirmed
// and written to TM_CHAIN_MISMATCHES_MAP.
type ModeBHandler struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// NewModeBHandler creates a new Mode B handler.
func NewModeBHandler(rdb *redis.Client, logger *slog.Logger) *ModeBHandler {
	return &ModeBHandler{
		rdb:    rdb,
		logger: logger.With("mode", "B"),
	}
}

// Confirm attempts to confirm a mismatch using Time Delta Mode.
// It reads CHAIN_COMPARE_DELAY_SECONDS from config, waits, then re-evaluates.
func (h *ModeBHandler) Confirm(ctx context.Context, pair ChainPair, mnem models.TmMnemonic, mismatch Mismatch) {
	// Read delay from TM_SOFTWARE_CFG_MAP
	delayStr, err := h.rdb.HGet(ctx, models.TMSoftwareCfgMap, models.CfgChainCompareDelaySeconds).Result()
	if err != nil {
		h.logger.Debug("cannot read CHAIN_COMPARE_DELAY_SECONDS, using default 1s", "error", err)
		delayStr = "1"
	}

	delaySec, err := strconv.Atoi(delayStr)
	if err != nil {
		h.logger.Warn("invalid CHAIN_COMPARE_DELAY_SECONDS value, using default 1s", "value", delayStr)
		delaySec = 1
	}

	// Wait for the configured delay
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(delaySec) * time.Second):
	}

	// Re-read values from both chains
	chain1Key := models.ChainMapKeyByName(pair.Chain1)
	chain2Key := models.ChainMapKeyByName(pair.Chain2)

	id := string(mnem.ID)
	v1, err := h.rdb.HGet(ctx, chain1Key, id).Result()
	if err != nil {
		h.logger.Warn("failed to re-read chain1 value after delay", "mnemonic", id, "error", err)
		return
	}

	v2, err := h.rdb.HGet(ctx, chain2Key, id).Result()
	if err != nil {
		h.logger.Warn("failed to re-read chain2 value after delay", "mnemonic", id, "error", err)
		return
	}

	// Re-evaluate match
	if h.valuesMatch(mnem, v1, v2) {
		h.logger.Debug("mismatch resolved after delay (transient)",
			"mnemonic", id,
			"delay_seconds", delaySec,
		)
		return
	}

	// Still mismatched after delay — confirmed
	mismatch.Value1 = v1
	mismatch.Value2 = v2
	WriteMismatch(ctx, h.rdb, mismatch, h.logger)
}

// valuesMatch checks value match using the same logic as the main comparator.
func (h *ModeBHandler) valuesMatch(mnem models.TmMnemonic, v1, v2 string) bool {
	if mnem.IsAnalog() {
		f1, err1 := strconv.ParseFloat(v1, 64)
		f2, err2 := strconv.ParseFloat(v2, 64)
		if err1 != nil || err2 != nil {
			return v1 == v2
		}
		return math.Abs(f1-f2) <= 2*float64(mnem.Tolerance)
	}
	return v1 == v2
}
