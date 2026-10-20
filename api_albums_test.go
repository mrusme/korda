package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

func TestAlbumRules(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	ownerID := h.do(fiber.MethodGet, "/api/users/me", nil, owner...).str(t, "id")
	memberID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))
	asset := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()

	h.do(fiber.MethodPost, "/api/albums", `{"albumName":"x","albumUsers":[{"userId":"00000000-0000-4000-8000-000000000000","role":"viewer"}]}`, owner...).
		expect(t, 400, `{"message":"Invalid user"}`)
	created := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Holiday","assetIds":["`+asset+`"],"albumUsers":[{"userId":"`+memberID+`","role":"viewer"}]}`, owner...)
	if got := normalizeAsset(string(created.Body)); created.Status != 201 || got != `{"albumName":"Holiday","description":"","albumThumbnailAssetId":"<uuid>","createdAt":"<time>","updatedAt":"<time>","id":"<uuid>","albumUsers":[{"user":{"id":"<uuid>","email":"admin@example.com","name":"Admin","profileImagePath":"","avatarColor":"<color>","profileChangedAt":"<pgtime>"},"role":"owner"},{"user":{"id":"<uuid>","email":"mira@example.com","name":"Mira","profileImagePath":"","avatarColor":"<color>","profileChangedAt":"<pgtime>"},"role":"viewer"}],"shared":true,"hasSharedLink":false,"startDate":"2024-05-06T07:08:09.123+00:00","endDate":"2024-05-06T07:08:09.123+00:00","assetCount":1,"isActivityEnabled":true,"order":"desc"}` {
		t.Fatalf("create = %d %s", created.Status, got)
	}
	album := "/api/albums/" + created.str(t, "id")

	h.do(fiber.MethodPut, album+"/users", `{"albumUsers":[{"userId":"`+memberID+`","role":"owner"}]}`, owner...).
		expect(t, 400, `{"message":"Cannot add another owner"}`)
	h.do(fiber.MethodPut, album+"/user/"+ownerID, `{"role":"editor"}`, owner...).expect(t, 400, `{"message":"User is owner"}`)
	h.do(fiber.MethodPatch, album, `{"albumThumbnailAssetId":"00000000-0000-4000-8000-000000000000"}`, owner...).
		expect(t, 400, `{"message":"Invalid album thumbnail"}`)
	h.do(fiber.MethodPut, album+"/assets", `{"ids":["`+asset+`"]}`, member...).
		expect(t, 400, `{"message":"Not found or no albumAsset.create access"}`)
	h.do(fiber.MethodPut, album+"/user/"+memberID, `{"role":"editor"}`, owner...).expect(t, 204, "")
	h.do(fiber.MethodPut, album+"/assets", `{"ids":["`+asset+`"]}`, member...).
		expect(t, 200, `[{"id":"`+asset+`","success":false,"error":"duplicate"}]`)
	h.do(fiber.MethodGet, "/api/albums/statistics", nil, member...).expect(t, 200, `{"owned":0,"shared":1,"notShared":0}`)
	h.do(fiber.MethodDelete, album+"/user/me", nil, owner...).expect(t, 400, `{"message":"Cannot remove the last album owner"}`)
	h.do(fiber.MethodDelete, album+"/user/"+memberID, nil, owner...).expect(t, 204, "")
	h.do(fiber.MethodDelete, album+"/user/"+memberID, nil, owner...).expect(t, 400, `{"message":"Album not shared with user"}`)
	h.do(fiber.MethodGet, album, nil, member...).expect(t, 400, `{"message":"Not found or no album.read access"}`)
	h.do(fiber.MethodGet, "/api/albums/statistics", nil, owner...).expect(t, 200, `{"owned":1,"shared":0,"notShared":1}`)
	h.do(fiber.MethodDelete, album, nil, owner...).expect(t, 204, "")
	h.do(fiber.MethodGet, album, nil, owner...).expect(t, 400, `{"message":"Not found or no album.read access"}`)
}

func TestAlbumSyncBackfillsALateMember(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	memberID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := h.login("mira@example.com", "mira-password")
	first := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	second := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z")
	h.runAllJobs()
	later := "/api/albums/" + h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Later","assetIds":["`+second+`"]}`, bearer(admin)...).str(t, "id")
	h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Shared","assetIds":["`+first+`"],"albumUsers":[{"userId":"`+memberID+`","role":"editor"}]}`, bearer(admin)...).
		expect(t, 201, "")

	_, lines := h.stream(member, map[string]any{"types": appSyncTypes})
	h.ackRuns(member, lines)
	h.do(fiber.MethodPut, later+"/users", `{"albumUsers":[{"userId":"`+memberID+`"}]}`, bearer(admin)...).expect(t, 200, "")

	_, lines = h.stream(member, map[string]any{"types": appSyncTypes})
	want := []string{
		"AlbumAssetBackfillV2", "SyncAckV1", "AlbumV2", "AlbumUserBackfillV1", "SyncAckV1", "AlbumUserV1",
		"AlbumToAssetBackfillV1", "SyncAckV1", "AlbumAssetExifBackfillV1", "SyncAckV1", "SyncCompleteV1",
	}
	if got := types(lines); !slices.Equal(got, want) {
		t.Fatalf("types = %v\n want %v", got, want)
	}
	for _, l := range lines {
		if l.Type == "AlbumAssetBackfillV2" && !strings.Contains(string(l.Data), `"id":"`+second+`"`) {
			t.Errorf("backfill = %s", l.Data)
		}
	}

	h.ackRuns(member, lines)
	if _, lines = h.stream(member, map[string]any{"types": appSyncTypes}); !slices.Equal(types(lines), []string{"SyncCompleteV1"}) {
		t.Errorf("after acks = %v", types(lines))
	}
}
