package upal

import "errors"

var (
	ErrAlreadyArchived = errors.New("session is already archived")
	ErrNotArchived     = errors.New("session is not archived")
	ErrInvalidStatus   = errors.New("invalid status for operation")
)
