package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

type upload struct {
	name     string
	build    func() ([]byte, error)
	created  string
	modified string
	fields   []string
}

var (
	conformanceUploads = []upload{
		{name: "korda-exif.jpg", build: testmedia.ExifJPEG, created: "2024-05-06T05:00:00.000Z", modified: "2024-05-06T05:00:00.000Z"},
		{name: "korda-plain.jpg", build: testmedia.PlainJPEG, created: "2020-02-03T04:05:06.000Z", modified: "2020-02-04T04:05:06.000Z", fields: []string{"isFavorite", "true"}},
		{name: "korda.png", build: testmedia.PlainPNG, created: "2021-01-02T03:04:05.000Z", modified: "2021-01-02T03:04:05.000Z", fields: []string{"filename", "renamed.png"}},
		{name: "korda-mutate-1.jpg", build: func() ([]byte, error) { return testmedia.Mutate(10) }, created: "2019-07-08T09:10:11.000Z", modified: "2019-07-08T09:10:11.000Z"},
		{name: "korda-mutate-2.jpg", build: func() ([]byte, error) { return testmedia.Mutate(90) }, created: "2019-07-08T09:10:11.000Z", modified: "2019-07-08T09:10:11.000Z"},
	}

	volatileFields = regexp.MustCompile(`"(id|ownerId|createdAt|updatedAt|originalPath|thumbhash|profileChangedAt|assetId|value)":"[^"]*"`)
	geocoded       = regexp.MustCompile(`"(city|state|country)":("[^"]*"|null)`)
	pathList       = regexp.MustCompile(`"[^"]*upload/[0-9a-f-]{36}/[0-9a-f]{2}/[0-9a-f]{2}"`)
)

func normalizeConformance(body string) string {
	body = volatileFields.ReplaceAllString(body, `"$1":"<v>"`)
	body = geocoded.ReplaceAllString(body, `"$1":"<geo>"`)
	body = pathList.ReplaceAllString(body, `"<dir>"`)

	return uuidPattern.ReplaceAllString(body, "<uuid>")
}

