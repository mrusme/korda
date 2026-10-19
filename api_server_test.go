package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

const (
	immichFeatures   = `{"smartSearch":false,"facialRecognition":false,"duplicateDetection":false,"map":false,"reverseGeocoding":false,"importFaces":false,"sidecar":true,"search":true,"trash":true,"oauth":false,"oauthAutoLaunch":false,"ocr":false,"passwordLogin":true,"configFile":true,"email":false,"realtimeTranscoding":false}`
	immichMediaTypes = `{"video":[".3gp",".3gpp",".avi",".flv",".insv",".m2t",".m2ts",".m4v",".mkv",".mov",".mp4",".mpe",".mpeg",".mpg",".mts",".mxf",".ts",".vob",".webm",".wmv"],"image":[".avif",".bmp",".gif",".jfif",".jpeg",".jpg",".png",".webp",".3fr",".ari",".arw",".cap",".cin",".cr2",".cr3",".crw",".dcr",".dng",".erf",".fff",".iiq",".k25",".kdc",".mrw",".nef",".nrw",".orf",".ori",".pef",".psd",".raf",".raw",".rw2",".rwl",".sr2",".srf",".srw",".x3f",".heic",".heif",".hif",".insp",".jp2",".jpe",".jxl",".mpo",".svg",".tif",".tiff"],"sidecar":[".xmp"]}`
	immichConfig     = `{"loginPageMessage":"","trashDays":30,"userDeleteDelay":7,"oauthButtonText":"Login with OAuth","oauthAccountManagementUrl":"","isInitialized":%s,"isOnboarded":false,"externalDomain":"","publicUsers":true,"mapDarkStyleUrl":"https://tiles.immich.cloud/v1/style/dark.json","mapLightStyleUrl":"https://tiles.immich.cloud/v1/style/light.json","maintenanceMode":false,"minFaces":3}`
	immichAPKLinks   = `{"arm64v8a":"https://github.com/immich-app/immich/releases/download/v3.3.3/app-arm64-v8a-release.apk","armeabiv7a":"https://github.com/immich-app/immich/releases/download/v3.3.3/app-armeabi-v7a-release.apk","universal":"https://github.com/immich-app/immich/releases/download/v3.3.3/app-release.apk","x86_64":"https://github.com/immich-app/immich/releases/download/v3.3.3/app-x86_64-release.apk"}`
)

func TestPublicServerEndpoints(t *testing.T) {
	h := newHarness(t)

	h.do(fiber.MethodGet, "/api/server/ping", nil).expect(t, 200, `{"res":"pong"}`)
	h.do(fiber.MethodGet, "/api/server/version", nil).expect(t, 200, `{"major":3,"minor":3,"patch":3,"prerelease":null}`)
	h.do(fiber.MethodGet, "/api/server/features", nil).expect(t, 200, immichFeatures)
	h.do(fiber.MethodGet, "/api/server/media-types", nil).expect(t, 200, immichMediaTypes)
	h.do(fiber.MethodGet, "/api/server/config", nil).expect(t, 200, strings.Replace(immichConfig, "%s", "false", 1))
	h.do(fiber.MethodGet, "/api/server/version-history", nil).
		expectShape(t, 200, `[{"id":"<uuid>","createdAt":"<time>","version":"dev"}]`)

	h.signUp()
	h.do(fiber.MethodGet, "/api/server/config", nil).expect(t, 200, strings.Replace(immichConfig, "%s", "true", 1))
}

func TestRoutingMatchesExpress(t *testing.T) {
	h := newHarness(t)

	h.do(fiber.MethodGet, "/api/server/ping/", nil).expect(t, 200, `{"res":"pong"}`)
	h.do(fiber.MethodGet, "/api/Server/Ping", nil).expect(t, 200, `{"res":"pong"}`)
	h.do(fiber.MethodGet, "/api/nope?x=1&y=<b>", nil).expect(t, 404, `{"message":"Cannot GET /api/nope?x=1&y=<b>"}`)
	h.do(fiber.MethodDelete, "/api/server/ping", nil).expect(t, 404, `{"message":"Cannot DELETE /api/server/ping"}`)
	h.do(fiber.MethodPost, "/api/server/nope", nil).expect(t, 404, `{"message":"Cannot POST /api/server/nope"}`)
	h.do(fiber.MethodGet, "/.well-known/immich", nil).expect(t, 200, `{"api":{"endpoint":"/api"}}`)

	res := h.do(fiber.MethodGet, "/api/server/ping", nil)
	if id := res.Header.Get("X-Correlation-ID"); len(id) != 8 {
		t.Errorf("X-Correlation-ID = %q, want eight characters", id)
	}
	if ct := res.Header.Get(fiber.HeaderContentType); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if got := h.do(fiber.MethodGet, "/api/server/ping", nil, "X-Correlation-ID", "mine-123").Header.Get("X-Correlation-ID"); got != "mine-123" {
		t.Errorf("X-Correlation-ID = %q, want the incoming mine-123", got)
	}
	if got := h.do(fiber.MethodGet, "/api/nope", nil).Header.Get("X-Correlation-ID"); len(got) != 8 {
		t.Errorf("a 404 has X-Correlation-ID %q", got)
	}
	if res = h.do(fiber.MethodHead, "/api/server/ping", nil); res.Status != 200 {
		t.Errorf("HEAD = %d", res.Status)
	}
}

