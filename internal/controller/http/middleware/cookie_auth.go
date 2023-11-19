package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/crypto"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

const (
	CookieAuthName = "Authorization-Token"
	CookieAuthAge  = 24 * time.Hour
)

type (
	CookieAuthConfig struct {
		Cipher crypto.EncryptorDecryptor
	}
)

func CookieAuth(cfg CookieAuthConfig, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token *entity.AuthToken
		var err error

		// Pre-processing
		cookie, err := c.Cookie(CookieAuthName)
		if err == nil && cookie != "" {
			// Get token from cookie
			token, _ = cfg.Cipher.Decrypt(c, cookie)
		}
		if token == nil {
			// Generate new
			token, err = generateToken()
			if err != nil {
				log.Error(c, err).Msg("generate cookie-auth token")
				c.Status(http.StatusInternalServerError)
				return
			}
		}

		// Add token to request context
		c.Set(string(entity.AuthTokenCtxKey), token)

		// Go to next middleware/handler
		c.Next()

		// Post-processing
		// Encrypt the token
		encrypted, err := cfg.Cipher.Encrypt(c, token)
		if err != nil {
			log.Error(c, err).Msg("encrypt cookie-auth token")
			c.Status(http.StatusInternalServerError)
			return
		}

		// Add encrypted token as a cookie
		c.SetCookie(CookieAuthName, encrypted, int(CookieAuthAge.Seconds()), "", "", false, false)
	}
}

func generateToken() (*entity.AuthToken, error) {
	// Generate random bytes for the user UID
	userUID := make([]byte, 16)
	_, err := rand.Read(userUID)
	if err != nil {
		return nil, errors.Wrap(err, "prepare random bytes for userUID")
	}

	return &entity.AuthToken{
		UserUID: hex.EncodeToString(userUID),
	}, nil
}
