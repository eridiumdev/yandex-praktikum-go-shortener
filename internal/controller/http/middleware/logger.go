package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

func Logger(log *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Pre-processing
		ctx := c.Context()

		// Prepare request clone
		req := new(fasthttp.Request)
		c.Request().CopyTo(req)

		reqMsg := log.Info(ctx).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Bytes("request", req.Body())

		if strings.Contains(string(req.Header.ContentEncoding()), "application/json") {
			reqMsg.RawJSON("request", req.Body())
		} else {
			reqMsg.Bytes("request", req.Body())
		}
		reqMsg.Msg("-> New request")

		// Go to next middleware/handler
		if err := c.Next(); err != nil {
			return err
		}

		// Post-processing
		// Prepare response clone
		resp := new(fasthttp.Response)
		c.Response().CopyTo(resp)

		code := resp.StatusCode()
		var respMsg *zerolog.Event

		switch {
		case code >= 400 && code < 500:
			respMsg = log.Warn(ctx)
		case code >= 500:
			respMsg = log.Error(ctx, errors.New(http.StatusText(code)))
		default:
			respMsg = log.Info(ctx)
		}

		respMsg = log.Info(ctx).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("code", code).
			Str("status", http.StatusText(code))

		if strings.Contains(string(resp.Header.ContentEncoding()), "application/json") {
			respMsg.RawJSON("response", req.Body())
		} else {
			respMsg.Bytes("response", req.Body())
		}
		respMsg.Msg("<- Request handled")

		return nil
	}
}
