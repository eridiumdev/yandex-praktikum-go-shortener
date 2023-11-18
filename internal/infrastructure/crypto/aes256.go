package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

var ErrInvalidSecretSize = errors.New("secret must be 16 or 32 bytes long")

type AES256 struct {
	cipher cipher.AEAD
	log    *logger.Logger
}

func NewAES256(secret string, log *logger.Logger) (*AES256, error) {
	if len(secret)%aes.BlockSize != 0 {
		return nil, ErrInvalidSecretSize
	}

	aesblock, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return nil, log.Wrap(err, "prepare aesblock")
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return nil, log.Wrap(err, "prepare aesgcm")
	}

	return &AES256{
		cipher: aesgcm,
		log:    log,
	}, nil
}

func (ae *AES256) Encrypt(ctx context.Context, token *entity.AuthToken) (string, error) {
	// Generate random bytes for the nonce
	nonce := make([]byte, ae.cipher.NonceSize())
	_, err := rand.Read(nonce)
	if err != nil {
		return "", ae.log.Wrap(err, "prepare nonce bytes")
	}

	// Sign the token
	// Note the nonce in the beginning - we will use it during decryption
	signedToken := ae.cipher.Seal(nonce, nonce, []byte(token.UserUID), nil)

	// Return as base64 string
	return base64.StdEncoding.EncodeToString(signedToken), nil
}

func (ae *AES256) Decrypt(ctx context.Context, encrypted string) (*entity.AuthToken, error) {
	// First, decode encrypted string (base64)
	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, ae.log.Wrap(err, "decode token")
	}

	// Cut off nonce bytes
	nonce, signedToken := decoded[:ae.cipher.NonceSize()], decoded[ae.cipher.NonceSize():]

	// Decrypt and verify signature
	original, err := ae.cipher.Open(nil, nonce, signedToken, nil)
	if err != nil {
		return nil, ae.log.Wrap(err, "decrypt token")
	}

	return &entity.AuthToken{
		UserUID: string(original),
	}, nil
}
