package main

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

const adminShape = `{"id":"<uuid>","email":"admin@example.com","name":"Admin","profileImagePath":"","avatarColor":"gray","profileChangedAt":"<time>","clusterGroupId":"<uuid>","storageLabel":"admin","shouldChangePassword":true,"isAdmin":true,"createdAt":"<time>","deletedAt":null,"updatedAt":"<time>","oauthId":"","quotaSizeInBytes":null,"quotaUsageInBytes":0,"status":"active","license":null}`

func TestSignUpOnlyOnce(t *testing.T) {
	h := newHarness(t)

	h.do(fiber.MethodPost, "/api/auth/admin-sign-up", `{"email":"x"}`).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"email","pattern":"/^[\\p{L}\\p{M}\\p{N}.!#$%&'*+/=?^_`+"`"+`{|}~-]+@[\\p{L}\\p{N}](?:[\\p{L}\\p{M}\\p{N}-]{0,61}[\\p{L}\\p{M}\\p{N}])?(?:\\.[\\p{L}\\p{N}](?:[\\p{L}\\p{M}\\p{N}-]{0,61}[\\p{L}\\p{M}\\p{N}])?)*$/u","path":["email"],"message":"Invalid input: expected email, received string"},{"expected":"string","code":"invalid_type","path":["password"],"message":"Invalid input: expected string, received undefined"},{"expected":"string","code":"invalid_type","path":["name"],"message":"Invalid input: expected string, received undefined"}]}`)

	h.do(fiber.MethodPost, "/api/auth/admin-sign-up", `{"email":"Admin@Example.com","password":"korda-admin","name":"Admin"}`).
		expectShape(t, 201, adminShape)

	h.do(fiber.MethodPost, "/api/auth/admin-sign-up", `{"email":"other@example.com","password":"x","name":"Other"}`).
		expect(t, 400, `{"message":"Admin setup is not available"}`)
	h.do(fiber.MethodPost, "/api/auth/admin-sign-up", `{"bad":1}`).
		expect(t, 400, `{"message":"Admin setup is not available"}`)
}

