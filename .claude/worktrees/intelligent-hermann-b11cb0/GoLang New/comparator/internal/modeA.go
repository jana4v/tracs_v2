package internal

import (
	"context"
	"log/slog"
	"math"
	"strconv"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// ModeAHandler implements Frame ID Sync Mode (SRS 6.3.1).
// On initial mismatch, it records the frame IDs and waits until both chains
// reach the same frame ID. Then it re-evaluates: if still mismatched, the
// mismatch is confirmed and written to TM_CHAIN_MISMATCHES_MAP.
// If resolved, the mismatch is discarded as transient.
type ModeAHandler struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// NewModeAHandler creates a new Mode A handler.
func NewModeAHandler(rdb *redis.Client, logger *slog.Logger) *ModeAHandler {
	return &ModeAHandler{
		rdb:    rdb,
		logger: logger.With("mode", "A"),
	}
}

// Confirm attempts to confirm a mismatch using Frame ID Sync Mode.
// It reads both chain maps to check for matching frame IDs, then re-evaluates the values.
func (h *ModeAHandler) Confirm(ctx context.Context, pair ChainPair, mnem models.TmMnemonic, mismatch Mismatch) {
	chain1Key := models.ChainMapKeyByName(pair.Chain1)
	chain2Key := models.ChainMapKeyByName(pair.Chain2)

	// Read frame IDs from both chains (using "FRAME_ID" field in the chain map)
	frameID1, err := h.rdb.HGet(ctx, chain1Key, "FRAME_ID").Result()
	if err != nil {
		h.logger.Debug("cannot read frame ID for chain1, confirming mismatch directly",
			"chain", pair.Chain1, "error", err)
		WriteMismatch(ctx, h.rdb, mismatch, h.logger)
		return
	}

	frameID2, err := h.rdb.HGet(ctx, chain2Key, "FRAME_ID").Result()
	if err != nil {
		h.logger.Debug("cannot read frame ID for chain2, confirming mismatch directly",
			"chain", pair.Chain2, "error", err)
		WriteMismatch(ctx, h.rdb, mismatch, h.logger)
		return
	}

	// If frame IDs don't match yet, skip confirmation for now (will re-check next cycle)
	if frameID1 != frameID2 {
		h.logger.Debug("frame IDs not synchronized, deferring confirmation",
			"mnemonic", string(mnem.ID),
			"frameID1", frameID1,
			"frameID2", frameID2,
		)
		return
	}

	// Frame IDs match — re-read values for final evaluation
	id := string(mnem.ID)
	v1, err := h.rdb.HGet(ctx, chain1Key, id).Result()
	if err != nil {
		h.logger.Warn("failed to re-read chain1 value", "mnemonic", id, "error", err)
		return
	}

	v2, err := h.rdb.HGet(ctx, chain2Key, id).Result()
	if err != nil {
		h.logger.Warn("failed to re-read chain2 value", "mnemonic", id, "error", err)
		return
	}

	// Re-evaluate match
	if valuesMatchForMode(mnem, v1, v2) {
		h.logger.Debug("mismatch resolved at same frame ID (transient)",
			"mnemonic", id,
			"frameID", frameID1,
		)
		return
	}

	// Still mismatched at same frame ID — confirmed
	mismatch.Value1 = v1
	mismatch.Value2 = v2
	WriteMismatch(ctx, h.rdb, mismatch, h.logger)
}

// valuesMatchForMode checks value match using the same logic as the main comparator.
func valuesMatchForMode(mnem models.TmMnemonic, v1, v2 string) bool {
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
