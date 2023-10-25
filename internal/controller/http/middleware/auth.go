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

func CookieAuth(cfg CookieAuthConfig) (fiber.Handler, error) {
	return func(c *fiber.Ctx) error {
		var token *entity.AuthToken
		var err error

		// Pre-processing
		cookie := c.Cookies(CookieAuthName)
		if cookie != "" {
			token, err = cfg.Cipher.Decrypt(c.Context(), cookie)
			if err != nil {
				// Ignore the error and generate new token
			}
		}
		if token == nil {
			token, err = generateToken()
			if err != nil {
				return errors.Wrapf(err, "[cookie-auth] generate token")
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
			return errors.Wrapf(err, "[cookie-auth] encrypt token")
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
	// Generate random bytes for the user ID
	userID := make([]byte, 16)
	_, err := rand.Read(userID)
	if err != nil {
		return nil, errors.Wrapf(err, "preparing random bytes for userID")
	}

	return &entity.AuthToken{
		UserID: hex.EncodeToString(userID),
	}, nil
}
