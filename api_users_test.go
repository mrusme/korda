package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	defaultPreferences = `{"albums":{"defaultAssetOrder":"desc"},"folders":{"enabled":false,"sidebarWeb":false},"memories":{"enabled":true,"duration":5,"sidebarWeb":false},"people":{"enabled":true,"sidebarWeb":false,"minimumFaces":3,"updateStrategy":"everyone"},"sharedLinks":{"enabled":true,"sidebarWeb":false},"ratings":{"enabled":false},"tags":{"enabled":false,"sidebarWeb":false},"emailNotifications":{"enabled":true,"albumInvite":true,"albumUpdate":true},"download":{"archiveSize":4294967296,"includeEmbeddedVideos":false},"purchase":{"showSupportBadge":true,"hideBuyButtonUntil":"2022-02-12T00:00:00.000Z"},"cast":{"gCastEnabled":false},"recentlyAdded":{"sidebarWeb":false}}`
	updatedPreferences = `{"albums":{"defaultAssetOrder":"desc"},"folders":{"enabled":false,"sidebarWeb":false},"memories":{"enabled":true,"duration":7,"sidebarWeb":false},"people":{"enabled":true,"sidebarWeb":false,"minimumFaces":3,"updateStrategy":"everyone"},"sharedLinks":{"enabled":true,"sidebarWeb":false},"ratings":{"enabled":false},"tags":{"enabled":false,"sidebarWeb":false},"emailNotifications":{"enabled":true,"albumInvite":true,"albumUpdate":true},"download":{"archiveSize":1,"includeEmbeddedVideos":false},"purchase":{"showSupportBadge":true,"hideBuyButtonUntil":"2022-02-12T00:00:00.000Z"},"cast":{"gCastEnabled":false},"recentlyAdded":{"sidebarWeb":false}`
	veraShape          = `{"id":"<uuid>","email":"vera@example.com","name":"%s","profileImagePath":"%s","avatarColor":"%s","profileChangedAt":"<time>","clusterGroupId":"<uuid>","storageLabel":null,"shouldChangePassword":true,"isAdmin":false,"createdAt":"<time>","deletedAt":null,"updatedAt":"<time>","oauthId":"","quotaSizeInBytes":null,"quotaUsageInBytes":0,"status":"active","license":null}`
)

func shape(template string, values ...string) string {
	for _, v := range values {
		template = strings.Replace(template, "%s", v, 1)
	}

	return template
}

func pngImage(t *testing.T, width int, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := range width {
		for y := range height {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 200, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestMyUser(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	h.createUser(token, "vera@example.com", "Vera", "vera-password")
	vera := bearer(h.login("vera@example.com", "vera-password"))

	h.do(fiber.MethodGet, "/api/users/me", nil, vera...).expectShape(t, 200, shape(veraShape, "Vera", "", "orange"))
	h.do(fiber.MethodPut, "/api/users/me", `{"name":"Vera Two","avatarColor":"green"}`, vera...).
		expectShape(t, 200, shape(veraShape, "Vera Two", "", "green"))
	h.do(fiber.MethodPatch, "/api/users/me", `{"avatarColor":null}`, vera...).
		expectShape(t, 200, shape(veraShape, "Vera Two", "", "orange"))
	h.do(fiber.MethodPut, "/api/users/me", `{"email":"admin@example.com"}`, vera...).
		expect(t, 400, `{"message":"Email is not available"}`)
	h.do(fiber.MethodPut, "/api/users/me", `{"avatarColor":"magenta"}`, vera...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["primary","pink","red","yellow","blue","green","purple","orange","gray","amber"],"path":["avatarColor"],"message":"Invalid option: expected one of \"primary\"|\"pink\"|\"red\"|\"yellow\"|\"blue\"|\"green\"|\"purple\"|\"orange\"|\"gray\"|\"amber\""}]}`)
	h.do(fiber.MethodPut, "/api/users/me", `{"email":"Vera@Example.com"}`, vera...).
		expectShape(t, 200, shape(veraShape, "Vera Two", "", "orange"))
}

func TestPreferences(t *testing.T) {
	h := newHarness(t)
	token := bearer(h.admin())

	h.do(fiber.MethodGet, "/api/users/me/preferences", nil, token...).expect(t, 200, defaultPreferences)
	h.do(fiber.MethodPut, "/api/users/me/preferences", `{"memories":{"duration":7},"avatar":{"color":"red"},"download":{"archiveSize":1}}`, token...).
		expect(t, 200, updatedPreferences+`,"avatar":{"color":"red"}}`)
	h.do(fiber.MethodGet, "/api/users/me/preferences", nil, token...).expect(t, 200, updatedPreferences+`}`)
	h.do(fiber.MethodPatch, "/api/users/me/preferences", `{"memories":{"duration":5},"download":{"archiveSize":4294967296}}`, token...).
		expect(t, 200, defaultPreferences)
	h.do(fiber.MethodPut, "/api/users/me/preferences", `{"memories":{"duration":0,"enabled":"yes"},"albums":{"defaultAssetOrder":"up"},"extra":1}`, token...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["asc","desc"],"path":["albums","defaultAssetOrder"],"message":"Invalid option: expected one of \"asc\"|\"desc\""},{"expected":"boolean","code":"invalid_type","path":["memories","enabled"],"message":"Invalid input: expected boolean, received string"},{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["memories","duration"],"message":"Too small: expected number to be >=1"}]}`)
}

