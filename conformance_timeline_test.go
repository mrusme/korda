package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	gorilla "github.com/gorilla/websocket"
	"tty.fail/mrus/korda/internal/testmedia"
)

func (c *socketClient) collect() <-chan string {
	ch := make(chan string, 64)
	_ = c.conn.SetReadDeadline(time.Time{})
	go func() {
		defer close(ch)
		for {
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				return
			}
			if string(data) == "2" {
				_ = c.conn.WriteMessage(gorilla.TextMessage, []byte("3"))
				continue
			}
			ch <- string(data)
		}
	}()

	return ch
}

func awaitFrames(t *testing.T, ch <-chan string, events map[string]int, timeout time.Duration) []string {
	t.Helper()

	var out []string
	seen := map[string]int{}
	deadline := time.After(timeout)
	for {
		done := true
		for event, n := range events {
			done = done && seen[event] >= n
		}
		if done {
			return out
		}
		select {
		case frame, ok := <-ch:
			if !ok {
				t.Fatalf("socket closed after %v", out)
			}
			out = append(out, frame)
			for event := range events {
				if strings.HasPrefix(frame, `42["`+event+`"`) {
					seen[event]++
				}
			}
		case <-deadline:
			t.Fatalf("frames after %v: %v, want %v", timeout, out, events)
		}
	}
}

func renameIDs(body string, names map[string]string) string {
	for id, name := range names {
		body = strings.ReplaceAll(body, id, "<"+name+">")
	}

	return body
}

func normalizeFrames(frames []string, names map[string]string) []string {
	out := make([]string, 0, len(frames))
	for _, f := range frames {
		if !strings.HasPrefix(f, `42["on_new_release"`) {
			out = append(out, normalizeConformance(renameIDs(f, names)))
		}
	}
	slices.Sort(out)

	return out
}

func normalizeBucket(t *testing.T, body []byte, names map[string]string) string {
	t.Helper()

	var columns map[string]json.RawMessage
	if err := json.Unmarshal(body, &columns); err != nil {
		return string(body)
	}
	for _, key := range []string{"createdAt", "ownerId", "thumbhash", "city", "country"} {
		if _, ok := columns[key]; ok {
			columns[key] = json.RawMessage(`"<v>"`)
		}
	}
	encoded, err := json.Marshal(columns)
	if err != nil {
		t.Fatal(err)
	}

	return renameIDs(string(encoded), names)
}

type zipEntry struct {
	Name     string
	Method   uint16
	Flags    uint16
	Attrs    uint32
	Size     uint64
	Modified string
}

func zipEntries(t *testing.T, body []byte) []zipEntry {
	t.Helper()

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	var out []zipEntry
	for _, f := range zr.File {
		out = append(out, zipEntry{f.Name, f.Method, f.Flags, f.ExternalAttrs, f.UncompressedSize64, f.Modified.Format(time.RFC3339)})
	}

	return out
}

