package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/internal/testmedia"
	"tty.fail/mrus/korda/models/queue"
)

func enumIssue(path string, values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, `\"`+v+`\"`)
	}

	return `{"message":"Validation failed","errors":[{"code":"invalid_value","values":["` + strings.Join(values, `","`) +
		`"],"path":[` + path + `],"message":"Invalid option: expected one of ` + strings.Join(quoted, "|") + `"}]}`
}

func idleQueues(exceptions map[queue.Name]string) (string, string) {
	const zero = `{"active":0,"completed":0,"failed":0,"delayed":0,"waiting":0,"paused":0}`

	var list, legacy []string
	for _, name := range queue.Names {
		stats, paused := zero, "false"
		if e, ok := exceptions[name]; ok {
			stats = e
		}
		list = append(list, `{"name":"`+string(name)+`","isPaused":`+paused+`,"statistics":`+stats+`}`)
		legacy = append(legacy, `"`+string(name)+`":{"queueStatus":{"isPaused":false,"isActive":false},"jobCounts":`+stats+`}`)
	}

	return "[" + strings.Join(list, ",") + "]", "{" + strings.Join(legacy, ",") + "}"
}

func TestQueueRoutes(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	h.createUser(admin, "mira@example.com", "Mira", "mira-password")
	member := bearer(h.login("mira@example.com", "mira-password"))
	limited := []string{"x-api-key", h.apiKey(admin, "server.about")}
	names := queue.Name("").Enum()

	list, legacy := idleQueues(nil)
	h.do(fiber.MethodGet, "/api/queues", nil, owner...).expect(t, 200, list)
	h.do(fiber.MethodGet, "/api/jobs", nil, owner...).expect(t, 200, legacy)
	h.do(fiber.MethodGet, "/api/jobs", nil, member...).expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodGet, "/api/jobs", nil, limited...).expect(t, 403, `{"message":"Missing required permission: job.read"}`)
	h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration", nil, limited...).
		expect(t, 403, `{"message":"Missing required permission: queue.read"}`)
	h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration", nil, owner...).
		expect(t, 200, `{"name":"thumbnailGeneration","isPaused":false,"statistics":{"active":0,"completed":0,"failed":0,"delayed":0,"waiting":0,"paused":0}}`)
	h.do(fiber.MethodGet, "/api/queues/ThumbnailGeneration", nil, owner...).expect(t, 400, enumIssue(`"name"`, names))

	h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration/jobs", nil, owner...).expect(t, 200, `[]`)
	h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration/jobs?status=active", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"array","code":"invalid_type","path":["status"],"message":"Invalid input: expected array, received string"}]}`)
	h.do(fiber.MethodGet, "/api/queues/nope/jobs?status=active", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"array","code":"invalid_type","path":["status"],"message":"Invalid input: expected array, received string"}]}`)
	statuses := queue.Status("").Enum()
	issue := enumIssue(`"status",1`, statuses)
	two := enumIssue(`"status",2`, statuses)
	h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration/jobs?status=waiting&status=nope&status=x", nil, owner...).
		expect(t, 400, issue[:len(issue)-2]+","+two[strings.Index(two, `[{`)+1:])
	h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration/jobs?status%5B%5D=active", nil, owner...).expect(t, 200, `[]`)

	h.do(fiber.MethodDelete, "/api/queues/nope/jobs", nil, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}]}`)
	h.do(fiber.MethodDelete, "/api/queues/nope/jobs", `{}`, owner...).expect(t, 400, enumIssue(`"name"`, names))
	h.do(fiber.MethodDelete, "/api/queues/thumbnailGeneration/jobs", `{"failed":"yes"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["failed"],"message":"Invalid input: expected boolean, received string"}]}`)
	h.do(fiber.MethodDelete, "/api/queues/thumbnailGeneration/jobs", `{"failed":"yes"}`, limited...).
		expect(t, 403, `{"message":"Missing required permission: queueJob.delete"}`)

	h.do(fiber.MethodPut, "/api/queues/thumbnailGeneration", `{}`, owner...).
		expect(t, 200, `{"name":"thumbnailGeneration","isPaused":false,"statistics":{"active":0,"completed":0,"failed":0,"delayed":0,"waiting":0,"paused":0}}`)
	h.do(fiber.MethodPut, "/api/queues/backgroundTask", `{"isPaused":true}`, owner...).
		expect(t, 400, `{"message":"The BackgroundTask queue cannot be paused"}`)
	h.do(fiber.MethodPut, "/api/queues/nope", `{"isPaused":"yes"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["isPaused"],"message":"Invalid input: expected boolean, received string"}]}`)
	h.do(fiber.MethodPut, "/api/queues/thumbnailGeneration", `{"isPaused":null}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["isPaused"],"message":"Invalid input: expected boolean, received null"}]}`)

	h.do(fiber.MethodPost, "/api/jobs", `{}`, owner...).expect(t, 400, enumIssue(`"name"`, queue.ManualJob("").Enum()))
	h.do(fiber.MethodPost, "/api/jobs", `{"name":"TagCleanup"}`, owner...).expect(t, 400, enumIssue(`"name"`, queue.ManualJob("").Enum()))
	h.do(fiber.MethodPost, "/api/jobs", `{"name":"nope"}`, member...).expect(t, 403, `{"message":"Forbidden"}`)
	h.do(fiber.MethodPost, "/api/jobs", `{"name":"integrity-missing-files"}`, owner...).
		expect(t, 400, `{"message":"Integrity reports are not supported by korda"}`)
	h.do(fiber.MethodPost, "/api/jobs", `{"name":"person-cleanup"}`, owner...).expect(t, 204, "")
	if ids := h.queue.ids(); len(ids) != 0 {
		t.Errorf("queued = %v", ids)
	}

	h.do(fiber.MethodPut, "/api/jobs/nope", `{"command":"nope"}`, owner...).
		expect(t, 400, enumIssue(`"command"`, queue.Command("").Enum()))
	h.do(fiber.MethodPut, "/api/jobs/nope", `{"command":"start"}`, owner...).expect(t, 400, enumIssue(`"name"`, names))
	h.do(fiber.MethodPut, "/api/jobs/thumbnailGeneration", `{"command":"start","force":"yes"}`, owner...).
		expect(t, 400, `{"message":"Validation failed","errors":[{"expected":"boolean","code":"invalid_type","path":["force"],"message":"Invalid input: expected boolean, received string"}]}`)
	h.do(fiber.MethodPut, "/api/jobs/notifications", `{"command":"start"}`, owner...).
		expect(t, 400, `{"message":"Invalid job name: notifications"}`)
	h.do(fiber.MethodPut, "/api/jobs/smartSearch", `{"command":"start"}`, owner...).
		expect(t, 200, `{"queueStatus":{"isPaused":false,"isActive":false},"jobCounts":{"active":0,"completed":0,"failed":0,"delayed":0,"waiting":0,"paused":0}}`)
	if ids := h.queue.ids(); len(ids) != 0 {
		t.Errorf("queued = %v", ids)
	}
}

