package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestConfigRoutes(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))

	cfg := h.do(fiber.MethodGet, "/api/admin/config", nil, owner...)
	legacy := h.do(fiber.MethodGet, "/api/system-config", nil, owner...)
	if cfg.Status != 200 || string(cfg.Body) != string(legacy.Body) {
		t.Fatalf("admin config = %d %s, legacy %s", cfg.Status, cfg.Body, legacy.Body)
	}
	for _, want := range []string{
		`{"backup":{"database":{"enabled":true,"cronExpression":"0 02 * * *","keepLastAmount":14}},"ffmpeg":{"crf":23,`,
		`"transcode":"required","accel":"disabled","accelDecode":true,"tonemap":"hable","realtime":{"enabled":false`,
		`"job":{"thumbnailGeneration":{"concurrency":1},"metadataExtraction":{"concurrency":1},"videoConversion":{"concurrency":1},"faceDetection":{"concurrency":2}`,
		`"machineLearning":{"enabled":false,"urls":["http://immich-machine-learning:3003"]`,
		`"image":{"thumbnail":{"format":"jpeg","quality":80,"size":250,"progressive":false},"preview":{"format":"jpeg","quality":80,"size":1440,"progressive":false},"fullsize":{"enabled":false,"format":"jpeg","quality":80,"progressive":false},"colorspace":"srgb","extractEmbedded":false}`,
		`"defaultStorageQuota":null,`,
		`"library":{"scan":{"enabled":false,"cronExpression":"0 0 * * *"},"watch":{"enabled":false}},"notifications":{`,
		`"server":{"externalDomain":"","loginPageMessage":"","publicUsers":true},"user":{"deleteDelay":7}}`,
	} {
		if !strings.Contains(string(cfg.Body), want) {
			t.Errorf("admin config lacks %s", want)
		}
	}

	defaults := h.do(fiber.MethodGet, "/api/admin/config/defaults", nil, owner...)
	if !strings.Contains(string(defaults.Body), `"image":{"thumbnail":{"format":"jpeg","size":250,"quality":80,"progressive":false},"preview":{"format":"jpeg","size":1440,"quality":80,"progressive":false},"colorspace":"srgb","extractEmbedded":false,"fullsize":`) ||
		!strings.Contains(string(defaults.Body), `"library":{"scan":{"enabled":false,"cronExpression":"0 0 * * *"},"watch":{"enabled":false}},"server":{`) {
		t.Errorf("defaults order = %s", defaults.Body)
	}
	var a, d map[string]any
	if json.Unmarshal(cfg.Body, &a) != nil || json.Unmarshal(defaults.Body, &d) != nil {
		t.Fatal("config bodies are not JSON")
	}

	h.do(fiber.MethodGet, "/api/config", nil, member...).
		expect(t, 200, `{"ffmpeg":{"realtime":{"enabled":false,"videoCodecs":["h264","hevc"],"resolutions":[480,720,1080]}},"machineLearning":{"enabled":false,"clip":{"enabled":false},"duplicateDetection":{"enabled":false},"facialRecognition":{"enabled":false,"minFaces":3},"ocr":{"enabled":false}},"map":{"enabled":false,"lightStyle":"https://tiles.immich.cloud/v1/style/light.json","darkStyle":"https://tiles.immich.cloud/v1/style/dark.json"},"reverseGeocoding":{"enabled":false},"oauth":{"autoLaunch":false,"buttonText":"Login with OAuth","enabled":false},"passwordLogin":{"enabled":true},"image":{"thumbnail":{"size":250},"preview":{"size":1440},"fullsize":{"enabled":false}},"trash":{"enabled":true,"days":30},"theme":{"customCss":""},"server":{"externalDomain":"","loginPageMessage":"","publicUsers":true},"user":{"deleteDelay":7}}`)
	h.do(fiber.MethodGet, "/api/public/config", nil).
		expect(t, 200, `{"oauth":{"autoLaunch":false,"buttonText":"Login with OAuth","enabled":false},"passwordLogin":{"enabled":true},"theme":{"customCss":""},"server":{"loginPageMessage":""}}`)
	h.do(fiber.MethodGet, "/api/public/config/defaults", nil, "Authorization", "Bearer nope").
		expect(t, 200, `{"oauth":{"autoLaunch":false,"buttonText":"Login with OAuth","enabled":false},"passwordLogin":{"enabled":true},"theme":{"customCss":""},"server":{"loginPageMessage":""}}`)
	h.do(fiber.MethodGet, "/api/admin/config", nil, member...).expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodGet, "/api/config", nil).expect(t, 401, `{"message":"Authentication required"}`)

	h.do(fiber.MethodPut, "/api/admin/config", cfg.Body, owner...).
		expect(t, 400, `{"message":"Cannot update configuration while IMMICH_CONFIG_FILE is in use"}`)
	h.do(fiber.MethodPut, "/api/system-config", strings.Replace(string(cfg.Body), `"days":30`, `"days":-1`, 1), owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"number","code":"too_small","minimum":0,"inclusive":true,"path":["trash","days"],"message":"Too small: expected number to be >=0"}]}`)
	if res := h.do(fiber.MethodGet, "/api/system-config/storage-template-options", nil, owner...); res.Status != 200 || !strings.HasPrefix(string(res.Body), `{"secondOptions":["s","ss","SSS"]`) || !strings.HasSuffix(string(res.Body), `"]}`) {
		t.Errorf("storage template options = %d %s", res.Status, res.Body)
	}

	h.do(fiber.MethodGet, "/api/system-metadata/admin-onboarding", nil, owner...).expect(t, 200, `{"isOnboarded":false}`)
	h.do(fiber.MethodPost, "/api/system-metadata/admin-onboarding", `{"isOnboarded":"yes"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["isOnboarded"],"message":"Invalid input: expected boolean, received string"}]}`)
	h.do(fiber.MethodPost, "/api/system-metadata/admin-onboarding", `{"isOnboarded":true}`, owner...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/system-metadata/admin-onboarding", nil, owner...).expect(t, 200, `{"isOnboarded":true}`)
	h.do(fiber.MethodGet, "/api/system-metadata/reverse-geocoding-state", nil, owner...).expect(t, 200, `{"lastUpdate":null,"lastImportFileName":null}`)
	h.do(fiber.MethodGet, "/api/system-metadata/version-check-state", nil, owner...).expect(t, 200, `{"checkedAt":null,"releaseVersion":null}`)
}
