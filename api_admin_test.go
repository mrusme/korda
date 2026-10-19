package main

import (
	"slices"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"xn--gckvb8fzb.com/glides/services/database"
)

const userAdminShape = `{"id":"<uuid>","email":"%s","name":"%s","profileImagePath":"","avatarColor":"%s","profileChangedAt":"<time>","clusterGroupId":"<uuid>","storageLabel":%s,"shouldChangePassword":%s,"isAdmin":false,"createdAt":"<time>","deletedAt":%s,"updatedAt":"<time>","oauthId":"","quotaSizeInBytes":%s,"quotaUsageInBytes":0,"status":"%s","license":null}`

func TestAdminCreatesAndUpdatesUsers(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	auth := bearer(token)

	h.do(fiber.MethodPost, "/api/admin/users", `{"email":"Vera@Example.com","password":"vera-password","name":"Vera"}`, auth...).
		expectShape(t, 201, shape(userAdminShape, "vera@example.com", "Vera", "orange", "null", "true", "null", "null", "active"))
	h.do(fiber.MethodPost, "/api/admin/users", `{"email":"vera@example.com","password":"x","name":"Dup"}`, auth...).
		expect(t, 400, `{"message":"Email is not available"}`)
	h.do(fiber.MethodPost, "/api/admin/users", `{"email":"nopass@example.com","password":"","name":"No"}`, auth...).
		expect(t, 400, `{"message":"password is required"}`)
	juno := h.do(fiber.MethodPost, "/api/admin/users", `{"email":"juno@example.com","password":"juno-password","name":"Juno","storageLabel":"ju/no","quotaSizeInBytes":1073741824,"avatarColor":"blue","shouldChangePassword":false,"isAdmin":false}`, auth...)
	juno.expectShape(t, 201, shape(userAdminShape, "juno@example.com", "Juno", "blue", `"juno"`, "false", "null", "1073741824", "active"))
	h.do(fiber.MethodPost, "/api/admin/users", `{"email":"a@b.c","password":"p","name":"n","quotaSizeInBytes":-1,"pinCode":"12"}`, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^\\d{6}$/","path":["pinCode"],"message":"Invalid string: must match pattern /^\\d{6}$/"},{"origin":"number","code":"too_small","minimum":0,"inclusive":true,"path":["quotaSizeInBytes"],"message":"Too small: expected number to be >=0"}]}`)

	list := h.do(fiber.MethodGet, "/api/admin/users", nil, auth...).array(t)
	emails := []string{}
	for _, u := range list {
		emails = append(emails, u.(map[string]any)["email"].(string))
	}
	if !slices.Equal(emails, []string{"juno@example.com", "vera@example.com", adminEmail}) {
		t.Errorf("admin users = %v, want the newest first", emails)
	}

	vera := list[1].(map[string]any)["id"].(string)
	if filtered := h.do(fiber.MethodGet, "/api/admin/users?id="+vera, nil, auth...).array(t); len(filtered) != 1 {
		t.Errorf("?id= gives %d users", len(filtered))
	}
	h.do(fiber.MethodGet, "/api/admin/users?withDeleted=yes", nil, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["withDeleted"],"message":"Invalid option: expected one of \"true\"|\"false\""}]}`)

	h.do(fiber.MethodPut, "/api/admin/users/"+vera, `{"name":"Vera V","storageLabel":"","quotaSizeInBytes":null}`, auth...).
		expectShape(t, 200, shape(userAdminShape, "vera@example.com", "Vera V", "orange", "null", "true", "null", "null", "active"))
	h.do(fiber.MethodPut, "/api/admin/users/"+vera, `{"storageLabel":"admin"}`, auth...).
		expect(t, 400, `{"message":"Storage label already in use by another account"}`)
	h.do(fiber.MethodPut, "/api/admin/users/"+vera, `{"email":"admin@example.com"}`, auth...).
		expect(t, 400, `{"message":"Email is not available"}`)
	h.do(fiber.MethodPatch, "/api/admin/users/"+vera, `{"storageLabel":"vera"}`, auth...).
		expectShape(t, 200, shape(userAdminShape, "vera@example.com", "Vera V", "orange", `"vera"`, "true", "null", "null", "active"))
	h.do(fiber.MethodPut, "/api/admin/users/not-a-uuid", `{"isAdmin":"x"}`, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["isAdmin"],"message":"Invalid input: expected boolean, received string"}]}`)

	me := h.do(fiber.MethodGet, "/api/users/me", nil, auth...).str(t, "id")
	h.do(fiber.MethodPut, "/api/admin/users/"+me, `{"isAdmin":false}`, auth...).
		expect(t, 400, `{"message":"Admin status can only be changed by another admin"}`)

	h.do(fiber.MethodPut, "/api/admin/users/"+vera, `{"password":"vera-new-password","pinCode":"123456"}`, auth...).expect(t, 200, "")
	veraToken := h.login("vera@example.com", "vera-new-password")
	h.do(fiber.MethodPost, "/api/auth/session/unlock", `{"pinCode":"123456"}`, bearer(veraToken)...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/admin/users", nil, bearer(veraToken)...).expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodPost, "/api/admin/users", `{}`, bearer(veraToken)...).expect(t, 403, `{"message":"Forbidden"}`)
}

