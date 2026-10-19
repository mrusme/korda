package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	envReferenceURL      = "KORDA_REFERENCE_URL"
	envReferenceEmail    = "KORDA_REFERENCE_EMAIL"
	envReferencePassword = "KORDA_REFERENCE_PASSWORD"
)

type compareMode int

const (
	compareExact compareMode = iota
	compareShape
)

type step struct {
	name    string
	method  string
	path    string
	body    string
	auth    string
	headers []string
	mode    compareMode
	capture string
}

type side struct {
	name   string
	send   func(method string, path string, body string, headers ...string) apiResponse
	tokens map[string]string
	vars   map[string]string
}

func (s *side) expand(text string) string {
	for k, v := range s.vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}

	return text
}

func (s *side) run(t *testing.T, st step) apiResponse {
	t.Helper()

	headers := append([]string{}, st.headers...)
	switch st.auth {
	case "":
	case "key":
		headers = append(headers, "x-api-key", s.tokens["key"])
	default:
		headers = append(headers, bearer(s.tokens[st.auth])...)
	}

	res := s.send(st.method, s.expand(st.path), s.expand(st.body), headers...)
	if st.capture != "" && res.Status < 300 {
		var body map[string]any
		if err := json.Unmarshal(res.Body, &body); err == nil {
			for _, field := range []string{"id", "secret", "token"} {
				if v, ok := body[field].(string); ok {
					s.vars[st.capture+"."+field] = v
				}
			}
		}
	}

	return res
}

func immichSide(t *testing.T, base string) *side {
	t.Helper()

	client := &http.Client{Timeout: 30 * time.Second}

	return &side{
		name: "immich",
		send: func(method string, path string, body string, headers ...string) apiResponse {
			var reader io.Reader
			if body != "" {
				reader = strings.NewReader(body)
			}
			req, err := http.NewRequest(method, base+path, reader)
			if err != nil {
				t.Fatal(err)
			}
			if body != "" {
				req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			}
			for i := 0; i+1 < len(headers); i += 2 {
				req.Header.Set(headers[i], headers[i+1])
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			return apiResponse{Status: resp.StatusCode, Header: resp.Header, Body: b}
		},
		tokens: map[string]string{},
		vars:   map[string]string{},
	}
}

func kordaSide(h *apiHarness) *side {
	return &side{
		name: "korda",
		send: func(method string, path string, body string, headers ...string) apiResponse {
			var payload any
			if body != "" {
				payload = body
			}
			return h.do(method, path, payload, headers...)
		},
		tokens: map[string]string{},
		vars:   map[string]string{},
	}
}

func shapeOf(raw []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var out strings.Builder
	if err := writeShape(dec, &out); err != nil {
		return "", err
	}

	return out.String(), nil
}

func writeShape(dec *json.Decoder, out *strings.Builder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}

	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			out.WriteByte('{')
			first := true
			for dec.More() {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				if !first {
					out.WriteByte(',')
				}
				first = false
				fmt.Fprintf(out, "%q:", key)
				if err = writeShape(dec, out); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			out.WriteByte('}')
			return err
		case '[':
			out.WriteByte('[')
			if dec.More() {
				if err = writeShape(dec, out); err != nil {
					return err
				}
				for dec.More() {
					var skip json.RawMessage
					if err = dec.Decode(&skip); err != nil {
						return err
					}
				}
			}
			_, err = dec.Token()
			out.WriteByte(']')
			return err
		}
	case string:
		out.WriteString("string")
	case json.Number:
		out.WriteString("number")
	case bool:
		out.WriteString("bool")
	case nil:
		out.WriteString("null")
	}

	return nil
}

