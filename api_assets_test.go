package main

import (
	"crypto/sha1"
	"encoding/base64"
	"io/fs"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
	"tty.fail/mrus/korda/services/metadata"
)

const (
	referenceExifInfo  = `{"id":"<uuid>","createdAt":"<time>","ownerId":"<uuid>","owner":{"id":"<uuid>","email":"admin@example.com","name":"Admin","profileImagePath":"","avatarColor":"<color>","profileChangedAt":"<pgtime>"},"libraryId":null,"type":"IMAGE","originalPath":"<path>","originalFileName":"korda-exif.jpg","originalMimeType":"image/jpeg","thumbhash":"<thumbhash>","fileCreatedAt":"2024-05-06T05:08:09.123Z","fileModifiedAt":"2024-05-06T05:00:00.000Z","localDateTime":"2024-05-06T07:08:09.123Z","updatedAt":"<time>","isFavorite":false,"isArchived":false,"isTrashed":false,"visibility":"timeline","duration":null,"exifInfo":{"make":"korda","model":"Fixture One","exifImageWidth":640,"exifImageHeight":480,"fileSizeInByte":15619,"orientation":"6","dateTimeOriginal":"2024-05-06T05:08:09.123+00:00","modifyDate":"2024-05-06T05:00:00+00:00","timeZone":"UTC+2","lensModel":"Fixture Lens 4.2mm f/2.8","fNumber":2.8,"focalLength":4.2,"iso":100,"exposureTime":"1/250","latitude":52.516272,"longitude":13.377722,"city":null,"state":null,"country":null,"description":"a fixture","projectionType":null,"rating":4},"livePhotoVideoId":null,"tags":[],"people":[],"checksum":"7lWy+xd67RvuCrGVWCyajV1LYuU=","stack":null,"isOffline":false,"hasMetadata":true,"duplicateId":null,"resized":true,"width":480,"height":640,"isEdited":false}`
	referencePlainInfo = `{"id":"<uuid>","createdAt":"<time>","ownerId":"<uuid>","owner":{"id":"<uuid>","email":"admin@example.com","name":"Admin","profileImagePath":"","avatarColor":"<color>","profileChangedAt":"<pgtime>"},"libraryId":null,"type":"IMAGE","originalPath":"<path>","originalFileName":"korda-plain.jpg","originalMimeType":"image/jpeg","thumbhash":"<thumbhash>","fileCreatedAt":"2020-02-03T04:05:06.000Z","fileModifiedAt":"2020-02-04T04:05:06.000Z","localDateTime":"2020-02-03T04:05:06.000Z","updatedAt":"<time>","isFavorite":true,"isArchived":false,"isTrashed":false,"visibility":"timeline","duration":null,"exifInfo":{"make":null,"model":null,"exifImageWidth":300,"exifImageHeight":200,"fileSizeInByte":4158,"orientation":null,"dateTimeOriginal":"2020-02-03T04:05:06+00:00","modifyDate":"2020-02-04T04:05:06+00:00","timeZone":null,"lensModel":null,"fNumber":null,"focalLength":null,"iso":null,"exposureTime":null,"latitude":null,"longitude":null,"city":null,"state":null,"country":null,"description":"","projectionType":null,"rating":null},"livePhotoVideoId":null,"tags":[],"people":[],"checksum":"fH6Q5NN0jzG5Sav36O5+hEiU6o8=","stack":null,"isOffline":false,"hasMetadata":true,"duplicateId":null,"resized":true,"width":300,"height":200,"isEdited":false}`
)

var (
	pgTimePattern    = regexp.MustCompile(`"profileChangedAt":"[^"]+"`)
	pathPattern      = regexp.MustCompile(`"originalPath":"[^"]+"`)
	thumbhashPattern = regexp.MustCompile(`"thumbhash":"[^"]+"`)
	colorPattern     = regexp.MustCompile(`"avatarColor":"[a-z]+"`)
	stampPattern     = regexp.MustCompile(`"(createdAt|updatedAt)":"[^"]+"`)
)

