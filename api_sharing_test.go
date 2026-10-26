package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/internal/testmedia"
	logicmemories "tty.fail/mrus/korda/logic/root/memories"
	"tty.fail/mrus/korda/models/systemmetadata"
)

func mutated(t *testing.T, shade uint8) []byte {
	t.Helper()

	return fixture(t, func() ([]byte, error) { return testmedia.Mutate(shade) })
}

func TestPartnerRules(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	ownerID := h.do(fiber.MethodGet, "/api/users/me", nil, owner...).str(t, "id")
	memberID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))

	h.do(fiber.MethodPost, "/api/partners", `{"sharedWithId":"00000000-0000-4000-8000-000000000000"}`, owner...).
		expect(t, 400, `{"message":"Invalid user"}`)
	created := h.do(fiber.MethodPost, "/api/partners", `{"sharedWithId":"`+memberID+`"}`, owner...)
	if got := normalizeAsset(string(created.Body)); created.Status != 201 || got != `{"id":"<uuid>","email":"mira@example.com","name":"Mira","profileImagePath":"","avatarColor":"<color>","profileChangedAt":"<pgtime>","inTimeline":false}` {
		t.Errorf("create = %d %s", created.Status, got)
	}
	h.do(fiber.MethodPost, "/api/partners", `{"sharedWithId":"`+memberID+`"}`, owner...).expect(t, 400, `{"message":"Partner already exists"}`)
	h.do(fiber.MethodGet, "/api/partners?direction=shared-with", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodPut, "/api/partners/"+ownerID, `{"inTimeline":true}`, member...).expect(t, 200, "")
	if res := h.do(fiber.MethodGet, "/api/partners?direction=shared-with", nil, member...); !strings.Contains(string(res.Body), `"inTimeline":true`) {
		t.Errorf("shared with = %s", res.Body)
	}
	h.do(fiber.MethodDelete, "/api/partners/"+memberID, nil, owner...).expect(t, 204, "")
	h.do(fiber.MethodDelete, "/api/partners/"+memberID, nil, owner...).expect(t, 400, `{"message":"Partner not found"}`)
}

func TestPartnerSyncBackfillsALatePartner(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	viewerID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	viewer := h.login("mira@example.com", "mira-password")
	h.createUser(admin, "carl@example.com", "Carl", "carl-password")
	carl := h.login("carl@example.com", "carl-password")

	old := h.uploadID(carl, "old.jpg", mutated(t, 1), "2021-01-01T00:00:00.000Z", "2021-01-01T00:00:00.000Z")
	first := h.uploadID(admin, "new.jpg", mutated(t, 2), "2022-01-01T00:00:00.000Z", "2022-01-01T00:00:00.000Z")
	second := h.uploadID(admin, "newer.jpg", mutated(t, 3), "2022-01-01T00:00:00.000Z", "2022-01-01T00:00:00.000Z")
	h.runAllJobs()
	h.do(fiber.MethodPost, "/api/stacks", `{"assetIds":["`+first+`","`+second+`"]}`, bearer(admin)...).expect(t, 201, "")
	h.do(fiber.MethodPost, "/api/partners", `{"sharedWithId":"`+viewerID+`"}`, bearer(admin)...).expect(t, 201, "")
	h.ackRuns(viewer, h.appStream(viewer))

	h.do(fiber.MethodPost, "/api/partners", `{"sharedWithId":"`+viewerID+`"}`, bearer(carl)...).expect(t, 201, "")
	lines := h.appStream(viewer)
	want := []string{
		"PartnerV1", "PartnerAssetBackfillV2", "SyncAckV1", "SyncAckV1", "PartnerAssetExifBackfillV1", "SyncAckV1",
		"SyncCompleteV1",
	}
	if got := types(lines); !slices.Equal(got, want) {
		t.Fatalf("types = %v\n want %v", got, want)
	}
	if !strings.Contains(string(lines[1].Data), `"id":"`+old+`"`) || !strings.HasSuffix(lines[3].Ack, "|complete") ||
		!strings.HasPrefix(lines[3].Ack, "PartnerStackBackfillV1|") {
		t.Errorf("backfill = %s, %s", lines[1].Data, lines[3].Ack)
	}

	h.ackRuns(viewer, lines)
	lines = h.appStream(viewer)
	if got := types(lines); !slices.Equal(got, []string{"SyncAckV1", "SyncCompleteV1"}) ||
		!strings.HasPrefix(lines[0].Ack, "PartnerAssetBackfillV2|") {
		t.Fatalf("second stream = %v %v", got, lines)
	}
	h.ackRuns(viewer, lines)
	if got := types(h.appStream(viewer)); !slices.Equal(got, []string{"SyncCompleteV1"}) {
		t.Errorf("third stream = %v", got)
	}
}

