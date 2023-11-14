package repository

import "errors"

var (
	ErrURLConflict = errors.New("long URL already exists")
)
