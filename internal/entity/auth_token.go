package entity

import "encoding/hex"

type AuthToken struct {
	UserID []byte
}

func (t *AuthToken) String() string {
	return hex.EncodeToString(t.UserID)
}