func TestStackRules(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	var ids []string
	for i := range 3 {
		ids = append(ids, h.uploadID(admin, "m.jpg", mutated(t, uint8(10+i)), "2021-01-01T00:00:00.000Z", "2021-01-01T00:00:00.000Z"))
	}
	h.runAllJobs()

	h.do(fiber.MethodPost, "/api/stacks", `{"assetIds":["`+ids[0]+`"]}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"array","code":"too_small","minimum":2,"inclusive":true,"path":["assetIds"],"message":"Too small: expected array to have >=2 items"}]}`)
	created := h.do(fiber.MethodPost, "/api/stacks", `{"assetIds":["`+ids[0]+`","`+ids[1]+`"]}`, owner...)
	if created.Status != 201 || created.str(t, "primaryAssetId") != ids[0] {
		t.Fatalf("create = %d %s", created.Status, created.Body)
	}
	stack := "/api/stacks/" + created.str(t, "id")

	h.do(fiber.MethodPut, stack, `{"primaryAssetId":"`+ids[2]+`"}`, owner...).expect(t, 400, `{"message":"Primary asset must be in the stack"}`)
	h.do(fiber.MethodDelete, stack+"/assets/"+ids[0], nil, owner...).expect(t, 400, `{"message":"Cannot remove stack's primary asset"}`)
	h.do(fiber.MethodDelete, stack+"/assets/"+ids[2], nil, owner...).expect(t, 400, `{"message":"Asset not in stack"}`)
	if res := h.do(fiber.MethodGet, "/api/assets/"+ids[1], nil, owner...); !strings.Contains(string(res.Body), `"stack":{"id":"`+created.str(t, "id")+`","primaryAssetId":"`+ids[0]+`","assetCount":2}`) {
		t.Errorf("stacked asset = %s", res.Body)
	}
	h.do(fiber.MethodDelete, stack+"/assets/"+ids[1], nil, owner...).expect(t, 204, "")
	if got := len(h.do(fiber.MethodGet, stack, nil, owner...).object(t)["assets"].([]any)); got != 1 {
		t.Errorf("stack after removal has %d assets", got)
	}
	h.do(fiber.MethodDelete, stack, nil, owner...).expect(t, 204, "")
	h.do(fiber.MethodGet, stack, nil, owner...).expect(t, 400, `{"message":"Not found or no stack.read access"}`)
	h.do(fiber.MethodDelete, "/api/stacks", `{"ids":[]}`, owner...).expect(t, 500, `{"message":"Failed to delete stacks"}`)
}

func TestTagRules(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	asset := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()

	created := h.do(fiber.MethodPost, "/api/tags", `{"name":"Travel","color":"ABC"}`, owner...)
	created.expectShape(t, 201, `{"id":"<uuid>","name":"Travel","value":"Travel","createdAt":"<time>","updatedAt":"<time>","color":"#ABC"}`)
	tag := "/api/tags/" + created.str(t, "id")
	h.do(fiber.MethodPost, "/api/tags", `{"name":"Travel"}`, owner...).expect(t, 400, `{"message":"A tag with that name already exists"}`)
	h.do(fiber.MethodPost, "/api/tags", `{"name":"a/b"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^[^/]*$/","path":["name"],"message":"Tag name cannot contain slash characters (\"/\")"}]}`)
	h.do(fiber.MethodPut, "/api/tags", `{"tags":["Travel/Japan","Travel 2026","travel"]}`, owner...).expect(t, 200, "")

	var values []string
	for _, item := range h.do(fiber.MethodGet, "/api/tags", nil, owner...).array(t) {
		values = append(values, item.(map[string]any)["value"].(string))
	}
	if want := []string{"travel", "Travel", "Travel 2026", "Travel/Japan"}; !slices.Equal(values, want) {
		t.Errorf("tags = %v, want %v", values, want)
	}

	h.do(fiber.MethodPut, tag+"/assets", `{"ids":["`+asset+`","`+asset+`","00000000-0000-4000-8000-000000000000"]}`, owner...).
		expect(t, 200, `[{"id":"`+asset+`","success":true},{"id":"`+asset+`","success":false,"error":"duplicate"},{"id":"00000000-0000-4000-8000-000000000000","success":false,"error":"no_permission"}]`)
	if res := h.do(fiber.MethodGet, "/api/assets/"+asset, nil, owner...); !strings.Contains(string(res.Body), `"tags":[{"id":"`+created.str(t, "id")+`","name":"Travel","value":"Travel"`) {
		t.Errorf("tagged asset = %s", res.Body)
	}
	h.do(fiber.MethodDelete, tag, nil, owner...).expect(t, 204, "")
	values = nil
	for _, item := range h.do(fiber.MethodGet, "/api/tags", nil, owner...).array(t) {
		values = append(values, item.(map[string]any)["value"].(string))
	}
	if want := []string{"travel", "Travel 2026"}; !slices.Equal(values, want) {
		t.Errorf("tags after delete = %v, want %v", values, want)
	}
}

