package internal

import (
	"strconv"

	"github.com/mainframe/tm-system/internal/models"
)

// AnalogCheck checks whether the given value falls within the mnemonic's configured
// range. Range[0] is the minimum and Range[1] is the maximum. Returns a LimitViolation
// if the value is out of range, or nil if within limits or actively suppressed by an
// EXPECTED pre-declaration.
func AnalogCheck(value string, mnemonic models.TmMnemonic, suppressions SuppressionMap) *LimitViolation {
	id := string(mnemonic.ID)
	if IsSuppressed(suppressions, id) {
		return nil
	}

	rangeStr := mnemonic.GetRangeStr()
	if len(rangeStr) < 2 {
		return nil
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}

	minVal, err := strconv.ParseFloat(rangeStr[0], 64)
	if err != nil {
		return nil
	}

	maxVal, err := strconv.ParseFloat(rangeStr[1], 64)
	if err != nil {
		return nil
	}

	if val < minVal || val > maxVal {
		return FormatAnalogViolation(id, value, rangeStr[0], rangeStr[1])
	}

	return nil
}