func TestQueueJobs(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)
	asset := h.uploadID(admin, "korda-exif.jpg", fixture(t, testmedia.ExifJPEG), "2024-05-06T05:00:00.000Z", "2024-05-06T05:00:00.000Z")
	h.runAllJobs()

	h.do(fiber.MethodPut, "/api/jobs/thumbnailGeneration", `{"command":"start","force":true}`, owner...).
		expect(t, 200, `{"queueStatus":{"isPaused":false,"isActive":false},"jobCounts":{"active":0,"completed":0,"failed":0,"delayed":0,"waiting":1,"paused":0}}`)
	h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration/jobs", nil, owner...).
		expectShape(t, 200, `[{"id":"thumbnails-all:<uuid>","name":"AssetGenerateThumbnailsQueueAll","timestamp":<ms>,"data":{"force":true}}]`)
	h.do(fiber.MethodPut, "/api/queues/thumbnailGeneration", `{"isPaused":true}`, owner...).
		expect(t, 200, `{"name":"thumbnailGeneration","isPaused":true,"statistics":{"active":0,"completed":0,"failed":0,"delayed":0,"waiting":0,"paused":1}}`)
	h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration/jobs?status=active&status=failed", nil, owner...).expect(t, 200, `[]`)
	if got := h.do(fiber.MethodGet, "/api/queues/thumbnailGeneration/jobs?status=waiting&status=active", nil, owner...).array(t); len(got) != 1 {
		t.Errorf("paused jobs = %v", got)
	}
	h.do(fiber.MethodPut, "/api/jobs/thumbnailGeneration", `{"command":"resume"}`, owner...).
		expect(t, 200, `{"queueStatus":{"isPaused":false,"isActive":false},"jobCounts":{"active":0,"completed":0,"failed":0,"delayed":0,"waiting":1,"paused":0}}`)
	h.runJobs()
	ids := h.queue.ids()
	if len(ids) != 1 || ids[0] != "thumbnails:"+asset {
		t.Errorf("sweep queued %v", ids)
	}

	req := httptest.NewRequest(fiber.MethodDelete, "/api/queues/thumbnailGeneration/jobs", nil)
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	req.Header.Set(fiber.HeaderContentLength, "0")
	req.Header.Set(fiber.HeaderAuthorization, owner[1])
	if res := h.send(req); res.Status != 204 {
		t.Errorf("empty with an empty body = %d %s", res.Status, res.Body)
	}
	if ids := h.queue.ids(); len(ids) != 0 {
		t.Errorf("after emptying = %v", ids)
	}

	h.do(fiber.MethodPut, "/api/jobs/metadataExtraction", `{"command":"start"}`, owner...)
	h.runJobs()
	if ids := h.queue.ids(); len(ids) != 0 {
		t.Errorf("metadata sweep without force = %v", ids)
	}
	h.do(fiber.MethodPut, "/api/jobs/metadataExtraction", `{"command":"start","force":true}`, owner...)
	h.runJobs()
	if ids := h.queue.ids(); len(ids) != 1 || ids[0] != "metadata:"+asset {
		t.Errorf("metadata sweep with force = %v", ids)
	}
	h.runAllJobs()

	tag := h.do(fiber.MethodPost, "/api/tags", `{"name":"empty"}`, owner...).str(t, "id")
	upserted := h.do(fiber.MethodPut, "/api/tags", `{"tags":["kept/child"]}`, owner...).array(t)
	child := upserted[len(upserted)-1].(map[string]any)["id"].(string)
	h.do(fiber.MethodPut, "/api/tags/"+child+"/assets", `{"ids":["`+asset+`"]}`, owner...).
		expect(t, 200, `[{"id":"`+asset+`","success":true}]`)
	h.do(fiber.MethodPost, "/api/jobs", `{"name":"tag-cleanup"}`, owner...).expect(t, 204, "")
	h.do(fiber.MethodGet, "/api/queues/backgroundTask/jobs?status=waiting&status=failed", nil, owner...).
		expectShape(t, 200, `[{"id":"tag-cleanup:<uuid>","name":"TagCleanup","timestamp":<ms>,"data":{}}]`)
	h.runJobs()
	var values []string
	for _, v := range h.do(fiber.MethodGet, "/api/tags", nil, owner...).array(t) {
		values = append(values, v.(map[string]any)["value"].(string))
	}
	if strings.Join(values, ",") != "kept,kept/child" {
		t.Errorf("tags after the cleanup = %v, %s gone", values, tag)
	}
}
