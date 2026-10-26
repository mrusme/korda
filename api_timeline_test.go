package main

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

var bucketTimePattern = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,6})?"`)

func bucketShape(body []byte, keep ...string) string {
	return normalize(bucketTimePattern.ReplaceAllStringFunc(string(body), func(m string) string {
		if slices.Contains(keep, m) {
			return m
		}
		return `"<pgtime>"`
	}))
}

func TestTimeline(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	png := h.uploadID(admin, "korda.png", fixture(t, testmedia.PlainPNG), "2023-01-03T03:04:05.000Z", "2023-01-03T03:04:05.000Z")
	h.runAllJobs()

	h.do(fiber.MethodGet, "/api/timeline/buckets", nil, owner...).
		expect(t, 200, `[{"timeBucket":"2024-05-01","count":1},{"timeBucket":"2023-01-01","count":2}]`)
	h.do(fiber.MethodGet, "/api/timeline/buckets?order=asc", nil, owner...).
		expect(t, 200, `[{"timeBucket":"2023-01-01","count":2},{"timeBucket":"2024-05-01","count":1}]`)

	may := h.do(fiber.MethodGet, "/api/timeline/bucket?timeBucket=2024-05-01T00:00:00.000Z&withCoordinates=true", nil, owner...)
	if may.Status != 200 || may.Header.Get(fiber.HeaderContentType) != "application/json; charset=utf-8" {
		t.Fatalf("bucket = %d %v %s", may.Status, may.Header, may.Body)
	}
	if got, want := bucketShape(may.Body, `"2024-05-06T05:08:09.123"`), `{"duration":[null],"id":["<uuid>"],"visibility":["timeline"],"isFavorite":[false],"isImage":[true],"isTrashed":[false],"livePhotoVideoId":[null],"fileCreatedAt":["2024-05-06T05:08:09.123"],"localOffsetHours":[2],"createdAt":["<pgtime>"],"ownerId":["<uuid>"],"projectionType":[null],"ratio":[0.750],"status":["active"],"thumbhash":["`+thumbhashOf(t, h, admin, exif)+`"],"city":[null],"country":[null],"latitude":[52.516272],"longitude":[13.377722]}`; got != want {
		t.Errorf("May bucket = %s\n          want %s", got, want)
	}
	for _, form := range []string{"2024-05-01", "2024-05-01T02:00:00%2B02:00", "%2B2024-05-01T00:00:00.000Z", "-2024-05-01T00:00:00.000Z", "+2024-05-01"} {
		if res := h.do(fiber.MethodGet, "/api/timeline/bucket?timeBucket="+form, nil, owner...); res.Status != 200 || !strings.Contains(string(res.Body), `"id":["`+exif+`"]`) {
			t.Errorf("bucket %s = %d %s", form, res.Status, res.Body)
		}
	}
	empty := `{"duration":[],"id":[],"visibility":[],"isFavorite":[],"isImage":[],"isTrashed":[],"livePhotoVideoId":[],"fileCreatedAt":[],"localOffsetHours":[],"createdAt":[],"ownerId":[],"projectionType":[],"ratio":[],"status":[],"thumbhash":[],"city":[],"country":[],"latitude":[],"longitude":[],"stack":[]}`
	h.do(fiber.MethodGet, "/api/timeline/bucket?timeBucket=2024-05-15T00:00:00.000Z&withCoordinates=true&withStacked=true", nil, owner...).expect(t, 200, empty)
	h.do(fiber.MethodGet, "/api/timeline/bucket?timeBucket=012345-01-01&withCoordinates=true&withStacked=true", nil, owner...).expect(t, 200, empty)
	h.do(fiber.MethodGet, "/api/timeline/bucket?timeBucket=abc", nil, owner...).expect(t, 500, `{"message":"Failed to get time bucket"}`)
	h.do(fiber.MethodGet, "/api/timeline/bucket", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["timeBucket"],"message":"Invalid input: expected string, received undefined"}]}`)
	h.do(fiber.MethodGet, "/api/timeline/bucket?isFavorite=x&userId=x", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/","path":["userId"],"message":"Invalid UUID"},{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["isFavorite"],"message":"Invalid option: expected one of \"true\"|\"false\""},{"expected":"string","code":"invalid_type","path":["timeBucket"],"message":"Invalid input: expected string, received undefined"}]}`)

	for query, want := range map[string]string{
		"1,2,3":          `[{"code":"custom","message":"bbox must have 4 comma-separated numbers: west,south,east,north","path":["bbox"]}]`,
		"1,2,3,x":        `[{"code":"custom","message":"bbox parts must be valid numbers","path":["bbox"]}]`,
		"200,0,10,10":    `[{"origin":"number","code":"too_big","maximum":180,"inclusive":true,"path":["bbox","west"],"message":"Too big: expected number to be <=180"}]`,
		"0,10,10,0":      `[{"code":"custom","path":["bbox","north"],"message":"North latitude must be greater than or equal to south latitude"}]`,
		"0,95,10,96":     `[{"origin":"number","code":"too_big","maximum":90,"inclusive":true,"path":["bbox","south"],"message":"Too big: expected number to be <=90"},{"origin":"number","code":"too_big","maximum":90,"inclusive":true,"path":["bbox","north"],"message":"Too big: expected number to be <=90"}]`,
		"Infinity,0,0,0": `[{"expected":"number","code":"invalid_type","received":"Infinity","path":["bbox","west"],"message":"Invalid input: expected number, received number"}]`,
	} {
		h.do(fiber.MethodGet, "/api/timeline/buckets?bbox="+query, nil, owner...).
			expect(t, 400, `{"message":"Validation failed","errors":`+want+`}`)
	}
	h.do(fiber.MethodGet, "/api/timeline/buckets?bbox=13,52,14,53", nil, owner...).expect(t, 200, `[{"timeBucket":"2024-05-01","count":1}]`)
	h.do(fiber.MethodGet, "/api/timeline/buckets?bbox=170,-10,-170,10", nil, owner...).expect(t, 200, `[]`)

	h.do(fiber.MethodGet, "/api/timeline/buckets?visibility=locked", nil, owner...).
		expect(t, 401, `{"message":"Elevated permission is required"}`)
	h.do(fiber.MethodGet, "/api/timeline/buckets?withPartners=true", nil, owner...).
		expect(t, 400, `{"message":"withPartners is only supported for non-archived, non-trashed, non-favorited, non-locked assets"}`)
	h.do(fiber.MethodGet, "/api/timeline/buckets?personId=0f0e0d0c-0b0a-4908-8706-050403020100", nil, owner...).
		expect(t, 400, `{"message":"Not found or no person.read access"}`)

	h.do(fiber.MethodPut, "/api/assets", `{"ids":["`+png+`"],"visibility":"archive"}`, owner...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/timeline/buckets?visibility=timeline", nil, owner...).
		expect(t, 200, `[{"timeBucket":"2024-05-01","count":1},{"timeBucket":"2023-01-01","count":1}]`)
	h.do(fiber.MethodGet, "/api/timeline/buckets", nil, owner...).
		expect(t, 200, `[{"timeBucket":"2024-05-01","count":1},{"timeBucket":"2023-01-01","count":2}]`)

	adminID := h.do(fiber.MethodGet, "/api/users/me", nil, owner...).str(t, "id")
	partnerID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	partner := bearer(h.login("mira@example.com", "mira-password"))
	h.do(fiber.MethodGet, "/api/timeline/buckets?userId="+adminID, nil, partner...).
		expect(t, 400, `{"message":"Not found or no timeline.read access"}`)
	h.do(fiber.MethodPost, "/api/partners", `{"sharedWithId":"`+partnerID+`"}`, owner...)
	h.do(fiber.MethodGet, "/api/timeline/buckets?userId="+adminID, nil, partner...).
		expect(t, 200, `[{"timeBucket":"2024-05-01","count":1},{"timeBucket":"2023-01-01","count":1}]`)
	h.do(fiber.MethodGet, "/api/timeline/buckets?userId="+adminID+"&visibility=archive", nil, partner...).
		expect(t, 400, `{"message":"Not found or no archive.read access"}`)
	h.do(fiber.MethodGet, "/api/timeline/buckets?userId="+adminID+"&visibility=timeline&withPartners=true", nil, partner...).
		expect(t, 200, `[{"timeBucket":"2024-05-01","count":1},{"timeBucket":"2023-01-01","count":1}]`)

	stack := h.do(fiber.MethodPost, "/api/stacks", `{"assetIds":["`+plain+`","`+exif+`"]}`, owner...)
	if stack.Status != 201 {
		t.Fatalf("stack = %d %s", stack.Status, stack.Body)
	}
	h.do(fiber.MethodGet, "/api/timeline/buckets?withStacked=true&visibility=timeline", nil, owner...).
		expect(t, 200, `[{"timeBucket":"2023-01-01","count":1}]`)
	jan := h.do(fiber.MethodGet, "/api/timeline/bucket?timeBucket=2023-01-01T00:00:00.000Z&withStacked=true", nil, owner...)
	if got := bucketShape(jan.Body); !regexp.MustCompile(`"id":\["<uuid>","<uuid>"\].*"stack":\[null, \n \["<uuid>","2"\]\]\}$`).MatchString(got) {
		t.Errorf("stacked January = %s", got)
	}
}

func thumbhashOf(t *testing.T, h *apiHarness, token string, id string) string {
	t.Helper()

	return h.do(fiber.MethodGet, "/api/assets/"+id, nil, bearer(token)...).str(t, "thumbhash")
}