func TestActivityRules(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	memberID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))
	h.createUser(admin, "carl@example.com", "Carl", "carl-password")
	stranger := bearer(h.login("carl@example.com", "carl-password"))
	asset := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()
	albumID := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Holiday","assetIds":["`+asset+`"],"albumUsers":[{"userId":"`+memberID+`","role":"viewer"}]}`, owner...).str(t, "id")

	h.do(fiber.MethodPost, "/api/activities", `{"albumId":"`+albumID+`","type":"comment"}`, member...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"custom","path":["comment"],"message":"Comment is required when type is COMMENT"}]}`)
	h.do(fiber.MethodPost, "/api/activities", `{"albumId":"`+albumID+`","type":"comment","comment":"Nice"}`, stranger...).
		expect(t, 400, `{"message":"Not found or no activity.create access"}`)
	comment := h.do(fiber.MethodPost, "/api/activities", `{"albumId":"`+albumID+`","type":"comment","comment":"Nice"}`, member...)
	if comment.Status != 201 {
		t.Fatalf("comment = %d %s", comment.Status, comment.Body)
	}
	like := h.do(fiber.MethodPost, "/api/activities", `{"albumId":"`+albumID+`","assetId":"`+asset+`","type":"like"}`, member...)
	again := h.do(fiber.MethodPost, "/api/activities", `{"albumId":"`+albumID+`","assetId":"`+asset+`","type":"like"}`, member...)
	if like.Status != 201 || again.Status != 200 || like.str(t, "id") != again.str(t, "id") {
		t.Errorf("likes = %d %s, %d %s", like.Status, like.Body, again.Status, again.Body)
	}
	h.do(fiber.MethodGet, "/api/activities/statistics?albumId="+albumID, nil, owner...).expect(t, 200, `{"comments":1,"likes":1}`)
	h.do(fiber.MethodDelete, "/api/activities/"+comment.str(t, "id"), nil, stranger...).
		expect(t, 400, `{"message":"Not found or no activity.delete access"}`)
	h.do(fiber.MethodDelete, "/api/activities/"+comment.str(t, "id"), nil, owner...).expect(t, 204, "")
	if got := len(h.do(fiber.MethodGet, "/api/activities?albumId="+albumID, nil, member...).array(t)); got != 1 {
		t.Errorf("activities = %d", got)
	}
}

func linkToken(id string, password string) string {
	sum := sha256.Sum256([]byte(id + "-" + password))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func TestSharedLinks(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z")
	h.runAllJobs()
	albumID := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Holiday","assetIds":["`+exif+`"]}`, owner...).str(t, "id")

	h.do(fiber.MethodPost, "/api/shared-links", `{"type":"ALBUM","assetIds":["`+exif+`"]}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"message":"albumId is required for type ALBUM","code":"custom","path":[]},{"message":"assetIds can only be used with type INDIVIDUAL","code":"custom","path":[]}]}`)
	h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["00000000-0000-4000-8000-000000000000"]}`, owner...).
		expect(t, 400, `{"message":"Not found or no asset.share access"}`)
	ind := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+exif+`"],"allowUpload":false,"slug":"one","description":""}`, owner...)
	if ind.Status != 201 || !strings.Contains(string(ind.Body), `"description":null,"password":null`) || len(ind.str(t, "key")) != 67 {
		t.Fatalf("individual = %d %s", ind.Status, ind.Body)
	}
	h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+plain+`"],"slug":"one"}`, owner...).
		expect(t, 400, `{"message":"Failed to save shared link"}`)
	alb := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"ALBUM","albumId":"`+albumID+`","password":"secret","showMetadata":false}`, owner...)
	if alb.Status != 201 || !strings.Contains(string(alb.Body), `"assets":[],"allowUpload":true,"allowDownload":false,"showMetadata":false,"slug":null}`) {
		t.Fatalf("album link = %d %s", alb.Status, alb.Body)
	}
	expired := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+plain+`"],"expiresAt":"2020-01-01T00:00:00Z"}`, owner...)
	indKey, albKey := "?key="+ind.str(t, "key"), "?key="+alb.str(t, "key")

	h.do(fiber.MethodGet, "/api/shared-links/me?key="+expired.str(t, "key"), nil).expect(t, 401, `{"message":"Invalid share key"}`)
	h.do(fiber.MethodGet, "/api/users/me"+indKey, nil).expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodGet, "/api/users/me?key=nope", nil, owner...).expect(t, 401, `{"message":"Invalid share key"}`)
	h.do(fiber.MethodGet, "/api/shared-links/me?slug=one", nil).expect(t, 200, "")
	h.do(fiber.MethodGet, "/api/shared-links/me", nil, owner...).expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodGet, "/api/shared-links/me"+albKey, nil).expect(t, 401, `{"message":"Password required"}`)
	h.do(fiber.MethodPost, "/api/shared-links/login"+indKey, `{"password":"x"}`).expect(t, 400, `{"message":"Shared link is not password protected"}`)
	h.do(fiber.MethodPost, "/api/shared-links/login"+albKey, `{"password":"Secret"}`).expect(t, 401, `{"message":"Invalid password"}`)
	login := h.do(fiber.MethodPost, "/api/shared-links/login"+albKey, `{"password":"secret"}`)
	cookie, _, _ := strings.Cut(login.Header.Get(fiber.HeaderSetCookie), ";")
	if want := "immich_shared_link_token=" + url.QueryEscape(linkToken(alb.str(t, "id"), "secret")); login.Status != 201 || cookie != want {
		t.Fatalf("login = %d %s, want %s", login.Status, cookie, want)
	}
	if res := h.do(fiber.MethodGet, "/api/shared-links/me"+albKey, nil, "Cookie", cookie); res.Status != 200 || !strings.Contains(string(res.Body), `"album":{"albumName":"Holiday"`) {
		t.Errorf("me with cookie = %d %s", res.Status, res.Body)
	}
	h.do(fiber.MethodGet, "/api/assets/"+exif+albKey, nil).expect(t, 401, `{"message":"Password required"}`)

	stripped := h.do(fiber.MethodGet, "/api/assets/"+exif+albKey, nil, "Cookie", cookie)
	if got := normalizeAsset(string(stripped.Body)); got != `{"id":"<uuid>","type":"IMAGE","originalMimeType":"image/jpeg","thumbhash":"<thumbhash>","localDateTime":"2024-05-06T07:08:09.123Z","duration":null,"livePhotoVideoId":null,"hasMetadata":false,"width":480,"height":640}` {
		t.Errorf("stripped = %s", got)
	}
	if res := h.do(fiber.MethodGet, "/api/assets/"+exif+indKey, nil); strings.Contains(string(res.Body), `"owner"`) || !strings.Contains(string(res.Body), `"exifInfo"`) {
		t.Errorf("asset through a link = %s", res.Body)
	}
	h.do(fiber.MethodGet, "/api/assets/"+plain+indKey, nil).expect(t, 400, `{"message":"Not found or no asset.read access"}`)
	h.do(fiber.MethodGet, "/api/assets/"+exif+"/original"+albKey, nil, "Cookie", cookie).expect(t, 404, `{"message":"Not Found"}`)
	if res := h.do(fiber.MethodGet, "/api/assets/"+exif+"/thumbnail"+albKey, nil, "Cookie", cookie); res.Status != 200 ||
		!strings.Contains(res.Header.Get(fiber.HeaderContentDisposition), exif+"_thumbnail") {
		t.Errorf("thumbnail = %d %v", res.Status, res.Header)
	}
	h.do(fiber.MethodGet, "/api/albums/"+albumID+"/map-markers"+albKey, nil, "Cookie", cookie).expect(t, 200, `[]`)
	h.do(fiber.MethodPost, "/api/search/metadata"+indKey, `{}`).
		expect(t, 400, `{"message":"Shared link access is only allowed in combination with an albumIds filter"}`)

	upload := func(key string, shade uint8, headers ...string) apiResponse {
		return h.multipart("/api/assets"+key, []part{
			{field: "deviceAssetId", content: []byte("link")}, {field: "deviceId", content: []byte("browser")},
			{field: "fileCreatedAt", content: []byte("2021-01-01T00:00:00Z")}, {field: "fileModifiedAt", content: []byte("2021-01-01T00:00:00Z")},
			{field: "assetData", filename: "via-link.jpg", content: mutated(t, shade)},
		}, headers...)
	}
	upload(indKey, 40).expect(t, 401, `{"message":"Unauthorized"}`)
	upload(albKey, 41).expect(t, 401, `{"message":"Password required"}`)
	if res := upload(albKey, 41, "Cookie", cookie); res.Status != 201 {
		t.Fatalf("upload through the album link = %d %s", res.Status, res.Body)
	}
	if got := h.do(fiber.MethodGet, "/api/albums/"+albumID, nil, owner...).object(t)["assetCount"]; got != float64(2) {
		t.Errorf("album asset count = %v", got)
	}

	h.do(fiber.MethodPut, "/api/shared-links/"+alb.str(t, "id")+"/assets", `{"assetIds":["`+plain+`"]}`, owner...).
		expect(t, 400, `{"message":"Invalid shared link type"}`)
	h.do(fiber.MethodPut, "/api/shared-links/"+ind.str(t, "id")+"/assets", `{"assetIds":["`+exif+`","`+plain+`"]}`, owner...).
		expect(t, 200, `[{"assetId":"`+exif+`","success":false,"error":"duplicate"},{"assetId":"`+plain+`","success":true}]`)
	h.do(fiber.MethodDelete, "/api/shared-links/"+ind.str(t, "id")+"/assets", `{"assetIds":[]}`, owner...).
		expect(t, 500, `{"message":"Failed to remove shared link assets"}`)
	if res := h.do(fiber.MethodPatch, "/api/shared-links/"+ind.str(t, "id"), `{}`, owner...); !strings.HasSuffix(string(res.Body), `"slug":null}`) {
		t.Errorf("update = %d %s", res.Status, res.Body)
	}
	h.do(fiber.MethodDelete, "/api/shared-links/"+ind.str(t, "id"), nil, owner...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/shared-links/me"+indKey, nil).expect(t, 401, `{"message":"Invalid share key"}`)
	h.do(fiber.MethodGet, "/api/shared-links/"+ind.str(t, "id"), nil, owner...).expect(t, 400, `{"message":"Shared link not found"}`)
}

