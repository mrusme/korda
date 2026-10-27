package main

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

func newLoggedHarness(t *testing.T) (*apiHarness, *os.File) {
	t.Helper()

	out, err := os.CreateTemp(t.TempDir(), "korda-*.log")
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = out
	defer func() { os.Stdout = stdout }()

	return newHarnessWith(t, harnessOptions{publicUsers: true, logLevel: "debug"}), out
}

func logLines(t *testing.T, out *os.File, wants ...[]string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		raw, err := os.ReadFile(out.Name())
		if err != nil {
			t.Fatal(err)
		}
		missing := []string{}
		for _, parts := range wants {
			found := false
			for line := range strings.SplitSeq(string(raw), "\n") {
				all := true
				for _, part := range parts {
					all = all && strings.Contains(line, part)
				}
				found = found || all
			}
			if !found {
				missing = append(missing, strings.Join(parts, " … "))
			}
		}
		if len(missing) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no log line with %s in\n%s", strings.Join(missing, "\nnor "), raw)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestLogCoversTheWebAppSocketsAndLinkPasswords(t *testing.T) {
	h, out := newLoggedHarness(t)
	admin := h.admin()
	addr := h.listen()

	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-03T04:05:06.000Z")
	h.runAllJobs()
	link := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+plain+`"],"password":"secret","slug":"s3cr3t-slug"}`, bearer(admin)...).object(t)
	key, linkID := link["key"].(string), link["id"].(string)

	for _, path := range []string{"/share/" + key, "/s/s3cr3t-slug", "/.well-known/immich", "/custom.css"} {
		res, err := http.Get("http://" + addr + path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}
	res, err := http.PostForm("http://"+addr+"/share/"+key, url.Values{"password": {"wrong"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	h.do(fiber.MethodPost, "/api/shared-links/login?key="+key, `{"password":"guess"}`).expect(t, 401, `{"message":"Invalid password"}`)

	socket := dialSocket(t, addr, http.Header{"Authorization": {"Bearer " + admin}})
	if first := socket.connect(); !strings.HasPrefix(first, `42["on_server_version"`) {
		t.Fatalf("first frame = %s", first)
	}
	anonymous := dialSocket(t, addr, nil)
	if got := anonymous.connect(); got != `42["error","unauthorized"]` {
		t.Errorf("anonymous = %s", got)
	}
	_ = socket.conn.Close()

	logLines(t, out,
		[]string{`"path":"/share/redacted"`, `"status":401`, `"ip":"127.0.0.1"`},
		[]string{`"path":"/s/redacted"`, `"ip":"127.0.0.1"`},
		[]string{`"path":"/.well-known/immich"`, `"status":200`},
		[]string{`"path":"/custom.css"`, `"status":200`},
		[]string{`"level":"WARN"`, `"sharedlinks":"failed login attempt"`, `"link":"` + linkID + `"`, `"ip":"127.0.0.1"`},
		[]string{`"level":"WARN"`, `"sharedlinks":"failed login attempt"`, `"link":"` + linkID + `"`, `"ip":"0.0.0.0"`},
		[]string{`"path":"/api/shared-links/login"`, `"query":"key=redacted"`, `"status":401`},
		[]string{`"socket":"connected"`, `"session":"`, `"ip":"127.0.0.1"`},
		[]string{`"level":"INFO"`, `"socket":"authentication failed"`, `"ip":"127.0.0.1"`},
		[]string{`"socket":"disconnected"`, `"reason":"transport close"`},
	)

	raw, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{key, "s3cr3t-slug"} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("the log contains %q", secret)
		}
	}
	if strings.Contains(string(raw), `"path":"/api/socket.io`) {
		t.Errorf("the socket's upgrade request is in the request log")
	}
}
