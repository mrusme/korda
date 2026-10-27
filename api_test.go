package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/hibiken/asynq"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/http/routes"
	"tty.fail/mrus/korda/http/web"
	logicserver "tty.fail/mrus/korda/logic/root/server"
	"tty.fail/mrus/korda/models/queue"
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/worker/targets"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/models/asyncjob"
	"xn--gckvb8fzb.com/glides/runtime"
	"xn--gckvb8fzb.com/glides/services/database/sqlite"
	"xn--gckvb8fzb.com/glides/services/dispatch"
	"xn--gckvb8fzb.com/glides/worker/targets/handler"
)

const (
	adminEmail    = "admin@example.com"
	adminPassword = "korda-admin"
	testTimeout   = 30 * time.Second
)

type jobRecorder struct {
	mu     sync.Mutex
	jobs   []recordedJob
	paused map[string]bool
}

type recordedJob struct {
	id   string
	job  asyncjob.AsyncJob
	opts []asynq.Option
}

func (r *jobRecorder) JobUnique(j *asyncjob.AsyncJob, id string, opts ...asynq.Option) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.jobs {
		if existing.id == id {
			return nil
		}
	}
	if _, err := j.SetID(); err != nil {
		return err
	}
	r.jobs = append(r.jobs, recordedJob{id: id, job: *j, opts: opts})

	return nil
}

func (r *jobRecorder) Remove(queue string, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.jobs = slices.DeleteFunc(r.jobs, func(j recordedJob) bool { return j.id == id })

	return nil
}

func (j recordedJob) queue() string {
	for _, o := range j.opts {
		if o.Type() == asynq.QueueOpt {
			return o.Value().(string)
		}
	}

	return dispatch.QueueDefault
}

func (r *jobRecorder) Stats(queue string) (jobs.Stats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	st := jobs.Stats{IsPaused: r.paused[queue]}
	for _, j := range r.jobs {
		switch {
		case j.queue() != queue:
		case st.IsPaused:
			st.Paused++
		default:
			st.Waiting++
		}
	}

	return st, nil
}

func (r *jobRecorder) Pause(queue string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused == nil {
		r.paused = map[string]bool{}
	}
	r.paused[queue] = true

	return nil
}

func (r *jobRecorder) Resume(queue string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.paused, queue)

	return nil
}

func (r *jobRecorder) ClearFailed(queue string) error {
	return nil
}

func (r *jobRecorder) Empty(queue string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.jobs = slices.DeleteFunc(r.jobs, func(j recordedJob) bool { return j.queue() == queue })

	return nil
}

func (r *jobRecorder) Tasks(name string, statuses []queue.Status) ([]jobs.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	listed := queue.StatusWaiting
	if r.paused[name] {
		listed = queue.StatusPaused
	}
	if !slices.Contains(statuses, listed) {
		return []jobs.Task{}, nil
	}

	out := []jobs.Task{}
	for _, j := range r.jobs {
		if j.queue() == name {
			out = append(out, jobs.Task{ID: j.id, Job: j.job})
		}
	}

	return jobs.Newest(out), nil
}

func (r *jobRecorder) drain() []recordedJob {
	r.mu.Lock()
	defer r.mu.Unlock()

	jobs := r.jobs
	r.jobs = nil

	return jobs
}

func (r *jobRecorder) ids() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	ids := make([]string, 0, len(r.jobs))
	for _, j := range r.jobs {
		ids = append(ids, j.id)
	}

	return ids
}

type apiHarness struct {
	t        *testing.T
	ctx      context.Context
	rt       *runtime.Runtime
	app      *fiber.App
	dir      string
	queue    *jobRecorder
	handlers handler.Handlers
}

type apiResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

type harnessOptions struct {
	rateLimitAuth int
	publicUsers   bool
	logLevel      string
}

