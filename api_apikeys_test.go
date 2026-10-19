package main

import (
	"testing"

	"github.com/gofiber/fiber/v3"
)

const keyShape = `{"id":"<uuid>","name":"%s","createdAt":"<time>","updatedAt":"<time>","permissions":%s}`

func TestAPIKeys(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	auth := bearer(token)

	created := h.do(fiber.MethodPost, "/api/api-keys", `{"permissions":["user.read"]}`, auth...)
	created.expectShape(t, 201, `{"id":"<uuid>","name":"API Key","createdAt":"<time>","updatedAt":"<time>","permissions":["user.read"],"secret":"<token>","apiKey":`+shape(keyShape, "API Key", `["user.read"]`)+`}`)
	id := created.str(t, "id")
	secret := created.str(t, "secret")

	h.do(fiber.MethodPost, "/api/api-keys", `{"name":"none"}`, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"array","code":"invalid_type","path":["permissions"],"message":"Invalid input: expected array, received undefined"}]}`)
	h.do(fiber.MethodGet, "/api/api-keys/"+id, nil, auth...).expectShape(t, 200, shape(keyShape, "API Key", `["user.read"]`))
	h.do(fiber.MethodGet, "/api/api-keys", nil, auth...).expectShape(t, 200, `[`+shape(keyShape, "API Key", `["user.read"]`)+`]`)
	h.do(fiber.MethodGet, "/api/api-keys/00000000-0000-4000-8000-000000000000", nil, auth...).
		expect(t, 400, `{"message":"API Key not found"}`)

	h.do(fiber.MethodGet, "/api/users/me", nil, "x-api-key", secret).expect(t, 200, "")
	h.do(fiber.MethodGet, "/api/sessions", nil, "x-api-key", secret).expect(t, 403, `{"message":"Missing required permission: session.read"}`)
	h.do(fiber.MethodGet, "/api/api-keys/me", nil, "x-api-key", secret).expectShape(t, 200, shape(keyShape, "API Key", `["user.read"]`))
	h.do(fiber.MethodGet, "/api/api-keys/me", nil, auth...).expect(t, 403, `{"message":"Not authenticated with an API Key"}`)

	h.do(fiber.MethodPut, "/api/api-keys/"+id, `{"name":"renamed"}`, auth...).expectShape(t, 200, shape(keyShape, "renamed", `["user.read"]`))
	h.do(fiber.MethodPatch, "/api/api-keys/"+id, `{"permissions":["user.read","session.read"]}`, auth...).
		expectShape(t, 200, shape(keyShape, "renamed", `["user.read","session.read"]`))
	h.do(fiber.MethodGet, "/api/sessions", nil, "x-api-key", secret).expect(t, 200, "")

	rotated := h.do(fiber.MethodPost, "/api/api-keys/"+id+"/rotate", nil, auth...)
	rotated.expectShape(t, 201, `{"id":"<uuid>","name":"renamed","createdAt":"<time>","updatedAt":"<time>","permissions":["user.read","session.read"],"secret":"<token>","apiKey":`+shape(keyShape, "renamed", `["user.read","session.read"]`)+`}`)
	h.do(fiber.MethodGet, "/api/users/me", nil, "x-api-key", secret).expect(t, 401, `{"message":"Invalid API key"}`)
	h.do(fiber.MethodGet, "/api/users/me", nil, "x-api-key", rotated.str(t, "secret")).expect(t, 200, "")
	h.do(fiber.MethodPost, "/api/api-keys/00000000-0000-4000-8000-000000000000/rotate", nil, auth...).
		expect(t, 400, `{"message":"API Key not found not found"}`)

	limited := h.apiKey(token, "apiKey.create", "apiKey.rotate", "apiKey.update")
	h.do(fiber.MethodPost, "/api/api-keys", `{"permissions":["all"]}`, "x-api-key", limited).
		expect(t, 400, `{"message":"Cannot grant permissions you do not have"}`)
	h.do(fiber.MethodPost, "/api/api-keys", `{"permissions":["apiKey.create"]}`, "x-api-key", limited).expect(t, 201, "")
	h.do(fiber.MethodPut, "/api/api-keys/"+id, `{"permissions":["all"]}`, "x-api-key", limited).
		expect(t, 400, `{"message":"Cannot grant permissions you do not have"}`)
	h.do(fiber.MethodPost, "/api/api-keys/"+id+"/rotate", nil, "x-api-key", limited).
		expect(t, 400, `{"message":"Cannot rotate an API Key with permissions you do not have"}`)
	h.do(fiber.MethodPost, "/api/sessions", `{}`, "x-api-key", h.apiKey(token, "session.create")).
		expect(t, 400, `{"message":"This endpoint can only be used with a session token"}`)

	h.do(fiber.MethodDelete, "/api/api-keys/"+id, nil, auth...).expect(t, 204, "")
	h.do(fiber.MethodDelete, "/api/api-keys/"+id, nil, auth...).expect(t, 400, `{"message":"API Key not found"}`)

	h.createUser(token, "vera@example.com", "Vera", "vera-password")
	vera := bearer(h.login("vera@example.com", "vera-password"))
	h.do(fiber.MethodGet, "/api/api-keys", nil, vera...).expect(t, 200, `[]`)
}
