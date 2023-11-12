package usecase

import "errors"

var (
	ErrInvalidURL    = errors.New("provided URL is invalid")
	ErrIncompleteURL = errors.New("provided URL is incomplete (e.g. missing scheme or host)")
	ErrUIDConflict   = errors.New("shortlink UID conflict")
	ErrDBUnavailable = errors.New("database is unavailable")
)
