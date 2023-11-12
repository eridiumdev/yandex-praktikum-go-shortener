package entity

const AuthTokenCtxKey = AuthTokenKey("auth-token")

type (
	AuthToken struct {
		UserUID string
	}
	AuthTokenKey string
)
