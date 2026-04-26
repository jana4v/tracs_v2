package internal

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// SuppressionEntry represents a single expected-TM declaration associated with a SEND command.
// Julia writes these to TM_LIMIT_SUPPRESSION_MAP before enqueuing a command and removes them
// after the post-command master-frame window has elapsed. The limiter reads them each check
// cycle to skip violation detection for suppressed mnemonics.
type SuppressionEntry struct {
	RequestID     string `json:"request_id"`
	ProcedureID   string `json:"procedure_id"`
	Operator      string `json:"operator"`
	ExpectedValue string `json:"expected_value"`
	MnemonicType  string `json:"mnemonic_type"`
	ExpiresAtUnix int64  `json:"expires_at_unix"`
}

// SuppressionMap maps mnemonic ID → slice of active (non-expired) suppression entries.
// Built fresh on every Check() cycle from TM_LIMIT_SUPPRESSION_MAP.
type SuppressionMap map[string][]SuppressionEntry

// cleanupItem pairs a mnemonic with a request_id that needs to be removed from Redis
// because the entry has expired. The full Entry is retained so that
// VerifyExpiredExpectations can check whether the expected outcome was achieved.
type cleanupItem struct {
	Mnemonic  string
	RequestID string
	Entry     SuppressionEntry // full entry for post-expiry verification
}

// LoadSuppressions reads TM_LIMIT_SUPPRESSION_MAP and partitions entries into:
//   - active: non-expired entries per mnemonic (used by the limiter this cycle)
//   - toCleanup: expired (request_id, mnemonic) pairs to remove from Redis
//
// Expired entries have ExpiresAtUnix < now(). They are retained in the returned
// toCleanup slice so CleanupExpiredSuppressions can remove them atomically.
func LoadSuppressions(
	ctx context.Context,
	rdb *redis.Client,
	logger *slog.Logger,
) (SuppressionMap, []cleanupItem) {
	active := make(SuppressionMap)
	var toCleanup []cleanupItem

	raw, err := rdb.HGetAll(ctx, models.TMLimitSuppressionMap).Result()
	if err != nil {
		if err != redis.Nil {
			logger.Warn("failed to read TM_LIMIT_SUPPRESSION_MAP", "error", err)
		}
		return active, toCleanup
	}

	now := time.Now().Unix()

	for mnemonic, arrayJSON := range raw {
		var entries []SuppressionEntry
		if err := json.Unmarshal([]byte(arrayJSON), &entries); err != nil {
			logger.Warn("failed to parse suppression entries",
				"mnemonic", mnemonic, "error", err)
			continue
		}

		var live []SuppressionEntry
		for _, e := range entries {
			if e.ExpiresAtUnix > now {
				live = append(live, e)
			} else {
				toCleanup = append(toCleanup, cleanupItem{
					Mnemonic:  mnemonic,
					RequestID: e.RequestID,
					Entry:     e,
				})
			}
		}
		if len(live) > 0 {
			active[mnemonic] = live
		}
	}

	return active, toCleanup
}

// IsSuppressed returns true if the mnemonic has at least one active suppression entry.
func IsSuppressed(suppressions SuppressionMap, mnemonic string) bool {
	entries, ok := suppressions[mnemonic]
	return ok && len(entries) > 0
}

// luaRemoveByRequestID atomically removes entries matching request_id from the
// mnemonic's JSON array in TM_LIMIT_SUPPRESSION_MAP.
// If the resulting array is empty, the hash field is deleted.
var luaRemoveByRequestID = redis.NewScript(`
local existing = redis.call('HGET', KEYS[1], ARGV[1])
if existing == false then return 0 end
local arr = cjson.decode(existing)
local filtered = {}
for _, v in ipairs(arr) do
    if v.request_id ~= ARGV[2] then
        table.insert(filtered, v)
    end
end
if #filtered == 0 then
    redis.call('HDEL', KEYS[1], ARGV[1])
else
    redis.call('HSET', KEYS[1], ARGV[1], cjson.encode(filtered))
end
return #filtered
`)

// CleanupExpiredSuppressions removes expired suppression entries from TM_LIMIT_SUPPRESSION_MAP.
// Called at the end of every Check() cycle with the toCleanup slice from LoadSuppressions.
// Uses a Lua script per (mnemonic, request_id) pair for atomic read-modify-write.
func CleanupExpiredSuppressions(
	ctx context.Context,
	rdb *redis.Client,
	logger *slog.Logger,
	toCleanup []cleanupItem,
) {
	if len(toCleanup) == 0 {
		return
	}

	for _, item := range toCleanup {
		if err := luaRemoveByRequestID.Run(
			ctx, rdb,
			[]string{models.TMLimitSuppressionMap},
			item.Mnemonic, item.RequestID,
		).Err(); err != nil && err != redis.Nil {
			logger.Warn("failed to cleanup expired suppression",
				"mnemonic", item.Mnemonic,
				"request_id", item.RequestID,
				"error", err)
		} else {
			logger.Debug("cleaned up expired suppression",
				"mnemonic", item.Mnemonic,
				"request_id", item.RequestID)
		}
	}
}
