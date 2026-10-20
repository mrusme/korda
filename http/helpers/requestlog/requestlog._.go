package requestlog

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

const Redacted = "redacted"

var Secrets = []string{"sessionKey", "apiKey", "key", "slug"}

type Config struct {
	DefaultLevel     slog.Level
	ClientErrorLevel slog.Level
	ServerErrorLevel slog.Level
}

func query(c fiber.Ctx) string {
	var b strings.Builder
	for k, v := range c.Request().URI().QueryArgs().All() {
		if b.Len() > 0 {
			b.WriteByte('&')
		}
		b.Write(k)
		b.WriteByte('=')
		if slices.Contains(Secrets, string(k)) {
			b.WriteString(Redacted)
			continue
		}
		b.Write(v)
	}

	return b.String()
}

func responseLength(c fiber.Ctx) int {
	if c.Response().IsBodyStream() {
		return c.Response().Header.ContentLength()
	}

	return len(c.Response().Body())
}

func New(logger *slog.Logger, cfg Config) fiber.Handler {
	var (
		once       sync.Once
		errHandler fiber.ErrorHandler
	)

	return func(c fiber.Ctx) error {
		once.Do(func() {
			errHandler = c.App().ErrorHandler
		})

		start := time.Now()
		path := c.Path()
		q := query(c)

		err := c.Next()
		if err != nil {
			if err = errHandler(c, err); err != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		end := time.Now()
		status := c.Response().StatusCode()

		request := []slog.Attr{
			slog.Time("time", start.UTC()),
			slog.String("method", c.Method()),
			slog.String("host", c.Hostname()),
			slog.String("path", path),
			slog.String("query", q),
			slog.String("route", c.Route().Path),
			slog.Int("length", c.Request().Header.ContentLength()),
		}
		response := []slog.Attr{
			slog.Time("time", end.UTC()),
			slog.Duration("latency", end.Sub(start)),
			slog.Int("status", status),
			slog.Int("length", responseLength(c)),
		}

		level, msg := cfg.DefaultLevel, "Incoming request"
		if status >= http.StatusBadRequest {
			level = cfg.ClientErrorLevel
			if status >= http.StatusInternalServerError {
				level = cfg.ServerErrorLevel
			}
			if msg = http.StatusText(status); msg == "" {
				msg = fmt.Sprintf("HTTP error: %d", status)
			}
		}

		logger.LogAttrs(c, level, msg,
			slog.Attr{Key: "request", Value: slog.GroupValue(request...)},
			slog.Attr{Key: "response", Value: slog.GroupValue(response...)},
		)

		return err
	}
}
