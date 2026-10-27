package maintenance

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/migrations"
)

var (
	sqliteHeader = []byte("SQLite format 3\x00")
	gzipHeader   = []byte{0x1f, 0x8b}
	pathEscaper  = strings.NewReplacer("%", "%25", "?", "%3F", "#", "%23")
)

type Restored struct {
	Source   string
	Database string
	Previous string
	Version  uint
	Sessions int64
}

func Restore(ctx context.Context, database string, backups string, file string, now time.Time) (Restored, error) {
	source, err := restoreSource(backups, file)
	if err != nil {
		return Restored{}, err
	}

	unpacked := database + ".restore"
	defer removeDatabase(unpacked)
	if err = unpack(source, unpacked); err != nil {
		return Restored{}, err
	}
	version, sessions, err := prepareBackup(ctx, unpacked)
	if err != nil {
		return Restored{}, err
	}

	out := Restored{Source: source, Database: database, Version: version, Sessions: sessions}
	info, err := os.Stat(database)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return Restored{}, err
	default:
		if err = checkUnused(ctx, database); err != nil {
			return Restored{}, err
		}
		if err = os.Chmod(unpacked, info.Mode().Perm()); err != nil {
			return Restored{}, err
		}
		out.Previous = database + ".before-restore-" + now.Format("20060102T150405")
		if err = os.Rename(database, out.Previous); err != nil {
			return Restored{}, err
		}
	}
	if err = os.Rename(unpacked, database); err != nil {
		return Restored{}, err
	}

	return out, nil
}

func restoreSource(backups string, file string) (string, error) {
	_, err := os.Stat(file)
	if errors.Is(err, fs.ErrNotExist) && filepath.Base(file) == file {
		inBackups := filepath.Join(backups, file)
		if _, err = os.Stat(inBackups); err == nil {
			return inBackups, nil
		}
	}
	if err != nil {
		return "", err
	}

	return filepath.Abs(file)
}

func unpack(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	buffered := bufio.NewReader(in)
	var r io.Reader = buffered
	if magic, _ := buffered.Peek(len(gzipHeader)); bytes.Equal(magic, gzipHeader) {
		zr, err := gzip.NewReader(buffered)
		if err != nil {
			return err
		}
		defer zr.Close()
		r = zr
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, r); err != nil {
		_ = out.Close()
		return err
	}
	if err = out.Sync(); err != nil {
		_ = out.Close()
		return err
	}

	return out.Close()
}

func openSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+pathEscaper.Replace(path))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	return db, nil
}

func prepareBackup(ctx context.Context, path string) (uint, int64, error) {
	header := make([]byte, len(sqliteHeader))
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	_, err = io.ReadFull(f, header)
	_ = f.Close()
	if err != nil || !bytes.Equal(header, sqliteHeader) {
		return 0, 0, errs.ErrRestoreNotSQLite
	}

	db, err := openSQLite(path)
	if err != nil {
		return 0, 0, err
	}
	defer db.Close()

	version, err := checkBackup(ctx, db)
	if err != nil {
		return 0, 0, err
	}
	sessions, err := endSessions(ctx, db)
	if err != nil {
		return 0, 0, err
	}

	var mode string
	if err = db.QueryRowContext(ctx, `PRAGMA journal_mode=DELETE`).Scan(&mode); err != nil {
		return 0, 0, err
	}

	return version, sessions, nil
}

func checkBackup(ctx context.Context, db *sql.DB) (uint, error) {
	var result string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&result); err != nil {
		return 0, fmt.Errorf("%w: %w", errs.ErrRestoreCorrupt, err)
	}
	if result != "ok" {
		return 0, fmt.Errorf("%w: %s", errs.ErrRestoreCorrupt, result)
	}

	var version uint
	var dirty bool
	if err := db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty); err != nil {
		return 0, fmt.Errorf("%w: %w", errs.ErrRestoreNotKorda, err)
	}
	known, err := migrationVersions()
	if err != nil {
		return 0, err
	}
	switch {
	case dirty:
		return 0, fmt.Errorf("%w: %d", errs.ErrRestoreDirty, version)
	case version > slices.Max(known):
		return 0, fmt.Errorf("%w: %d, this korda's latest is %d", errs.ErrRestoreNewer, version, slices.Max(known))
	case !slices.Contains(known, version):
		return 0, fmt.Errorf("%w: unknown migration %d", errs.ErrRestoreNotKorda, version)
	}

	return version, nil
}

func endSessions(ctx context.Context, db *sql.DB) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, `DELETE FROM sync_checkpoints`); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM sessions`)
	if err != nil {
		return 0, err
	}
	ended, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return ended, tx.Commit()
}

func migrationVersions() ([]uint, error) {
	entries, err := fs.ReadDir(migrations.FS, migrations.Path)
	if err != nil {
		return nil, err
	}

	var out []uint
	for _, e := range entries {
		prefix, _, ok := strings.Cut(e.Name(), "_")
		if !ok || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		version, err := strconv.ParseUint(prefix, 10, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, uint(version))
	}
	if len(out) == 0 {
		return nil, errs.ErrRestoreNotKorda
	}

	return out, nil
}

func checkUnused(ctx context.Context, path string) error {
	db, err := openSQLite(path)
	if err != nil {
		return err
	}
	defer db.Close()

	var mode string
	if err = db.QueryRowContext(ctx, `PRAGMA journal_mode=DELETE`).Scan(&mode); err != nil {
		return fmt.Errorf("%w: %w", errs.ErrRestoreInUse, err)
	}
	if mode != "delete" {
		return fmt.Errorf("%w: journal mode %s", errs.ErrRestoreInUse, mode)
	}

	return nil
}

func removeDatabase(path string) {
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		_ = os.Remove(path + suffix)
	}
}
