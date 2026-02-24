package upal

import "errors"

var (
	ErrAlreadyArchived = errors.New("session is already archived")
	ErrNotArchived     = errors.New("session is not archived")
	ErrMustBeArchived  = errors.New("session must be archived before deletion")
)
