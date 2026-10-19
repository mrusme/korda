package deadline

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestNewSetsTheDeadlineOfTheRequestContext(t *testing.T) {
	app := fiber.New()
	app.Use(New(time.Second))

	var deadline time.Time
	var ok bool
	app.Get("/", func(c fiber.Ctx) error {
		deadline, ok = c.Context().Deadline()
		return nil
	})

	start := time.Now()
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("the request = %v, %v", resp, err)
	}
	if !ok || deadline.Before(start.Add(time.Second)) || deadline.After(time.Now().Add(time.Second)) {
		t.Errorf("the deadline = %v, %v, want one second after %v", deadline, ok, start)
	}
}

func TestNewWithoutATimeoutSetsNoDeadline(t *testing.T) {
	app := fiber.New()
	app.Use(New(0))

	ok := true
	app.Get("/", func(c fiber.Ctx) error {
		_, ok = c.Context().Deadline()
		return nil
	})

	if _, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil)); err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("a timeout of zero set a deadline")
	}
}
