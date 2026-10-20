package main

import (
	"bytes"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/internal/testmedia"
)

var appSyncTypes = []string{
	"AuthUsersV2", "UsersV1", "AssetsV2", "AssetExifsV1", "AssetEditsV1", "AssetMetadataV1", "PartnersV1",
	"PartnerAssetsV2", "PartnerAssetExifsV1", "AlbumsV2", "AlbumUsersV1", "AlbumAssetsV2", "AlbumAssetExifsV1",
	"AlbumToAssetsV1", "MemoriesV2", "MemoryToAssetsV2", "StacksV1", "PartnerStacksV1", "UserMetadataV1", "PeopleV1",
	"AssetFacesV3", "AssetOcrV1",
}

var syncTimePattern = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z"`)

type syncLine struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Ack  string          `json:"ack"`
}

func (h *apiHarness) stream(token string, body any) (apiResponse, []syncLine) {
	h.t.Helper()

	time.Sleep(3 * time.Millisecond)
	res := h.do(fiber.MethodPost, "/api/sync/stream", body, bearer(token)...)
	if res.Status != fiber.StatusOK {
		return res, nil
	}
	if !bytes.HasSuffix(res.Body, []byte("\n")) {
		h.t.Fatalf("stream does not end with a newline: %q", res.Body)
	}

	var lines []syncLine
	for raw := range bytes.SplitSeq(bytes.TrimSuffix(res.Body, []byte("\n")), []byte("\n")) {
		var l syncLine
		if err := json.Unmarshal(raw, &l); err != nil {
			h.t.Fatalf("line %q: %v", raw, err)
		}
		lines = append(lines, l)
	}

	return res, lines
}

func (h *apiHarness) appStream(token string) []syncLine {
	h.t.Helper()

	res, lines := h.stream(token, map[string]any{"types": appSyncTypes})
	if res.Status != fiber.StatusOK {
		h.t.Fatalf("stream = %d %s", res.Status, res.Body)
	}

	return lines
}

func (h *apiHarness) ackRuns(token string, lines []syncLine) {
	h.t.Helper()

	var acks []string
	for i, l := range lines {
		if i+1 == len(lines) || lines[i+1].Type != l.Type {
			acks = append(acks, l.Ack)
		}
	}
	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": acks}, bearer(token)...).expect(h.t, fiber.StatusNoContent, "")
}

func types(lines []syncLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.Type)
	}

	return out
}

func normalizeSync(raw []byte) string {
	return uuidPattern.ReplaceAllString(syncTimePattern.ReplaceAllString(string(raw), `"<time>"`), "<uuid>")
}

