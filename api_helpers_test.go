package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	logicadminusers "tty.fail/mrus/korda/logic/root/admin/users"
	logicserver "tty.fail/mrus/korda/logic/root/server"
	logicsessions "tty.fail/mrus/korda/logic/root/sessions"
)

func recordVersion(h *apiHarness) error {
	return logicserver.RecordVersion(h.ctx, h.rt)
}

func formRequest(path string, body string) *http.Request {
	req := httptest.NewRequest(fiber.MethodPost, path, strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	return req
}

func fileExists(h *apiHarness, relative string) bool {
	path, err := gh.MediaRoot(h.rt).Abs(relative)
	if err != nil {
		return false
	}

	_, err = os.Stat(path)
	return err == nil
}

func cleanupSessions(h *apiHarness) error {
	return logicsessions.Cleanup(h.ctx, h.rt)
}

func queueDueDeletions(h *apiHarness) error {
	return logicadminusers.QueueDueDeletions(h.ctx, h.rt)
}

func nullUUID() uuid.NullUUID {
	return uuid.NullUUID{}
}
