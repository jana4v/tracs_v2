package ingest

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/mainframe/tm-system/internal/models"
)

// WriteToUnifiedMap writes a parameter value to the unified TM_MAP in Redis,
// applying the SRS Section 9.2 fallback/suffix rules:
//
//   - TM chains (TM1, TM2, ...): write directly to TM_MAP without suffix.
//     Both TM1 and TM2 write to the same key; the last writer wins, which
//     is the expected behaviour for primary chain failover.
//
//   - SMON chains: SMON1 writes directly to TM_MAP (primary SCOS source).
//     SMON2 and above write with a _SMON<N> suffix so downstream consumers
//     can distinguish redundant SCOS sources.
//
//   - ADC chains: ADC1 writes directly to TM_MAP.
//     ADC2 and above write with a _ADC<N> suffix.
func WriteToUnifiedMap(ctx context.Context, pipe redis.Pipeliner, chainName, param, value string) {
	upper := strings.ToUpper(chainName)

	switch {
	// TM chains always write directly (old code: both TM1 and TM2 write to TM_MAP)
	case strings.HasPrefix(upper, "TM"):
		pipe.HSet(ctx, models.TMMap, param, value)

	// SMON1 is the primary SCOS chain — writes directly
	case upper == "SMON1":
		pipe.HSet(ctx, models.TMMap, param, value)

	// SMON2+ write with suffix so values don't overwrite the primary
	case strings.HasPrefix(upper, "SMON"):
		suffixedParam := param + "_" + upper
		pipe.HSet(ctx, models.TMMap, suffixedParam, value)

	// ADC1 is the primary ADC chain — writes directly
	case upper == "ADC1":
		pipe.HSet(ctx, models.TMMap, param, value)

	// ADC2+ write with suffix
	case strings.HasPrefix(upper, "ADC"):
		suffixedParam := param + "_" + upper
		pipe.HSet(ctx, models.TMMap, suffixedParam, value)
	}
}
