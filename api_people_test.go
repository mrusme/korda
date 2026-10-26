package main

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
)

func TestPeopleEmptyState(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	asset := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2023-01-02T03:04:05.000Z", "2023-01-02T03:04:05.000Z")
	adminID := h.do(fiber.MethodGet, "/api/users/me", nil, owner...).str(t, "id")
	otherID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	const person = "0f0e0d0c-0b0a-4908-8706-050403020100"

	h.do(fiber.MethodGet, "/api/people", nil, owner...).expect(t, 200, `{"people":[],"hasNextPage":false,"total":0,"hidden":0}`)
	h.do(fiber.MethodGet, "/api/people?closestPersonId="+person, nil, owner...).expect(t, 404, `{"message":"Person not found"}`)
	h.do(fiber.MethodPost, "/api/people", `{"name":"Ann"}`, owner...).expect(t, 400, `{"message":"People and faces are not supported by korda"}`)
	h.do(fiber.MethodPut, "/api/people", `{"people":[{"id":"`+person+`","name":"x"}]}`, owner...).
		expect(t, 200, `[{"id":"`+person+`","success":false,"error":"unknown"}]`)
	h.do(fiber.MethodDelete, "/api/people", `{"ids":[]}`, owner...).expect(t, 204, "")
	h.do(fiber.MethodDelete, "/api/people", `{"ids":["`+person+`"]}`, owner...).expect(t, 400, `{"message":"Not found or no person.delete access"}`)
	h.do(fiber.MethodDelete, "/api/people", `{"ids":["`+person+`"],"userId":"x"}`, owner...).expect(t, 500, `{"message":"Failed to delete people"}`)
	h.do(fiber.MethodPost, "/api/people/merge", `{"ids":["`+person+`"]}`, owner...).expect(t, 400, `{"message":"At least two people are required for merging"}`)
	h.do(fiber.MethodPost, "/api/people/merge", `{"ids":["`+person+`","`+person+`"]}`, owner...).expect(t, 400, `{"message":"Cannot merge a person into themselves"}`)
	h.do(fiber.MethodPost, "/api/people/"+person+"/merge", `{"ids":["`+asset+`"]}`, owner...).
		expect(t, 200, `[{"id":"`+person+`","success":false,"error":"no_permission"},{"id":"`+asset+`","success":false,"error":"no_permission"}]`)
	h.do(fiber.MethodGet, "/api/people/users", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodPut, "/api/people/users", `{"type":"everyone","sharedWithIds":["`+adminID+`"],"role":"read"}`, owner...).
		expect(t, 400, `{"message":"Cannot share a person with yourself"}`)
	h.do(fiber.MethodPut, "/api/people/users", `{"type":"everyone","sharedWithIds":["`+otherID+`"],"role":"read"}`, owner...).
		expect(t, 400, `{"message":"All users must be in the same cluster group"}`)
	h.do(fiber.MethodPut, "/api/people/users", `{"type":"everyone","sharedWithIds":[],"role":"read"}`, owner...).expect(t, 200, "")
	h.do(fiber.MethodDelete, "/api/people/users", `[]`, owner...).expect(t, 200, "")
	h.do(fiber.MethodDelete, "/api/people/users", `[{"personId":"`+person+`","sharedWithId":"`+otherID+`"}]`, owner...).
		expect(t, 400, `{"message":"Not found or no person.update access"}`)
	h.do(fiber.MethodGet, "/api/people/"+person, nil, owner...).expect(t, 400, `{"message":"Not found or no person.read access"}`)
	h.do(fiber.MethodGet, "/api/people/"+person+"/statistics", nil, owner...).expect(t, 400, `{"message":"Not found or no person.read access"}`)
	h.do(fiber.MethodGet, "/api/people/"+person+"/thumbnail", nil, owner...).expect(t, 404, `{"message":"Not Found"}`)
	h.do(fiber.MethodPut, "/api/people/"+person, `{"isHidden":true,"userId":"`+otherID+`"}`, owner...).
		expect(t, 400, `{"message":"Only name and birthDate can be updated for other users"}`)
	h.do(fiber.MethodPatch, "/api/people/"+person, `{"name":"x"}`, owner...).expect(t, 400, `{"message":"Not found or no person.update access"}`)
	h.do(fiber.MethodPut, "/api/people/abc", `{}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"custom","path":[],"message":"At least one of the following fields is required: name, birthDate, isHidden, isFavorite, color, featureFaceAssetId"}]}`)
	h.do(fiber.MethodDelete, "/api/people/"+person, nil, owner...).expect(t, 400, `{"message":"Not found or no person.delete access"}`)
	h.do(fiber.MethodPut, "/api/people/"+person+"/reassign", `{"data":[]}`, owner...).expect(t, 400, `{"message":"Not found or no person.update access"}`)

	h.do(fiber.MethodGet, "/api/faces?id="+asset, nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/faces?id="+person, nil, owner...).expect(t, 400, `{"message":"Not found or no asset.read access"}`)
	h.do(fiber.MethodPost, "/api/faces", `{"personId":"`+person+`","assetId":"`+person+`","imageWidth":10,"imageHeight":10,"x":0,"y":0,"width":5,"height":5}`, owner...).
		expect(t, 400, `{"message":"Not found or no asset.update access"}`)
	h.do(fiber.MethodPost, "/api/faces", `{"personId":"`+person+`","assetId":"`+asset+`","imageWidth":10,"imageHeight":10,"x":0,"y":0,"width":5,"height":5}`, owner...).
		expect(t, 400, `{"message":"People and faces are not supported by korda"}`)
	h.do(fiber.MethodPut, "/api/faces/"+person, `{"id":"`+person+`"}`, owner...).expect(t, 400, `{"message":"Not found or no person.update access"}`)
	h.do(fiber.MethodDelete, "/api/faces/"+person, `{"force":false}`, owner...).expect(t, 400, `{"message":"Not found or no face.delete access"}`)
}