func normalizeAsset(body string) string {
	body = pgTimePattern.ReplaceAllString(body, `"profileChangedAt":"<pgtime>"`)
	body = pathPattern.ReplaceAllString(body, `"originalPath":"<path>"`)
	body = thumbhashPattern.ReplaceAllString(body, `"thumbhash":"<thumbhash>"`)
	body = colorPattern.ReplaceAllString(body, `"avatarColor":"<color>"`)
	body = stampPattern.ReplaceAllString(body, `"$1":"<time>"`)

	return uuidPattern.ReplaceAllString(body, "<uuid>")
}

func fixture(t *testing.T, build func() ([]byte, error)) []byte {
	t.Helper()

	data, err := build()
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func (h *apiHarness) upload(token string, name string, content []byte, fields ...string) apiResponse {
	h.t.Helper()

	parts := []part{
		{field: "deviceAssetId", content: []byte("test-" + name)},
		{field: "deviceId", content: []byte("korda-test")},
	}
	for i := 0; i+1 < len(fields); i += 2 {
		parts = append(parts, part{field: fields[i], content: []byte(fields[i+1])})
	}
	parts = append(parts, part{field: "assetData", filename: name, content: content})

	return h.multipart("/api/assets", parts, bearer(token)...)
}

func (h *apiHarness) uploadID(token string, name string, content []byte, created string, modified string, fields ...string) string {
	h.t.Helper()

	res := h.upload(token, name, content, append([]string{"fileCreatedAt", created, "fileModifiedAt", modified}, fields...)...)
	if res.Status != fiber.StatusCreated {
		h.t.Fatalf("upload %s = %d %s", name, res.Status, res.Body)
	}

	return res.str(h.t, "id")
}

func (h *apiHarness) runAllJobs() {
	h.t.Helper()

	for range 10 {
		if len(h.queue.ids()) == 0 {
			return
		}
		h.runJobs()
	}
	h.t.Fatalf("jobs keep queueing: %v", h.queue.ids())
}

func TestUploadMatchesImmich(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	exif := fixture(t, testmedia.ExifJPEG)

	res := h.upload(token, "korda-exif.jpg", exif, "fileCreatedAt", "2024-05-06T05:00:00.000Z", "fileModifiedAt", "2024-05-06T05:00:00.000Z")
	res.expectShape(t, fiber.StatusCreated, `{"id":"<uuid>","status":"created"}`)
	id := res.str(t, "id")

	h.upload(token, "korda-exif.jpg", exif, "fileCreatedAt", "2024-05-06T05:00:00.000Z", "fileModifiedAt", "2024-05-06T05:00:00.000Z").
		expect(t, fiber.StatusOK, `{"status":"duplicate","id":"`+id+`"}`)

	sum := sha1.Sum(exif)
	checksumRes := h.multipart("/api/assets", []part{{field: "assetData", filename: "korda-exif.jpg", content: exif}},
		append(bearer(token), "x-immich-checksum", base64.StdEncoding.EncodeToString(sum[:]))...)
	checksumRes.expect(t, fiber.StatusOK, `{"status":"duplicate","id":"`+id+`"}`)

	h.runAllJobs()

	got := h.do(fiber.MethodGet, "/api/assets/"+id, nil, bearer(token)...)
	if body := normalizeAsset(string(got.Body)); got.Status != fiber.StatusOK || body != referenceExifInfo {
		t.Errorf("asset info = %d\n got %s\nwant %s", got.Status, body, referenceExifInfo)
	}

	plain := h.uploadID(token, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG),
		"2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z", "isFavorite", "true", "duration", "0")
	h.runAllJobs()
	got = h.do(fiber.MethodGet, "/api/assets/"+plain, nil, bearer(token)...)
	if body := normalizeAsset(string(got.Body)); body != referencePlainInfo {
		t.Errorf("plain asset info\n got %s\nwant %s", body, referencePlainInfo)
	}
}

