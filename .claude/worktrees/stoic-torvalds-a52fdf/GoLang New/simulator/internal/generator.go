package internal

import (
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"

	"github.com/mainframe/tm-system/internal/models"
)

var logger *slog.Logger

func SetLogger(l *slog.Logger) {
	logger = l
}

// GenerateValue produces a simulated value for a mnemonic based on the current mode.
//
// RANDOM mode (SRS 3.3.1):
//   - ANALOG: random float64 between range[0] and range[1]
//   - BINARY: random selection from range array
//
// FIXED mode (SRS 3.3.2):
//   - All mnemonics return range[0]
func GenerateValue(m models.TmMnemonic, mode string) string {
	rangeStr := m.GetRangeStr()
	if len(rangeStr) == 0 {
		return ""
	}

	if mode == models.SimModeFixed {
		return rangeStr[0]
	}

	if logger != nil {
		logger.Debug("generate_value debug", "mode", mode, "mnemonic", m.CdbMnemonic, "type", m.Type, "range_len", len(rangeStr))
	}

	// RANDOM mode
	if m.IsAnalog() && len(rangeStr) >= 2 {
		lo, errLo := strconv.ParseFloat(rangeStr[0], 64)
		hi, errHi := strconv.ParseFloat(rangeStr[1], 64)
		if errLo == nil && errHi == nil && hi > lo {
			val := lo + rand.Float64()*(hi-lo)
			return fmt.Sprintf("%.6f", val)
		}
		if logger != nil {
			logger.Debug("analog parse failed", "lo", rangeStr[0], "hi", rangeStr[1], "errLo", errLo, "errHi", errHi)
		}
		// Fallback: return range[0] if parsing fails
		return rangeStr[0]
	}

	if m.IsBinary() && len(rangeStr) > 0 {
		idx := rand.Intn(len(rangeStr))
		return rangeStr[idx]
	}

	if logger != nil {
		logger.Debug("fallback to range[0]", "mnemonic", m.CdbMnemonic, "type", m.Type, "is_analog", m.IsAnalog(), "is_binary", m.IsBinary())
	}

	return rangeStr[0]
}

// InitialValue returns the initial value for a mnemonic (range[0]).
// Used for FIXED mode initialization and POST /simulator/reset (SRS 3.4.2).
func InitialValue(m models.TmMnemonic) string {
	rangeStr := m.GetRangeStr()
	if len(rangeStr) == 0 {
		return ""
	}
	return rangeStr[0]
}