func TestLogin(t *testing.T) {
	h := newHarness(t)
	h.signUp()

	h.do(fiber.MethodPost, "/api/auth/login", `{"email":"admin@example.com","password":"wrong"}`).
		expect(t, 401, `{"message":"Incorrect email or password"}`)
	h.do(fiber.MethodPost, "/api/auth/login", `{"email":"nobody@example.com","password":"wrong"}`).
		expect(t, 401, `{"message":"Incorrect email or password"}`)
	h.do(fiber.MethodPost, "/api/auth/login", `{"email":"admin@example.com"}`).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["password"],"message":"Invalid input: expected string, received undefined"}]}`)
	h.do(fiber.MethodPost, "/api/auth/login", `not json`).
		expect(t, 400, `{"message":"Unexpected token 'n', \"not json\" is not valid JSON"}`)

	res := h.do(fiber.MethodPost, "/api/auth/login", `{"email":"ADMIN@Example.com","password":"korda-admin"}`)
	res.expectShape(t, 201, `{"accessToken":"<token>","userId":"<uuid>","userEmail":"admin@example.com","name":"Admin","isAdmin":true,"profileImagePath":"","shouldChangePassword":true,"isOnboarded":false}`)

	token := res.str(t, "accessToken")
	cookies := strings.Join(res.Header.Values(fiber.HeaderSetCookie), "\n")
	for _, want := range []string{"immich_access_token=" + token, "immich_auth_type=password", "immich_is_authenticated=true"} {
		if !strings.Contains(cookies, want) {
			t.Errorf("Set-Cookie = %s, want %s", cookies, want)
		}
	}
	if res.Header.Get("X-Ratelimit-Limit") != "" {
		t.Error("login sends rate limit headers")
	}

	form := h.send(formRequest("/api/auth/login", "email=admin%40example.com&password=korda-admin"))
	if form.Status != 201 {
		t.Errorf("a form login = %d %s", form.Status, form.Body)
	}
}

func TestEveryCredentialForm(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	key := h.apiKey(token, "all")

	ok := `{"pinCode":false,"password":true,"isElevated":false}`
	for name, headers := range map[string][]string{
		"bearer":        bearer(token),
		"user token":    {"x-immich-user-token", token},
		"session token": {"x-immich-session-token", token},
		"cookie":        {"Cookie", "immich_access_token=" + token},
		"api key":       {"x-api-key", key},
	} {
		t.Run(name, func(t *testing.T) {
			h.do(fiber.MethodGet, "/api/auth/status", nil, headers...).expect(t, 200, ok)
		})
	}

	h.do(fiber.MethodGet, "/api/auth/status?sessionKey="+token, nil).expect(t, 200, ok)
	h.do(fiber.MethodGet, "/api/auth/status?apiKey="+key, nil).expect(t, 200, ok)

	h.do(fiber.MethodGet, "/api/users/me", nil).expect(t, 401, `{"message":"Authentication required"}`)
	h.do(fiber.MethodGet, "/api/users/me", nil, bearer("nope")...).expect(t, 401, `{"message":"Invalid user token"}`)
	h.do(fiber.MethodGet, "/api/users/me", nil, "x-api-key", "nope").expect(t, 401, `{"message":"Invalid API key"}`)
	h.do(fiber.MethodGet, "/api/users/me?key=x", nil, bearer(token)...).expect(t, 401, `{"message":"Invalid share key"}`)
	h.do(fiber.MethodGet, "/api/users/me", nil, append(bearer(token), "x-immich-share-slug", "x")...).expect(t, 401, `{"message":"Invalid share slug"}`)
	h.do(fiber.MethodGet, "/api/users/me", nil, append(bearer("nope"), "x-api-key", key)...).expect(t, 401, `{"message":"Invalid user token"}`)

	h.do(fiber.MethodPost, "/api/auth/validateToken", nil, bearer(token)...).expect(t, 200, `{"authStatus":true}`)
}

func TestSessionFollowsTheAppVersion(t *testing.T) {
	h := newHarness(t)
	h.signUp()
	token := h.login(adminEmail, adminPassword, fiber.HeaderUserAgent, "Immich_Android_3.3.3", "devicemodel", "Pixel 9")

	sessions := h.do(fiber.MethodGet, "/api/sessions", nil, append(bearer(token), fiber.HeaderUserAgent, "Immich_Android_3.3.3")...)
	sessions.expectShape(t, 200, `[{"id":"<uuid>","createdAt":"<time>","updatedAt":"<time>","current":true,"appVersion":"3.3.3","deviceOS":"Android","deviceType":"Pixel 9","isPendingSyncReset":false}]`)

	sessions = h.do(fiber.MethodGet, "/api/sessions", nil, append(bearer(token), fiber.HeaderUserAgent, "Immich_Android_3.4.0")...)
	sessions.expectShape(t, 200, `[{"id":"<uuid>","createdAt":"<time>","updatedAt":"<time>","current":true,"appVersion":"3.4.0","deviceOS":"Android","deviceType":"","isPendingSyncReset":false}]`)
}

func TestPinCodeLifecycle(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	auth := bearer(token)

	h.do(fiber.MethodPost, "/api/auth/pin-code", `{"pinCode":"12345"}`, auth...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^\\d{6}$/","path":["pinCode"],"message":"Invalid string: must match pattern /^\\d{6}$/"}]}`)
	h.do(fiber.MethodPost, "/api/auth/session/unlock", `{"pinCode":"123456"}`, auth...).
		expect(t, 400, `{"message":"User does not have a PIN code"}`)
	h.do(fiber.MethodPost, "/api/auth/pin-code", `{"pinCode":"123456"}`, auth...).expect(t, 204, "")
	h.do(fiber.MethodPost, "/api/auth/pin-code", `{"pinCode":"654321"}`, auth...).
		expect(t, 400, `{"message":"User already has a PIN code"}`)
	h.do(fiber.MethodPost, "/api/auth/session/unlock", `{"pinCode":"000000"}`, auth...).
		expect(t, 400, `{"message":"Wrong PIN code"}`)
	h.do(fiber.MethodPost, "/api/auth/session/unlock", `{}`, auth...).
		expect(t, 400, `{"message":"Either password or pinCode is required"}`)
	h.do(fiber.MethodPost, "/api/auth/session/unlock", `{"pinCode":"123456"}`, auth...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/auth/status", nil, auth...).
		expectShape(t, 200, `{"pinCode":true,"password":true,"isElevated":true,"pinExpiresAt":"<time>"}`)
	h.do(fiber.MethodPost, "/api/auth/session/lock", nil, auth...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/auth/status", nil, auth...).
		expect(t, 200, `{"pinCode":true,"password":true,"isElevated":false}`)
	h.do(fiber.MethodPut, "/api/auth/pin-code", `{"pinCode":"123456","newPinCode":"111111"}`, auth...).expect(t, 204, "")
	h.do(fiber.MethodPut, "/api/auth/pin-code", `{"password":"wrong","newPinCode":"222222"}`, auth...).
		expect(t, 400, `{"message":"Wrong password"}`)
	h.do(fiber.MethodPost, "/api/auth/session/unlock", `{"pinCode":"111111"}`, auth...).expect(t, 204, "")
	h.do(fiber.MethodDelete, "/api/auth/pin-code", `{"password":"korda-admin"}`, auth...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/auth/status", nil, auth...).
		expect(t, 200, `{"pinCode":false,"password":true,"isElevated":false}`)

	key := h.apiKey(token, "all")
	h.do(fiber.MethodPost, "/api/auth/session/lock", nil, "x-api-key", key).
		expect(t, 400, `{"message":"This endpoint can only be used with a session token"}`)
}

func TestChangePasswordInvalidatesOtherSessions(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	other := h.login(adminEmail, adminPassword)

	h.do(fiber.MethodPost, "/api/auth/change-password", `{"password":"wrong","newPassword":"korda-admin-2"}`, bearer(token)...).
		expect(t, 400, `{"message":"Wrong password"}`)
	h.do(fiber.MethodPost, "/api/auth/change-password", `{"password":"korda-admin","newPassword":"short"}`, bearer(token)...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"too_small","minimum":8,"inclusive":true,"path":["newPassword"],"message":"Too small: expected string to have >=8 characters"}]}`)

	res := h.do(fiber.MethodPost, "/api/auth/change-password", `{"password":"korda-admin","newPassword":"korda-admin-2","invalidateSessions":false}`, bearer(token)...)
	res.expectShape(t, 200, strings.Replace(adminShape, `"shouldChangePassword":true`, `"shouldChangePassword":false`, 1))

	h.do(fiber.MethodGet, "/api/users/me", nil, bearer(other)...).expect(t, 401, `{"message":"Invalid user token"}`)
	h.do(fiber.MethodGet, "/api/auth/status", nil, bearer(token)...).expect(t, 200, `{"pinCode":false,"password":true,"isElevated":false}`)
	h.login(adminEmail, "korda-admin-2")
}

func TestLogout(t *testing.T) {
	h := newHarness(t)
	token := h.admin()

	res := h.do(fiber.MethodPost, "/api/auth/logout", nil, bearer(token)...)
	res.expect(t, 200, `{"successful":true,"redirectUri":"/auth/login?autoLaunch=0"}`)
	cookies := strings.Join(res.Header.Values(fiber.HeaderSetCookie), "\n")
	for _, want := range []string{"immich_access_token=;", "immich_auth_type=;", "immich_is_authenticated=;", "1970"} {
		if !strings.Contains(cookies, want) {
			t.Errorf("Set-Cookie = %s, want %s", cookies, want)
		}
	}

	h.do(fiber.MethodPost, "/api/auth/logout", nil, bearer(token)...).expect(t, 401, `{"message":"Invalid user token"}`)
}

func TestLoginRateLimit(t *testing.T) {
	h := newHarnessWith(t, harnessOptions{rateLimitAuth: 2, publicUsers: true})
	h.signUp()

	for range 2 {
		h.do(fiber.MethodPost, "/api/auth/login", `{"email":"admin@example.com","password":"wrong"}`).expect(t, 401, "")
	}
	h.do(fiber.MethodPost, "/api/auth/login", `{"email":"admin@example.com","password":"korda-admin"}`).
		expect(t, 429, `{"message":"Too Many Requests"}`)
	h.do(fiber.MethodGet, "/api/server/ping", nil).expect(t, 200, `{"res":"pong"}`)
}
