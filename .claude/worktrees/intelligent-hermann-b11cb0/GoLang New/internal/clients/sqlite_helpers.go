package clients

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNoRows is re-exported so callers don't need to import database/sql directly.
var ErrNoRows = sql.ErrNoRows

// IsNoRows returns true when err indicates no rows were found.
func IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// IsDuplicateKey returns true when err is a SQLite UNIQUE constraint violation.
func IsDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return len(msg) >= 24 && msg[:24] == "UNIQUE constraint failed:" ||
		containsStr(msg, "UNIQUE constraint failed")
}

func containsStr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// MarshalJSON serialises v to a JSON string.
func MarshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalJSON deserialises raw JSON string into v.
func UnmarshalJSON(raw string, v any) error {
	return json.Unmarshal([]byte(raw), v)
}

// NewID returns a new random UUID string.
func NewID() string {
	return uuid.NewString()
}

// TimeToUnix converts a time.Time to a Unix epoch (seconds).
func TimeToUnix(t time.Time) int64 {
	return t.Unix()
}

// UnixToTime converts a Unix epoch (seconds) to a UTC time.Time.
func UnixToTime(ts int64) time.Time {
	return time.Unix(ts, 0).UTC()
}
