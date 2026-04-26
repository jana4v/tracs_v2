package internal

import (
	"github.com/mainframe/tm-system/internal/models"
)

// DigitalCheck checks whether the given value matches the expected digital state
// for this mnemonic. The expected states are read from TM_EXPECTED_DIGITAL_STATES_MAP.
// If the mnemonic is not in the expected states map, it is skipped (returns nil).
// If the value does not match the expected state, a violation is returned.
// Returns nil without checking if the mnemonic is actively suppressed by an EXPECTED
// pre-declaration.
func DigitalCheck(value string, mnemonic models.TmMnemonic, expectedStates map[string]string, suppressions SuppressionMap) *LimitViolation {
	id := string(mnemonic.ID)
	if IsSuppressed(suppressions, id) {
		return nil
	}

	expected, exists := expectedStates[id]
	if !exists {
		// Mnemonic not in expected states map — skip check
		return nil
	}

	if value != expected {
		return FormatDigitalViolation(id, value, expected)
	}

	return nil
}
