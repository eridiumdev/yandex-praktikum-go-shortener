package usecase

import "errors"

var ErrIDConflict = errors.New("failed to generate shortlink because of ID conflict")
