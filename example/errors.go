package example

import "errors"

//go:generate go run github.com/vasilesk/allgen/cmd/allgen -name AllErrors
var (
	// ErrNotFound means the requested item does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict means the item was changed concurrently.
	ErrConflict = errors.New("conflict")
)