func TestUploadRejectsLikeImmich(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	plain := fixture(t, testmedia.PlainJPEG)
	dates := []string{"fileCreatedAt", "2020-02-03T04:05:06.000Z", "fileModifiedAt", "2020-02-04T04:05:06.000Z"}

	h.do(fiber.MethodPost, "/api/assets", nil, bearer(token)...).expect(t, fiber.StatusBadRequest,
		`{"message":"Validation failed","errors":[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}]}`)

	h.multipart("/api/assets", []part{{field: "fileCreatedAt", content: []byte("2020-02-03T04:05:06.000Z")},
		{field: "fileModifiedAt", content: []byte("2020-02-03T04:05:06.000Z")}}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"File is required"}`)

	h.multipart("/api/assets", []part{{field: "fileCreatedAt", content: []byte("2020-02-03T04:05:06.000Z")},
		{field: "fileModifiedAt", content: []byte("2020-02-03T04:05:06.000Z")},
		{field: "sidecarData", filename: "x.xmp", content: []byte("<x/>")}}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Field(s) assetData should not be empty"}`)

	h.upload(token, "notes.txt", []byte("hello"), dates...).
		expect(t, fiber.StatusBadRequest, `{"message":"Unsupported file type notes.txt"}`)
	h.upload(token, "empty.jpg", nil, dates...).
		expect(t, fiber.StatusBadRequest, `{"message":"File is empty"}`)
	h.upload(token, "renamed.bin", plain, append([]string{"filename", "x.txt"}, dates...)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Unsupported file type x.txt"}`)

	h.upload(token, "korda-plain.jpg", plain).expect(t, fiber.StatusBadRequest,
		`{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["fileCreatedAt"],"message":"Invalid input: expected ISO 8601 datetime string, received undefined"},{"expected":"string","code":"invalid_type","path":["fileModifiedAt"],"message":"Invalid input: expected ISO 8601 datetime string, received undefined"}]}`)

	h.upload(token, "korda-plain.jpg", plain, append([]string{"isFavorite", "true", "isFavorite", "true"}, dates...)...).
		expect(t, fiber.StatusBadRequest,
			`{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["isFavorite"],"message":"Invalid input: expected string, received array"}]}`)

	userID := h.createUser(token, "quota@example.com", "Quota", "quota-password")
	h.do(fiber.MethodPut, "/api/admin/users/"+userID, map[string]any{"quotaSizeInBytes": 1000}, bearer(token)...).expect(t, fiber.StatusOK, "")
	quota := h.login("quota@example.com", "quota-password")
	h.upload(quota, "korda-plain.jpg", plain, dates...).
		expect(t, fiber.StatusBadRequest, `{"message":"Quota has been exceeded!"}`)

	entries, err := walkFiles(h.dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("rejected uploads left files: %v", entries)
	}
}

