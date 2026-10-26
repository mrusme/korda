package main

import (
	"archive/zip"
	"bytes"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

func (h *apiHarness) form(path string, values url.Values, headers ...string) apiResponse {
	h.t.Helper()

	req := httptest.NewRequest(fiber.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	return h.send(req)
}

func expectPage(t *testing.T, res apiResponse, status int, contains ...string) {
	t.Helper()

	if res.Status != status || res.Header.Get(fiber.HeaderContentType) != fiber.MIMETextHTMLCharsetUTF8 || res.Header.Get(fiber.HeaderCacheControl) != "no-store" {
		t.Errorf("page = %d %v\n%s", res.Status, res.Header, res.Body)
		return
	}
	for _, want := range contains {
		if !strings.Contains(string(res.Body), want) {
			t.Errorf("page has no %s\n%s", want, res.Body)
		}
	}
}

func TestSharePage(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2020-02-03T04:05:06.000Z", "2020-02-04T04:05:06.000Z")
	h.runAllJobs()
	albumID := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"<b>Trip</b>","description":"Two days","assetIds":["`+exif+`","`+plain+`"]}`, owner...).str(t, "id")
	alb := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"ALBUM","albumId":"`+albumID+`","slug":"family trip"}`, owner...)
	ind := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+plain+`"],"password":"secret","allowDownload":false}`, owner...)
	expired := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+plain+`"],"expiresAt":"2020-01-01T00:00:00Z"}`, owner...)
	albKey, indKey := alb.str(t, "key"), ind.str(t, "key")

	expectPage(t, h.do(fiber.MethodGet, "/share/unknown", nil), 404, "<title>Link not found</title>")
	expectPage(t, h.do(fiber.MethodGet, "/share/"+expired.str(t, "key"), nil), 404, "Link not found")
	expectPage(t, h.do(fiber.MethodGet, "/s/nothing", nil), 404, "Link not found")

	escaped := "&lt;b&gt;Trip&lt;/b&gt;"
	for path, query := range map[string]string{"/share/" + albKey: "key=" + albKey, "/s/family%20trip": "slug=family%20trip"} {
		res := h.do(fiber.MethodGet, path, nil)
		expectPage(t, res, 200,
			"<title>"+escaped+"</title>",
			"<h1>"+escaped+"</h1>",
			`<p class="description">Two days</p>`,
			`<p class="count">2 photos</p>`,
			`<meta property="og:title" content="`+escaped+`">`,
			`<meta property="og:description" content="2 shared photos &amp; videos">`,
			`<meta property="og:image" content="http://example.com/api/assets/`,
			`/thumbnail?key=`+albKey+`">`,
			`<form method="post" action="/api/download/archive?`+query+`">`,
			`<input type="hidden" name="archiveName" value="`+escaped+`">`,
			`<a href="/api/assets/`+exif+`/thumbnail?size=preview&amp;`+query+`"><img src="/api/assets/`+exif+`/thumbnail?`+query+`" alt="" loading="lazy"></a>`,
		)
		if strings.Contains(string(res.Body), "<b>") {
			t.Errorf("%s isn't escaped:\n%s", path, res.Body)
		}
		ids := regexp.MustCompile(`name="assetIds" value="([^"]*)"`).FindStringSubmatch(string(res.Body))
		if got := strings.Split(ids[len(ids)-1], ","); len(ids) != 2 || len(got) != 2 || !slices.Contains(got, exif) || !slices.Contains(got, plain) {
			t.Errorf("%s download ids = %v", path, ids)
		}
	}

	locked := h.do(fiber.MethodGet, "/share/"+indKey, nil)
	expectPage(t, locked, 401, "<title>Password required</title>", `<input type="password" id="password" name="password"`)
	if strings.Contains(string(locked.Body), plain) || strings.Contains(string(locked.Body), "og:") {
		t.Errorf("password page shows the link:\n%s", locked.Body)
	}
	expectPage(t, h.form("/share/"+indKey, url.Values{"password": {"wrong"}}), 401, `<p class="error">Invalid password</p>`)

	login := h.form("/share/"+indKey, url.Values{"password": {"secret"}})
	cookie, _, _ := strings.Cut(login.Header.Get(fiber.HeaderSetCookie), ";")
	if login.Status != fiber.StatusSeeOther || login.Header.Get(fiber.HeaderLocation) != "/share/"+indKey ||
		cookie != "immich_shared_link_token="+url.QueryEscape(linkToken(ind.str(t, "id"), "secret")) {
		t.Fatalf("login = %d %v", login.Status, login.Header)
	}
	opened := h.do(fiber.MethodGet, "/share/"+indKey, nil, "Cookie", cookie)
	expectPage(t, opened, 200, "<title>Shared photos</title>", `<p class="count">1 photo</p>`, `/api/assets/`+plain+`/thumbnail?key=`+indKey)
	if body := string(opened.Body); strings.Contains(body, "<form") || strings.Contains(body, "og:") {
		t.Errorf("link without download or with a password shows a form or tags:\n%s", body)
	}
	if res := h.do(fiber.MethodGet, "/api/assets/"+plain+"/thumbnail?key="+indKey, nil, "Cookie", cookie); res.Status != 200 {
		t.Errorf("thumbnail with the cookie = %d %s", res.Status, res.Body)
	}

	archive := h.form("/api/download/archive?slug=family+trip", url.Values{"assetIds": {exif + "," + plain}, "archiveName": {"Trip"}, "edited": {"true"}})
	if archive.Status != 200 || archive.Header.Get(fiber.HeaderContentDisposition) != "attachment; filename*=UTF-8''Trip.zip" {
		t.Fatalf("archive = %d %v", archive.Status, archive.Header)
	}
	if zr, err := zip.NewReader(bytes.NewReader(archive.Body), int64(len(archive.Body))); err != nil || len(zr.File) != 2 {
		t.Errorf("archive entries = %v %v", zr, err)
	}

	css := h.do(fiber.MethodGet, "/static/share/share.css", nil)
	if css.Status != 200 || !strings.HasPrefix(css.Header.Get(fiber.HeaderContentType), "text/css") || !bytes.Contains(css.Body, []byte(".grid")) {
		t.Errorf("stylesheet = %d %v", css.Status, css.Header)
	}
}

func TestSharePageLoginRateLimit(t *testing.T) {
	h := newHarnessWith(t, harnessOptions{rateLimitAuth: 2, publicUsers: true})
	admin := h.admin()
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	key := h.do(fiber.MethodPost, "/api/shared-links", `{"type":"INDIVIDUAL","assetIds":["`+exif+`"],"password":"secret"}`, bearer(admin)...).str(t, "key")

	expectPage(t, h.form("/share/"+key, url.Values{"password": {"guess"}}), 401, "Invalid password")
	h.do(fiber.MethodPost, "/api/shared-links/login?key="+key, `{"password":"guess"}`).expect(t, 401, `{"message":"Invalid password"}`)
	h.form("/share/"+key, url.Values{"password": {"secret"}}).expect(t, 429, `{"message":"Too Many Requests"}`)
	h.do(fiber.MethodPost, "/api/shared-links/login?key="+key, `{"password":"secret"}`).expect(t, 429, `{"message":"Too Many Requests"}`)
	expectPage(t, h.do(fiber.MethodGet, "/share/"+key, nil), 401, "Password required")
}
