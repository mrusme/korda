package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

var (
	syncVolatile = regexp.MustCompile(`"(createdAt|deletedAt|profileChangedAt|thumbhash)":"[^"]*"`)
	syncGeocoded = regexp.MustCompile(`"(city|state|country)":("[^"]*"|null)`)
)

func syncBody() string {
	encoded, _ := json.Marshal(map[string]any{"types": appSyncTypes})
	return string(encoded)
}

func normalizeStream(t *testing.T, body []byte, userID string) string {
	t.Helper()

	var groups [][]string
	last := ""
	for raw := range bytes.SplitSeq(bytes.TrimSuffix(body, []byte("\n")), []byte("\n")) {
		var l struct {
			Type string         `json:"type"`
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(raw, &l); err != nil {
			t.Fatalf("line %q: %v", raw, err)
		}
		if l.Type == "UserDeleteV1" || (l.Type == "UserV1" && l.Data["id"] != userID) {
			continue
		}

		line := syncGeocoded.ReplaceAllString(syncVolatile.ReplaceAllString(string(raw), `"$1":"<v>"`), `"$1":"<geo>"`)
		line = uuidPattern.ReplaceAllString(line, "<uuid>")
		if l.Type != last {
			groups = append(groups, nil)
			last = l.Type
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], line)
	}

	var out []string
	for _, g := range groups {
		slices.Sort(g)
		out = append(out, g...)
	}

	return strings.Join(out, "\n")
}

func syncAckRuns(body []byte) string {
	var acks []string
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	for i, raw := range lines {
		var l syncLine
		_ = json.Unmarshal([]byte(raw), &l)
		next := ""
		if i+1 < len(lines) {
			var n syncLine
			_ = json.Unmarshal([]byte(lines[i+1]), &n)
			next = n.Type
		}
		if next != l.Type {
			acks = append(acks, l.Ack)
		}
	}
	encoded, _ := json.Marshal(map[string]any{"acks": acks})

	return string(encoded)
}

func currentSession(t *testing.T, s *side) string {
	t.Helper()

	res := s.send(fiber.MethodGet, "/api/sessions", "", bearer(s.tokens["session"])...)
	var list []map[string]any
	if err := json.Unmarshal(res.Body, &list); err != nil {
		t.Fatalf("%s sessions = %s", s.name, res.Body)
	}
	for _, session := range list {
		if session["current"] == true {
			return session["id"].(string)
		}
	}
	t.Fatalf("%s has no current session: %s", s.name, res.Body)

	return ""
}

