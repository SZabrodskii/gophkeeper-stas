package repository

import "errors"

// Repository-level sentinel errors.
var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
)