func TestSyncStreamOfAFreshSession(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	id := h.uploadID(token, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()

	res, lines := h.stream(token, map[string]any{"types": appSyncTypes})
	if res.Header.Get(fiber.HeaderContentType) != "application/jsonlines+json" {
		t.Errorf("content type = %q", res.Header.Get(fiber.HeaderContentType))
	}
	if got := types(lines); !slices.Equal(got, []string{"AuthUserV2", "UserV1", "AssetV2", "AssetExifV1", "SyncCompleteV1"}) {
		t.Fatalf("types = %v", got)
	}

	want := []string{
		`{"type":"AuthUserV2","data":{"id":"<uuid>","name":"Admin","email":"admin@example.com","avatarColor":null,"deletedAt":null,"profileChangedAt":"<time>","isAdmin":true,"pinCode":null,"oauthId":null,"storageLabel":"admin","quotaSizeInBytes":null,"quotaUsageInBytes":15619,"hasProfileImage":false},"ack":"AuthUserV2|<uuid>"}`,
		`{"type":"UserV1","data":{"id":"<uuid>","name":"Admin","email":"admin@example.com","avatarColor":null,"deletedAt":null,"profileChangedAt":"<time>","hasProfileImage":false},"ack":"UserV1|<uuid>"}`,
		`{"type":"AssetV2","data":{"id":"<uuid>","ownerId":"<uuid>","originalFileName":"korda-exif.jpg","fileCreatedAt":"<time>","fileModifiedAt":"<time>","createdAt":"<time>","localDateTime":"<time>","type":"IMAGE","deletedAt":null,"isFavorite":false,"visibility":"timeline","duration":null,"livePhotoVideoId":null,"stackId":null,"libraryId":null,"width":480,"height":640,"isEdited":false,"checksum":"7lWy+xd67RvuCrGVWCyajV1LYuU=","thumbhash":"`,
		`{"type":"AssetExifV1","data":{"assetId":"<uuid>","description":"a fixture","exifImageWidth":640,"exifImageHeight":480,"fileSizeInByte":15619,"orientation":"6","dateTimeOriginal":"<time>","modifyDate":"<time>","timeZone":"UTC+2","latitude":52.516272,"longitude":13.377722,"projectionType":null,"city":null,"state":null,"country":null,"make":"korda","model":"Fixture One","lensModel":"Fixture Lens 4.2mm f/2.8","fNumber":2.8,"focalLength":4.2,"iso":100,"exposureTime":"1/250","profileDescription":null,"rating":4,"fps":null},"ack":"AssetExifV1|<uuid>"}`,
		`{"type":"SyncCompleteV1","data":{},"ack":"SyncCompleteV1|<uuid>"}`,
	}
	body := strings.Split(strings.TrimSuffix(string(res.Body), "\n"), "\n")
	for i, w := range want {
		if got := normalizeSync([]byte(body[i])); !strings.HasPrefix(got, w) {
			t.Errorf("line %d\n got %s\nwant %s", i, got, w)
		}
	}
	if !strings.Contains(body[2], `"id":"`+id+`"`) || !strings.Contains(body[2], `"fileCreatedAt":"2024-05-06T05:08:09.123Z"`) ||
		!strings.Contains(body[2], `"localDateTime":"2024-05-06T07:08:09.123Z"`) {
		t.Errorf("asset line = %s", body[2])
	}

	for _, l := range lines[:4] {
		ackID := strings.TrimPrefix(l.Ack, l.Type+"|")
		if parsed, err := uuid.Parse(ackID); err != nil || parsed.Version() != 7 {
			t.Errorf("ack %s is not a UUIDv7", l.Ack)
		}
	}
	complete := strings.TrimPrefix(lines[4].Ack, "SyncCompleteV1|")
	for _, l := range lines[:4] {
		if strings.TrimPrefix(l.Ack, l.Type+"|") >= complete {
			t.Errorf("ack %s is not below the stream's nowId %s", l.Ack, complete)
		}
	}
}

func TestSyncAcksLimitTheNextStream(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	first := h.uploadID(token, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()

	h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...).expect(t, fiber.StatusOK, `[]`)
	lines := h.appStream(token)
	h.ackRuns(token, lines)

	stored := h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...)
	var acks []map[string]string
	if err := json.Unmarshal(stored.Body, &acks); err != nil {
		t.Fatal(err)
	}
	var stored_ []string
	for _, a := range acks {
		stored_ = append(stored_, a["type"])
		if !strings.HasPrefix(a["ack"], a["type"]+"|") {
			t.Errorf("ack %v", a)
		}
	}
	if !slices.Equal(stored_, []string{"AuthUserV2", "UserV1", "AssetV2", "AssetExifV1", "SyncCompleteV1"}) {
		t.Errorf("stored acks = %s", stored.Body)
	}

	if got := types(h.appStream(token)); !slices.Equal(got, []string{"SyncCompleteV1"}) {
		t.Errorf("acked stream = %v", got)
	}

	h.do(fiber.MethodPut, "/api/assets/"+first, map[string]any{"isFavorite": true}, bearer(token)...)
	lines = h.appStream(token)
	if got := types(lines); !slices.Equal(got, []string{"AssetV2", "SyncCompleteV1"}) || !strings.Contains(string(lines[0].Data), `"isFavorite":true`) {
		t.Errorf("stream after a favorite = %v %s", got, lines[0].Data)
	}

	h.do(fiber.MethodDelete, "/api/sync/ack", map[string]any{"types": []string{"AssetV2"}}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	if got := types(h.appStream(token)); !slices.Equal(got, []string{"AssetV2", "SyncCompleteV1"}) {
		t.Errorf("stream after deleting the AssetV2 ack = %v", got)
	}

	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{"AssetV2|01a15504-c72b-71fa-9f54-9ab9079d9dca", "AssetV2|00000000-0000-7000-8000-000000000000"}}, bearer(token)...).
		expect(t, fiber.StatusNoContent, "")
	res := h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...)
	if !strings.Contains(string(res.Body), `{"type":"AssetV2","ack":"AssetV2|00000000-0000-7000-8000-000000000000"}`) {
		t.Errorf("the last ack of a type does not win: %s", res.Body)
	}

	for _, pre := range [][]string{
		{"AlbumV1"},
		{"AssetExifV1", "PartnerAssetExifV1", "AlbumAssetExifCreateV1", "AlbumAssetExifUpdateV1"},
		{"AssetV1", "PartnerAssetV1", "AlbumAssetCreateV1", "AlbumAssetUpdateV1"},
		{"AssetV1", "AssetV2"},
	} {
		h.do(fiber.MethodDelete, "/api/sync/ack", map[string]any{"types": pre}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	}
	res = h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...)
	if strings.Contains(string(res.Body), `"AssetV2"`) || strings.Contains(string(res.Body), `"AssetExifV1"`) || !strings.Contains(string(res.Body), `"UserV1"`) {
		t.Errorf("acks after the app's pre-sync deletes = %s", res.Body)
	}

	h.do(fiber.MethodDelete, "/api/sync/ack", map[string]any{}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...).expect(t, fiber.StatusOK, `[]`)
}

