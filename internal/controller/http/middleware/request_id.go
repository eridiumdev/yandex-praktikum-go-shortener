package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

func RequestID(log *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Pre-processing

		return nil
	}
}