func TestSharedLinkPasswordOnEveryRoute(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()
	albumID := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Guarded","assetIds":["`+exif+`"]}`, owner...).str(t, "id")
	alb := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"ALBUM","albumId":"`+albumID+`","password":"secret","slug":"guarded"}`, owner...)
	ind := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+exif+`"],"password":"secret"}`, owner...)
	albCookie := "immich_shared_link_token=" + url.QueryEscape(linkToken(alb.str(t, "id"), "secret"))
	indCookie := "immich_shared_link_token=" + url.QueryEscape(linkToken(ind.str(t, "id"), "secret"))

	type route struct {
		method string
		path   string
		body   any
		status int
	}
	stream := "/api/assets/" + exif + "/video/stream/"
	session := "00000000-0000-4000-8000-000000000000"
	routes := []route{
		{fiber.MethodGet, "/api/shared-links/me", nil, 200},
		{fiber.MethodGet, "/api/assets/" + exif, nil, 200},
		{fiber.MethodGet, "/api/assets/" + exif + "/thumbnail", nil, 200},
		{fiber.MethodGet, "/api/assets/" + exif + "/original", nil, 200},
		{fiber.MethodGet, "/api/assets/" + exif + "/video/playback", nil, 404},
		{fiber.MethodGet, stream + "main.m3u8", nil, 400},
		{fiber.MethodGet, stream + session + "/0/playlist.m3u8", nil, 404},
		{fiber.MethodGet, stream + session + "/0/segment.ts", nil, 400},
		{fiber.MethodDelete, stream + session, nil, 204},
		{fiber.MethodPost, "/api/search/metadata", `{"albumIds":["` + albumID + `"]}`, 200},
		{fiber.MethodGet, "/api/albums/" + albumID, nil, 200},
		{fiber.MethodGet, "/api/albums/" + albumID + "/map-markers", nil, 200},
		{fiber.MethodGet, "/api/timeline/buckets?albumId=" + albumID, nil, 200},
		{fiber.MethodGet, "/api/timeline/bucket?albumId=" + albumID + "&timeBucket=2024-05-01T00:00:00.000Z", nil, 200},
		{fiber.MethodPost, "/api/download/info", `{"albumId":"` + albumID + `"}`, 201},
		{fiber.MethodPost, "/api/download/archive", `{"assetIds":["` + exif + `"]}`, 200},
	}
	with := func(path string, credential string) string {
		if strings.Contains(path, "?") {
			return path + "&" + credential
		}
		return path + "?" + credential
	}
	for _, credential := range []string{"key=" + alb.str(t, "key"), "slug=guarded"} {
		for _, r := range routes {
			path := with(r.path, credential)
			h.do(r.method, path, r.body).expect(t, 401, `{"message":"Password required"}`)
			h.do(r.method, path, r.body, "Cookie", indCookie).expect(t, 401, `{"message":"Password required"}`)
			if res := h.do(r.method, path, r.body, "Cookie", albCookie); res.Status != r.status {
				t.Errorf("%s %s with the token = %d %s, want %d", r.method, path, res.Status, res.Body, r.status)
			}
		}
	}

	indKey := "key=" + ind.str(t, "key")
	for _, path := range []string{"/api/shared-links/me", "/api/assets/" + exif, "/api/assets/" + exif + "/original"} {
		h.do(fiber.MethodGet, with(path, indKey), nil).expect(t, 401, `{"message":"Password required"}`)
		if res := h.do(fiber.MethodGet, with(path, indKey), nil, "Cookie", indCookie); res.Status != 200 {
			t.Errorf("%s with the token = %d %s", path, res.Status, res.Body)
		}
	}

	upload := func(credential string, shade uint8, headers ...string) apiResponse {
		return h.multipart(with("/api/assets", credential), []part{
			{field: "deviceAssetId", content: []byte("link")}, {field: "deviceId", content: []byte("browser")},
			{field: "fileCreatedAt", content: []byte("2021-01-01T00:00:00Z")}, {field: "fileModifiedAt", content: []byte("2021-01-01T00:00:00Z")},
			{field: "assetData", filename: "via-link.jpg", content: mutated(t, shade)},
		}, headers...)
	}
	upload("slug=guarded", 50).expect(t, 401, `{"message":"Password required"}`)
	if res := upload("slug=guarded", 50, "Cookie", albCookie); res.Status != 201 {
		t.Errorf("upload with the token = %d %s", res.Status, res.Body)
	}

	if res := h.do(fiber.MethodPost, with("/api/shared-links/login", "slug=guarded"), `{"password":"secret"}`); res.Status != 201 {
		t.Errorf("login by slug = %d %s", res.Status, res.Body)
	}
	h.do(fiber.MethodPatch, "/api/shared-links/"+alb.str(t, "id"), `{"password":"changed"}`, owner...).expect(t, 200, "")
	albKey := "key=" + alb.str(t, "key")
	h.do(fiber.MethodGet, with("/api/assets/"+exif, albKey), nil, "Cookie", albCookie).expect(t, 401, `{"message":"Password required"}`)
	both := albCookie + url.QueryEscape(","+linkToken(alb.str(t, "id"), "changed"))
	if res := h.do(fiber.MethodGet, with("/api/assets/"+exif, albKey), nil, "Cookie", both); res.Status != 200 {
		t.Errorf("asset with both tokens = %d %s", res.Status, res.Body)
	}
}

