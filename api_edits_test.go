package main

import (
	"bytes"
	"image/jpeg"
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

var editTaskPattern = regexp.MustCompile(`^edit:[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func (h *apiHarness) assetSize(token string, id string) (int, int, bool) {
	h.t.Helper()

	o := h.do(fiber.MethodGet, "/api/assets/"+id, nil, bearer(token)...).object(h.t)
	return int(o["width"].(float64)), int(o["height"].(float64)), o["isEdited"].(bool)
}

func (h *apiHarness) jpegSize(token string, path string) (int, int) {
	h.t.Helper()

	res := h.do(fiber.MethodGet, path, nil, bearer(token)...)
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(res.Body))
	if err != nil {
		h.t.Fatalf("%s = %d %s: %v", path, res.Status, res.Header, err)
	}

	return cfg.Width, cfg.Height
}

func TestAssetEdits(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()
	_, initial := h.stream(admin, `{"types":["AssetEditsV1"]}`)
	h.ackRuns(admin, initial)
	rec := h.recordSockets()

	h.do(fiber.MethodGet, "/api/assets/"+plain+"/edits", nil, owner...).expect(t, 200, `{"assetId":"`+plain+`","edits":[]}`)
	h.do(fiber.MethodPut, "/api/assets/"+plain+"/edits", `{"edits":[{"action":"crop","parameters":{"x":101,"y":0,"width":200,"height":120}}]}`, owner...).
		expect(t, 400, `{"message":"Crop parameters are out of bounds"}`)
	h.do(fiber.MethodPut, "/api/assets/"+plain+"/edits", `{"edits":[{"action":"rotate","parameters":{"angle":90}},{"action":"crop","parameters":{"x":0,"y":0,"width":10,"height":10}}]}`, owner...).
		expect(t, 400, `{"message":"Crop action must be the first edit action"}`)
	h.do(fiber.MethodPut, "/api/assets/abc/edits", `{"edits":[]}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"array","code":"too_small","minimum":1,"inclusive":true,"path":["edits"],"message":"Too small: expected array to have >=1 items"}]}`)
	h.do(fiber.MethodPut, "/api/assets/"+exif+"/edits", `{"edits":[{"action":"crop","parameters":{"x":0,"y":0,"width":640,"height":480}}]}`, owner...).
		expect(t, 400, `{"message":"Crop parameters are out of bounds"}`)

	h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	stranger := bearer(h.login("mira@example.com", "mira-password"))
	h.do(fiber.MethodPut, "/api/assets/"+plain+"/edits", `{"edits":[{"action":"rotate","parameters":{"angle":90}}]}`, stranger...).
		expect(t, 400, `{"message":"Not found or no asset.edit.create access"}`)
	h.do(fiber.MethodGet, "/api/assets/"+plain+"/edits", nil, stranger...).expect(t, 400, `{"message":"Not found or no asset.read access"}`)
	h.do(fiber.MethodDelete, "/api/assets/"+plain+"/edits", nil, stranger...).
		expect(t, 400, `{"message":"Not found or no asset.edit.delete access"}`)
	if ids := h.queue.ids(); len(ids) != 0 {
		t.Fatalf("queued = %v", ids)
	}

	res := h.do(fiber.MethodPut, "/api/assets/"+plain+"/edits", `{"edits":[{"action":"crop","parameters":{"height":120,"width":200,"y":20,"x":30}},{"action":"rotate","parameters":{"angle":90}}]}`, owner...)
	res.expectShape(t, 200, `{"assetId":"<uuid>","edits":[{"id":"<uuid>","action":"crop","parameters":{"x":30,"y":20,"width":200,"height":120}},{"id":"<uuid>","action":"rotate","parameters":{"angle":90}}]}`)
	if got := h.do(fiber.MethodGet, "/api/assets/"+plain+"/edits", nil, owner...); string(got.Body) != string(res.Body) {
		t.Errorf("edits = %s, want %s", got.Body, res.Body)
	}
	if ids := h.queue.ids(); len(ids) != 1 || !editTaskPattern.MatchString(ids[0]) {
		t.Fatalf("queued = %v", ids)
	}
	if w, hh, edited := h.assetSize(admin, plain); w != 300 || hh != 200 || !edited {
		t.Errorf("before the job = %d by %d, edited %v", w, hh, edited)
	}
	rec.take()
	h.runAllJobs()
	if w, hh, edited := h.assetSize(admin, plain); w != 120 || hh != 200 || !edited {
		t.Errorf("after the job = %d by %d, edited %v", w, hh, edited)
	}
	got := rec.take()
	if len(got) != 1 || got[0].event != "AssetEditReadyV2" {
		t.Fatalf("events = %+v", got)
	}
	if want := `{"asset":{"id":"<uuid>","ownerId":"<uuid>","originalFileName":"korda-plain.jpg","thumbhash":"<thumbhash>","checksum":"fH6Q5NN0jzG5Sav36O5+hEiU6o8=","fileCreatedAt":"2023-01-02T03:04:05.000Z","fileModifiedAt":"2023-01-02T03:04:05.000Z","createdAt":"<time>","localDateTime":"2023-01-02T03:04:05.000Z","duration":null,"type":"IMAGE","deletedAt":null,"isFavorite":false,"visibility":"timeline","livePhotoVideoId":null,"stackId":null,"libraryId":null,"width":120,"height":200,"isEdited":true},"edit":[{"id":"<uuid>","assetId":"<uuid>","sequence":0,"action":"crop","parameters":{"x":30,"y":20,"width":200,"height":120}},{"id":"<uuid>","assetId":"<uuid>","sequence":1,"action":"rotate","parameters":{"angle":90}}]}`; normalizeAsset(got[0].args[0]) != want {
		t.Errorf("AssetEditReadyV2 = %s", normalizeAsset(got[0].args[0]))
	}

	if w, hh := h.jpegSize(admin, "/api/assets/"+plain+"/thumbnail?size=preview&edited=true"); w != 120 || hh != 200 {
		t.Errorf("edited preview = %d by %d", w, hh)
	}
	if w, hh := h.jpegSize(admin, "/api/assets/"+plain+"/thumbnail?size=preview"); w != 300 || hh != 200 {
		t.Errorf("unedited preview = %d by %d", w, hh)
	}
	if w, hh := h.jpegSize(admin, "/api/assets/"+plain+"/original?edited=true"); w != 120 || hh != 200 {
		t.Errorf("edited original = %d by %d", w, hh)
	}

	_, lines := h.stream(admin, `{"types":["AssetEditsV1"]}`)
	if ts := types(lines); len(ts) != 3 || ts[0] != "AssetEditV1" || ts[1] != "AssetEditV1" || ts[2] != "SyncCompleteV1" {
		t.Fatalf("sync = %v", ts)
	}
	if got := normalizeSync(lines[0].Data); got != `{"id":"<uuid>","assetId":"<uuid>","sequence":0,"action":"crop","parameters":{"x":30,"y":20,"width":200,"height":120}}` {
		t.Errorf("AssetEditV1 = %s", got)
	}
	h.ackRuns(admin, lines)

	h.do(fiber.MethodDelete, "/api/assets/"+plain+"/edits", nil, owner...).expect(t, 204, "")
	rec.take()
	h.runAllJobs()
	if w, hh, edited := h.assetSize(admin, plain); w != 300 || hh != 200 || edited {
		t.Errorf("after the removal = %d by %d, edited %v", w, hh, edited)
	}
	if got := rec.take(); len(got) != 1 || got[0].event != "AssetEditReadyV2" || !bytes.Contains([]byte(got[0].args[0]), []byte(`"edit":[]`)) {
		t.Errorf("removal events = %+v", got)
	}
	if w, hh := h.jpegSize(admin, "/api/assets/"+plain+"/thumbnail?size=preview&edited=true"); w != 300 || hh != 200 {
		t.Errorf("preview after the removal = %d by %d", w, hh)
	}
	_, lines = h.stream(admin, `{"types":["AssetEditsV1"]}`)
	if ts := types(lines); len(ts) != 3 || ts[0] != "AssetEditDeleteV1" || ts[1] != "AssetEditDeleteV1" {
		t.Errorf("sync after the removal = %v", ts)
	}

	h.do(fiber.MethodPut, "/api/assets/"+exif+"/edits", `{"edits":[{"action":"crop","parameters":{"x":0,"y":0,"width":480,"height":320}},{"action":"mirror","parameters":{"axis":"horizontal"}}]}`, owner...).
		expectShape(t, 200, `{"assetId":"<uuid>","edits":[{"id":"<uuid>","action":"crop","parameters":{"x":0,"y":0,"width":480,"height":320}},{"id":"<uuid>","action":"mirror","parameters":{"axis":"horizontal"}}]}`)
	h.runAllJobs()
	if w, hh, _ := h.assetSize(admin, exif); w != 480 || hh != 320 {
		t.Errorf("edited EXIF image = %d by %d", w, hh)
	}
	h.do(fiber.MethodPost, "/api/assets/jobs", `{"assetIds":["`+exif+`"],"name":"regenerate-thumbnail"}`, owner...)
	h.runAllJobs()
	if w, hh, edited := h.assetSize(admin, exif); w != 480 || hh != 320 || !edited {
		t.Errorf("edited EXIF image after a new thumbnail job = %d by %d, edited %v", w, hh, edited)
	}
}

func TestAssetFiles(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	h.runAllJobs()
	partnerID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	partner := bearer(h.login("mira@example.com", "mira-password"))
	h.do(fiber.MethodPost, "/api/partners", `{"sharedWithId":"`+partnerID+`"}`, owner...)

	h.do(fiber.MethodGet, "/api/asset-files", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["assetId"],"message":"Invalid input: expected string, received undefined"}]}`)
	h.do(fiber.MethodGet, "/api/asset-files?assetId="+plain+"&type=original", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["fullsize","preview","thumbnail","sidecar","encoded_video"],"path":["type"],"message":"Invalid option: expected one of \"fullsize\"|\"preview\"|\"thumbnail\"|\"sidecar\"|\"encoded_video\""}]}`)
	files := h.do(fiber.MethodGet, "/api/asset-files?assetId="+plain, nil, owner...)
	files.expectShape(t, 200, `[{"id":"<uuid>","createdAt":"<time>","updatedAt":"<time>","type":"preview","path":"thumbs/<uuid>/`+plain[:2]+`/`+plain[2:4]+`/<uuid>_preview.jpeg","isEdited":false,"isProgressive":false,"isTransparent":false},{"id":"<uuid>","createdAt":"<time>","updatedAt":"<time>","type":"thumbnail","path":"thumbs/<uuid>/`+plain[:2]+`/`+plain[2:4]+`/<uuid>_thumbnail.jpeg","isEdited":false,"isProgressive":false,"isTransparent":false}]`)
	if got := h.do(fiber.MethodGet, "/api/asset-files?assetId="+plain, nil, partner...); string(got.Body) != string(files.Body) {
		t.Errorf("partner list = %d %s", got.Status, got.Body)
	}
	h.do(fiber.MethodGet, "/api/asset-files?assetId="+plain+"&isEdited=true", nil, owner...).expect(t, 200, `[]`)

	preview := files.array(t)[0].(map[string]any)["id"].(string)
	h.do(fiber.MethodGet, "/api/asset-files/"+preview, nil, partner...).
		expect(t, 400, `{"message":"Not found or no assetFile.read access"}`)
	if got := h.do(fiber.MethodGet, "/api/asset-files/"+preview, nil, owner...); got.Status != 200 || got.str(t, "type") != "preview" {
		t.Errorf("file = %d %s", got.Status, got.Body)
	}
	h.do(fiber.MethodGet, "/api/asset-files/"+preview+"/download", nil, partner...).expect(t, 404, `{"message":"Not Found"}`)
	if got := h.do(fiber.MethodGet, "/api/asset-files/"+preview+"/download", nil, owner...); got.Status != 200 ||
		got.Header.Get(fiber.HeaderContentType) != "image/jpeg" ||
		got.Header.Get(fiber.HeaderContentDisposition) != "inline; filename*=UTF-8''"+plain+"_preview.jpeg" {
		t.Errorf("download = %d %v", got.Status, got.Header)
	}
	h.do(fiber.MethodDelete, "/api/asset-files/"+preview, nil, partner...).
		expect(t, 400, `{"message":"Not found or no assetFile.delete access"}`)
	h.do(fiber.MethodDelete, "/api/asset-files/"+preview, nil, owner...).expect(t, 204, "")
	h.do(fiber.MethodDelete, "/api/asset-files/"+preview, nil, owner...).
		expect(t, 400, `{"message":"Not found or no assetFile.delete access"}`)
	h.do(fiber.MethodGet, "/api/assets/"+plain+"/thumbnail?size=preview", nil, owner...).expect(t, 404, `{"message":"Asset media not found"}`)
}

func TestEditsQueueOneRenderEach(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	h.runAllJobs()

	h.do(fiber.MethodPut, "/api/assets/"+plain+"/edits", `{"edits":[{"action":"rotate","parameters":{"angle":90}}]}`, owner...).expect(t, 200, "")
	h.do(fiber.MethodPut, "/api/assets/"+plain+"/edits", `{"edits":[{"action":"rotate","parameters":{"angle":180}}]}`, owner...).expect(t, 200, "")
	if ids := h.queue.ids(); len(ids) != 2 || !editTaskPattern.MatchString(ids[0]) || !editTaskPattern.MatchString(ids[1]) || ids[0] == ids[1] {
		t.Fatalf("queued = %v", ids)
	}
	h.runAllJobs()
	if w, hh, edited := h.assetSize(admin, plain); w != 300 || hh != 200 || !edited {
		t.Errorf("after both renders = %d by %d, edited %v", w, hh, edited)
	}
}
