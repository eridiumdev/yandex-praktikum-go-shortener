package entity

const AuthTokenCtxKey = AuthTokenKey("auth-token")

type (
	AuthToken struct {
		UserID string
	}
	AuthTokenKey string
)