func TestAssetMediaRoutes(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	id := h.uploadID(token, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()

	thumb := h.do(fiber.MethodGet, "/api/assets/"+id+"/thumbnail", nil, bearer(token)...)
	if thumb.Status != fiber.StatusOK || thumb.Header.Get(fiber.HeaderContentType) != "image/jpeg" ||
		thumb.Header.Get(fiber.HeaderContentDisposition) != "inline; filename*=UTF-8''korda-exif_thumbnail.jpeg" ||
		thumb.Header.Get(fiber.HeaderCacheControl) != "private, max-age=86400, no-transform, stale-while-revalidate=2592000, stale-if-error=2592000" {
		t.Errorf("thumbnail = %d %v", thumb.Status, thumb.Header)
	}

	original := h.do(fiber.MethodGet, "/api/assets/"+id+"/original", nil, bearer(token)...)
	if original.Status != fiber.StatusOK || original.Header.Get(fiber.HeaderETag) != `W/"3d03-18f4c45e480"` ||
		original.Header.Get(fiber.HeaderLastModified) != "Mon, 06 May 2024 05:00:00 GMT" || len(original.Body) != 15619 {
		t.Errorf("original = %d %v", original.Status, original.Header)
	}

	ranged := httptest.NewRequest(fiber.MethodGet, "/api/assets/"+id+"/original", nil)
	ranged.Header.Set(fiber.HeaderAuthorization, "Bearer "+token)
	ranged.Header.Set(fiber.HeaderRange, "bytes=0-9")
	if res := h.send(ranged); res.Status != fiber.StatusPartialContent || len(res.Body) != 10 ||
		res.Header.Get(fiber.HeaderContentRange) != "bytes 0-9/15619" {
		t.Errorf("range = %d %v", res.Status, res.Header)
	}

	fullsize := h.do(fiber.MethodGet, "/api/assets/"+id+"/thumbnail?size=fullsize&c=x", nil, bearer(token)...)
	if fullsize.Status != fiber.StatusFound || fullsize.Header.Get(fiber.HeaderLocation) != "original?c=x" ||
		string(fullsize.Body) != "Found. Redirecting to original?c=x" {
		t.Errorf("fullsize = %d %v %s", fullsize.Status, fullsize.Header, fullsize.Body)
	}

	h.do(fiber.MethodGet, "/api/assets/"+id+"/video/playback", nil, bearer(token)...).
		expect(t, fiber.StatusNotFound, `{"message":"Not Found"}`)
	h.do(fiber.MethodGet, "/api/assets/0f0e0d0c-0b0a-4908-8706-050403020100/thumbnail", nil, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Not found or no asset.view access"}`)
	h.do(fiber.MethodGet, "/api/assets/0f0e0d0c-0b0a-4908-8706-050403020100/original", nil, bearer(token)...).
		expect(t, fiber.StatusNotFound, `{"message":"Not Found"}`)
	h.do(fiber.MethodGet, "/api/assets/"+id+"/video/stream/main.m3u8", nil, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Real-time transcoding is not enabled"}`)
}

func TestAssetLifecycle(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	ids := make([]string, 0, 3)
	for i, shade := range []uint8{10, 90, 170} {
		data := fixture(t, func() ([]byte, error) { return testmedia.Mutate(shade) })
		ids = append(ids, h.uploadID(token, "korda-mutate-"+string(rune('1'+i))+".jpg", data,
			"2019-07-08T09:10:11.000Z", "2019-07-08T09:10:11.000Z"))
	}
	h.runAllJobs()
	a1, a2, a3 := ids[0], ids[1], ids[2]

	res := h.do(fiber.MethodPut, "/api/assets/"+a1, map[string]any{"isFavorite": true, "description": "edited", "rating": 5}, bearer(token)...)
	body := res.object(t)
	exifInfo, _ := body["exifInfo"].(map[string]any)
	if res.Status != fiber.StatusOK || body["isFavorite"] != true || exifInfo["description"] != "edited" || exifInfo["rating"] != float64(5) {
		t.Errorf("update = %d %s", res.Status, res.Body)
	}
	h.do(fiber.MethodPut, "/api/assets/"+a1, map[string]any{"rating": 7}, bearer(token)...).expect(t, fiber.StatusBadRequest,
		`{"message":"Validation failed","errors":[{"origin":"number","code":"too_big","maximum":5,"inclusive":true,"path":["rating"],"message":"Too big: expected number to be <=5"}]}`)

	res = h.do(fiber.MethodPut, "/api/assets/"+a1, map[string]any{"dateTimeOriginal": "2020-01-02T03:04:05.000+02:00"}, bearer(token)...)
	exifInfo, _ = res.object(t)["exifInfo"].(map[string]any)
	if exifInfo["dateTimeOriginal"] != "2020-01-02T01:04:05+00:00" || exifInfo["timeZone"] != "UTC+2" {
		t.Errorf("date edit = %s", res.Body)
	}
	h.runAllJobs()
	res = h.do(fiber.MethodGet, "/api/assets/"+a1, nil, bearer(token)...)
	if b := res.object(t); b["fileCreatedAt"] != "2020-01-02T01:04:05.000Z" || b["localDateTime"] != "2020-01-02T03:04:05.000Z" {
		t.Errorf("dates after extraction = %s", res.Body)
	}

	h.do(fiber.MethodPut, "/api/assets", map[string]any{"ids": []string{a2}, "dateTimeRelative": 60}, bearer(token)...).
		expect(t, fiber.StatusNoContent, "")
	h.runAllJobs()
	if b := h.do(fiber.MethodGet, "/api/assets/"+a2, nil, bearer(token)...).object(t); b["fileCreatedAt"] != "2019-07-08T10:10:11.000Z" {
		t.Errorf("relative shift = %v", b["fileCreatedAt"])
	}

	h.do(fiber.MethodGet, "/api/assets/statistics", nil, bearer(token)...).expect(t, fiber.StatusOK, `{"images":3,"videos":0,"total":3}`)

	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{a3}}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	if b := h.do(fiber.MethodGet, "/api/assets/"+a3, nil, bearer(token)...).object(t); b["isTrashed"] != true {
		t.Errorf("trashed asset = %v", b["isTrashed"])
	}
	h.do(fiber.MethodGet, "/api/assets/statistics?isTrashed=true", nil, bearer(token)...).expect(t, fiber.StatusOK, `{"images":1,"videos":0,"total":1}`)
	h.do(fiber.MethodPost, "/api/trash/restore/assets", map[string]any{"ids": []string{a3, a3}}, bearer(token)...).expect(t, fiber.StatusOK, `{"count":2}`)
	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{a3}}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.do(fiber.MethodPost, "/api/trash/restore", nil, bearer(token)...).expect(t, fiber.StatusOK, `{"count":1}`)
	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{a3}}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.do(fiber.MethodPost, "/api/trash/empty", nil, bearer(token)...).expect(t, fiber.StatusOK, `{"count":1}`)
	h.runAllJobs()
	h.do(fiber.MethodGet, "/api/assets/"+a3, nil, bearer(token)...).expect(t, fiber.StatusBadRequest, `{"message":"Not found or no asset.read access"}`)

	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{a2}, "force": true}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.runAllJobs()
	h.do(fiber.MethodGet, "/api/assets/"+a2, nil, bearer(token)...).expect(t, fiber.StatusBadRequest, `{"message":"Not found or no asset.read access"}`)

	me := h.do(fiber.MethodGet, "/api/users/me", nil, bearer(token)...).object(t)
	remaining := h.do(fiber.MethodGet, "/api/assets/"+a1, nil, bearer(token)...).object(t)["exifInfo"].(map[string]any)["fileSizeInByte"]
	if me["quotaUsageInBytes"] != remaining {
		t.Errorf("quota usage = %v, want %v", me["quotaUsageInBytes"], remaining)
	}
	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{"0f0e0d0c-0b0a-4908-8706-050403020100"}}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Not found or no asset.delete access"}`)
}

