package main

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

func TestSessions(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	auth := bearer(token)

	created := h.do(fiber.MethodPost, "/api/sessions", `{"deviceOS":"Cast","deviceType":"Chromecast","duration":3600}`, auth...)
	created.expectShape(t, 201, `{"id":"<uuid>","createdAt":"<time>","updatedAt":"<time>","expiresAt":"<time>","current":false,"appVersion":null,"deviceOS":"Cast","deviceType":"Chromecast","isPendingSyncReset":false,"token":"<token>"}`)
	child := created.str(t, "id")
	childToken := created.str(t, "token")

	h.do(fiber.MethodGet, "/api/auth/status", nil, bearer(childToken)...).
		expectShape(t, 200, `{"pinCode":false,"password":true,"isElevated":false,"expiresAt":"<time>"}`)

	list := h.do(fiber.MethodGet, "/api/sessions", nil, auth...).array(t)
	if len(list) != 2 || list[0].(map[string]any)["id"] != child || list[1].(map[string]any)["current"] != true {
		t.Errorf("sessions = %v, want the child first and the caller's marked current", list)
	}

	h.do(fiber.MethodPut, "/api/sessions/"+child, `{"isPendingSyncReset":true}`, auth...).
		expectShape(t, 200, `{"id":"<uuid>","createdAt":"<time>","updatedAt":"<time>","expiresAt":"<time>","current":false,"appVersion":null,"deviceOS":"Cast","deviceType":"Chromecast","isPendingSyncReset":true}`)
	h.do(fiber.MethodPatch, "/api/sessions/"+child, `{"isPendingSyncReset":false}`, auth...).expect(t, 200, "")
	h.do(fiber.MethodPut, "/api/sessions/"+child, `{}`, auth...).expect(t, 400, `{"message":"No fields to update"}`)
	h.do(fiber.MethodPut, "/api/sessions/not-a-uuid", `{"isPendingSyncReset":"x"}`, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["isPendingSyncReset"],"message":"Invalid input: expected boolean, received string"}]}`)
	h.do(fiber.MethodPost, "/api/sessions/"+child+"/lock", nil, auth...).expect(t, 204, "")
	h.do(fiber.MethodPost, "/api/sessions", `{"duration":0}`, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["duration"],"message":"Too small: expected number to be >=1"}]}`)

	if _, err := database.Exec(h.ctx, database.From(h.rt), `UPDATE sessions SET expires_at = $1 WHERE id = $2`,
		time.Now().UTC().Add(-time.Minute), uuid.MustParse(child)); err != nil {
		t.Fatal(err)
	}
	h.do(fiber.MethodGet, "/api/auth/status", nil, bearer(childToken)...).expect(t, 401, `{"message":"Invalid user token"}`)

	h.do(fiber.MethodDelete, "/api/sessions/00000000-0000-4000-8000-000000000000", nil, auth...).
		expect(t, 400, `{"message":"Not found or no authDevice.delete access"}`)
	h.do(fiber.MethodDelete, "/api/sessions/"+child, nil, auth...).expect(t, 204, "")

	h.createUser(token, "vera@example.com", "Vera", "vera-password")
	vera := bearer(h.login("vera@example.com", "vera-password"))
	mine := h.do(fiber.MethodGet, "/api/sessions", nil, auth...).array(t)[0].(map[string]any)["id"].(string)
	h.do(fiber.MethodDelete, "/api/sessions/"+mine, nil, vera...).
		expect(t, 400, `{"message":"Not found or no authDevice.delete access"}`)
	h.do(fiber.MethodPut, "/api/sessions/"+mine, `{"isPendingSyncReset":true}`, vera...).
		expect(t, 400, `{"message":"Not found or no session.update access"}`)
	h.do(fiber.MethodPost, "/api/sessions/"+mine+"/lock", nil, vera...).
		expect(t, 400, `{"message":"Not found or no session.lock access"}`)

	second := h.login(adminEmail, adminPassword)
	h.do(fiber.MethodDelete, "/api/sessions", nil, auth...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/auth/status", nil, bearer(second)...).expect(t, 401, `{"message":"Invalid user token"}`)
	h.do(fiber.MethodGet, "/api/auth/status", nil, auth...).expect(t, 200, "")
}

func TestSessionCleanup(t *testing.T) {
	h := newHarness(t)
	token := h.admin()

	child := h.do(fiber.MethodPost, "/api/sessions", `{"duration":1}`, bearer(token)...).str(t, "id")
	if _, err := database.Exec(h.ctx, database.From(h.rt), `UPDATE sessions SET expires_at = $1 WHERE id = $2`,
		time.Now().UTC().Add(-time.Minute), uuid.MustParse(child)); err != nil {
		t.Fatal(err)
	}

	if err := cleanupSessions(h); err != nil {
		t.Fatal(err)
	}

	if list := h.do(fiber.MethodGet, "/api/sessions", nil, bearer(token)...).array(t); len(list) != 1 {
		t.Errorf("after the cleanup %d sessions are left, want the caller's", len(list))
	}
}