func TestSharedLinkLoginRateLimit(t *testing.T) {
	h := newHarnessWith(t, harnessOptions{rateLimitAuth: 2, publicUsers: true})
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	link := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+exif+`"],"password":"secret"}`, owner...)
	login := "/api/shared-links/login?key=" + link.str(t, "key")

	for range 2 {
		h.do(fiber.MethodPost, login, `{"password":"guess"}`).expect(t, 401, `{"message":"Invalid password"}`)
	}
	h.do(fiber.MethodPost, login, `{"password":"secret"}`).expect(t, 429, `{"message":"Too Many Requests"}`)
	if res := h.do(fiber.MethodPost, "/api/auth/login", `{"email":"admin@example.com","password":"korda-admin"}`); res.Status != 201 {
		t.Errorf("account login = %d %s", res.Status, res.Body)
	}
}

func TestSharedLinkChecksumNeedsUploadAccess(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	content := fixture(t, testmedia.ExifJPEG)
	exif := h.uploadID(admin, "korda-exif.jpg", content, "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	albumID := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Uploads","assetIds":["`+exif+`"]}`, owner...).str(t, "id")
	readOnly := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+exif+`"],"allowUpload":false}`, owner...)
	uploads := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"ALBUM","albumId":"`+albumID+`"}`, owner...)

	sum := sha1.Sum(content)
	probe := func(key string) apiResponse {
		return h.multipart("/api/assets?key="+key, []part{{field: "assetData", filename: "probe.jpg", content: content}},
			"x-immich-checksum", base64.StdEncoding.EncodeToString(sum[:]))
	}
	probe(readOnly.str(t, "key")).expect(t, 400, `{"message":"Not found or no asset.upload access"}`)
	probe(uploads.str(t, "key")).expect(t, 200, `{"status":"duplicate","id":"`+exif+`"}`)
}

func TestSharedLinkSearchHidesMetadata(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()
	albumID := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Hidden","assetIds":["`+exif+`"]}`, owner...).str(t, "id")
	hidden := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"ALBUM","albumId":"`+albumID+`","showMetadata":false}`, owner...)
	shown := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"ALBUM","albumId":"`+albumID+`"}`, owner...)

	search := func(key string) string {
		res := h.do(fiber.MethodPost, "/api/search/metadata?key="+key, `{"albumIds":["`+albumID+`"],"withExif":true}`)
		if res.Status != 200 {
			t.Fatalf("search = %d %s", res.Status, res.Body)
		}
		return normalizeAsset(string(res.Body))
	}
	if got, want := search(hidden.str(t, "key")), `{"albums":{"total":0,"count":0,"items":[],"facets":[]},"assets":{"total":1,"count":1,"items":[{"id":"<uuid>","type":"IMAGE","originalMimeType":"image/jpeg","thumbhash":"<thumbhash>","localDateTime":"2024-05-06T07:08:09.123Z","duration":null,"livePhotoVideoId":null,"hasMetadata":false,"width":480,"height":640}],"facets":[],"nextPage":null,"nextCursor":null}}`; got != want {
		t.Errorf("search without metadata = %s\n                   want %s", got, want)
	}
	if got := search(shown.str(t, "key")); !strings.Contains(got, `"exifInfo":{`) || !strings.Contains(got, `"originalFileName":"korda-exif.jpg"`) {
		t.Errorf("search with metadata = %s", got)
	}
}

