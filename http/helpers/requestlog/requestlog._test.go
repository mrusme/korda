package requestlog

import (
	"bytes"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

const streamSize = 4 << 20

var testConfig = fiber.TestConfig{Timeout: 30 * time.Second}

type countingReader struct {
	read atomic.Int64
	left int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	if r.left <= 0 {
		return 0, io.EOF
	}
	n := min(int64(len(p)), r.left)
	r.left -= n
	r.read.Add(n)

	return int(n), nil
}

func logger(out *bytes.Buffer) fiber.Handler {
	l := slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return New(l, Config{DefaultLevel: slog.LevelDebug, ClientErrorLevel: slog.LevelInfo, ServerErrorLevel: slog.LevelError})
}

func TestQuerySecretsAndRefererStayOutOfTheLog(t *testing.T) {
	var out bytes.Buffer
	app := fiber.New()
	app.Use(logger(&out))
	app.Get("/assets", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusUnauthorized)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/assets?sessionKey=s3cr3t-session&apiKey=s3cr3t-key&key=s3cr3t-share&slug=s3cr3t-slug&size=preview", nil)
	req.Header.Set(fiber.HeaderReferer, "https://example.com/share/s3cr3t-referer")
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}

	logged := out.String()
	for _, secret := range []string{"s3cr3t-session", "s3cr3t-key", "s3cr3t-share", "s3cr3t-slug", "s3cr3t-referer"} {
		if strings.Contains(logged, secret) {
			t.Errorf("the log contains %q: %s", secret, logged)
		}
	}
	want := `"query":"sessionKey=redacted&apiKey=redacted&key=redacted&slug=redacted&size=preview"`
	if !strings.Contains(logged, want) || !strings.Contains(logged, `"status":401`) {
		t.Errorf("log = %s, want %s", logged, want)
	}
}

func TestStreamedBodiesAreNotRead(t *testing.T) {
	var out bytes.Buffer
	request := &countingReader{left: streamSize}
	response := &countingReader{left: streamSize}
	var readBeforeWrite [2]int64

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		err := c.Next()
		readBeforeWrite = [2]int64{request.read.Load(), response.read.Load()}
		return err
	})
	app.Use(logger(&out))
	app.Post("/upload", func(c fiber.Ctx) error {
		c.Request().SetBodyStream(request, -1)
		return c.SendStatus(fiber.StatusRequestEntityTooLarge)
	})
	app.Get("/original", func(c fiber.Ctx) error {
		return c.SendStream(response, streamSize)
	})

	if _, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/upload", nil), testConfig); err != nil {
		t.Fatal(err)
	}
	if readBeforeWrite[0] != 0 {
		t.Errorf("the request log read %d bytes of a rejected request body", readBeforeWrite[0])
	}

	if _, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/original", nil), testConfig); err != nil {
		t.Fatal(err)
	}
	if readBeforeWrite[1] != 0 {
		t.Errorf("the request log read %d bytes of a streamed response", readBeforeWrite[1])
	}
	if !strings.Contains(out.String(), `"length":4194304`) {
		t.Errorf("log = %s, want the declared length of the stream", out.String())
	}
}