func TestStatisticsAndHeatmapsCountAssets(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	me := h.do(fiber.MethodGet, "/api/users/me", nil, bearer(token)...).str(t, "id")

	h.uploadID(token, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	plain := h.uploadID(token, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG),
		"2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z", "isFavorite", "true")
	h.runAllJobs()
	today := time.Now().UTC().Format("2006-01-02")

	h.do(fiber.MethodGet, "/api/server/statistics", nil, bearer(token)...).expect(t, fiber.StatusOK,
		`{"photos":2,"videos":0,"usage":19777,"usagePhotos":19777,"usageVideos":0,"usageByUser":[{"userId":"`+me+
			`","userName":"Admin","photos":2,"videos":0,"usage":19777,"usagePhotos":19777,"usageVideos":0,"quotaSizeInBytes":null}]}`)
	h.do(fiber.MethodGet, "/api/admin/users/"+me+"/statistics?isFavorite=true", nil, bearer(token)...).
		expect(t, fiber.StatusOK, `{"images":1,"videos":0,"total":1}`)
	h.do(fiber.MethodGet, "/api/users/me/calendar-heatmap?from=2024-05-05&to=2024-05-07&type=Taken", nil, bearer(token)...).
		expect(t, fiber.StatusOK, `{"from":"2024-05-05","to":"2024-05-07","series":[{"date":"2024-05-05","count":0},`+
			`{"date":"2024-05-06","count":1},{"date":"2024-05-07","count":0}],"totalCount":1}`)
	h.do(fiber.MethodGet, "/api/admin/users/"+me+"/calendar-heatmap?from="+today+"&to="+today, nil, bearer(token)...).
		expect(t, fiber.StatusOK, `{"from":"`+today+`","to":"`+today+`","series":[{"date":"`+today+`","count":2}],"totalCount":2}`)

	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{plain}}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.do(fiber.MethodGet, "/api/server/statistics", nil, bearer(token)...).expect(t, fiber.StatusOK,
		`{"photos":1,"videos":0,"usage":15619,"usagePhotos":15619,"usageVideos":0,"usageByUser":[{"userId":"`+me+
			`","userName":"Admin","photos":1,"videos":0,"usage":15619,"usagePhotos":15619,"usageVideos":0,"quotaSizeInBytes":null}]}`)
	h.do(fiber.MethodGet, "/api/admin/users/"+me+"/statistics?isTrashed=true", nil, bearer(token)...).
		expect(t, fiber.StatusOK, `{"images":1,"videos":0,"total":1}`)
	h.do(fiber.MethodGet, "/api/users/me/calendar-heatmap?from="+today+"&to="+today, nil, bearer(token)...).
		expect(t, fiber.StatusOK, `{"from":"`+today+`","to":"`+today+`","series":[{"date":"`+today+`","count":1}],"totalCount":1}`)
}