func dated(t *testing.T, width int, date string) []byte {
	t.Helper()

	data, err := testmedia.JPEG(testmedia.Pattern(width, 48), testmedia.Tags{Make: "korda", DateTimeOriginal: date}.TIFF())
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func TestMemoryRules(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	asset := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()

	h.do(fiber.MethodPost, "/api/memories", `{"type":"birthday","data":{"year":99},"memoryAt":"2024-05-06"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"number","code":"too_small","minimum":1000,"inclusive":true,"path":["data","year"],"message":"Too small: expected number to be >=1000"},{"origin":"string","code":"invalid_format","format":"datetime","pattern":"/^(?:(?:\\d\\d[2468][048]|\\d\\d[13579][26]|\\d\\d0[48]|[02468][048]00|[13579][26]00)-02-29|\\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\\d|30)|(?:02)-(?:0[1-9]|1\\d|2[0-8])))T(?:(?:[01]\\d|2[0-3]):[0-5]\\d(?::[0-5]\\d(?:\\.\\d+)?)?(?:Z|([+-](?:[01]\\d|2[0-3]):[0-5]\\d)))$/","path":["memoryAt"],"message":"Invalid input: expected ISO 8601 datetime string, received string"},{"code":"custom","path":["data","personId"],"message":"Required for birthday memories"},{"code":"custom","path":["data","personName"],"message":"Required for birthday memories"}]}`)
	created := h.do(fiber.MethodPost, "/api/memories", `{"type":"on_this_day","data":{"year":2024,"extra":1},"memoryAt":"2024-05-06T00:00:00Z","assetIds":["`+asset+`","00000000-0000-4000-8000-000000000000"]}`, owner...)
	if created.Status != 201 || !strings.Contains(string(created.Body), `"type":"on_this_day","data":{"year":2024},"isSaved":false,"assets":[{"id":"`+asset+`"`) {
		t.Fatalf("create = %d %s", created.Status, created.Body)
	}
	memory := "/api/memories/" + created.str(t, "id")
	h.do(fiber.MethodPut, memory, `{}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"custom","path":[],"message":"At least one of the following fields is required: isSaved, seenAt, memoryAt"}]}`)
	h.do(fiber.MethodPut, memory+"/assets", `{"ids":["`+asset+`"]}`, owner...).expect(t, 200, `[{"id":"`+asset+`","success":false,"error":"duplicate"}]`)
	h.do(fiber.MethodDelete, memory+"/assets", `{"ids":["`+asset+`","`+asset+`"]}`, owner...).
		expect(t, 200, `[{"id":"`+asset+`","success":true},{"id":"`+asset+`","success":false,"error":"not_found"}]`)
	h.do(fiber.MethodGet, "/api/memories", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/memories/statistics", nil, owner...).expect(t, 200, `{"total":1}`)
	h.do(fiber.MethodDelete, memory, nil, owner...).expect(t, 204, "")
	h.do(fiber.MethodGet, memory, nil, owner...).expect(t, 400, `{"message":"Not found or no memory.read access"}`)
}

func TestMemoryGeneration(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	for i, date := range []string{"2025:03:15 10:00:00", "2024:03:16 23:30:00", "2020:03:12 10:00:00", "2026:03:15 08:00:00", "2024:02:29 12:00:00", "2027:02:28 12:00:00"} {
		h.uploadID(admin, "d.jpg", dated(t, 64+i, date), "2021-01-01T00:00:00Z", "2021-01-01T00:00:00Z")
	}
	h.runAllJobs()

	generate := func(at string) {
		t.Helper()
		parsed, err := time.Parse(time.RFC3339, at)
		if err != nil {
			t.Fatal(err)
		}
		if err = logicmemories.GenerateAt(h.ctx, h.rt, parsed); err != nil {
			t.Fatal(err)
		}
	}
	summary := func() []string {
		t.Helper()
		var out []string
		for _, item := range h.do(fiber.MethodGet, "/api/memories?order=asc", nil, owner...).array(t) {
			m := item.(map[string]any)
			out = append(out, fmt.Sprintf("%v %v %v %v %d", m["memoryAt"], m["showAt"], m["hideAt"], m["data"], len(m["assets"].([]any))))
		}
		return out
	}

	generate("2026-03-15T00:05:00Z")
	want := []string{
		"2025-03-15T00:00:00.000Z 2026-03-15T00:00:00.000Z 2026-03-15T23:59:59.999Z map[year:2025] 1",
		"2024-03-16T00:00:00.000Z 2026-03-16T00:00:00.000Z 2026-03-16T23:59:59.999Z map[year:2024] 1",
	}
	if got := summary(); !slices.Equal(got, want) {
		t.Fatalf("memories = %v\n want %v", got, want)
	}
	state, err := gh.Repositories(h.rt).SystemMetadata.Get(h.ctx, systemmetadata.MemoriesState)
	if err != nil || state != `{"lastOnThisDayDate":"2026-03-18T00:00:00.000Z"}` {
		t.Errorf("state = %s, %v", state, err)
	}

	generate("2026-03-15T23:00:00Z")
	if got := summary(); !slices.Equal(got, want) {
		t.Errorf("after a second run = %v", got)
	}
	h.do(fiber.MethodGet, "/api/memories/statistics?for=2026-03-16", nil, owner...).expect(t, 200, `{"total":1}`)

	generate("2028-03-01T00:05:00Z")
	got := summary()
	if len(got) != 3 || got[2] != "2027-02-28T00:00:00.000Z 2028-02-28T00:00:00.000Z 2028-02-28T23:59:59.999Z map[year:2027] 1" {
		t.Errorf("after the leap day = %v", got)
	}
}
