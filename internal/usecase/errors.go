package usecase

import "errors"

var (
	ErrInvalidURL    = errors.New("provided URL is invalid")
	ErrIncompleteURL = errors.New("provided URL is incomplete (e.g. missing scheme or host)")
	ErrIDConflict    = errors.New("shortlink ID conflict")
	ErrDbUnavailable = errors.New("database is unavailable")
)
