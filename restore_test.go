package main

import (
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tty.fail/mrus/korda/errs"
	logicmaintenance "tty.fail/mrus/korda/logic/root/maintenance"
	"xn--gckvb8fzb.com/glides/runtime"
	"xn--gckvb8fzb.com/glides/services/database"
	"xn--gckvb8fzb.com/glides/services/database/sqlite"
)

func openRuntime(t *testing.T, cfgstr string) *runtime.Runtime {
	t.Helper()

	rt, err := newRuntime(cfgstr, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = prepare(rt, nil); err != nil {
		t.Fatal(err)
	}
	if err = database.From(rt).Startup(); err != nil {
		t.Fatal(err)
	}

	return rt
}

func alterBackup(t *testing.T, src string, dst string, query string) {
	t.Helper()

	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	zr, err := gzip.NewReader(in)
	if err != nil {
		t.Fatal(err)
	}
	plain := dst + ".db"
	out, err := os.Create(plain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = out.ReadFrom(zr); err != nil {
		t.Fatal(err)
	}
	_ = out.Close()

	db, err := sql.Open("sqlite", plain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if err = os.Rename(plain, dst); err != nil {
		t.Fatal(err)
	}
}

func TestRestore(t *testing.T) {
	cfgstr := fileRuntimeConfig(t)
	dir := filepath.Dir(strings.TrimPrefix(cfgstr, "file://"))
	dbPath := filepath.Join(dir, "korda.db")
	backups := filepath.Join(dir, "media", "backups")
	ctx := context.Background()
	now := time.Date(2026, 10, 27, 2, 0, 0, 0, time.Local)

	if err := runCreateUser(t, cfgstr, `{"email":"admin@example.com","name":"Admin","password":"admin-password","admin":true}`); err != nil {
		t.Fatal(err)
	}

	rt := openRuntime(t, cfgstr)
	db := database.From(rt)
	adminID, err := database.Value[string](ctx, db, `SELECT id FROM users`)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UTC()
	if _, err = database.Exec(ctx, db, `INSERT INTO sessions (id, token, user_id, created_at, updated_at, update_id)
		VALUES ('019a0000-0000-7000-8000-000000000001', X'01', $1, $2, $2, '019a0000-0000-7000-8000-000000000002')`, adminID, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err = database.Exec(ctx, db, `INSERT INTO sync_checkpoints (session_id, type, ack, created_at, updated_at, update_id)
		VALUES ('019a0000-0000-7000-8000-000000000001', 'AssetV1', 'ack', $1, $1, '019a0000-0000-7000-8000-000000000003')`, stamp); err != nil {
		t.Fatal(err)
	}
	backup, err := logicmaintenance.CreateBackup(ctx, rt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = logicmaintenance.Restore(ctx, dbPath, backups, filepath.Base(backup), now); !errors.Is(err, errs.ErrRestoreInUse) {
		t.Errorf("a restore while korda runs = %v", err)
	}
	if err = database.From(rt).Shutdown(); err != nil {
		t.Fatal(err)
	}

	if err = runCreateUser(t, cfgstr, `{"email":"vera@example.com","name":"Vera","password":"vera-password"}`); err != nil {
		t.Fatal(err)
	}
	if list := usersOf(t, cfgstr); len(list) != 2 {
		t.Fatalf("users before the restore = %d", len(list))
	}

	broken := filepath.Join(t.TempDir(), "broken")
	for name, content := range map[string]string{"plain.sql": "SELECT 1;", "empty.sql": ""} {
		path := filepath.Join(broken, name)
		if err = os.MkdirAll(broken, 0o750); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err = logicmaintenance.Restore(ctx, dbPath, backups, path, now); !errors.Is(err, errs.ErrRestoreNotSQLite) {
			t.Errorf("%s = %v", name, err)
		}
	}
	newer := filepath.Join(broken, "newer.sql")
	alterBackup(t, backup, newer, `UPDATE schema_migrations SET version = version + 1`)
	if _, err = logicmaintenance.Restore(ctx, dbPath, backups, newer, now); !errors.Is(err, errs.ErrRestoreNewer) {
		t.Errorf("a newer backup = %v", err)
	}
	dirty := filepath.Join(broken, "dirty.sql")
	alterBackup(t, backup, dirty, `UPDATE schema_migrations SET dirty = 1`)
	if _, err = logicmaintenance.Restore(ctx, dbPath, backups, dirty, now); !errors.Is(err, errs.ErrRestoreDirty) {
		t.Errorf("a dirty backup = %v", err)
	}
	other := filepath.Join(broken, "other.sql")
	alterBackup(t, backup, other, `DROP TABLE schema_migrations`)
	if _, err = logicmaintenance.Restore(ctx, dbPath, backups, other, now); !errors.Is(err, errs.ErrRestoreNotKorda) {
		t.Errorf("a database of another application = %v", err)
	}
	if _, err = logicmaintenance.Restore(ctx, dbPath, backups, "missing.sql.gz", now); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a missing backup = %v", err)
	}
	if leftovers, _ := filepath.Glob(dbPath + ".*"); len(leftovers) != 0 {
		t.Errorf("leftovers after the failed restores = %v", leftovers)
	}

	restored, err := logicmaintenance.Restore(ctx, dbPath, backups, filepath.Base(backup), now)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Source != backup || restored.Database != dbPath || restored.Previous != dbPath+".before-restore-20261027T020000" || restored.Sessions != 1 {
		t.Errorf("restored = %+v", restored)
	}
	if list := usersOf(t, cfgstr); len(list) != 1 || list[0].Email != "admin@example.com" {
		t.Errorf("users after the restore = %v", list)
	}
	plain, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	var sessions, checkpoints int
	if err = plain.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM sessions), (SELECT COUNT(*) FROM sync_checkpoints)`).Scan(&sessions, &checkpoints); err != nil || sessions != 0 || checkpoints != 0 {
		t.Errorf("after the restore: %d sessions, %d checkpoints, %v", sessions, checkpoints, err)
	}
	_ = plain.Close()
	if _, err = os.Stat(restored.Previous); err != nil {
		t.Errorf("previous = %v", err)
	}
	if leftovers, _ := filepath.Glob(dbPath + ".restore*"); len(leftovers) != 0 {
		t.Errorf("leftovers after the restore = %v", leftovers)
	}
}
