package maintenance

import (
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/event"
	"tty.fail/mrus/korda/models/notification"
	"tty.fail/mrus/korda/services/mediaroot"
	reponotification "tty.fail/mrus/korda/services/repositories/notification"
	"xn--gckvb8fzb.com/glides/runtime"
	"xn--gckvb8fzb.com/glides/services/database"
)

const (
	BackupPrefix   = "korda-db-backup-"
	UploadedPrefix = "uploaded-"
	JobName        = "DatabaseBackup"
)

var (
	routineBackup = regexp.MustCompile(`^korda-db-backup-\d{8}T\d{6}-v.*-sqlite.*\.sql\.gz$`)
	failedBackup  = regexp.MustCompile(`^korda-db-backup-.*\.tmp$`)
	unsafeName    = regexp.MustCompile(`[^a-zA-Z0-9_.\-]`)
)

type Backup struct {
	Filename string
	Filesize int64
	Timezone string
}

func backupsDir(rt *runtime.Runtime) (string, error) {
	dir, err := gh.MediaRoot(rt).Abs(string(mediaroot.Backups))
	if err != nil {
		return "", err
	}

	return dir, os.MkdirAll(dir, 0o750)
}

func zoneName() string {
	if tz := strings.TrimPrefix(os.Getenv("TZ"), ":"); tz != "" {
		return tz
	}
	if target, err := os.Readlink("/etc/localtime"); err == nil {
		if _, zone, ok := strings.Cut(target, "zoneinfo/"); ok {
			return zone
		}
	}

	return "UTC"
}

func BackupName(at time.Time, version string, sqliteVersion string) string {
	version = unsafeName.ReplaceAllString(strings.TrimPrefix(version, "v"), "_")
	if version == "" {
		version = "dev"
	}

	return BackupPrefix + at.Format("20060102T150405") + "-v" + version + "-sqlite" +
		unsafeName.ReplaceAllString(sqliteVersion, "_") + ".sql.gz"
}

func CreateBackup(ctx context.Context, rt *runtime.Runtime) (string, error) {
	dir, err := backupsDir(rt)
	if err != nil {
		return "", err
	}

	db := database.From(rt)
	sqliteVersion, err := database.Value[string](ctx, db, `SELECT sqlite_version()`)
	if err != nil {
		return "", err
	}
	version, _, _, _ := rt.GetBuild()
	name := BackupName(time.Now(), version, sqliteVersion)
	final := filepath.Join(dir, name)
	copied := final + ".db.tmp"
	compressed := final + ".tmp"

	fail := func(err error) (string, error) {
		_ = os.Remove(copied)
		_ = os.Remove(compressed)
		return "", err
	}
	if _, err = database.Exec(ctx, db, `VACUUM INTO $1`, copied); err != nil {
		return fail(err)
	}
	if err = gzipFile(copied, compressed); err != nil {
		return fail(err)
	}
	if err = os.Rename(compressed, final); err != nil {
		return fail(err)
	}
	if err = os.Remove(copied); err != nil {
		return final, err
	}

	rt.Info("backups", "created", "file", name)
	return final, nil
}

func gzipFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(out)
	if _, err = io.Copy(zw, in); err != nil {
		_ = out.Close()
		return err
	}
	if err = zw.Close(); err != nil {
		_ = out.Close()
		return err
	}
	if err = out.Sync(); err != nil {
		_ = out.Close()
		return err
	}

	return out.Close()
}

func CleanupBackups(rt *runtime.Runtime, keep int) error {
	dir, err := backupsDir(rt)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var routine, remove []string
	for _, e := range entries {
		switch {
		case routineBackup.MatchString(e.Name()):
			routine = append(routine, e.Name())
		case failedBackup.MatchString(e.Name()):
			remove = append(remove, e.Name())
		}
	}
	slices.Sort(routine)
	slices.Reverse(routine)
	if len(routine) > keep {
		remove = append(remove, routine[keep:]...)
	}
	for _, name := range remove {
		if err = os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	return nil
}

func RunBackup(ctx context.Context, rt *runtime.Runtime) error {
	err := runBackup(ctx, rt)
	if err != nil {
		rt.Error("backups", "failed", "error", err)
		if nerr := notifyFailure(ctx, rt, err); nerr != nil {
			rt.Error("backups", "notify", "error", nerr)
		}
	}

	return err
}

func runBackup(ctx context.Context, rt *runtime.Runtime) error {
	cfg, err := gh.BackupConfigFrom(rt.Config())
	if err != nil {
		return err
	}
	if _, err = CreateBackup(ctx, rt); err != nil {
		return err
	}

	return CleanupBackups(rt, cfg.KeepLastAmount)
}

func notifyFailure(ctx context.Context, rt *runtime.Runtime, cause error) error {
	repos := gh.Repositories(rt)
	admin, err := repos.User.GetAdmin(ctx)
	if errors.Is(err, errs.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	n, err := repos.Notification.Create(ctx, reponotification.Create{
		UserID:      admin.ID,
		Level:       notification.LevelError,
		Type:        notification.TypeJobFailed,
		Title:       "Job Failed",
		Description: sql.Null[string]{V: fmt.Sprintf("Job %s failed with error: %s", JobName, cause), Valid: true},
	}, time.Now().UTC())
	if err != nil {
		return err
	}
	gh.Emit(ctx, rt, event.NotificationCreated{Notification: n})

	return nil
}

func ListBackups(rt *runtime.Runtime) ([]Backup, error) {
	dir, err := backupsDir(rt)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var routine, uploaded []string
	for _, e := range entries {
		switch {
		case e.IsDir() || !ValidBackupName(e.Name()):
		case strings.HasPrefix(e.Name(), UploadedPrefix):
			uploaded = append(uploaded, e.Name())
		default:
			routine = append(routine, e.Name())
		}
	}
	slices.Sort(routine)
	slices.Reverse(routine)
	slices.Sort(uploaded)
	slices.Reverse(uploaded)

	zone := zoneName()
	out := []Backup{}
	for _, name := range append(routine, uploaded...) {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		out = append(out, Backup{Filename: name, Filesize: info.Size(), Timezone: zone})
	}

	return out, nil
}

func BackupPath(rt *runtime.Runtime, name string) (string, error) {
	if !ValidBackupName(name) {
		return "", errs.ErrInvalidBackupName
	}
	dir, err := backupsDir(rt)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, name), nil
}

func DeleteBackups(rt *runtime.Runtime, in *DeleteBackupsInput) error {
	for _, name := range in.Backups {
		if !ValidBackupName(name) {
			return errs.ErrInvalidBackupName
		}
	}
	dir, err := backupsDir(rt)
	if err != nil {
		return err
	}
	for _, name := range in.Backups {
		err = os.Remove(filepath.Join(dir, name))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			rt.Warn("backups", "delete", "missing", name)
		case err != nil:
			return err
		}
	}

	return nil
}

func UploadBackup(rt *runtime.Runtime, name string, src io.Reader) error {
	name = filepath.Base(name)
	if !ValidBackupName(name) {
		return errs.ErrInvalidBackupName
	}
	dir, err := backupsDir(rt)
	if err != nil {
		return err
	}

	final := filepath.Join(dir, UploadedPrefix+name)
	tmp := final + ".upload.tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, src); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err = out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, final)
}
