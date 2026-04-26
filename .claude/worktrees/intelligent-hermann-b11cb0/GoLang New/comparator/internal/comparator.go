package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// ChainPair defines two chains to compare.
type ChainPair struct {
	Chain1 string
	Chain2 string
}

// Mismatch represents a confirmed mismatch between two chain values.
type Mismatch struct {
	Mnemonic  string `json:"mnemonic"`
	Chain1    string `json:"chain1"`
	Chain2    string `json:"chain2"`
	Value1    string `json:"value1"`
	Value2    string `json:"value2"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

// Comparator performs chain-to-chain value comparison for mnemonics
// with enable_comparison=true.
type Comparator struct {
	rdb    *redis.Client
	loader *MnemonicLoader
	pairs  []ChainPair
	logger *slog.Logger
	modeA  *ModeAHandler
	modeB  *ModeBHandler
}

// NewComparator creates a new chain comparator.
func NewComparator(rdb *redis.Client, loader *MnemonicLoader, pairs []ChainPair, logger *slog.Logger) *Comparator {
	return &Comparator{
		rdb:    rdb,
		loader: loader,
		pairs:  pairs,
		logger: logger.With("component", "comparator"),
		modeA:  NewModeAHandler(rdb, logger),
		modeB:  NewModeBHandler(rdb, logger),
	}
}

// Compare reads the comparison mode and performs value comparison for each chain pair.
func (c *Comparator) Compare(ctx context.Context) {
	// Read comparison mode from TM_SOFTWARE_CFG_MAP
	mode, err := c.rdb.HGet(ctx, models.TMSoftwareCfgMap, models.CfgChainCompareMode).Result()
	if err != nil {
		if err != redis.Nil {
			c.logger.Error("failed to read CHAIN_COMPARE_MODE", "error", err)
		}
		mode = models.CompareModeA // default to Mode A
	}

	mnemonics := c.loader.Get()
	if len(mnemonics) == 0 {
		return
	}

	for _, pair := range c.pairs {
		c.comparePair(ctx, pair, mnemonics, mode)
	}
}

// comparePair compares values between two chains for all enabled mnemonics.
func (c *Comparator) comparePair(ctx context.Context, pair ChainPair, mnemonics []models.TmMnemonic, mode string) {
	// Read both chain maps from Redis
	chain1Key := models.ChainMapKeyByName(pair.Chain1)
	chain2Key := models.ChainMapKeyByName(pair.Chain2)

	data1, err := c.rdb.HGetAll(ctx, chain1Key).Result()
	if err != nil {
		c.logger.Error("failed to read chain map", "chain", pair.Chain1, "error", err)
		return
	}

	data2, err := c.rdb.HGetAll(ctx, chain2Key).Result()
	if err != nil {
		c.logger.Error("failed to read chain map", "chain", pair.Chain2, "error", err)
		return
	}

	if len(data1) == 0 || len(data2) == 0 {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	for _, mnem := range mnemonics {
		id := string(mnem.ID)
		v1, ok1 := data1[id]
		v2, ok2 := data2[id]
		if !ok1 || !ok2 {
			continue
		}

		matched := c.valuesMatch(mnem, v1, v2)
		if matched {
			// Clear any existing mismatch for this mnemonic
			c.clearMismatch(ctx, id)
			continue
		}

		// Values don't match — delegate to mode handler for confirmation
		mismatch := Mismatch{
			Mnemonic:  id,
			Chain1:    pair.Chain1,
			Chain2:    pair.Chain2,
			Value1:    v1,
			Value2:    v2,
			Type:      mnem.Type,
			Timestamp: now,
		}

		switch mode {
		case models.CompareModeA:
			c.modeA.Confirm(ctx, pair, mnem, mismatch)
		case models.CompareModeB:
			c.modeB.Confirm(ctx, pair, mnem, mismatch)
		default:
			c.logger.Warn("unknown comparison mode, defaulting to A", "mode", mode)
			c.modeA.Confirm(ctx, pair, mnem, mismatch)
		}
	}
}

// valuesMatch checks whether two values match based on the mnemonic type.
// ANALOG: abs(v1-v2) <= 2*tolerance
// BINARY: exact string match
func (c *Comparator) valuesMatch(mnem models.TmMnemonic, v1, v2 string) bool {
	if mnem.IsAnalog() {
		f1, err1 := strconv.ParseFloat(v1, 64)
		f2, err2 := strconv.ParseFloat(v2, 64)
		if err1 != nil || err2 != nil {
			return v1 == v2 // fallback to string comparison
		}
		return math.Abs(f1-f2) <= 2*float64(mnem.Tolerance)
	}

	// BINARY: exact string match
	return v1 == v2
}

// writeMismatch writes a confirmed mismatch to TM_CHAIN_MISMATCHES_MAP.
func WriteMismatch(ctx context.Context, rdb *redis.Client, mismatch Mismatch, logger *slog.Logger) {
	data, err := json.Marshal(mismatch)
	if err != nil {
		logger.Error("failed to marshal mismatch", "mnemonic", mismatch.Mnemonic, "error", err)
		return
	}

	if err := rdb.HSet(ctx, models.TMChainMismatchesMap, mismatch.Mnemonic, string(data)).Err(); err != nil {
		logger.Error("failed to write mismatch", "mnemonic", mismatch.Mnemonic, "error", err)
		return
	}

	logger.Info("confirmed mismatch recorded",
		"mnemonic", mismatch.Mnemonic,
		"chain1", mismatch.Chain1,
		"chain2", mismatch.Chain2,
		"value1", mismatch.Value1,
		"value2", mismatch.Value2,
	)
}

// clearMismatch removes a resolved mismatch from TM_CHAIN_MISMATCHES_MAP.
func (c *Comparator) clearMismatch(ctx context.Context, mnemonic string) {
	deleted, err := c.rdb.HDel(ctx, models.TMChainMismatchesMap, mnemonic).Result()
	if err != nil {
		c.logger.Error("failed to clear mismatch", "mnemonic", mnemonic, "error", err)
		return
	}
	if deleted > 0 {
		c.logger.Debug("mismatch cleared", "mnemonic", mnemonic)
	}
}

// FormatMismatchKey returns the hash field key for a mismatch entry.
func FormatMismatchKey(mnemonic string) string {
	return fmt.Sprintf("%s", mnemonic)
}
