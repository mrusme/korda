package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/internal/testmedia"
)

type searchPage struct {
	Albums struct {
		Total int   `json:"total"`
		Items []any `json:"items"`
	} `json:"albums"`
	Assets struct {
		Total      int              `json:"total"`
		Count      int              `json:"count"`
		Items      []map[string]any `json:"items"`
		NextPage   *string          `json:"nextPage"`
		NextCursor *string          `json:"nextCursor"`
	} `json:"assets"`
}

func (h *apiHarness) search(token string, body string) searchPage {
	h.t.Helper()

	res := h.do(fiber.MethodPost, "/api/search/metadata", body, bearer(token)...)
	if res.Status != fiber.StatusOK {
		h.t.Fatalf("search %s = %d %s", body, res.Status, res.Body)
	}

	var page searchPage
	if err := json.Unmarshal(res.Body, &page); err != nil {
		h.t.Fatal(err)
	}

	return page
}

func names(page searchPage) []string {
	out := make([]string, 0, len(page.Assets.Items))
	for _, item := range page.Assets.Items {
		out = append(out, item["originalFileName"].(string))
	}

	return out
}

func seedLibrary(t *testing.T, h *apiHarness, token string) {
	t.Helper()

	h.uploadID(token, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.uploadID(token, "renamed.png", fixture(t, testmedia.PlainPNG), "2021-01-02T03:04:05.000Z", "2021-01-02T03:04:05.000Z")
	h.uploadID(token, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z", "isFavorite", "true")
	h.runAllJobs()
}

func TestSearchMatchesImmich(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	seedLibrary(t, h, token)

	page := h.search(token, `{"page":1,"size":2}`)
	if got := names(page); len(got) != 2 || got[0] != "korda-exif.jpg" || got[1] != "renamed.png" ||
		page.Assets.NextPage == nil || *page.Assets.NextPage != "2" || page.Assets.NextCursor != nil {
		t.Errorf("first page = %v next %v", got, page.Assets.NextPage)
	}
	if _, ok := page.Assets.Items[0]["exifInfo"]; ok {
		t.Error("search items carry exifInfo without withExif")
	}
	if _, ok := page.Assets.Items[0]["tags"]; ok {
		t.Error("search items carry tags")
	}

	page = h.search(token, `{"page":2,"size":2}`)
	if got := names(page); len(got) != 1 || got[0] != "korda-plain.jpg" || page.Assets.NextPage != nil {
		t.Errorf("second page = %v", got)
	}

	if got := names(h.search(token, `{"originalFileName":"KORDA"}`)); len(got) != 2 {
		t.Errorf("file name search = %v", got)
	}
	if got := names(h.search(token, `{"isFavorite":true}`)); len(got) != 1 || got[0] != "korda-plain.jpg" {
		t.Errorf("favorite search = %v", got)
	}
	if got := names(h.search(token, `{"make":"korda","model":"Fixture One"}`)); len(got) != 1 {
		t.Errorf("camera search = %v", got)
	}
	if got := names(h.search(token, `{"takenAfter":"2021-01-01T00:00:00.000Z","takenBefore":"2022-01-01T00:00:00.000Z"}`)); len(got) != 1 || got[0] != "renamed.png" {
		t.Errorf("date search = %v", got)
	}
	exif := h.search(token, `{"withExif":true,"make":"korda"}`)
	if info, ok := exif.Assets.Items[0]["exifInfo"].(map[string]any); !ok || info["timeZone"] != "UTC+2" {
		t.Errorf("withExif = %v", exif.Assets.Items[0])
	}

	v3 := h.search(token, `{"size":2,"filter":{"type":{"eq":"IMAGE"}},"orderBy":{"direction":"asc"}}`)
	if got := names(v3); len(got) != 2 || got[0] != "korda-plain.jpg" || v3.Assets.NextCursor == nil || *v3.Assets.NextCursor != "eyJvZmZzZXQiOjJ9" {
		t.Errorf("v3 page = %v cursor %v", got, v3.Assets.NextCursor)
	}
	if got := names(h.search(token, `{"size":2,"filter":{"type":{"eq":"IMAGE"}},"orderBy":{"direction":"asc"},"cursor":"eyJvZmZzZXQiOjJ9"}`)); len(got) != 1 || got[0] != "korda-exif.jpg" {
		t.Errorf("v3 next page = %v", got)
	}
	h.do(fiber.MethodPost, "/api/search/metadata", `{"cursor":"nope"}`, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Invalid cursor"}`)
	h.do(fiber.MethodPost, "/api/search/metadata", `{"isFavorite":true,"page":2,"filter":{},"cursor":"x"}`, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Validation failed","errors":[{"code":"custom","path":["isFavorite"],"message":"Deprecated field isFavorite cannot be combined with filter/cursor"},{"code":"custom","path":["page"],"message":"Deprecated field page cannot be combined with filter/cursor"}]}`)
	h.do(fiber.MethodPost, "/api/search/metadata", `{"filter":{"or":[{"or":[]}]}}`, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Validation failed","errors":[{"code":"unrecognized_keys","keys":["or"],"path":["filter","or",0],"message":"Unrecognized key: \"or\""}]}`)
	h.do(fiber.MethodPost, "/api/search/metadata", `{"filter":{"id":{"foo":"x"}}}`, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Validation failed","errors":[{"code":"custom","path":["filter","id"],"message":"At least one of the following fields is required: eq, ne"}]}`)
	h.do(fiber.MethodPost, "/api/search/metadata", `{"filter":{"visibility":{"ne":"timeline"}}}`, bearer(token)...).
		expect(t, fiber.StatusUnauthorized, `{"message":"Elevated permission is required"}`)

	h.do(fiber.MethodPost, "/api/search/statistics", `{}`, bearer(token)...).expect(t, fiber.StatusOK, `{"total":3}`)
	h.do(fiber.MethodPost, "/api/search/statistics", `{"type":"VIDEO"}`, bearer(token)...).expect(t, fiber.StatusOK, `{"total":0}`)
	if got := h.do(fiber.MethodPost, "/api/search/random", `{"size":2}`, bearer(token)...).array(t); len(got) != 2 {
		t.Errorf("random = %d items", len(got))
	}
	h.do(fiber.MethodPost, "/api/search/smart", `{"query":"cat"}`, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Smart search is not enabled"}`)

	h.do(fiber.MethodGet, "/api/search/suggestions?type=camera-make", nil, bearer(token)...).expect(t, fiber.StatusOK, `["korda"]`)
	h.do(fiber.MethodGet, "/api/search/suggestions?type=camera-lens-model&includeNull=true", nil, bearer(token)...).
		expect(t, fiber.StatusOK, `["Fixture Lens 4.2mm f/2.8",null]`)
	h.do(fiber.MethodGet, "/api/search/suggestions?type=city2", nil, bearer(token)...).expect(t, fiber.StatusBadRequest,
		`{"message":"Validation failed","errors":[{"code":"invalid_value","values":["country","state","city","camera-make","camera-model","camera-lens-model"],"path":["type"],"message":"Invalid option: expected one of \"country\"|\"state\"|\"city\"|\"camera-make\"|\"camera-model\"|\"camera-lens-model\""}]}`)

	explore := h.do(fiber.MethodGet, "/api/search/explore", nil, bearer(token)...).array(t)
	if len(explore) != 2 || len(explore[1].(map[string]any)["items"].([]any)) != 3 {
		t.Errorf("explore = %v", explore)
	}

	paths := h.do(fiber.MethodGet, "/api/view/folder/unique-paths", nil, bearer(token)...).array(t)
	if len(paths) != 3 {
		t.Fatalf("unique paths = %v", paths)
	}
	if folder := h.do(fiber.MethodGet, "/api/view/folder?path="+paths[0].(string), nil, bearer(token)...).array(t); len(folder) != 1 {
		t.Errorf("folder = %v", folder)
	}
	h.do(fiber.MethodGet, "/api/view/folder", nil, bearer(token)...).
		expect(t, fiber.StatusInternalServerError, `{"message":"Failed to get assets by original path"}`)
}

func TestVideoUpload(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "korda.mp4")
	cmd := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "testsrc=size=320x240:rate=25:duration=2",
		"-c:v", "mpeg4", "-pix_fmt", "yuv420p", "-metadata", "creation_time=2022-03-04T05:06:07Z", "-y", out)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg cannot encode a test video here: %v %s", err, output)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	h := newHarness(t)
	token := h.admin()
	id := h.uploadID(token, "korda.mp4", data, "2022-03-04T05:06:07.000Z", "2022-03-04T05:06:07.000Z", "duration", "0")
	h.runAllJobs()

	info := h.do(fiber.MethodGet, "/api/assets/"+id, nil, bearer(token)...).object(t)
	exifInfo := info["exifInfo"].(map[string]any)
	if info["type"] != "VIDEO" || info["duration"] != float64(2000) || info["fileCreatedAt"] != "2022-03-04T05:06:07.000Z" ||
		exifInfo["timeZone"] != "UTC" || exifInfo["orientation"] != "1" || info["width"] != float64(320) || info["thumbhash"] == nil {
		t.Errorf("video = %v", info)
	}

	thumb := h.do(fiber.MethodGet, "/api/assets/"+id+"/thumbnail", nil, bearer(token)...)
	if thumb.Status != fiber.StatusOK || thumb.Header.Get(fiber.HeaderContentType) != "image/jpeg" {
		t.Errorf("video thumbnail = %d %v", thumb.Status, thumb.Header)
	}
	play := h.do(fiber.MethodGet, "/api/assets/"+id+"/video/playback", nil, bearer(token)...)
	if play.Status != fiber.StatusOK || play.Header.Get(fiber.HeaderContentType) != "video/mp4" {
		t.Errorf("playback = %d %v", play.Status, play.Header)
	}
	if encoder := gh.Media(h.rt).FFmpeg(); encoder != nil && encoder.Encoder != "" {
		encoded := filepath.Join(dir, "encoded.mp4")
		if err := os.WriteFile(encoded, play.Body, 0o600); err != nil {
			t.Fatal(err)
		}
		probe, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name,height",
			"-of", "csv=p=0", encoded).Output()
		if err != nil || string(probe) != "h264,240\n" {
			t.Errorf("playback of an MPEG-4 Part 2 upload = %q %v, want the H.264 transcode", probe, err)
		}
	}
	redirect := h.do(fiber.MethodGet, "/api/assets/"+id+"/thumbnail?size=fullsize", nil, bearer(token)...)
	if redirect.Status != fiber.StatusFound || redirect.Header.Get(fiber.HeaderLocation) != "/api/assets/"+id+"/thumbnail?size=preview" {
		t.Errorf("fullsize = %d %v", redirect.Status, redirect.Header)
	}
}