func testConfig(dir string, opts harnessOptions) string {
	return fmt.Sprintf(`[General]
Mode = "production"

[Logging]
Level = %q

[Web]
RequestTimeout = 30

[API]
RateLimitAuth = %d

[[Target]]
ID = "jobs"
Type = "handler"

[Media]
Location = %q
Concurrency = 1
ThumbnailSize = 250
ThumbnailQuality = 80

[Server]
PublicUsers = %t
TrashDays = 30
UserDeleteDelay = 7
`, cmp.Or(opts.logLevel, "error"), opts.rateLimitAuth, filepath.Join(dir, "media"), opts.publicUsers)
}

func newHarness(t *testing.T) *apiHarness {
	t.Helper()
	return newHarnessWith(t, harnessOptions{publicUsers: true})
}

func newHarnessWith(t *testing.T, opts harnessOptions) *apiHarness {
	t.Helper()

	h := &apiHarness{t: t, ctx: context.Background(), dir: t.TempDir(), queue: &jobRecorder{}}

	cfgPath := filepath.Join(h.dir, "korda.toml")
	if err := os.WriteFile(cfgPath, []byte(testConfig(h.dir, opts)), 0o600); err != nil {
		t.Fatal(err)
	}

	var err error
	if h.rt, err = newRuntime("file://"+cfgPath, sqlite.Options{InMemory: true}); err != nil {
		t.Fatal(err)
	}
	if err = prepare(h.rt, h.queue); err != nil {
		t.Fatal(err)
	}
	if err = h.rt.Startup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := h.rt.Shutdown(); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})

	route.Use(routes.Routes, routes.DefaultTitle)
	targets.Install(h.rt, &h.handlers)

	if err = logicserver.RecordVersion(h.ctx, h.rt); err != nil {
		t.Fatal(err)
	}

	webcfg, err := gh.WebConfigFrom(h.rt.Config())
	if err != nil {
		t.Fatal(err)
	}
	apicfg, err := gh.APIConfigFrom(h.rt.Config())
	if err != nil {
		t.Fatal(err)
	}

	srv, err := web.New(h.rt, webcfg, apicfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = srv.Startup(); err != nil {
		t.Fatal(err)
	}
	h.app = srv.App()

	return h
}

func (h *apiHarness) send(req *http.Request) apiResponse {
	h.t.Helper()

	resp, err := h.app.Test(req, fiber.TestConfig{Timeout: testTimeout, FailOnTimeout: true})
	if err != nil {
		h.t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.t.Fatal(err)
	}

	return apiResponse{Status: resp.StatusCode, Header: resp.Header, Body: body}
}

func (h *apiHarness) do(method string, path string, body any, headers ...string) apiResponse {
	h.t.Helper()

	var reader io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil:
	case string:
		reader, contentType = strings.NewReader(b), fiber.MIMEApplicationJSON
	case []byte:
		reader, contentType = bytes.NewReader(b), fiber.MIMEApplicationJSON
	default:
		encoded, err := json.Marshal(b)
		if err != nil {
			h.t.Fatal(err)
		}
		reader, contentType = bytes.NewReader(encoded), fiber.MIMEApplicationJSON
	}

	req := httptest.NewRequest(method, path, reader)
	if contentType != "" {
		req.Header.Set(fiber.HeaderContentType, contentType)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	return h.send(req)
}

type part struct {
	field    string
	filename string
	content  []byte
}

func (h *apiHarness) multipart(path string, parts []part, headers ...string) apiResponse {
	h.t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, p := range parts {
		var w io.Writer
		var err error
		if p.filename == "" {
			w, err = mw.CreateFormField(p.field)
		} else {
			w, err = mw.CreateFormFile(p.field, p.filename)
		}
		if err != nil {
			h.t.Fatal(err)
		}
		if _, err = w.Write(p.content); err != nil {
			h.t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		h.t.Fatal(err)
	}

	req := httptest.NewRequest(fiber.MethodPost, path, &buf)
	req.Header.Set(fiber.HeaderContentType, mw.FormDataContentType())
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	return h.send(req)
}

func bearer(token string) []string {
	return []string{fiber.HeaderAuthorization, "Bearer " + token}
}

func (res apiResponse) object(t *testing.T) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(res.Body, &body); err != nil {
		t.Fatalf("response is not a JSON object: %v\n%s", err, res.Body)
	}

	return body
}