func TestAuthenticatedServerEndpoints(t *testing.T) {
	h := newHarness(t)
	h.do(fiber.MethodGet, "/api/server/about", nil).expect(t, 401, `{"message":"Authentication required"}`)

	token := h.admin()

	h.do(fiber.MethodGet, "/api/server/about", nil, bearer(token)...).
		expect(t, 200, `{"version":"dev","versionUrl":"https://tty.fail/mrus/korda/releases","licensed":false,"repository":"mrus/korda","repositoryUrl":"https://tty.fail/mrus/korda","sourceRef":"dev"}`)
	h.do(fiber.MethodGet, "/api/server/apk-links", nil, bearer(token)...).expect(t, 200, immichAPKLinks)

	storage := h.do(fiber.MethodGet, "/api/server/storage", nil, bearer(token)...)
	body := storage.object(t)
	for _, key := range []string{"diskAvailable", "diskSize", "diskUse", "diskAvailableRaw", "diskSizeRaw", "diskUseRaw", "diskUsagePercentage"} {
		if _, ok := body[key]; !ok {
			t.Errorf("storage has no %s: %s", key, storage.Body)
		}
	}
	if !strings.HasPrefix(string(storage.Body), `{"diskAvailable":"`) {
		t.Errorf("storage = %s", storage.Body)
	}

	h.do(fiber.MethodGet, "/api/server/statistics", nil, bearer(token)...).
		expectShape(t, 200, `{"photos":0,"videos":0,"usage":0,"usagePhotos":0,"usageVideos":0,"usageByUser":[{"userId":"<uuid>","userName":"Admin","photos":0,"videos":0,"usage":0,"usagePhotos":0,"usageVideos":0,"quotaSizeInBytes":null}]}`)

	h.do(fiber.MethodGet, "/api/server/license", nil, bearer(token)...).expect(t, 404, `{"message":"Not Found"}`)
	h.do(fiber.MethodPut, "/api/server/license", `{"licenseKey":"IMSV-AAAA-BBBB-CCCC-DDDD-EEEE-FFFF-GGGG-HHHH","activationKey":"x"}`, bearer(token)...).
		expect(t, 400, `{"message":"Invalid license key"}`)
	h.do(fiber.MethodDelete, "/api/server/license", nil, bearer(token)...).expect(t, 204, "")

	h.createUser(token, "vera@example.com", "Vera", "vera-password")
	vera := h.login("vera@example.com", "vera-password")
	h.do(fiber.MethodGet, "/api/server/statistics", nil, bearer(vera)...).expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodGet, "/api/server/license", nil, bearer(vera)...).expect(t, 403, `{"message":"Forbidden"}`)

	stats := h.do(fiber.MethodGet, "/api/server/statistics", nil, bearer(token)...).object(t)
	users := stats["usageByUser"].([]any)
	if len(users) != 2 || users[0].(map[string]any)["userName"] != "Admin" || users[1].(map[string]any)["userName"] != "Vera" {
		t.Errorf("usageByUser = %v, want Admin then Vera", users)
	}
}

func TestVersionHistoryRecordsKordasVersion(t *testing.T) {
	h := newHarness(t)

	if err := recordVersion(h); err != nil {
		t.Fatal(err)
	}
	if err := recordVersion(h); err != nil {
		t.Fatal(err)
	}

	h.do(fiber.MethodGet, "/api/server/version-history", nil).
		expectShape(t, http.StatusOK, `[{"id":"<uuid>","createdAt":"<time>","version":"dev"}]`)
}