func TestHEICUpload(t *testing.T) {
	if !testmedia.HEICSupported {
		t.Skip("HEIC needs the build tag heic")
	}

	h := newHarness(t)
	token := h.admin()
	id := h.uploadID(token, "korda.heic", fixture(t, testmedia.ExifHEIC), "2023-12-24T23:30:00.000Z", "2023-12-24T23:30:00.000Z")
	h.runAllJobs()

	info := h.do(fiber.MethodGet, "/api/assets/"+id, nil, bearer(token)...).object(t)
	exifInfo := info["exifInfo"].(map[string]any)
	if info["localDateTime"] != "2023-12-24T18:30:00.000Z" || exifInfo["timeZone"] != "UTC-5" || exifInfo["orientation"] != nil ||
		info["width"] != float64(800) || info["thumbhash"] == nil {
		t.Errorf("heic = %v", info)
	}
	if preview := h.do(fiber.MethodGet, "/api/assets/"+id+"/thumbnail?size=preview", nil, bearer(token)...); preview.Status != fiber.StatusOK {
		t.Errorf("heic preview = %d %s", preview.Status, preview.Body)
	}
}

func TestSearchLeavesOutPartnersLockedAssets(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	viewer := bearer(admin)
	viewerID := h.do(fiber.MethodGet, "/api/users/me", nil, viewer...).str(t, "id")
	sharerID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	sharerToken := h.login("mira@example.com", "mira-password")
	sharer := bearer(sharerToken)
	h.uploadID(admin, "own.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	locked := h.uploadID(sharerToken, "partner-locked.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.uploadID(sharerToken, "partner-open.png", fixture(t, testmedia.PlainPNG), "2023-01-03T03:04:05.000Z", "2023-01-03T03:04:05.000Z")
	h.runAllJobs()

	for _, s := range [][]string{sharer, viewer} {
		h.do(fiber.MethodPost, "/api/auth/pin-code", `{"pinCode":"123456"}`, s...).expect(t, 204, "")
		h.do(fiber.MethodPost, "/api/auth/session/unlock", `{"pinCode":"123456"}`, s...).expect(t, 204, "")
	}
	h.do(fiber.MethodPut, "/api/assets", `{"ids":["`+locked+`"],"visibility":"locked"}`, sharer...).expect(t, 204, "")
	h.do(fiber.MethodPost, "/api/partners", `{"sharedWithId":"`+viewerID+`"}`, sharer...)
	if res := h.do(fiber.MethodPut, "/api/partners/"+sharerID, `{"inTimeline":true}`, viewer...); res.Status != 200 {
		t.Fatalf("in timeline = %d %s", res.Status, res.Body)
	}

	for _, q := range []struct{ path, body string }{
		{"/api/search/metadata", `{}`},
		{"/api/search/metadata", `{"originalFileName":"partner-locked"}`},
		{"/api/search/random", `{}`},
		{"/api/search/large-assets?minFileSize=0", ``},
		{"/api/search/metadata", `{"filter":{}}`},
	} {
		var body any
		if q.body != "" {
			body = q.body
		}
		res := h.do(fiber.MethodPost, q.path, body, viewer...)
		if res.Status != 200 || bytes.Contains(res.Body, []byte(locked)) || !bytes.Contains(res.Body, []byte("partner-open.png")) && q.body != `{"originalFileName":"partner-locked"}` {
			t.Errorf("%s %s = %d %s", q.path, q.body, res.Status, res.Body)
		}
	}
	h.do(fiber.MethodPost, "/api/search/statistics", `{}`, viewer...).expect(t, 200, `{"total":2}`)
	h.do(fiber.MethodPost, "/api/search/statistics", `{"filter":{}}`, viewer...).expect(t, 200, `{"total":2}`)
	h.do(fiber.MethodPost, "/api/search/statistics", `{}`, sharer...).expect(t, 200, `{"total":2}`)
}