func TestGPSZone(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	id := h.uploadID(token, "korda-gps.jpg", fixture(t, testmedia.GPSJPEG), "2024-07-01T12:00:00.000Z", "2024-07-01T12:00:00.000Z")
	h.runAllJobs()

	want := map[string]any{
		"fileCreatedAt":    "2024-07-01T12:00:00.000Z",
		"localDateTime":    "2024-07-01T12:00:00.000Z",
		"dateTimeOriginal": "2024-07-01T12:00:00+00:00",
		"timeZone":         nil,
	}
	if metadata.GPSZones {
		want["fileCreatedAt"] = "2024-07-01T10:00:00.000Z"
		want["dateTimeOriginal"] = "2024-07-01T10:00:00+00:00"
		want["timeZone"] = "Europe/Berlin"
	}

	info := h.do(fiber.MethodGet, "/api/assets/"+id, nil, bearer(token)...).object(t)
	exifInfo := info["exifInfo"].(map[string]any)
	got := map[string]any{
		"fileCreatedAt":    info["fileCreatedAt"],
		"localDateTime":    info["localDateTime"],
		"dateTimeOriginal": exifInfo["dateTimeOriginal"],
		"timeZone":         exifInfo["timeZone"],
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %v, want %v", key, got[key], value)
		}
	}
}

func TestAssetMetadataEndpoints(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	id := h.uploadID(token, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z",
		"metadata", `[{"key":"mobile-app","value":{"iCloudId":"x","createdAt":"c","adjustmentTime":"a","latitude":"1","longitude":"2"}}]`)

	res := h.do(fiber.MethodGet, "/api/assets/"+id+"/metadata", nil, bearer(token)...)
	if got := normalize(string(res.Body)); got != `[{"key":"mobile-app","value":{"iCloudId":"x","latitude":"1","createdAt":"c","longitude":"2","adjustmentTime":"a"},"updatedAt":"<time>"}]` {
		t.Errorf("metadata = %s", got)
	}

	h.do(fiber.MethodPut, "/api/assets/"+id+"/metadata", map[string]any{"items": []any{
		map[string]any{"key": "k", "value": map[string]any{}}, map[string]any{"key": "k", "value": map[string]any{}},
	}}, bearer(token)...).expect(t, fiber.StatusBadRequest, `{"message":"Duplicate items are not allowed: \"k\""}`)

	h.do(fiber.MethodGet, "/api/assets/"+id+"/metadata/nope", nil, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Metadata with key \"nope\" not found for asset with id \"`+id+`\""}`)
	h.do(fiber.MethodDelete, "/api/assets/"+id+"/metadata/mobile-app", nil, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.do(fiber.MethodGet, "/api/assets/"+id+"/metadata", nil, bearer(token)...).expect(t, fiber.StatusOK, `[]`)
	h.do(fiber.MethodPut, "/api/assets/metadata", map[string]any{"items": []any{}}, bearer(token)...).
		expect(t, fiber.StatusInternalServerError, `{"message":"Failed to update bulk asset metadata"}`)

	h.runAllJobs()
	h.do(fiber.MethodGet, "/api/assets/"+id+"/ocr", nil, bearer(token)...).expect(t, fiber.StatusOK, `[]`)
	h.do(fiber.MethodGet, "/api/assets/"+id+"/edits", nil, bearer(token)...).expect(t, fiber.StatusOK, `{"assetId":"`+id+`","edits":[]}`)
}

func walkFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.Contains(path, "/upload/") {
			out = append(out, path)
		}
		return nil
	})

	return out, err
}
