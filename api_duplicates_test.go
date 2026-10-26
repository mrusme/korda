package main

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

func TestDuplicatesAndMap(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	exif := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	plain := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	png := h.uploadID(admin, "korda.png", fixture(t, testmedia.PlainPNG), "2023-01-03T03:04:05.000Z", "2023-01-03T03:04:05.000Z")
	h.runAllJobs()
	const group = "5d000000-0000-4000-8000-000000000001"
	const single = "5d000000-0000-4000-8000-000000000002"

	h.do(fiber.MethodGet, "/api/duplicates", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodPut, "/api/assets", `{"ids":["`+exif+`","`+plain+`"],"duplicateId":"`+group+`"}`, owner...).expect(t, 204, "")
	h.do(fiber.MethodPut, "/api/assets", `{"ids":["`+png+`"],"duplicateId":"`+single+`"}`, owner...).expect(t, 204, "")
	res := h.do(fiber.MethodGet, "/api/duplicates", nil, owner...)
	groups := res.array(t)
	if len(groups) != 1 {
		t.Fatalf("groups = %s", res.Body)
	}
	g := groups[0].(map[string]any)
	assets := g["assets"].([]any)
	if g["duplicateId"] != group || len(assets) != 2 || assets[0].(map[string]any)["id"] != plain {
		t.Errorf("group = %s", res.Body)
	}
	if keep := g["suggestedKeepAssetIds"].([]any); len(keep) != 1 || keep[0] != exif {
		t.Errorf("suggested = %v", keep)
	}
	if !strings.Contains(string(res.Body), `"createdAt":"`) || strings.Contains(string(res.Body), `"owner":`) {
		t.Errorf("asset shape = %s", res.Body)
	}
	if got := h.do(fiber.MethodGet, "/api/assets/"+png, nil, owner...).object(t)["duplicateId"]; got != nil {
		t.Errorf("singleton duplicateId = %v", got)
	}

	h.do(fiber.MethodPost, "/api/duplicates/resolve", `{"groups":[]}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"array","code":"too_small","minimum":1,"inclusive":true,"path":["groups"],"message":"Too small: expected array to have >=1 items"}]}`)
	h.do(fiber.MethodPost, "/api/duplicates/resolve", `{"groups":[{"duplicateId":"`+single+`","keepAssetIds":[],"trashAssetIds":[]}]}`, owner...).
		expect(t, 400, `{"message":"Not found or no duplicate.delete access"}`)
	h.do(fiber.MethodPost, "/api/duplicates/resolve", `{"groups":[{"duplicateId":"`+group+`","keepAssetIds":["`+exif+`"],"trashAssetIds":["`+exif+`","`+plain+`"]}]}`, owner...).
		expect(t, 200, `[{"id":"`+group+`","success":false,"error":"validation","errorMessage":"An asset cannot be in both keepAssetIds and trashAssetIds"}]`)
	h.do(fiber.MethodPost, "/api/duplicates/resolve", `{"groups":[{"duplicateId":"`+group+`","keepAssetIds":["`+exif+`"],"trashAssetIds":[]}]}`, owner...).
		expect(t, 200, `[{"id":"`+group+`","success":false,"error":"validation","errorMessage":"Every asset must be in either keepAssetIds or trashAssetIds"}]`)
	h.do(fiber.MethodPut, "/api/assets/"+plain, `{"isFavorite":true,"description":"from the copy"}`, owner...)
	h.runAllJobs()
	h.do(fiber.MethodPost, "/api/duplicates/resolve", `{"groups":[{"duplicateId":"`+group+`","keepAssetIds":["`+exif+`"],"trashAssetIds":["`+plain+`"]}]}`, owner...).
		expect(t, 200, `[{"id":"`+group+`","success":true}]`)
	kept := h.do(fiber.MethodGet, "/api/assets/"+exif, nil, owner...).object(t)
	if kept["duplicateId"] != nil || kept["isFavorite"] != true || kept["exifInfo"].(map[string]any)["description"] != "from the copy\na fixture" {
		t.Errorf("kept = %v", kept)
	}
	if trashed := h.do(fiber.MethodGet, "/api/assets/"+plain, nil, owner...).object(t); trashed["isTrashed"] != true || trashed["duplicateId"] != nil {
		t.Errorf("trashed = %v", trashed)
	}
	h.do(fiber.MethodDelete, "/api/duplicates/"+group, nil, owner...).expect(t, 400, `{"message":"Not found or no duplicate.delete access"}`)
	h.do(fiber.MethodDelete, "/api/duplicates", `{"ids":[]}`, owner...).expect(t, 204, "")

	h.do(fiber.MethodGet, "/api/map/markers", nil, owner...).
		expect(t, 200, `[{"id":"`+exif+`","lat":52.516272,"lon":13.377722,"city":null,"state":null,"country":null}]`)
	h.do(fiber.MethodGet, "/api/map/markers?fileCreatedBefore=2024-01-01T00:00:00Z", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/map/markers?isArchived=1", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["isArchived"],"message":"Invalid option: expected one of \"true\"|\"false\""}]}`)
	h.do(fiber.MethodGet, "/api/map/reverse-geocode?lat=52.5&lon=13.4", nil, owner...).expect(t, 200, `[{"country":null,"state":null,"city":null}]`)
	h.do(fiber.MethodGet, "/api/map/reverse-geocode?lat=91&lon=13.4", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"number","code":"too_big","maximum":90,"inclusive":true,"path":["lat"],"message":"Too big: expected number to be <=90"}]}`)
}