func TestSyncDeletesTrashAndMetadata(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	trashed := h.uploadID(token, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z")
	deleted := h.uploadID(token, "korda.png", fixture(t, testmedia.PlainPNG), "2021-01-02T03:04:05.000Z", "2021-01-02T03:04:05.000Z")
	h.runAllJobs()
	h.ackRuns(token, h.appStream(token))

	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{trashed}}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{deleted}, "force": true}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.runAllJobs()
	h.do(fiber.MethodPut, "/api/assets/"+trashed+"/metadata", map[string]any{"items": []map[string]any{{"key": "mobile-app", "value": map[string]any{"zeta": 1, "a": true}}}}, bearer(token)...)
	h.do(fiber.MethodPut, "/api/users/me/onboarding", map[string]any{"isOnboarded": true}, bearer(token)...)

	lines := h.appStream(token)
	got := types(lines)
	if !slices.Equal(got, []string{"AuthUserV2", "UserV1", "AssetDeleteV1", "AssetV2", "UserMetadataV1", "AssetMetadataV1", "SyncCompleteV1"}) {
		t.Fatalf("types = %v", got)
	}
	if string(lines[2].Data) != `{"assetId":"`+deleted+`"}` {
		t.Errorf("delete = %s", lines[2].Data)
	}
	if !strings.Contains(string(lines[3].Data), `"id":"`+trashed+`"`) || strings.Contains(string(lines[3].Data), `"deletedAt":null`) {
		t.Errorf("trashed asset = %s", lines[3].Data)
	}
	if !strings.HasSuffix(string(lines[4].Data), `"key":"onboarding","value":{"isOnboarded":true}}`) {
		t.Errorf("user metadata = %s", lines[4].Data)
	}
	if string(lines[5].Data) != `{"assetId":"`+trashed+`","key":"mobile-app","value":{"a":true,"zeta":1}}` {
		t.Errorf("asset metadata = %s", lines[5].Data)
	}
	h.ackRuns(token, lines)

	h.do(fiber.MethodDelete, "/api/assets/"+trashed+"/metadata/mobile-app", nil, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.do(fiber.MethodDelete, "/api/users/me/onboarding", nil, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	lines = h.appStream(token)
	if got := types(lines); !slices.Equal(got, []string{"UserMetadataDeleteV1", "AssetMetadataDeleteV1", "SyncCompleteV1"}) {
		t.Fatalf("types after metadata deletes = %v", got)
	}
	if !strings.HasSuffix(string(lines[0].Data), `,"key":"onboarding"}`) || string(lines[1].Data) != `{"assetId":"`+trashed+`","key":"mobile-app"}` {
		t.Errorf("metadata deletes = %s %s", lines[0].Data, lines[1].Data)
	}
}

func TestSyncResets(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	h.ackRuns(token, h.appStream(token))

	reset := `{"type":"SyncResetV1","data":{},"ack":"SyncResetV1|reset"}` + "\n"

	res, lines := h.stream(token, map[string]any{"types": appSyncTypes, "reset": true})
	if res.Status != fiber.StatusOK || len(lines) != 3 || lines[2].Type != "SyncCompleteV1" {
		t.Errorf("reset stream = %d %s", res.Status, res.Body)
	}
	h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...).expect(t, fiber.StatusOK, `[]`)

	h.ackRuns(token, lines)
	sessions := h.do(fiber.MethodGet, "/api/sessions", nil, bearer(token)...)
	var list []map[string]any
	if err := json.Unmarshal(sessions.Body, &list); err != nil || len(list) == 0 {
		t.Fatalf("sessions = %s", sessions.Body)
	}
	session := list[0]["id"].(string)
	h.do(fiber.MethodPut, "/api/sessions/"+session, map[string]any{"isPendingSyncReset": true}, bearer(token)...)
	for range 2 {
		if res, _ := h.stream(token, map[string]any{"types": []string{}}); string(res.Body) != reset {
			t.Errorf("pending stream = %s", res.Body)
		}
	}
	if res, _ := h.stream(token, map[string]any{"types": []string{"AssetsV1"}}); string(res.Body) != reset {
		t.Errorf("pending stream with a deprecated type = %d %s", res.Status, res.Body)
	}
	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{"UserV1|01a15504-c72b-71fa-9f54-9ab9079d9dca", "SyncResetV1|reset"}}, bearer(token)...).
		expect(t, fiber.StatusNoContent, "")
	h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...).expect(t, fiber.StatusOK, `[]`)
	if got := types(h.appStream(token)); len(got) != 3 {
		t.Errorf("stream after the reset ack = %v", got)
	}

	old := uuidV7At(time.Now().AddDate(0, 0, -31))
	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{"SyncCompleteV1|" + old}}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	if res, _ := h.stream(token, map[string]any{"types": appSyncTypes}); string(res.Body) != reset {
		t.Errorf("stream with a 31 day old SyncCompleteV1 = %s", res.Body)
	}
	recent := uuidV7At(time.Now().AddDate(0, 0, -29))
	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{"SyncCompleteV1|" + recent}}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	if res, lines := h.stream(token, map[string]any{"types": appSyncTypes}); len(lines) != 3 {
		t.Errorf("stream with a 29 day old SyncCompleteV1 = %s", res.Body)
	}
}

