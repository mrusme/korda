package main

import (
	"slices"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/hibiken/asynq"
	"tty.fail/mrus/korda/internal/testmedia"
	"tty.fail/mrus/korda/services/jobs"
)

func TestNotificationRules(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	memberID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))
	h.createUser(admin, "carl@example.com", "Carl", "carl-password")
	stranger := bearer(h.login("carl@example.com", "carl-password"))

	h.do(fiber.MethodGet, "/api/notifications", nil, member...).expect(t, 200, `[]`)
	h.do(fiber.MethodPost, "/api/admin/notifications", `{}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["title"],"message":"Invalid input: expected string, received undefined"},{"expected":"string","code":"invalid_type","path":["userId"],"message":"Invalid input: expected string, received undefined"}]}`)
	h.do(fiber.MethodPost, "/api/admin/notifications", `{"userId":"`+memberID+`","title":"x","level":"x"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["success","error","warning","info"],"path":["level"],"message":"Invalid option: expected one of \"success\"|\"error\"|\"warning\"|\"info\""}]}`)
	h.do(fiber.MethodPost, "/api/admin/notifications", `{"userId":"`+memberID+`","title":"x"}`, member...).
		expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodPost, "/api/admin/notifications", `{"userId":"36ab44e4-048e-4a22-9129-b3794472c9dd","title":"x"}`, owner...).
		expect(t, 500, `{"message":"Failed to create notification"}`)

	one := h.do(fiber.MethodPost, "/api/admin/notifications", `{"userId":"`+memberID+`","title":"Probe one"}`, owner...)
	one.expectShape(t, 201, `{"id":"<uuid>","createdAt":"<time>","level":"info","type":"Custom","title":"Probe one"}`)
	two := h.do(fiber.MethodPost, "/api/admin/notifications", `{"userId":"`+memberID+`","title":"Probe two","level":"error","type":"SystemMessage","description":"second","data":{"b":1,"a":[1,2],"10":"x","albumId":"36ab44e4-048e-4a22-9129-b3794472c9dd"},"readAt":"2026-10-25T10:00:00.000Z"}`, owner...)
	two.expectShape(t, 201, `{"id":"<uuid>","createdAt":"<time>","level":"error","type":"SystemMessage","title":"Probe two","description":"second","data":{"10":"x","a":[1,2],"b":1,"albumId":"<uuid>"}}`)
	three := h.do(fiber.MethodPost, "/api/admin/notifications", `{"userId":"`+memberID+`","title":"Probe three","level":"warning","type":"Custom","description":null,"data":{}}`, owner...)
	three.expectShape(t, 201, `{"id":"<uuid>","createdAt":"<time>","level":"warning","type":"Custom","title":"Probe three","data":{}}`)
	n1, n2, n3 := one.str(t, "id"), two.str(t, "id"), three.str(t, "id")

	h.do(fiber.MethodGet, "/api/notifications?unread=true", nil, member...).
		expectShape(t, 200, `[{"id":"<uuid>","createdAt":"<time>","level":"warning","type":"Custom","title":"Probe three","data":{}},{"id":"<uuid>","createdAt":"<time>","level":"error","type":"SystemMessage","title":"Probe two","description":"second","data":{"10":"x","a":[1,2],"b":1,"albumId":"<uuid>"}},{"id":"<uuid>","createdAt":"<time>","level":"info","type":"Custom","title":"Probe one"}]`)
	h.do(fiber.MethodGet, "/api/notifications?level=error", nil, member...).
		expectShape(t, 200, `[{"id":"<uuid>","createdAt":"<time>","level":"error","type":"SystemMessage","title":"Probe two","description":"second","data":{"10":"x","a":[1,2],"b":1,"albumId":"<uuid>"}}]`)
	if got := h.do(fiber.MethodGet, "/api/notifications?type=Custom&id="+n1, nil, member...).array(t); len(got) != 1 {
		t.Errorf("by id = %v", got)
	}
	h.do(fiber.MethodGet, "/api/notifications?unread=yes", nil, member...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["unread"],"message":"Invalid option: expected one of \"true\"|\"false\""}]}`)
	h.do(fiber.MethodGet, "/api/notifications", nil, stranger...).expect(t, 200, `[]`)

	h.do(fiber.MethodGet, "/api/notifications/"+n1, nil, member...).expect(t, 400, `{"message":"Notification not found"}`)
	h.do(fiber.MethodGet, "/api/notifications/"+n1, nil, stranger...).
		expect(t, 400, `{"message":"Not found or no notification.read access"}`)
	h.do(fiber.MethodPut, "/api/notifications/"+n1, `{"readAt":"2026-10-25T10:00:00.5+02:00"}`, member...).
		expect(t, 200, `{"id":"`+n1+`","createdAt":"`+one.str(t, "createdAt")+`","level":"info","type":"Custom","title":"Probe one","readAt":"2026-10-25T08:00:00.500Z"}`)
	h.do(fiber.MethodPut, "/api/notifications/"+n1, `{"readAt":null}`, member...).
		expect(t, 200, `{"id":"`+n1+`","createdAt":"`+one.str(t, "createdAt")+`","level":"info","type":"Custom","title":"Probe one"}`)
	h.do(fiber.MethodPut, "/api/notifications/"+n1, `{}`, member...).expect(t, 500, `{"message":"Failed to update notification"}`)
	h.do(fiber.MethodPut, "/api/notifications/"+n1, nil, member...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}]}`)
	h.do(fiber.MethodPut, "/api/notifications/"+n1, `{"readAt":null}`, stranger...).
		expect(t, 400, `{"message":"Not found or no notification.update access"}`)

	h.do(fiber.MethodPut, "/api/notifications", `{"ids":["`+n1+`","`+n2+`"],"readAt":"2026-10-25T11:00:00.000Z"}`, member...).expect(t, 204, "")
	h.do(fiber.MethodPut, "/api/notifications", `{"ids":["`+n1+`"]}`, member...).expect(t, 500, `{"message":"Failed to update notifications"}`)
	h.do(fiber.MethodPut, "/api/notifications", `{"ids":[]}`, member...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"array","code":"too_small","minimum":1,"inclusive":true,"path":["ids"],"message":"Too small: expected array to have >=1 items"}]}`)
	h.do(fiber.MethodPut, "/api/notifications", `{"ids":["`+n1+`"],"readAt":null}`, stranger...).
		expect(t, 400, `{"message":"Not found or no notification.update access"}`)
	if got := h.do(fiber.MethodGet, "/api/notifications?unread=true", nil, member...).array(t); len(got) != 1 {
		t.Errorf("unread after the updates = %v", got)
	}

	h.do(fiber.MethodDelete, "/api/notifications/"+n2, nil, member...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/notifications/"+n2, nil, member...).
		expect(t, 200, `{"id":"`+n2+`","createdAt":"`+two.str(t, "createdAt")+`","level":"error","type":"SystemMessage","title":"Probe two","description":"second","data":{"10":"x","a":[1,2],"b":1,"albumId":"36ab44e4-048e-4a22-9129-b3794472c9dd"},"readAt":"2026-10-25T11:00:00.000Z"}`)
	h.do(fiber.MethodPut, "/api/notifications/"+n2, `{"readAt":null}`, member...).expect(t, 500, `{"message":"Failed to update notification"}`)
	h.do(fiber.MethodPut, "/api/notifications", `{"ids":["`+n2+`"],"readAt":null}`, member...).expect(t, 204, "")
	h.do(fiber.MethodDelete, "/api/notifications/"+n2, nil, member...).expect(t, 204, "")
	h.do(fiber.MethodDelete, "/api/notifications/"+n1, nil, stranger...).
		expect(t, 400, `{"message":"Not found or no notification.delete access"}`)
	h.do(fiber.MethodDelete, "/api/notifications", `{"ids":[]}`, member...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"origin":"array","code":"too_small","minimum":1,"inclusive":true,"path":["ids"],"message":"Too small: expected array to have >=1 items"}]}`)
	h.do(fiber.MethodDelete, "/api/notifications", nil, member...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}]}`)
	h.do(fiber.MethodDelete, "/api/notifications", `{"ids":["`+n1+`","`+n3+`"]}`, member...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/notifications", nil, member...).expect(t, 200, `[]`)
}

func TestNotificationAdminRoutes(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))

	for _, name := range []string{"welcome", "album-invite", "album-update", "test", "reset-password", "foo"} {
		h.do(fiber.MethodPost, "/api/admin/notifications/templates/"+name, `{"template":""}`, owner...).
			expect(t, 200, `{"name":"`+name+`","html":""}`)
	}
	h.do(fiber.MethodPost, "/api/admin/notifications/templates/welcome", `{}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"string","code":"invalid_type","path":["template"],"message":"Invalid input: expected string, received undefined"}]}`)
	h.do(fiber.MethodPost, "/api/admin/notifications/templates/welcome", `{"template":""}`, member...).
		expect(t, 403, `{"message":"Forbidden"}`)

	h.do(fiber.MethodPost, "/api/admin/notifications/test-email", `{}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["enabled"],"message":"Invalid input: expected boolean, received undefined"},{"expected":"string","code":"invalid_type","path":["from"],"message":"Invalid input: expected string, received undefined"},{"expected":"string","code":"invalid_type","path":["replyTo"],"message":"Invalid input: expected string, received undefined"},{"expected":"object","code":"invalid_type","path":["transport"],"message":"Invalid input: expected object, received undefined"}]}`)
	h.do(fiber.MethodPost, "/api/admin/notifications/test-email", `{"enabled":"true","from":"korda@example.com","replyTo":"","transport":{"ignoreCert":"false","host":"127.0.0.1","port":1,"secure":false,"username":"","password":""}}`, owner...).
		expect(t, 400, `{"message":"Failed to verify SMTP configuration"}`)
	h.do(fiber.MethodPost, "/api/admin/notifications/test-email", `{}`, member...).expect(t, 403, `{"message":"Forbidden"}`)
}

func TestAlbumNotifications(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	memberID := h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))
	juneID := h.createUser(admin, "june@example.com", "June", "june-password")
	june := bearer(h.login("june@example.com", "june-password"))
	asset := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	other := h.uploadID(admin, "korda-plain.jpg", fixture(t, testmedia.PlainJPEG), "2024-05-07T05:00:00.000Z", "2024-05-07T05:00:00.000Z")
	h.runAllJobs()
	rec := h.recordSockets()

	albumID := h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Holiday","albumUsers":[{"userId":"`+memberID+`","role":"editor"}]}`, owner...).str(t, "id")
	h.runAllJobs()
	invite := `{"id":"<uuid>","createdAt":"<time>","level":"success","type":"AlbumInvite","title":"Shared Album Invitation","description":"Admin shared an album (Holiday) with you","data":"{\"albumId\":\"<uuid>\"}"}`
	h.do(fiber.MethodGet, "/api/notifications", nil, member...).expectShape(t, 200, "["+invite+"]")
	got := rec.take()
	if len(got) != 1 || got[0].room != memberID || got[0].event != "on_notification" || normalize(got[0].args[0]) != invite {
		t.Errorf("invite events = %+v", got)
	}

	h.do(fiber.MethodPut, "/api/albums/"+albumID+"/users", `{"albumUsers":[{"userId":"`+juneID+`","role":"viewer"}]}`, owner...)
	h.runAllJobs()
	h.do(fiber.MethodGet, "/api/notifications", nil, june...).expectShape(t, 200, "["+invite+"]")
	rec.take()

	h.do(fiber.MethodPut, "/api/albums/"+albumID+"/assets", `{"ids":["`+asset+`"]}`, owner...)
	h.do(fiber.MethodPut, "/api/albums/"+albumID+"/assets", `{"ids":["`+other+`"]}`, owner...)
	queued := h.queue.drain()
	var ids []string
	for _, j := range queued {
		ids = append(ids, j.id)
	}
	slices.Sort(ids)
	want := []string{string(jobs.NotifyAlbumUpdate) + ":" + albumID + "/" + juneID, string(jobs.NotifyAlbumUpdate) + ":" + albumID + "/" + memberID}
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Errorf("queued = %v, want %v", ids, want)
	}
	for _, j := range queued {
		if !slices.ContainsFunc(j.opts, func(o asynq.Option) bool {
			return o.Type() == asynq.ProcessInOpt && o.Value() == jobs.AlbumUpdateDelay
		}) {
			t.Errorf("job %s has no delay of %s: %v", j.id, jobs.AlbumUpdateDelay, j.opts)
		}
		if err := h.handlers.Execute(h.ctx, j.job); err != nil {
			t.Fatal(err)
		}
	}

	update := `{"id":"<uuid>","createdAt":"<time>","level":"info","type":"AlbumUpdate","title":"Shared Album Update","description":"New media has been added to the album (Holiday)","data":"{\"albumId\":\"<uuid>\"}"}`
	h.do(fiber.MethodGet, "/api/notifications", nil, june...).expectShape(t, 200, "["+update+","+invite+"]")
	h.do(fiber.MethodGet, "/api/notifications", nil, member...).expectShape(t, 200, "["+update+","+invite+"]")
	h.do(fiber.MethodGet, "/api/notifications", nil, owner...).expect(t, 200, "[]")

	h.do(fiber.MethodDelete, "/api/albums/"+albumID, nil, owner...)
	h.do(fiber.MethodPost, "/api/albums", `{"albumName":"Gone","albumUsers":[{"userId":"`+memberID+`"}]}`, owner...)
	for _, j := range h.queue.drain() {
		if j.job.Type == jobs.NotifyAlbumInvite {
			h.do(fiber.MethodDelete, "/api/albums/"+albumIDOf(t, h, owner, "Gone"), nil, owner...)
			if err := h.handlers.Execute(h.ctx, j.job); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := h.do(fiber.MethodGet, "/api/notifications", nil, member...).array(t); len(got) != 2 {
		t.Errorf("notifications after a deleted album = %v", got)
	}
}

func albumIDOf(t *testing.T, h *apiHarness, headers []string, name string) string {
	t.Helper()

	for _, a := range h.do(fiber.MethodGet, "/api/albums", nil, headers...).array(t) {
		if album, ok := a.(map[string]any); ok && album["albumName"] == name {
			return album["id"].(string)
		}
	}
	t.Fatalf("no album %q", name)

	return ""
}
