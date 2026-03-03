package service

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrValidation         = errors.New("validation error")
	ErrNotFound           = errors.New("entry not found")
	ErrAccessDenied       = errors.New("access denied")
	ErrPayloadTooLarge    = errors.New("payload too large")
)