func TestConformance(t *testing.T) {
	base := os.Getenv(envReferenceURL)
	email := os.Getenv(envReferenceEmail)
	password := os.Getenv(envReferencePassword)
	if base == "" || email == "" || password == "" {
		t.Skipf("set %s, %s and %s to compare korda with an Immich", envReferenceURL, envReferenceEmail, envReferencePassword)
	}

	immich := immichSide(t, strings.TrimSuffix(base, "/"))
	login := immich.send(fiber.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	if login.Status != fiber.StatusCreated {
		t.Fatalf("login to %s = %d %s", base, login.Status, login.Body)
	}
	immich.tokens["session"] = login.str(t, "accessToken")
	name := immich.send(fiber.MethodGet, "/api/users/me", "", bearer(immich.tokens["session"])...).str(t, "name")
	defer immich.send(fiber.MethodPost, "/api/auth/logout", "", bearer(immich.tokens["session"])...)

	h := newHarness(t)
	korda := kordaSide(h)
	h.do(fiber.MethodPost, "/api/auth/admin-sign-up", map[string]string{"email": email, "password": password, "name": name}).expect(t, 201, "")
	korda.tokens["session"] = h.login(email, password)

	for _, s := range []*side{immich, korda} {
		key := s.send(fiber.MethodPost, "/api/api-keys", `{"name":"conformance","permissions":["user.read"]}`, bearer(s.tokens["session"])...)
		if key.Status != fiber.StatusCreated {
			t.Fatalf("%s api key = %d %s", s.name, key.Status, key.Body)
		}
		s.tokens["key"] = key.str(t, "secret")
		s.vars["readkey.id"] = key.str(t, "id")
	}
	defer immich.send(fiber.MethodDelete, "/api/api-keys/"+immich.vars["readkey.id"], "", bearer(immich.tokens["session"])...)

	steps := []step{
		{name: "ping", method: "GET", path: "/api/server/ping"},
		{name: "version", method: "GET", path: "/api/server/version"},
		{name: "config", method: "GET", path: "/api/server/config"},
		{name: "features", method: "GET", path: "/api/server/features", mode: compareShape},
		{name: "media types", method: "GET", path: "/api/server/media-types"},
		{name: "version history", method: "GET", path: "/api/server/version-history", mode: compareShape},
		{name: "storage", method: "GET", path: "/api/server/storage", auth: "session", mode: compareShape},
		{name: "apk links", method: "GET", path: "/api/server/apk-links", auth: "session"},
		{name: "statistics", method: "GET", path: "/api/server/statistics", auth: "session", mode: compareShape},
		{name: "server license", method: "GET", path: "/api/server/license", auth: "session"},
		{name: "not found", method: "GET", path: "/api/nope"},
		{name: "wrong method", method: "DELETE", path: "/api/server/ping"},
		{name: "sign-up again", method: "POST", path: "/api/auth/admin-sign-up", body: `{"email":"x@example.com","password":"x","name":"X"}`},
		{name: "login wrong password", method: "POST", path: "/api/auth/login", body: fmt.Sprintf(`{"email":%q,"password":"wrong"}`, email)},
		{name: "login unknown", method: "POST", path: "/api/auth/login", body: `{"email":"nobody@example.com","password":"wrong"}`},
		{name: "login invalid", method: "POST", path: "/api/auth/login", body: `{"email":"not-an-email","password":42}`},
		{name: "login empty", method: "POST", path: "/api/auth/login", body: `{}`},
		{name: "me", method: "GET", path: "/api/users/me", auth: "session"},
		{name: "me with key", method: "GET", path: "/api/users/me", auth: "key"},
		{name: "preferences", method: "GET", path: "/api/users/me/preferences", auth: "session"},
		{name: "onboarding", method: "GET", path: "/api/users/me/onboarding", auth: "session"},
		{name: "license", method: "GET", path: "/api/users/me/license", auth: "session"},
		{name: "license invalid", method: "PUT", path: "/api/users/me/license", body: `{"licenseKey":"nope","activationKey":"x"}`, auth: "session"},
		{name: "status", method: "GET", path: "/api/auth/status", auth: "session"},
		{name: "validate", method: "POST", path: "/api/auth/validateToken", auth: "session"},
		{name: "users", method: "GET", path: "/api/users", auth: "session", mode: compareShape},
		{name: "user missing", method: "GET", path: "/api/users/00000000-0000-4000-8000-000000000000", auth: "session"},
		{name: "user invalid", method: "GET", path: "/api/users/not-a-uuid", auth: "session"},
		{name: "no auth", method: "GET", path: "/api/users/me"},
		{name: "invalid token", method: "GET", path: "/api/users/me", headers: bearer("nope")},
		{name: "invalid key", method: "GET", path: "/api/users/me", headers: []string{"x-api-key", "nope"}},
		{name: "share key", method: "GET", path: "/api/users/me?key=x", auth: "session"},
		{name: "missing permission", method: "GET", path: "/api/sessions", auth: "key"},
		{name: "my api key", method: "GET", path: "/api/api-keys/me", auth: "key"},
		{name: "my api key with session", method: "GET", path: "/api/api-keys/me", auth: "session"},
		{name: "api key invalid", method: "POST", path: "/api/api-keys", body: `{"permissions":[]}`, auth: "session"},
		{name: "api key missing", method: "GET", path: "/api/api-keys/00000000-0000-4000-8000-000000000000", auth: "session"},
		{name: "api key rotate missing", method: "POST", path: "/api/api-keys/00000000-0000-4000-8000-000000000000/rotate", auth: "session"},
		{name: "session create invalid", method: "POST", path: "/api/sessions", body: `{"duration":1.5}`, auth: "session"},
		{name: "session create", method: "POST", path: "/api/sessions", body: `{"deviceOS":"Cast","deviceType":"Chromecast","duration":3600}`, auth: "session", capture: "child"},
		{name: "session update", method: "PUT", path: "/api/sessions/{child.id}", body: `{"isPendingSyncReset":true}`, auth: "session"},
		{name: "session update empty", method: "PUT", path: "/api/sessions/{child.id}", body: `{}`, auth: "session"},
		{name: "session lock", method: "POST", path: "/api/sessions/{child.id}/lock", auth: "session"},
		{name: "sessions", method: "GET", path: "/api/sessions", auth: "session", mode: compareShape},
		{name: "session delete", method: "DELETE", path: "/api/sessions/{child.id}", auth: "session"},
		{name: "session delete missing", method: "DELETE", path: "/api/sessions/00000000-0000-4000-8000-000000000000", auth: "session"},
		{name: "api key create", method: "POST", path: "/api/api-keys", body: `{"name":"conformance-2","permissions":["all"]}`, auth: "session", capture: "key2"},
		{name: "api key get", method: "GET", path: "/api/api-keys/{key2.id}", auth: "session"},
		{name: "api key rename", method: "PUT", path: "/api/api-keys/{key2.id}", body: `{"name":"renamed"}`, auth: "session"},
		{name: "api key rotate", method: "POST", path: "/api/api-keys/{key2.id}/rotate", auth: "session"},
		{name: "api key delete", method: "DELETE", path: "/api/api-keys/{key2.id}", auth: "session"},
		{name: "heatmap", method: "GET", path: "/api/users/me/calendar-heatmap?from=2026-10-01&to=2026-10-03", auth: "session"},
		{name: "heatmap order", method: "GET", path: "/api/users/me/calendar-heatmap?from=2026-10-04&to=2026-10-03", auth: "session"},
		{name: "admin users invalid", method: "GET", path: "/api/admin/users?withDeleted=yes&id=nope", auth: "session"},
		{name: "admin user missing", method: "GET", path: "/api/admin/users/00000000-0000-4000-8000-000000000000", auth: "session"},
		{name: "admin create invalid", method: "POST", path: "/api/admin/users", body: `{"email":"a@b.c","password":"p","name":"n","quotaSizeInBytes":-1,"pinCode":"12"}`, auth: "session"},
		{name: "admin update order", method: "PUT", path: "/api/admin/users/not-a-uuid", body: `{"isAdmin":"x"}`, auth: "session"},
		{name: "admin delete without body", method: "DELETE", path: "/api/admin/users/00000000-0000-4000-8000-000000000000", auth: "session"},
		{name: "preferences invalid", method: "PUT", path: "/api/users/me/preferences", body: `{"memories":{"duration":0,"enabled":"yes"},"albums":{"defaultAssetOrder":"up"}}`, auth: "session"},
		{name: "pin invalid", method: "POST", path: "/api/auth/pin-code", body: `{"pinCode":"12345"}`, auth: "session"},
		{name: "unlock without pin", method: "POST", path: "/api/auth/session/unlock", body: `{}`, auth: "session"},
	}

	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			want := immich.run(t, st)
			got := korda.run(t, st)
			t.Logf("%s %s: korda %d, Immich %d", st.method, st.path, got.Status, want.Status)

			if want.Status != got.Status {
				t.Errorf("status = %d, Immich %d\nkorda  %s\nImmich %s", got.Status, want.Status, got.Body, want.Body)
				return
			}

			var a, b string
			switch st.mode {
			case compareShape:
				var err error
				if a, err = shapeOf(got.Body); err != nil && len(got.Body) > 0 {
					t.Fatalf("korda body: %v", err)
				}
				if b, err = shapeOf(want.Body); err != nil && len(want.Body) > 0 {
					t.Fatalf("Immich body: %v", err)
				}
			default:
				a, b = normalize(string(got.Body)), normalize(string(want.Body))
			}

			if a != b {
				t.Errorf("body differs\nkorda  %s\nImmich %s", a, b)
			}
		})
	}
}