func TestOnboardingAndLicense(t *testing.T) {
	h := newHarness(t)
	token := bearer(h.admin())

	h.do(fiber.MethodGet, "/api/users/me/onboarding", nil, token...).expect(t, 200, `{"isOnboarded":false}`)
	h.do(fiber.MethodPut, "/api/users/me/onboarding", `{"isOnboarded":true}`, token...).expect(t, 200, `{"isOnboarded":true}`)
	h.do(fiber.MethodGet, "/api/users/me/onboarding", nil, token...).expect(t, 200, `{"isOnboarded":true}`)

	login := h.do(fiber.MethodPost, "/api/auth/login", map[string]string{"email": adminEmail, "password": adminPassword})
	if !strings.HasSuffix(string(login.Body), `"isOnboarded":true}`) {
		t.Errorf("login = %s, want isOnboarded true", login.Body)
	}

	h.do(fiber.MethodDelete, "/api/users/me/onboarding", nil, token...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/users/me/onboarding", nil, token...).expect(t, 200, `{"isOnboarded":false}`)

	h.do(fiber.MethodGet, "/api/users/me/license", nil, token...).expect(t, 404, `{"message":"Not Found"}`)
	h.do(fiber.MethodPut, "/api/users/me/license", `{"licenseKey":"IMCL-AAAA-BBBB-CCCC-DDDD-EEEE-FFFF-GGGG-HHHH","activationKey":"x"}`, token...).
		expect(t, 400, `{"message":"Invalid license key"}`)
	h.do(fiber.MethodPut, "/api/users/me/license", `{"licenseKey":"nope","activationKey":"x"}`, token...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^IM(SV|CL)(-[\\dA-Za-z]{4}){8}$/","path":["licenseKey"],"message":"Invalid string: must match pattern /^IM(SV|CL)(-[\\dA-Za-z]{4}){8}$/"}]}`)
	h.do(fiber.MethodDelete, "/api/users/me/license", nil, token...).expect(t, 204, "")
}

func TestProfileImage(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	auth := bearer(token)
	me := h.do(fiber.MethodGet, "/api/users/me", nil, auth...).str(t, "id")

	h.multipart("/api/users/profile-image", []part{{field: "file", filename: "x.gif", content: []byte("GIF89a")}}, auth...).
		expect(t, 400, `{"message":"Unsupported file type x.gif"}`)
	h.multipart("/api/users/profile-image", []part{{field: "image", filename: "a.png", content: pngImage(t, 10, 10)}}, auth...).
		expect(t, 400, `{"message":"Unexpected file field - image"}`)
	h.multipart("/api/users/profile-image", []part{{field: "file", filename: "a.png"}}, auth...).
		expect(t, 400, `{"message":"File is empty"}`)
	h.multipart("/api/users/profile-image", []part{{field: "file", filename: "a.jpg", content: []byte("not an image")}}, auth...).
		expect(t, 400, `{"message":"Unable to process profile image"}`)
	h.multipart("/api/users/profile-image", []part{{field: "other", content: []byte("1")}}, auth...).
		expect(t, 500, `{"message":"Failed to create profile image"}`)
	h.do(fiber.MethodPost, "/api/users/profile-image", `{}`, auth...).
		expect(t, 500, `{"message":"Failed to create profile image"}`)

	res := h.multipart("/api/users/profile-image", []part{{field: "other", content: []byte("1")}, {field: "file", filename: "a.PNG", content: pngImage(t, 400, 300)}}, auth...)
	res.expectShape(t, 201, `{"userId":"<uuid>","profileImagePath":"profile/<uuid>/<uuid>.jpg","profileChangedAt":"<time>"}`)
	first := res.str(t, "profileImagePath")

	image := h.do(fiber.MethodGet, "/api/users/"+me+"/profile-image", nil, auth...)
	if image.Status != 200 || image.Header.Get(fiber.HeaderContentType) != "image/jpeg" ||
		image.Header.Get(fiber.HeaderCacheControl) != "public, max-age=0" || !bytes.HasPrefix(image.Body, []byte{0xff, 0xd8, 0xff}) {
		t.Errorf("profile image = %d %v", image.Status, image.Header)
	}

	second := h.multipart("/api/users/profile-image", []part{{field: "file", filename: "b.png", content: pngImage(t, 300, 400)}}, auth...).str(t, "profileImagePath")
	if second == first || h.do(fiber.MethodGet, "/api/users/me", nil, auth...).str(t, "profileImagePath") != second {
		t.Errorf("the second upload left %s", first)
	}
	if fileExists(h, first) {
		t.Errorf("the replaced profile image %s is still on disk", first)
	}

	h.do(fiber.MethodDelete, "/api/users/profile-image", nil, auth...).expect(t, 204, "")
	h.do(fiber.MethodDelete, "/api/users/profile-image", nil, auth...).expect(t, 400, `{"message":"Can't delete a missing profile Image"}`)
	h.do(fiber.MethodGet, "/api/users/"+me+"/profile-image", nil, auth...).expect(t, 404, `{"message":"Not Found"}`)
	if fileExists(h, second) {
		t.Errorf("the deleted profile image %s is still on disk", second)
	}
}

func TestSearchAndGetUsers(t *testing.T) {
	h := newHarness(t)
	token := h.admin()
	vera := h.createUser(token, "vera@example.com", "Vera", "vera-password")
	veraToken := bearer(h.login("vera@example.com", "vera-password"))

	h.do(fiber.MethodGet, "/api/users", nil, veraToken...).
		expectShape(t, 200, `[{"id":"<uuid>","email":"vera@example.com","name":"Vera","profileImagePath":"","avatarColor":"orange","profileChangedAt":"<time>"},{"id":"<uuid>","email":"admin@example.com","name":"Admin","profileImagePath":"","avatarColor":"gray","profileChangedAt":"<time>"}]`)
	h.do(fiber.MethodGet, "/api/users/"+vera, nil, bearer(token)...).
		expectShape(t, 200, `{"id":"<uuid>","email":"vera@example.com","name":"Vera","profileImagePath":"","avatarColor":"orange","profileChangedAt":"<time>"}`)
	h.do(fiber.MethodGet, "/api/users/00000000-0000-4000-8000-000000000000", nil, bearer(token)...).
		expect(t, 400, `{"message":"User not found"}`)
	h.do(fiber.MethodGet, "/api/users/not-a-uuid", nil, bearer(token)...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["id"],"message":"Invalid UUID"}]}`)

	private := newHarnessWith(t, harnessOptions{publicUsers: false})
	privateAdmin := private.admin()
	private.createUser(privateAdmin, "vera@example.com", "Vera", "vera-password")
	privateVera := bearer(private.login("vera@example.com", "vera-password"))
	if users := private.do(fiber.MethodGet, "/api/users", nil, privateVera...).array(t); len(users) != 1 {
		t.Errorf("without public users vera sees %d users", len(users))
	}
	if users := private.do(fiber.MethodGet, "/api/users", nil, bearer(privateAdmin)...).array(t); len(users) != 2 {
		t.Errorf("without public users the admin sees %d users", len(users))
	}
}

func TestCalendarHeatmap(t *testing.T) {
	h := newHarness(t)
	token := bearer(h.admin())

	h.do(fiber.MethodGet, "/api/users/me/calendar-heatmap?from=2026-10-01&to=2026-10-03", nil, token...).
		expect(t, 200, `{"from":"2026-10-01","to":"2026-10-03","series":[{"date":"2026-10-01","count":0},{"date":"2026-10-02","count":0},{"date":"2026-10-03","count":0}],"totalCount":0}`)
	h.do(fiber.MethodGet, "/api/users/me/calendar-heatmap?from=2026-02-01&to=2026-01-01", nil, token...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"custom","path":["from"],"message":"from must be before to"}]}`)
	h.do(fiber.MethodGet, "/api/users/me/calendar-heatmap?from=2026-02-30", nil, token...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"date","pattern":"/^(?:(?:\\d\\d[2468][048]|\\d\\d[13579][26]|\\d\\d0[48]|[02468][048]00|[13579][26]00)-02-29|\\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\\d|30)|(?:02)-(?:0[1-9]|1\\d|2[0-8])))$/","path":["from"],"message":"Invalid input: expected ISO date string (YYYY-MM-DD), received string"}]}`)
	h.do(fiber.MethodGet, "/api/users/me/calendar-heatmap?type=Nope", nil, token...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["Upload","Taken"],"path":["type"],"message":"Invalid option: expected one of \"Upload\"|\"Taken\""}]}`)

	body := h.do(fiber.MethodGet, "/api/users/me/calendar-heatmap", nil, token...).object(t)
	today := time.Now().UTC().Format("2006-01-02")
	if body["to"] != today || len(body["series"].([]any)) != 364 {
		t.Errorf("the default heatmap runs from %v to %v over %d days", body["from"], body["to"], len(body["series"].([]any)))
	}
}