func uuidV7At(t time.Time) string {
	u := uuid.Must(uuid.NewV7())
	ms := uint64(t.UnixMilli())
	for i := range 6 {
		u[i] = byte(ms >> (40 - 8*i))
	}

	return u.String()
}

func TestSyncErrors(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	key := h.apiKey(token, "all")

	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{fiber.MethodPost, "/api/sync/stream", map[string]any{"types": []string{}}},
		{fiber.MethodGet, "/api/sync/ack", nil},
		{fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{"AssetV2|x"}}},
		{fiber.MethodDelete, "/api/sync/ack", map[string]any{}},
	} {
		h.do(tc.method, tc.path, tc.body, "x-api-key", key).
			expect(t, fiber.StatusForbidden, `{"message":"Sync endpoints cannot be used with API keys"}`)
	}
	h.do(fiber.MethodPost, "/api/sync/stream", map[string]any{"types": "AssetsV2"}, "x-api-key", key).
		expect(t, fiber.StatusBadRequest, `{"message":"Validation failed","errors":[{"expected":"array","code":"invalid_type","path":["types"],"message":"Invalid input: expected array, received string"}]}`)
	limited := h.apiKey(token, "asset.read")
	h.do(fiber.MethodPost, "/api/sync/stream", map[string]any{"types": []string{}}, "x-api-key", limited).
		expect(t, fiber.StatusForbidden, `{"message":"Missing required permission: sync.stream"}`)

	h.do(fiber.MethodPost, "/api/sync/stream", map[string]any{"types": []string{"AssetsV1"}}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"SyncRequestType.AssetsV1 is deprecated, use SyncRequestType.AssetsV2 instead"}`)
	h.do(fiber.MethodPost, "/api/sync/stream", map[string]any{"types": []string{"AssetFacesV1"}}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"SyncRequestType.AssetFacesV1 is deprecated, use SyncRequestType.AssetFacesV2 instead"}`)
	h.do(fiber.MethodPost, "/api/sync/stream", map[string]any{}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Validation failed","errors":[{"expected":"array","code":"invalid_type","path":["types"],"message":"Invalid input: expected array, received undefined"}]}`)
	h.do(fiber.MethodPost, "/api/sync/stream", map[string]any{"types": []string{}, "reset": "yes"}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["reset"],"message":"Invalid input: expected boolean, received string"}]}`)
	res := h.do(fiber.MethodPost, "/api/sync/stream", map[string]any{"types": []string{"Foo"}}, bearer(token)...)
	if res.Status != fiber.StatusBadRequest || !strings.Contains(string(res.Body), `"code":"invalid_value","values":["AlbumsV1","AlbumsV2","AlbumUsersV1",`) ||
		!strings.HasSuffix(string(res.Body), `"path":["types",0],"message":"Invalid option: expected one of \"AlbumsV1\"|\"AlbumsV2\"|\"AlbumUsersV1\"|\"AlbumToAssetsV1\"|\"AlbumAssetsV1\"|\"AlbumAssetsV2\"|\"AlbumAssetExifsV1\"|\"AssetsV1\"|\"AssetsV2\"|\"AssetExifsV1\"|\"AssetEditsV1\"|\"AssetMetadataV1\"|\"AssetOcrV1\"|\"AuthUsersV1\"|\"AuthUsersV2\"|\"MemoriesV1\"|\"MemoriesV2\"|\"MemoryToAssetsV1\"|\"MemoryToAssetsV2\"|\"PartnersV1\"|\"PartnerAssetsV1\"|\"PartnerAssetsV2\"|\"PartnerAssetExifsV1\"|\"PartnerStacksV1\"|\"StacksV1\"|\"UsersV1\"|\"PeopleV1\"|\"AssetFacesV1\"|\"AssetFacesV2\"|\"AssetFacesV3\"|\"UserMetadataV1\""}]}`) {
		t.Errorf("unknown type = %d %s", res.Status, res.Body)
	}
	if res, lines := h.stream(token, map[string]any{"types": []string{}}); len(lines) != 1 || lines[0].Type != "SyncCompleteV1" {
		t.Errorf("empty types = %s", res.Body)
	}

	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{}}, bearer(token)...).
		expect(t, fiber.StatusInternalServerError, `{"message":"Failed to send sync ack"}`)
	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{"AssetV2|01a15504-c72b-71fa-9f54-9ab9079d9dca", "Nope|x"}}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Invalid ack type: Nope"}`)
	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{""}}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Invalid ack type: "}`)
	h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...).expect(t, fiber.StatusOK, `[]`)
	many := make([]string, 1001)
	for i := range many {
		many[i] = "AssetV2|x"
	}
	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": many}, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Validation failed","errors":[{"origin":"array","code":"too_big","maximum":1000,"inclusive":true,"path":["acks"],"message":"Too big: expected array to have <=1000 items"}]}`)
	h.do(fiber.MethodPost, "/api/sync/ack", nil, bearer(token)...).
		expect(t, fiber.StatusBadRequest, `{"message":"Validation failed","errors":[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}]}`)
	h.do(fiber.MethodDelete, "/api/sync/ack", map[string]any{"types": []string{}}, bearer(token)...).
		expect(t, fiber.StatusInternalServerError, `{"message":"Failed to delete sync ack"}`)

	h.do(fiber.MethodPost, "/api/sync/ack", map[string]any{"acks": []string{"AssetV2|not-a-uuid|extra|more"}}, bearer(token)...).
		expect(t, fiber.StatusNoContent, "")
	if res := h.do(fiber.MethodGet, "/api/sync/ack", nil, bearer(token)...); !strings.Contains(string(res.Body), `"ack":"AssetV2|not-a-uuid|extra|more"`) {
		t.Errorf("stored ack = %s", res.Body)
	}
	h.do(fiber.MethodPost, "/api/sync/stream", map[string]any{"types": []string{"AssetsV2"}}, bearer(token)...).
		expect(t, fiber.StatusInternalServerError, `{"message":"Internal server error"}`)
	if res, lines := h.stream(token, map[string]any{"types": []string{"UsersV1"}}); len(lines) == 0 {
		t.Errorf("a stream without the broken type = %d %s", res.Status, res.Body)
	}
}

func TestSyncAuditCleanup(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	id := h.uploadID(token, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z")
	h.runAllJobs()
	h.do(fiber.MethodDelete, "/api/assets", map[string]any{"ids": []string{id}, "force": true}, bearer(token)...).expect(t, fiber.StatusNoContent, "")
	h.runAllJobs()

	repo := gh.Repositories(h.rt).Sync
	if n, err := repo.CleanupAudit(h.ctx, time.Now().UTC().AddDate(0, 0, -31)); err != nil || n != 0 {
		t.Errorf("cleanup of fresh rows = %d %v", n, err)
	}
	if n, err := repo.CleanupAudit(h.ctx, time.Now().UTC().Add(time.Minute)); err != nil || n != 1 {
		t.Errorf("cleanup of every row = %d %v", n, err)
	}
	if got := types(h.appStream(token)); slices.Contains(got, "AssetDeleteV1") {
		t.Errorf("stream after the cleanup = %v", got)
	}
}
