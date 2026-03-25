package model

import "errors"

// Domain-level sentinel errors.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrAccessDenied  = errors.New("access denied")
)
