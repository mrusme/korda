package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestOAuthDisabled(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)

	h.do(fiber.MethodPost, "/api/oauth/authorize", `{"redirectUri":""}`).expect(t, 400, `{"message":"OAuth is not enabled"}`)
	h.do(fiber.MethodPost, "/api/oauth/authorize", `{}`).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["redirectUri"],"message":"Invalid input: expected string, received undefined"}]}`)
	h.do(fiber.MethodPost, "/api/oauth/callback", `{"url":"x"}`).expect(t, 400, `{"message":"OAuth is not enabled"}`)
	h.do(fiber.MethodPost, "/api/oauth/callback", `{"url":""}`).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"too_small","minimum":1,"inclusive":true,"path":["url"],"message":"Too small: expected string to have >=1 characters"}]}`)
	h.do(fiber.MethodPost, "/api/oauth/link", `{"url":"x"}`, owner...).expect(t, 400, `{"message":"OAuth state is missing"}`)
	h.do(fiber.MethodPost, "/api/oauth/link", `{"url":"x","state":"s"}`, owner...).expect(t, 400, `{"message":"OAuth code verifier is missing"}`)
	h.do(fiber.MethodPost, "/api/oauth/link", `{"url":"x","state":"s","codeVerifier":"v"}`, owner...).
		expect(t, 500, `{"message":"Error in OAuth discovery: TypeError: Invalid URL"}`)
	if u := h.do(fiber.MethodPost, "/api/oauth/unlink", nil, owner...); u.Status != 200 || u.str(t, "oauthId") != "" {
		t.Errorf("unlink = %d %s", u.Status, u.Body)
	}
	h.do(fiber.MethodPost, "/api/admin/auth/unlink-all", nil, owner...).expect(t, 204, "")

	res := h.do(fiber.MethodGet, "/api/oauth/mobile-redirect?code=abc&state=def", nil)
	if res.Status != 307 || res.Header.Get(fiber.HeaderLocation) != "app.immich:///oauth-callback?code=abc&state=def" ||
		string(res.Body) != "Temporary Redirect. Redirecting to app.immich:///oauth-callback?code=abc&state=def" {
		t.Errorf("mobile redirect = %d %v %s", res.Status, res.Header, res.Body)
	}
	req := httptest.NewRequest(fiber.MethodGet, "/api/oauth/mobile-redirect?code=a%20b&x=1?y=2", nil)
	if res = h.send(req); res.Header.Get(fiber.HeaderLocation) != "app.immich:///oauth-callback?code=a%20b&x=1" {
		t.Errorf("second question mark = %v", res.Header)
	}

	h.do(fiber.MethodPost, "/api/oauth/backchannel-logout", `{"logout_token":"x"}`).
		expect(t, 400, `{"message":"Received backchannel logout request but OAuth is not enabled"}`)
	form := formRequest("/api/oauth/backchannel-logout", "logout_token=x")
	if res = h.send(form); res.Status != 400 || string(res.Body) != `{"message":"Received backchannel logout request but OAuth is not enabled"}` {
		t.Errorf("form logout = %d %s", res.Status, res.Body)
	}
	h.do(fiber.MethodGet, "/api/server/version-check", nil, owner...).expect(t, 200, `{"checkedAt":null,"releaseVersion":null}`)
}
