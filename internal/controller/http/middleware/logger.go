package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Pre-processing
		// Inject io.TeeReader so we have access to the request body later
		reqBody := bytes.NewBuffer(nil)
		c.Request.Body = io.NopCloser(io.TeeReader(c.Request.Body, reqBody))

		// Inject responseRecorder for the response body
		respRecorder := &responseRecorder{ResponseWriter: c.Writer, body: bytes.NewBuffer(nil)}
		c.Writer = respRecorder

		// Log request, except for the body (not optimal to read now, will log it later)
		log.Info(c).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Msg("-> New request")

		// Go to next middleware/handler
		c.Next()

		// Post-processing
		var msg *zerolog.Event

		select {
		case <-c.Done():
			// Request was canceled
			log.Warn(c).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Int("code", 499).
				Str("status", "Client Closed Request").
				Msg("Request canceled")
			return
		default:
		}

		// Different log level depending on status code
		code := respRecorder.Status()
		switch {
		case code >= 400 && code < 500:
			msg = log.Warn(c)
		case code >= 500:
			msg = log.Error(c, errors.New(http.StatusText(code)))
		default:
			msg = log.Info(c)
		}

		// Add common fields
		msg.Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("code", code).
			Str("status", http.StatusText(code))

		// Add url query
		if c.Request.URL.RawQuery != "" {
			msg.Str("query", c.Request.URL.RawQuery)
		}

		// Add request body
		if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
			msg.RawJSON("request", reqBody.Bytes())
		} else {
			msg.Bytes("request", reqBody.Bytes())
		}

		// Add response body
		if strings.Contains(respRecorder.Header().Get("Content-Type"), "application/json") {
			msg.RawJSON("response", respRecorder.body.Bytes())
		} else {
			msg.Bytes("response", respRecorder.body.Bytes())
		}

		msg.Msg("<- Request handled")
	}
}

type responseRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseRecorder) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
