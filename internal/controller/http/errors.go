package http

import "errors"

var (
	ErrUnauthenticatedUser = errors.New("user is not authenticated")
)