func TestAdminDeletesAndRestoresUsers(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	auth := bearer(token)
	me := h.do(fiber.MethodGet, "/api/users/me", nil, auth...).str(t, "id")
	juno := h.createUser(token, "juno@example.com", "Juno", "juno-password")

	h.do(fiber.MethodDelete, "/api/admin/users/"+me, `{}`, auth...).expect(t, 403, `{"message":"Cannot delete your own account"}`)
	h.do(fiber.MethodDelete, "/api/admin/users/"+juno, nil, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}]}`)
	h.do(fiber.MethodDelete, "/api/admin/users/"+juno, `{}`, auth...).
		expectShape(t, 200, shape(userAdminShape, "juno@example.com", "Juno", "pink", "null", "true", `"<time>"`, "null", "deleted"))

	if len(h.queue.ids()) != 0 {
		t.Errorf("a delete without force queued %v", h.queue.ids())
	}
	if list := h.do(fiber.MethodGet, "/api/admin/users", nil, auth...).array(t); len(list) != 1 {
		t.Errorf("admin users without deleted = %d", len(list))
	}
	if list := h.do(fiber.MethodGet, "/api/admin/users?withDeleted=true", nil, auth...).array(t); len(list) != 2 {
		t.Errorf("admin users with deleted = %d", len(list))
	}
	h.do(fiber.MethodGet, "/api/users/"+juno, nil, auth...).expect(t, 400, `{"message":"User not found"}`)
	h.do(fiber.MethodGet, "/api/admin/users/"+juno, nil, auth...).expect(t, 200, "")
	h.do(fiber.MethodPost, "/api/auth/login", `{"email":"juno@example.com","password":"juno-password"}`).expect(t, 401, "")

	h.do(fiber.MethodPost, "/api/admin/users/"+juno+"/restore", nil, auth...).
		expectShape(t, 200, shape(userAdminShape, "juno@example.com", "Juno", "pink", "null", "true", "null", "null", "active"))
	h.login("juno@example.com", "juno-password")

	h.do(fiber.MethodDelete, "/api/admin/users/"+juno, `{"force":true}`, auth...).
		expectShape(t, 200, shape(userAdminShape, "juno@example.com", "Juno", "pink", "null", "true", `"<time>"`, "null", "removing"))
	if ids := h.queue.ids(); !slices.Equal(ids, []string{"delete-user:" + juno}) {
		t.Errorf("queued = %v", ids)
	}
	h.runJobs()

	h.do(fiber.MethodGet, "/api/admin/users/"+juno, nil, auth...).expect(t, 400, `{"message":"User not found"}`)
	audited, err := database.Value[int](h.ctx, database.From(h.rt), `SELECT COUNT(*) FROM user_audit WHERE user_id = $1`, uuid.MustParse(juno))
	if err != nil || audited != 1 {
		t.Errorf("user_audit rows = %d, %v", audited, err)
	}

	h.do(fiber.MethodDelete, "/api/admin/users/"+juno, `{}`, auth...).expect(t, 400, `{"message":"User not found"}`)
	h.do(fiber.MethodPost, "/api/admin/users/"+juno+"/restore", nil, auth...).expect(t, 400, `{"message":"User not found"}`)
}

func TestNightlyUserDeletion(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	auth := bearer(token)
	due := h.createUser(token, "due@example.com", "Due", "due-password")
	recent := h.createUser(token, "recent@example.com", "Recent", "recent-password")

	for _, id := range []string{due, recent} {
		h.do(fiber.MethodDelete, "/api/admin/users/"+id, `{}`, auth...).expect(t, 200, "")
	}
	if _, err := database.Exec(h.ctx, database.From(h.rt), `UPDATE users SET deleted_at = $1 WHERE id = $2`,
		time.Now().UTC().AddDate(0, 0, -8), uuid.MustParse(due)); err != nil {
		t.Fatal(err)
	}

	if err := queueDueDeletions(h); err != nil {
		t.Fatal(err)
	}
	if ids := h.queue.ids(); !slices.Equal(ids, []string{"delete-user:" + due}) {
		t.Fatalf("queued = %v, want only the user deleted eight days ago", ids)
	}
	h.runJobs()

	h.do(fiber.MethodGet, "/api/admin/users/"+due, nil, auth...).expect(t, 400, `{"message":"User not found"}`)
	h.do(fiber.MethodGet, "/api/admin/users/"+recent, nil, auth...).expect(t, 200, "")
}

func TestAdminUserDetails(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	auth := bearer(token)
	vera := h.createUser(token, "vera@example.com", "Vera", "vera-password")
	h.login("vera@example.com", "vera-password")
	missing := "00000000-0000-4000-8000-000000000000"

	h.do(fiber.MethodGet, "/api/admin/users/"+vera+"/preferences", nil, auth...).expect(t, 200, defaultPreferences)
	h.do(fiber.MethodPut, "/api/admin/users/"+vera+"/preferences", `{"memories":{"duration":7},"avatar":{"color":"red"},"download":{"archiveSize":1}}`, auth...).
		expect(t, 200, updatedPreferences+`,"avatar":{"color":"red"}}`)
	h.do(fiber.MethodPatch, "/api/admin/users/"+vera+"/preferences", `{}`, auth...).expect(t, 200, updatedPreferences+`}`)
	h.do(fiber.MethodGet, "/api/admin/users/"+missing+"/preferences", nil, auth...).expect(t, 400, `{"message":"User not found"}`)

	if sessions := h.do(fiber.MethodGet, "/api/admin/users/"+vera+"/sessions", nil, auth...).array(t); len(sessions) != 1 ||
		sessions[0].(map[string]any)["current"] != false {
		t.Errorf("vera's sessions = %v", sessions)
	}
	h.do(fiber.MethodGet, "/api/admin/users/"+missing+"/sessions", nil, auth...).expect(t, 200, `[]`)

	h.do(fiber.MethodGet, "/api/admin/users/"+vera+"/statistics", nil, auth...).expect(t, 200, `{"images":0,"videos":0,"total":0}`)
	h.do(fiber.MethodGet, "/api/admin/users/"+missing+"/statistics?isFavorite=true&visibility=timeline", nil, auth...).
		expect(t, 200, `{"images":0,"videos":0,"total":0}`)
	h.do(fiber.MethodGet, "/api/admin/users/not-a-uuid/statistics?visibility=nope&isTrashed=1", nil, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["archive","timeline","hidden","locked"],"path":["visibility"],"message":"Invalid option: expected one of \"archive\"|\"timeline\"|\"hidden\"|\"locked\""},{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["isTrashed"],"message":"Invalid option: expected one of \"true\"|\"false\""}]}`)

	h.do(fiber.MethodGet, "/api/admin/users/"+vera+"/calendar-heatmap?from=2026-10-01&to=2026-10-01", nil, auth...).
		expect(t, 200, `{"from":"2026-10-01","to":"2026-10-01","series":[{"date":"2026-10-01","count":0}],"totalCount":0}`)
	h.do(fiber.MethodGet, "/api/admin/users/"+missing+"/calendar-heatmap", nil, auth...).expect(t, 400, `{"message":"User not found"}`)
	h.do(fiber.MethodGet, "/api/admin/users/not-a-uuid/calendar-heatmap?type=x", nil, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["Upload","Taken"],"path":["type"],"message":"Invalid option: expected one of \"Upload\"|\"Taken\""}]}`)
}
