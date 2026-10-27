package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/hibiken/asynq"
)

func backupNames(t *testing.T, h *apiHarness, headers ...string) []string {
	t.Helper()

	body := h.do(fiber.MethodGet, "/api/admin/database-backups", nil, headers...).object(t)
	names := []string{}
	for _, item := range body["backups"].([]any) {
		b := item.(map[string]any)
		if b["timezone"] == "" || b["filesize"].(float64) <= 0 {
			t.Errorf("backup = %v", b)
		}
		names = append(names, b["filename"].(string))
	}

	return names
}

func TestDatabaseBackups(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)

	dir := filepath.Join(h.dir, "media", "backups")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	write := func(name string, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for day := 1; day <= 15; day++ {
		write(fmt.Sprintf("korda-db-backup-202601%02dT020000-v0.1.0-sqlite3.50.0.sql.gz", day), "old")
	}
	write("korda-db-backup-20260101T020000-v0.1.0-sqlite3.50.0.sql.gz.tmp", "failed")
	write("immich-db-backup-20260101T020000-v2.5.0-pg17.sql.gz", "immich")
	write("notes.txt", "notes")

	h.do(fiber.MethodPost, "/api/jobs", `{"name":"backup-database"}`, owner...).expect(t, 204, "")
	queued := h.queue.drain()
	if len(queued) != 1 || queued[0].queue() != "backupDatabase" {
		t.Fatalf("queued = %v", queued)
	}
	retries := -1
	for _, o := range queued[0].opts {
		if o.Type() == asynq.MaxRetryOpt {
			retries = o.Value().(int)
		}
	}
	if retries != 0 {
		t.Errorf("max retry = %d", retries)
	}
	if err := h.handlers.Execute(h.ctx, queued[0].job); err != nil {
		t.Fatal(err)
	}

	names := backupNames(t, h, owner...)
	if len(names) != 15 || !regexp.MustCompile(`^korda-db-backup-\d{8}T\d{6}-vdev-sqlite3\.\d+\.\d+\.sql\.gz$`).MatchString(names[0]) ||
		names[1] != "korda-db-backup-20260115T020000-v0.1.0-sqlite3.50.0.sql.gz" ||
		names[13] != "korda-db-backup-20260103T020000-v0.1.0-sqlite3.50.0.sql.gz" ||
		names[14] != "immich-db-backup-20260101T020000-v2.5.0-pg17.sql.gz" {
		t.Fatalf("backups = %v", names)
	}
	for _, gone := range []string{"korda-db-backup-20260101T020000-v0.1.0-sqlite3.50.0.sql.gz", "korda-db-backup-20260101T020000-v0.1.0-sqlite3.50.0.sql.gz.tmp"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
			t.Errorf("%s = %v", gone, err)
		}
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if err != nil || len(leftovers) != 0 {
		t.Errorf("leftovers = %v %v", leftovers, err)
	}

	res := h.do(fiber.MethodGet, "/api/admin/database-backups/"+names[0], nil, owner...)
	if res.Status != 200 || res.Header.Get(fiber.HeaderContentType) != "application/gzip" ||
		res.Header.Get(fiber.HeaderCacheControl) != "private, no-cache, no-transform" ||
		res.Header.Get(fiber.HeaderContentDisposition) != "inline; filename*=UTF-8''"+names[0] {
		t.Fatalf("download = %d %v", res.Status, res.Header)
	}
	zr, err := gzip.NewReader(bytes.NewReader(res.Body))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(t.TempDir(), "copy.db")
	if err = os.WriteFile(copied, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", copied)
	if err != nil {
		t.Fatal(err)
	}
	var users int
	if err = db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM users`).Scan(&users); err != nil || users != 1 {
		t.Errorf("users in the backup = %d %v", users, err)
	}
	_ = db.Close()

	h.do(fiber.MethodGet, "/api/admin/database-backups/notes.txt", nil, owner...).expect(t, 404, `{"message":"Not Found"}`)
	h.do(fiber.MethodGet, "/api/admin/database-backups/missing.sql", nil, owner...).expect(t, 404, `{"message":"Not Found"}`)

	h.multipart("/api/admin/database-backups/upload", []part{{field: "file", filename: "mine.sql.gz", content: []byte("first")}}, owner...).
		expect(t, 201, "")
	h.multipart("/api/admin/database-backups/upload", []part{{field: "file", filename: "mine.sql.gz", content: []byte("second")}}, owner...).
		expect(t, 201, "")
	h.do(fiber.MethodGet, "/api/admin/database-backups/uploaded-mine.sql.gz", nil, owner...).expect(t, 200, "second")
	if names = backupNames(t, h, owner...); len(names) != 16 || names[15] != "uploaded-mine.sql.gz" {
		t.Errorf("backups after the upload = %v", names)
	}

	h.do(fiber.MethodDelete, "/api/admin/database-backups", `{"backups":["uploaded-mine.sql.gz","`+names[14]+`"]}`, owner...).expect(t, 200, "")
	if names = backupNames(t, h, owner...); len(names) != 14 || slices.Contains(names, "uploaded-mine.sql.gz") {
		t.Errorf("backups after the delete = %v", names)
	}

	h.do(fiber.MethodPut, "/api/jobs/backupDatabase", `{"command":"start"}`, owner...).
		expect(t, 200, `{"queueStatus":{"isPaused":false,"isActive":false},"jobCounts":{"active":0,"completed":0,"failed":0,"delayed":0,"waiting":1,"paused":0}}`)
	if ids := h.queue.ids(); len(ids) != 1 || !strings.HasPrefix(ids[0], "backup-database:") {
		t.Errorf("queued = %v", ids)
	}
}

func TestDatabaseBackupFailure(t *testing.T) {
	h := newHarness(t)
	admin := h.admin()
	owner := bearer(admin)

	if err := os.MkdirAll(filepath.Join(h.dir, "media"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.dir, "media", "backups"), []byte("not a folder"), 0o600); err != nil {
		t.Fatal(err)
	}

	h.do(fiber.MethodPost, "/api/jobs", `{"name":"backup-database"}`, owner...).expect(t, 204, "")
	queued := h.queue.drain()
	if len(queued) != 1 {
		t.Fatalf("queued = %v", queued)
	}
	if err := h.handlers.Execute(h.ctx, queued[0].job); err == nil {
		t.Fatal("the backup into a file instead of a folder succeeded")
	}

	list := h.do(fiber.MethodGet, "/api/notifications", nil, owner...).array(t)
	if len(list) != 1 {
		t.Fatalf("notifications = %v", list)
	}
	n := list[0].(map[string]any)
	if n["type"] != "JobFailed" || n["level"] != "error" || n["title"] != "Job Failed" ||
		!strings.HasPrefix(n["description"].(string), "Job DatabaseBackup failed with error: ") {
		t.Errorf("notification = %v", n)
	}
}