func (res apiResponse) array(t *testing.T) []any {
	t.Helper()

	var body []any
	if err := json.Unmarshal(res.Body, &body); err != nil {
		t.Fatalf("response is not a JSON array: %v\n%s", err, res.Body)
	}

	return body
}

func (res apiResponse) str(t *testing.T, key string) string {
	t.Helper()

	value, _ := res.object(t)[key].(string)
	return value
}

func (res apiResponse) expect(t *testing.T, status int, body string) {
	t.Helper()

	if res.Status != status || (body != "" && string(res.Body) != body) {
		t.Errorf("response = %d %s\n     want %d %s", res.Status, res.Body, status, body)
	}
}

func (res apiResponse) expectShape(t *testing.T, status int, shape string) {
	t.Helper()

	if got := normalize(string(res.Body)); res.Status != status || got != shape {
		t.Errorf("response = %d %s\n     want %d %s", res.Status, got, status, shape)
	}
}

var (
	uuidPattern  = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	timePattern  = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z`)
	tokenPattern = regexp.MustCompile(`"(accessToken|secret|token)":"[A-Za-z0-9]+"`)
	msPattern    = regexp.MustCompile(`"timestamp":\d{13}`)
)

func normalize(body string) string {
	body = uuidPattern.ReplaceAllString(body, "<uuid>")
	body = timePattern.ReplaceAllString(body, "<time>")
	body = msPattern.ReplaceAllString(body, `"timestamp":<ms>`)
	return tokenPattern.ReplaceAllString(body, `"$1":"<token>"`)
}

func (h *apiHarness) signUp() apiResponse {
	h.t.Helper()

	return h.do(fiber.MethodPost, "/api/auth/admin-sign-up",
		map[string]string{"email": adminEmail, "password": adminPassword, "name": "Admin"})
}

func (h *apiHarness) login(email string, password string, headers ...string) string {
	h.t.Helper()

	res := h.do(fiber.MethodPost, "/api/auth/login", map[string]string{"email": email, "password": password}, headers...)
	if res.Status != fiber.StatusCreated {
		h.t.Fatalf("login %s = %d %s", email, res.Status, res.Body)
	}

	return res.str(h.t, "accessToken")
}

func (h *apiHarness) admin() string {
	h.t.Helper()

	if res := h.signUp(); res.Status != fiber.StatusCreated {
		h.t.Fatalf("sign-up = %d %s", res.Status, res.Body)
	}

	return h.login(adminEmail, adminPassword)
}

func (h *apiHarness) createUser(token string, email string, name string, password string) string {
	h.t.Helper()

	res := h.do(fiber.MethodPost, "/api/admin/users",
		map[string]any{"email": email, "password": password, "name": name}, bearer(token)...)
	if res.Status != fiber.StatusCreated {
		h.t.Fatalf("create %s = %d %s", email, res.Status, res.Body)
	}

	return res.str(h.t, "id")
}

func (h *apiHarness) apiKey(token string, permissions ...string) string {
	h.t.Helper()

	res := h.do(fiber.MethodPost, "/api/api-keys", map[string]any{"permissions": permissions}, bearer(token)...)
	if res.Status != fiber.StatusCreated {
		h.t.Fatalf("api key %v = %d %s", permissions, res.Status, res.Body)
	}

	return res.str(h.t, "secret")
}

func (h *apiHarness) runJobs() {
	h.t.Helper()

	for _, j := range h.queue.drain() {
		if err := h.handlers.Execute(h.ctx, j.job); err != nil {
			h.t.Fatalf("job %s: %v", j.id, err)
		}
	}
}