func TestConformanceSync(t *testing.T) {
	base := strings.TrimSuffix(os.Getenv(envReferenceURL), "/")
	email := os.Getenv(envReferenceEmail)
	password := os.Getenv(envReferencePassword)
	if base == "" || email == "" || password == "" {
		t.Skipf("set %s, %s and %s to compare korda with an Immich", envReferenceURL, envReferenceEmail, envReferencePassword)
	}

	immich := immichSide(t, base)
	admin := immich.send(fiber.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	if admin.Status != fiber.StatusCreated {
		t.Fatalf("login to %s = %d %s", base, admin.Status, admin.Body)
	}
	adminToken := admin.str(t, "accessToken")
	defer immich.send(fiber.MethodPost, "/api/auth/logout", "", bearer(adminToken)...)

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	userEmail := "conformance-sync-" + hex.EncodeToString(suffix) + "@example.com"
	userPassword := "conformance-password"
	created := immich.send(fiber.MethodPost, "/api/admin/users",
		fmt.Sprintf(`{"email":%q,"password":%q,"name":"Conformance"}`, userEmail, userPassword), bearer(adminToken)...)
	if created.Status != fiber.StatusCreated {
		t.Fatalf("create Immich user = %d %s", created.Status, created.Body)
	}
	defer immich.send(fiber.MethodDelete, "/api/admin/users/"+created.str(t, "id"), `{"force":true}`, bearer(adminToken)...)
	immich.vars["user"] = created.str(t, "id")
	immich.tokens["session"] = immich.send(fiber.MethodPost, "/api/auth/login",
		fmt.Sprintf(`{"email":%q,"password":%q}`, userEmail, userPassword)).str(t, "accessToken")

	h := newHarness(t)
	korda := kordaSide(h)
	korda.vars["user"] = h.createUser(h.admin(), userEmail, "Conformance", userPassword)
	korda.tokens["session"] = h.login(userEmail, userPassword)

	sides := []*side{immich, korda}
	for _, s := range sides {
		key := s.send(fiber.MethodPost, "/api/api-keys", `{"name":"conformance-sync","permissions":["all"]}`, bearer(s.tokens["session"])...)
		s.tokens["key"] = key.str(t, "secret")
	}

	stream := func(s *side) apiResponse {
		time.Sleep(3 * time.Millisecond)
		return s.send(fiber.MethodPost, "/api/sync/stream", syncBody(), bearer(s.tokens["session"])...)
	}
	compareStreams := func(name string) (apiResponse, apiResponse) {
		t.Helper()
		want, got := stream(immich), stream(korda)
		if want.Status != fiber.StatusOK || got.Status != fiber.StatusOK {
			t.Fatalf("%s: korda %d %s, Immich %d %s", name, got.Status, got.Body, want.Status, want.Body)
		}
		a, b := normalizeStream(t, got.Body, korda.vars["user"]), normalizeStream(t, want.Body, immich.vars["user"])
		if a != b {
			t.Errorf("%s differs\nkorda\n%s\nImmich\n%s", name, a, b)
		}
		return want, got
	}
	same := func(name string, method string, path string, body string, auth string) {
		t.Helper()
		headers := func(s *side) []string {
			if auth == "key" {
				return []string{"x-api-key", s.tokens["key"]}
			}
			return bearer(s.tokens["session"])
		}
		want := immich.send(method, path, body, headers(immich)...)
		got := korda.send(method, path, body, headers(korda)...)
		if want.Status != got.Status || string(want.Body) != string(got.Body) {
			t.Errorf("%s = %d %s, Immich %d %s", name, got.Status, got.Body, want.Status, want.Body)
		}
	}

	compareStreams("fresh session")

	files := []struct {
		name  string
		build func() ([]byte, error)
		date  string
	}{
		{"korda-exif.jpg", testmedia.ExifJPEG, "2024-05-06T05:00:00.000Z"},
		{"korda-plain.jpg", testmedia.PlainJPEG, "2020-02-03T04:05:06.000Z"},
		{"korda.png", testmedia.PlainPNG, "2021-01-02T03:04:05.000Z"},
	}
	for _, f := range files {
		content := fixture(t, f.build)
		fields := []string{"fileCreatedAt", f.date, "fileModifiedAt", f.date}
		immich.vars[f.name] = immichUpload(t, base, immich.tokens["session"], f.name, content, fields...).str(t, "id")
		korda.vars[f.name] = h.upload(korda.tokens["session"], f.name, content, fields...).str(t, "id")
	}
	h.runAllJobs()
	deadline := time.Now().Add(60 * time.Second)
	for _, f := range files {
		for {
			info := immich.send(fiber.MethodGet, "/api/assets/"+immich.vars[f.name], "", bearer(immich.tokens["session"])...)
			if !strings.Contains(string(info.Body), `"thumbhash":null`) && strings.Contains(string(info.Body), `"dateTimeOriginal":"`) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("Immich did not process %s: %s", f.name, info.Body)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	immichStream, kordaStream := compareStreams("after uploads")

	for _, pair := range [][2]any{{immich, immichStream}, {korda, kordaStream}} {
		s, res := pair[0].(*side), pair[1].(apiResponse)
		ack := s.send(fiber.MethodPost, "/api/sync/ack", syncAckRuns(res.Body), bearer(s.tokens["session"])...)
		if ack.Status != fiber.StatusNoContent {
			t.Fatalf("%s ack = %d %s", s.name, ack.Status, ack.Body)
		}
	}
	compareStreams("acked, nothing new")

	for _, s := range sides {
		s.send(fiber.MethodPut, "/api/assets/"+s.vars["korda-plain.jpg"], `{"isFavorite":true}`, bearer(s.tokens["session"])...)
		s.send(fiber.MethodDelete, "/api/assets", fmt.Sprintf(`{"ids":[%q]}`, s.vars["korda.png"]), bearer(s.tokens["session"])...)
		s.send(fiber.MethodPut, "/api/assets/"+s.vars["korda-exif.jpg"]+"/metadata",
			`{"items":[{"key":"mobile-app","value":{"zeta":1,"a":true,"nested":{"bb":2,"a":1}}}]}`, bearer(s.tokens["session"])...)
		s.send(fiber.MethodPut, "/api/users/me/preferences", `{"ratings":{"enabled":true},"folders":{"enabled":true,"sidebarWeb":true}}`, bearer(s.tokens["session"])...)
		s.send(fiber.MethodPut, "/api/users/me/onboarding", `{"isOnboarded":true}`, bearer(s.tokens["session"])...)
	}
	h.runAllJobs()
	immichStream, kordaStream = compareStreams("after changes")

	for _, pair := range [][2]any{{immich, immichStream}, {korda, kordaStream}} {
		s, res := pair[0].(*side), pair[1].(apiResponse)
		s.send(fiber.MethodPost, "/api/sync/ack", syncAckRuns(res.Body), bearer(s.tokens["session"])...)
		s.send(fiber.MethodDelete, "/api/assets", fmt.Sprintf(`{"ids":[%q],"force":true}`, s.vars["korda-exif.jpg"]), bearer(s.tokens["session"])...)
		s.send(fiber.MethodDelete, "/api/users/me/onboarding", "", bearer(s.tokens["session"])...)
	}
	h.runAllJobs()
	deadline = time.Now().Add(60 * time.Second)
	for !strings.Contains(string(stream(immich).Body), `"AssetDeleteV1"`) {
		if time.Now().After(deadline) {
			t.Fatal("Immich did not delete korda-exif.jpg")
		}
		time.Sleep(500 * time.Millisecond)
	}
	compareStreams("after a force delete")

	ownAcks := func(s *side) string {
		res := s.send(fiber.MethodGet, "/api/sync/ack", "", bearer(s.tokens["session"])...)
		var list []map[string]string
		if err := json.Unmarshal(res.Body, &list); err != nil {
			t.Fatalf("%s acks = %s", s.name, res.Body)
		}
		list = slices.DeleteFunc(list, func(a map[string]string) bool { return a["type"] == "UserDeleteV1" })
		encoded, _ := json.Marshal(list)
		return uuidPattern.ReplaceAllString(string(encoded), "<uuid>")
	}
	if a, b := ownAcks(korda), ownAcks(immich); a != b {
		t.Errorf("acks = %s, Immich %s", a, b)
	}

	for _, s := range sides {
		s.vars["session"] = currentSession(t, s)
	}
	same("stream with an API key", fiber.MethodPost, "/api/sync/stream", `{"types":[]}`, "key")
	same("acks with an API key", fiber.MethodGet, "/api/sync/ack", "", "key")
	same("deprecated type", fiber.MethodPost, "/api/sync/stream", `{"types":["AssetsV1"]}`, "session")
	same("deprecated faces", fiber.MethodPost, "/api/sync/stream", `{"types":["AssetFacesV1"]}`, "session")
	same("unknown type", fiber.MethodPost, "/api/sync/stream", `{"types":["Foo"]}`, "session")
	same("no types", fiber.MethodPost, "/api/sync/stream", `{}`, "session")
	same("reset not a boolean", fiber.MethodPost, "/api/sync/stream", `{"types":[],"reset":"yes"}`, "session")
	same("empty acks", fiber.MethodPost, "/api/sync/ack", `{"acks":[]}`, "session")
	same("invalid ack type", fiber.MethodPost, "/api/sync/ack", `{"acks":["Nope|x"]}`, "session")
	same("ack not a string", fiber.MethodPost, "/api/sync/ack", `{"acks":[1]}`, "session")
	same("delete without types", fiber.MethodDelete, "/api/sync/ack", `{"types":[]}`, "session")
	same("delete unknown type", fiber.MethodDelete, "/api/sync/ack", `{"types":["Nope"]}`, "session")
	same("delete without a body", fiber.MethodDelete, "/api/sync/ack", "", "session")

	reset := `{"type":"SyncResetV1","data":{},"ack":"SyncResetV1|reset"}` + "\n"
	for _, s := range sides {
		s.send(fiber.MethodPut, "/api/sessions/"+s.vars["session"], `{"isPendingSyncReset":true}`, bearer(s.tokens["session"])...)
		if res := stream(s); string(res.Body) != reset {
			t.Errorf("%s pending reset stream = %d %s", s.name, res.Status, res.Body)
		}
		s.send(fiber.MethodPost, "/api/sync/ack", `{"acks":["SyncResetV1|reset"]}`, bearer(s.tokens["session"])...)
	}
	same("acks after a reset", fiber.MethodGet, "/api/sync/ack", "", "session")
	compareStreams("after a reset")

	old := uuidV7At(time.Now().AddDate(0, 0, -31))
	for _, s := range sides {
		s.send(fiber.MethodPost, "/api/sync/ack", `{"acks":["SyncCompleteV1|`+old+`"]}`, bearer(s.tokens["session"])...)
		if res := stream(s); string(res.Body) != reset {
			t.Errorf("%s stream after 31 days = %d %s", s.name, res.Status, res.Body)
		}
	}
	for _, s := range sides {
		s.send(fiber.MethodPost, "/api/sync/stream", `{"types":[],"reset":true}`, bearer(s.tokens["session"])...)
	}
	same("acks after reset true", fiber.MethodGet, "/api/sync/ack", "", "session")

	for _, s := range sides {
		s.send(fiber.MethodPost, "/api/sync/ack", `{"acks":["AuthUserV2|ffffffff-ffff-7fff-bfff-ffffffffffff","UserDeleteV1|ffffffff-ffff-7fff-bfff-ffffffffffff","UserV1|ffffffff-ffff-7fff-bfff-ffffffffffff","AssetDeleteV1|ffffffff-ffff-7fff-bfff-ffffffffffff","AssetV2|not-a-uuid"]}`, bearer(s.tokens["session"])...)
	}
	same("broken stored ack", fiber.MethodPost, "/api/sync/stream", `{"types":["AuthUsersV2","UsersV1","AssetsV2"]}`, "session")
}