func TestConformanceTimeline(t *testing.T) {
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

	h := newHarness(t)
	korda := kordaSide(h)
	kordaAdmin := h.admin()
	addr := h.listen()

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	userEmail := "conformance-timeline-" + hex.EncodeToString(suffix) + "@example.com"
	userPassword := "conformance-password"
	created := immich.send(fiber.MethodPost, "/api/admin/users",
		fmt.Sprintf(`{"email":%q,"password":%q,"name":"Timeline"}`, userEmail, userPassword), bearer(adminToken)...)
	if created.Status != fiber.StatusCreated {
		t.Fatalf("create Immich user = %d %s", created.Status, created.Body)
	}
	defer immich.send(fiber.MethodDelete, "/api/admin/users/"+created.str(t, "id"), `{"force":true}`, bearer(adminToken)...)
	immich.vars["owner"] = created.str(t, "id")
	immich.tokens["owner"] = immich.send(fiber.MethodPost, "/api/auth/login",
		fmt.Sprintf(`{"email":%q,"password":%q}`, userEmail, userPassword)).str(t, "accessToken")
	korda.vars["owner"] = h.createUser(kordaAdmin, userEmail, "Timeline", userPassword)
	korda.tokens["owner"] = h.login(userEmail, userPassword)

	sockets := map[string]<-chan string{}
	for name, target := range map[string]string{"immich": strings.TrimPrefix(base, "http://"), "korda": addr} {
		s := immich
		if name == "korda" {
			s = korda
		}
		client := dialSocket(t, target, http.Header{"Authorization": {"Bearer " + s.tokens["owner"]}})
		if first := client.connect(); !strings.HasPrefix(first, `42["on_server_version"`) {
			t.Fatalf("%s first frame = %s", name, first)
		}
		sockets[name] = client.collect()
	}

	files := []struct {
		name  string
		build func() ([]byte, error)
		date  string
	}{
		{"exif", testmedia.ExifJPEG, "2024-05-06T05:00:00.000Z"},
		{"plain", testmedia.PlainJPEG, "2020-02-03T04:05:06.000Z"},
		{"png", testmedia.PlainPNG, "2021-01-02T03:04:05.000Z"},
	}
	fileNames := map[string]string{"exif": "korda-exif.jpg", "plain": "korda-plain.jpg", "png": "korda.png"}
	names := map[*side]map[string]string{immich: {}, korda: {}}
	for _, f := range files {
		content := fixture(t, f.build)
		fields := []string{"fileCreatedAt", f.date, "fileModifiedAt", f.date}
		immich.vars[f.name] = immichUpload(t, base, immich.tokens["owner"], fileNames[f.name], content, fields...).str(t, "id")
		korda.vars[f.name] = h.upload(korda.tokens["owner"], fileNames[f.name], content, fields...).str(t, "id")
		for _, s := range []*side{immich, korda} {
			names[s][s.vars[f.name]] = f.name
		}
	}
	for _, s := range []*side{immich, korda} {
		names[s][s.vars["owner"]] = "owner"
	}
	h.runAllJobs()

	uploads := map[string]int{"on_upload_success": len(files), "AssetUploadReadyV2": len(files)}
	want := normalizeFrames(awaitFrames(t, sockets["immich"], uploads, 60*time.Second), names[immich])
	got := normalizeFrames(awaitFrames(t, sockets["korda"], uploads, 10*time.Second), names[korda])
	t.Run("upload frames", func(t *testing.T) {
		if !slices.Equal(got, want) {
			t.Errorf("frames differ\nkorda  %s\nImmich %s", strings.Join(got, "\n       "), strings.Join(want, "\n       "))
		}
	})

	compare := func(t *testing.T, st step, normalize func(t *testing.T, s *side, res apiResponse) string) {
		t.Helper()
		t.Run(st.name, func(t *testing.T) {
			wantRes, gotRes := immich.run(t, st), korda.run(t, st)
			if wantRes.Status != gotRes.Status {
				t.Fatalf("status = %d, Immich %d\nkorda  %s\nImmich %s", gotRes.Status, wantRes.Status, gotRes.Body, wantRes.Body)
			}
			if a, b := normalize(t, korda, gotRes), normalize(t, immich, wantRes); a != b {
				t.Errorf("body differs\nkorda  %s\nImmich %s", a, b)
			}
		})
	}
	plain := func(t *testing.T, s *side, res apiResponse) string {
		return normalizeConformance(renameIDs(string(res.Body), names[s]))
	}
	bucket := func(t *testing.T, s *side, res apiResponse) string {
		return normalizeBucket(t, res.Body, names[s])
	}
	archive := func(t *testing.T, s *side, res apiResponse) string {
		encoded, _ := json.Marshal(zipEntries(t, res.Body))
		return res.Header.Get(fiber.HeaderContentType) + " " + string(encoded)
	}
	type downloadArchive struct {
		Size     int64    `json:"size"`
		AssetIDs []string `json:"assetIds"`
	}
	unorderedDownload := func(t *testing.T, s *side, res apiResponse) string {
		var info struct {
			TotalSize int64             `json:"totalSize"`
			Archives  []downloadArchive `json:"archives"`
		}
		if err := json.Unmarshal([]byte(plain(t, s, res)), &info); err != nil {
			t.Fatalf("%s download info: %v", s.name, err)
		}
		for i := range info.Archives {
			slices.Sort(info.Archives[i].AssetIDs)
		}
		slices.SortFunc(info.Archives, func(a, b downloadArchive) int {
			return strings.Compare(strings.Join(a.AssetIDs, ","), strings.Join(b.AssetIDs, ","))
		})
		encoded, _ := json.Marshal(info)
		return string(encoded)
	}
	unordered := map[string]bool{"download info": true, "download info split": true}

	for _, st := range []step{
		{name: "buckets", method: "GET", path: "/api/timeline/buckets", auth: "owner"},
		{name: "buckets ascending", method: "GET", path: "/api/timeline/buckets?order=asc", auth: "owner"},
		{name: "buckets by box", method: "GET", path: "/api/timeline/buckets?bbox=13,52,14,53", auth: "owner"},
		{name: "buckets across the antimeridian", method: "GET", path: "/api/timeline/buckets?bbox=170,-10,-170,10", auth: "owner"},
		{name: "buckets archived", method: "GET", path: "/api/timeline/buckets?visibility=archive", auth: "owner"},
		{name: "buckets favorites", method: "GET", path: "/api/timeline/buckets?isFavorite=false&withStacked=true", auth: "owner"},
		{name: "buckets of the user", method: "GET", path: "/api/timeline/buckets?userId={owner}", auth: "owner"},
		{name: "buckets locked", method: "GET", path: "/api/timeline/buckets?visibility=locked", auth: "owner"},
		{name: "buckets with partners", method: "GET", path: "/api/timeline/buckets?withPartners=true&isFavorite=true", auth: "owner"},
		{name: "people", method: "GET", path: "/api/people", auth: "owner"},
		{name: "duplicates", method: "GET", path: "/api/duplicates", auth: "owner"},
		{name: "edits", method: "GET", path: "/api/assets/{exif}/edits", auth: "owner"},
		{name: "download info", method: "POST", path: "/api/download/info", body: `{"assetIds":["{png}","{plain}","{exif}"]}`, auth: "owner"},
		{name: "download info split", method: "POST", path: "/api/download/info", body: `{"assetIds":["{png}","{plain}","{exif}"],"archiveSize":1}`, auth: "owner"},
		{name: "download info of the user", method: "POST", path: "/api/download/info", body: `{"userId":"{owner}"}`, auth: "owner"},
		{name: "download info empty", method: "POST", path: "/api/download/info", body: `{"assetIds":[]}`, auth: "owner"},
		{name: "download archive empty", method: "POST", path: "/api/download/archive", body: `{"assetIds":[]}`, auth: "owner"},
		{name: "map markers", method: "GET", path: "/api/map/markers", auth: "owner"},
		{name: "map markers archived", method: "GET", path: "/api/map/markers?isArchived=true&withPartners=true", auth: "owner"},
		{name: "map markers by date", method: "GET", path: "/api/map/markers?fileCreatedAfter=2025-01-01T00:00:00.000Z", auth: "owner"},
	} {
		if unordered[st.name] {
			compare(t, st, unorderedDownload)
			continue
		}
		compare(t, st, plain)
	}
	for _, month := range []string{"2024-05-01", "2021-01-01", "2020-02-01"} {
		compare(t, step{name: "bucket " + month, method: "GET", path: "/api/timeline/bucket?timeBucket=" + month + "T00:00:00.000Z&withCoordinates=true&withStacked=true", auth: "owner"}, bucket)
	}
	compare(t, step{name: "download archive", method: "POST", path: "/api/download/archive", body: `{"assetIds":["{png}","{plain}","{exif}","{png}"]}`, auth: "owner"}, archive)

	compare(t, step{name: "favorite", method: "PUT", path: "/api/assets/{exif}", body: `{"isFavorite":true}`, auth: "owner"}, plain)
	update := map[string]int{"on_asset_update": 1}
	want = normalizeFrames(awaitFrames(t, sockets["immich"], update, 30*time.Second), names[immich])
	got = normalizeFrames(awaitFrames(t, sockets["korda"], update, 10*time.Second), names[korda])
	t.Run("update frame", func(t *testing.T) {
		if !slices.Equal(got, want) {
			t.Errorf("frames differ\nkorda  %s\nImmich %s", strings.Join(got, "\n       "), strings.Join(want, "\n       "))
		}
	})

	compare(t, step{name: "trash", method: "DELETE", path: "/api/assets", body: `{"ids":["{png}"]}`, auth: "owner"}, plain)
	trash := map[string]int{"on_asset_trash": 1}
	want = normalizeFrames(awaitFrames(t, sockets["immich"], trash, 30*time.Second), names[immich])
	got = normalizeFrames(awaitFrames(t, sockets["korda"], trash, 10*time.Second), names[korda])
	t.Run("trash frame", func(t *testing.T) {
		if !slices.Equal(got, want) {
			t.Errorf("frames differ\nkorda  %s\nImmich %s", strings.Join(got, "\n       "), strings.Join(want, "\n       "))
		}
	})
	compare(t, step{name: "buckets trashed", method: "GET", path: "/api/timeline/buckets?isTrashed=true", auth: "owner"}, plain)
	compare(t, step{name: "map markers after trash", method: "GET", path: "/api/map/markers?isFavorite=true", auth: "owner"}, plain)

	revoked := map[string]<-chan string{}
	for name, target := range map[string]string{"immich": strings.TrimPrefix(base, "http://"), "korda": addr} {
		s := immich
		if name == "korda" {
			s = korda
		}
		token := s.send(fiber.MethodPost, "/api/auth/login",
			fmt.Sprintf(`{"email":%q,"password":%q}`, userEmail, userPassword)).str(t, "accessToken")
		var sessions []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		}
		if err := json.Unmarshal(s.send(fiber.MethodGet, "/api/sessions", "", bearer(token)...).Body, &sessions); err != nil {
			t.Fatal(err)
		}
		for _, session := range sessions {
			if session.Current {
				s.vars["second"] = session.ID
			}
		}
		client := dialSocket(t, target, http.Header{"Authorization": {"Bearer " + token}})
		if first := client.connect(); !strings.HasPrefix(first, `42["on_server_version"`) {
			t.Fatalf("%s second session first frame = %s", name, first)
		}
		revoked[name] = client.collect()
	}
	compare(t, step{name: "session delete", method: "DELETE", path: "/api/sessions/{second}", auth: "owner"}, plain)
	compare(t, step{name: "unfavorite", method: "PUT", path: "/api/assets/{exif}", body: `{"isFavorite":false}`, auth: "owner"}, plain)
	t.Run("deleted session socket", func(t *testing.T) {
		frames := awaitFrames(t, revoked["immich"], map[string]int{"on_asset_update": 1}, 30*time.Second)
		if slices.Contains(frames, "41") {
			t.Errorf("Immich closes the socket of a deleted session as well, the deviation is gone: %v", frames)
		}
		t.Logf("deviation: korda closes the socket of a deleted session, Immich keeps it open and sends %d frames", len(frames))

		var got []string
		deadline := time.After(10 * time.Second)
		for !slices.Contains(got, "41") {
			select {
			case frame, ok := <-revoked["korda"]:
				if !ok {
					t.Fatalf("korda socket closed after %v without 41", got)
				}
				got = append(got, frame)
			case <-deadline:
				t.Fatalf("korda frames = %v, want 41", got)
			}
		}
		grace := time.After(time.Second)
		for {
			select {
			case frame, ok := <-revoked["korda"]:
				if ok && strings.HasPrefix(frame, `42[`) {
					t.Errorf("korda sent %s after 41", frame)
				}
				if !ok {
					return
				}
			case <-grace:
				return
			}
		}
	})
}
