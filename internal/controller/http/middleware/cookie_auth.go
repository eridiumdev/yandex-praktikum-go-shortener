package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
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

func CookieAuth(cfg CookieAuthConfig, log *logger.Logger) (fiber.Handler, error) {
	return func(c *fiber.Ctx) error {
		var token *entity.AuthToken
		var err error

		// Pre-processing
		cookie := c.Cookies(CookieAuthName)
		if cookie != "" {
			// Ignore the error and generate new token
			token, _ = cfg.Cipher.Decrypt(c.Context(), cookie)
		}
		if token == nil {
			token, err = generateToken()
			if err != nil {
				return log.Wrap(err, "generate token")
			}
		}

		// Add token to request context
		c.SetUserContext(
			context.WithValue(c.UserContext(), entity.AuthTokenCtxKey, token))

		// Go to next middleware/handler
		if err := c.Next(); err != nil {
			return err
		}

		// Post-processing
		// Encrypt the token
		encrypted, err := cfg.Cipher.Encrypt(c.Context(), token)
		if err != nil {
			return log.Wrap(err, "encrypt token")
		}

		// Add encrypted token as a cookie
		c.Cookie(&fiber.Cookie{
			Name:    CookieAuthName,
			Value:   encrypted,
			Expires: time.Now().Add(CookieAuthAge),
		})

		return nil
	}, nil
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
