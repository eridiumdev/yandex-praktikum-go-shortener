package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

func RequestID(log *logger.Logger) gin.HandlerFunc {
	log.RegisterHook(func(ctx context.Context) (string, string) {
		if val, ok := ctx.Value("request_id").(string); ok {
			return "request_id", val
		}
		return "request_id", ""
	})

	return func(c *gin.Context) {
		c.Set("request_id", uuid.NewString())
		c.Next()
	}
}