func immichUpload(t *testing.T, base string, token string, name string, content []byte, fields ...string) apiResponse {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for i := 0; i+1 < len(fields); i += 2 {
		if err := mw.WriteField(fields[i], fields[i+1]); err != nil {
			t.Fatal(err)
		}
	}
	w, err := mw.CreateFormFile("assetData", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err = mw.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(fiber.MethodPost, base+"/api/assets", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(fiber.HeaderContentType, mw.FormDataContentType())
	req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return apiResponse{Status: resp.StatusCode, Header: resp.Header, Body: body}
}

func TestConformanceAssets(t *testing.T) {
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

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	userEmail := "conformance-" + hex.EncodeToString(suffix) + "@example.com"
	userPassword := "conformance-password"
	created := immich.send(fiber.MethodPost, "/api/admin/users",
		fmt.Sprintf(`{"email":%q,"password":%q,"name":"Conformance"}`, userEmail, userPassword), bearer(adminToken)...)
	if created.Status != fiber.StatusCreated {
		t.Fatalf("create Immich user = %d %s", created.Status, created.Body)
	}
	defer immich.send(fiber.MethodDelete, "/api/admin/users/"+created.str(t, "id"), `{"force":true}`, bearer(adminToken)...)
	userLogin := immich.send(fiber.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, userEmail, userPassword))
	immich.tokens["session"] = userLogin.str(t, "accessToken")
	immich.tokens["admin"] = adminToken
	immich.vars["user"] = created.str(t, "id")

	h := newHarness(t)
	korda := kordaSide(h)
	kordaAdmin := h.admin()
	korda.vars["user"] = h.createUser(kordaAdmin, userEmail, "Conformance", userPassword)
	korda.tokens["session"] = h.login(userEmail, userPassword)
	korda.tokens["admin"] = kordaAdmin

	for _, u := range conformanceUploads {
		content := fixture(t, u.build)
		fields := append([]string{"fileCreatedAt", u.created, "fileModifiedAt", u.modified}, u.fields...)
		want := immichUpload(t, base, immich.tokens["session"], u.name, content, fields...)
		got := h.upload(korda.tokens["session"], u.name, content, fields...)
		if want.Status != got.Status || normalize(string(want.Body)) != normalize(string(got.Body)) {
			t.Errorf("upload %s = %d %s, Immich %d %s", u.name, got.Status, got.Body, want.Status, want.Body)
		}
		immich.vars[u.name] = want.str(t, "id")
		korda.vars[u.name] = got.str(t, "id")

		again := immichUpload(t, base, immich.tokens["session"], u.name, content, fields...)
		if dup := h.upload(korda.tokens["session"], u.name, content, fields...); again.Status != dup.Status ||
			normalize(string(again.Body)) != normalize(string(dup.Body)) {
			t.Errorf("duplicate %s = %d %s, Immich %d %s", u.name, dup.Status, dup.Body, again.Status, again.Body)
		}
	}

	h.runAllJobs()
	deadline := time.Now().Add(60 * time.Second)
	for _, u := range conformanceUploads {
		for {
			info := immich.send(fiber.MethodGet, "/api/assets/"+immich.vars[u.name], "", bearer(immich.tokens["session"])...)
			if info.Status == fiber.StatusOK && !strings.Contains(string(info.Body), `"thumbhash":null`) &&
				strings.Contains(string(info.Body), `"dateTimeOriginal":"`) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("Immich did not process %s: %s", u.name, info.Body)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	steps := []step{}
	for _, u := range conformanceUploads {
		steps = append(steps,
			step{name: "info " + u.name, method: "GET", path: "/api/assets/{" + u.name + "}", auth: "session"},
			step{name: "metadata " + u.name, method: "GET", path: "/api/assets/{" + u.name + "}/metadata", auth: "session"},
		)
	}
	steps = append(steps,
		step{name: "statistics", method: "GET", path: "/api/assets/statistics", auth: "session"},
		step{name: "statistics favorite", method: "GET", path: "/api/assets/statistics?isFavorite=true", auth: "session"},
		step{name: "admin statistics", method: "GET", path: "/api/admin/users/{user}/statistics", auth: "admin"},
		step{name: "admin statistics favorite", method: "GET", path: "/api/admin/users/{user}/statistics?isFavorite=true", auth: "admin"},
		step{name: "heatmap upload", method: "GET", path: "/api/users/me/calendar-heatmap", auth: "session"},
		step{name: "heatmap taken", method: "GET", path: "/api/users/me/calendar-heatmap?from=2019-07-07&to=2019-07-09&type=Taken", auth: "session"},
		step{name: "heatmap taken admin", method: "GET", path: "/api/admin/users/{user}/calendar-heatmap?from=2020-02-02&to=2020-02-04&type=Taken", auth: "admin"},
		step{name: "search page 1", method: "POST", path: "/api/search/metadata", body: `{"page":1,"size":2}`, auth: "session"},
		step{name: "search page 3", method: "POST", path: "/api/search/metadata", body: `{"page":3,"size":2,"order":"asc"}`, auth: "session"},
		step{name: "search exif", method: "POST", path: "/api/search/metadata", body: `{"withExif":true,"make":"korda","model":"Fixture One"}`, auth: "session"},
		step{name: "search name", method: "POST", path: "/api/search/metadata", body: `{"originalFileName":"MUTATE-1"}`, auth: "session"},
		step{name: "search taken", method: "POST", path: "/api/search/metadata", body: `{"takenAfter":"2020-01-01T00:00:00.000Z","takenBefore":"2022-01-01T00:00:00.000Z"}`, auth: "session"},
		step{name: "search v3", method: "POST", path: "/api/search/metadata", body: `{"size":2,"filter":{"type":{"eq":"IMAGE"},"isFavorite":{"eq":false}},"orderBy":{"field":"fileCreatedAt","direction":"desc"}}`, auth: "session"},
		step{name: "search v3 invalid", method: "POST", path: "/api/search/metadata", body: `{"filter":{"or":[{}]},"page":2}`, auth: "session"},
		step{name: "search statistics", method: "POST", path: "/api/search/statistics", body: `{"type":"IMAGE"}`, auth: "session"},
		step{name: "search smart", method: "POST", path: "/api/search/smart", body: `{"query":"cat"}`, auth: "session"},
		step{name: "suggestions make", method: "GET", path: "/api/search/suggestions?type=camera-make", auth: "session"},
		step{name: "suggestions model", method: "GET", path: "/api/search/suggestions?type=camera-model&make=korda&includeNull=true", auth: "session"},
		step{name: "explore", method: "GET", path: "/api/search/explore", auth: "session", mode: compareShape},
		step{name: "unique paths", method: "GET", path: "/api/view/folder/unique-paths", auth: "session"},
		step{name: "folder missing path", method: "GET", path: "/api/view/folder", auth: "session"},
		step{name: "bulk upload check", method: "POST", path: "/api/assets/bulk-upload-check", body: `{"assets":[{"id":"a","checksum":"7lWy+xd67RvuCrGVWCyajV1LYuU="},{"id":"b","checksum":"00"}]}`, auth: "session"},
		step{name: "update", method: "PUT", path: "/api/assets/{korda-mutate-1.jpg}", body: `{"isFavorite":true,"description":"edited","rating":5}`, auth: "session"},
		step{name: "update rating 0", method: "PUT", path: "/api/assets/{korda-mutate-1.jpg}", body: `{"rating":0,"latitude":1}`, auth: "session"},
		step{name: "update date", method: "PUT", path: "/api/assets/{korda-mutate-1.jpg}", body: `{"dateTimeOriginal":"2020-01-02T03:04:05.000+02:00","latitude":10.5,"longitude":20.25}`, auth: "session"},
		step{name: "bulk update", method: "PUT", path: "/api/assets", body: `{"ids":["{korda-mutate-2.jpg}"],"isFavorite":true,"visibility":"archive"}`, auth: "session"},
		step{name: "bulk update sibling", method: "PUT", path: "/api/assets", body: `{"ids":["{korda-mutate-2.jpg}"],"dateTimeRelative":60,"dateTimeOriginal":"2020-01-01T00:00:00Z"}`, auth: "session"},
		step{name: "metadata upsert", method: "PUT", path: "/api/assets/{korda-mutate-2.jpg}/metadata", body: `{"items":[{"key":"mobile-app","value":{"b":1,"a":{"z":1,"y":2},"10":3,"2":4}}]}`, auth: "session"},
		step{name: "metadata key", method: "GET", path: "/api/assets/{korda-mutate-2.jpg}/metadata/mobile-app", auth: "session"},
		step{name: "metadata missing", method: "GET", path: "/api/assets/{korda-mutate-2.jpg}/metadata/nope", auth: "session"},
		step{name: "metadata duplicate", method: "PUT", path: "/api/assets/{korda-mutate-2.jpg}/metadata", body: `{"items":[{"key":"k","value":{}},{"key":"k","value":{}}]}`, auth: "session"},
		step{name: "ocr", method: "GET", path: "/api/assets/{korda-mutate-2.jpg}/ocr", auth: "session"},
		step{name: "edits", method: "GET", path: "/api/assets/{korda-mutate-2.jpg}/edits", auth: "session"},
		step{name: "jobs", method: "POST", path: "/api/assets/jobs", body: `{"assetIds":["{korda-mutate-2.jpg}"],"name":"refresh-faces"}`, auth: "session"},
		step{name: "copy same", method: "PUT", path: "/api/assets/copy", body: `{"sourceId":"{korda-mutate-2.jpg}","targetId":"{korda-mutate-2.jpg}"}`, auth: "session"},
		step{name: "trash", method: "DELETE", path: "/api/assets", body: `{"ids":["{korda-mutate-2.jpg}"]}`, auth: "session"},
		step{name: "trashed info", method: "GET", path: "/api/assets/{korda-mutate-2.jpg}", auth: "session"},
		step{name: "trashed statistics", method: "GET", path: "/api/assets/statistics?isTrashed=true", auth: "session"},
		step{name: "restore", method: "POST", path: "/api/trash/restore/assets", body: `{"ids":["{korda-mutate-2.jpg}","{korda-mutate-2.jpg}"]}`, auth: "session"},
		step{name: "restore none", method: "POST", path: "/api/trash/restore", auth: "session"},
		step{name: "thumbnail missing", method: "GET", path: "/api/assets/00000000-0000-4000-8000-000000000000/thumbnail", auth: "session"},
		step{name: "original missing", method: "GET", path: "/api/assets/00000000-0000-4000-8000-000000000000/original", auth: "session"},
		step{name: "playback image", method: "GET", path: "/api/assets/{korda-exif.jpg}/video/playback", auth: "session"},
		step{name: "hls", method: "GET", path: "/api/assets/{korda-exif.jpg}/video/stream/main.m3u8", auth: "session"},
		step{name: "hls segment", method: "GET", path: "/api/assets/{korda-exif.jpg}/video/stream/00000000-0000-4000-8000-000000000000/0/bogus.ts", auth: "session"},
	)

	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			want := immich.run(t, st)
			got := korda.run(t, st)
			h.runAllJobs()

			if want.Status != got.Status {
				t.Errorf("status = %d, Immich %d\nkorda  %s\nImmich %s", got.Status, want.Status, got.Body, want.Body)
				return
			}

			var a, b string
			if st.mode == compareShape {
				a, _ = shapeOf(got.Body)
				b, _ = shapeOf(want.Body)
			} else {
				a, b = normalizeConformance(string(got.Body)), normalizeConformance(string(want.Body))
			}
			if a != b {
				t.Errorf("body differs\nkorda  %s\nImmich %s", a, b)
			}
		})
	}

	t.Run("server statistics", func(t *testing.T) {
		entry := func(s *side) string {
			res := s.send(fiber.MethodGet, "/api/server/statistics", "", bearer(s.tokens["admin"])...)
			var body struct {
				UsageByUser []map[string]any `json:"usageByUser"`
			}
			if err := json.Unmarshal(res.Body, &body); err != nil || res.Status != fiber.StatusOK {
				t.Fatalf("%s statistics = %d %s", s.name, res.Status, res.Body)
			}
			for _, u := range body.UsageByUser {
				if u["userId"] == s.vars["user"] {
					delete(u, "userId")
					encoded, _ := json.Marshal(u)
					return string(encoded)
				}
			}
			t.Fatalf("%s statistics have no entry for the conformance user: %s", s.name, res.Body)
			return ""
		}
		if got, want := entry(korda), entry(immich); got != want {
			t.Errorf("usage of the conformance user = %s, Immich %s", got, want)
		}
	})

	for _, check := range []struct {
		path    string
		headers []string
	}{
		{path: "/api/assets/{korda-exif.jpg}/original", headers: []string{"Content-Type", "Content-Length", "Content-Disposition", "Cache-Control", "Etag", "Last-Modified", "Accept-Ranges"}},
		{path: "/api/assets/{korda-exif.jpg}/thumbnail?size=preview", headers: []string{"Content-Type", "Content-Disposition", "Cache-Control"}},
		{path: "/api/assets/{korda-exif.jpg}/thumbnail?size=fullsize&c=x", headers: []string{"Location", "Content-Type"}},
		{path: "/api/assets/{korda.png}/thumbnail?size=original&edited=false", headers: []string{"Location"}},
	} {
		want := immich.send(fiber.MethodGet, immich.expand(check.path), "", bearer(immich.tokens["session"])...)
		got := korda.send(fiber.MethodGet, korda.expand(check.path), "", bearer(korda.tokens["session"])...)
		if want.Status != got.Status {
			t.Errorf("%s = %d, Immich %d", check.path, got.Status, want.Status)
			continue
		}
		for _, header := range check.headers {
			if want.Header.Get(header) != got.Header.Get(header) {
				t.Errorf("%s %s = %q, Immich %q", check.path, header, got.Header.Get(header), want.Header.Get(header))
			}
		}
	}
}
