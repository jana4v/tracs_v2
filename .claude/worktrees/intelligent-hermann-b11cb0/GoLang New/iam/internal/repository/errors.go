package repository

import "errors"

// Sentinel errors returned by all repository functions.
var (
	// ErrNotFound is returned when the requested document does not exist.
	ErrNotFound = errors.New("not found")

	// ErrDuplicateKey is returned when a unique-index constraint is violated.
	ErrDuplicateKey = errors.New("duplicate key")
)
