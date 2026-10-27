package maintenance

import (
	"os"
	"path/filepath"
	"regexp"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/refine"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Action string

func (Action) Enum() []string {
	return []string{"start", "end", "select_database_restore", "restore_database"}
}

type SetModeInput struct {
	Action                Action  `json:"action"`
	RestoreBackupFilename *string `json:"restoreBackupFilename"`
}

func (SetModeInput) Refine(in refine.Fields) []refine.Issue {
	action, _ := in.Get("action")
	name, _ := in.Get("restoreBackupFilename")
	if s, _ := name.(string); action == "restore_database" && s == "" {
		return []refine.Issue{refine.At([]any{"restoreBackupFilename"}, "Backup filename is required when action is restore_database")}
	}

	return nil
}

type LoginInput struct {
	Token *string `json:"token"`
}

type ReportType string

func (ReportType) Enum() []string {
	return []string{"untracked_file", "missing_file", "checksum_mismatch"}
}

type ReportInput struct {
	Type   ReportType `json:"type"`
	Cursor *string    `json:"cursor"`
	Limit  *int64     `json:"limit" v:"coerce,positive"`
}

type ReportTypeParams struct {
	Type ReportType `json:"type"`
}

type ReportIDParams struct {
	ID string `json:"id" v:"uuidv7"`
}

type FilenameParams struct {
	Filename string `json:"filename" v:"regex=filename"`
}

type DeleteBackupsInput struct {
	Backups []string `json:"backups"`
}

type Folder struct {
	Folder   string
	Readable bool
	Writable bool
	Files    int
}

var (
	folders    = []string{"encoded-video", "library", "upload", "profile", "thumbs", "backups"}
	backupName = regexp.MustCompile(`^[a-zA-Z0-9_\-.]+\.sql(\.gz)?$`)
)

func ValidBackupName(name string) bool {
	return backupName.MatchString(name)
}

func SetMode(in *SetModeInput) error {
	if in.Action == "end" {
		return nil
	}

	return errs.UnsupportedMaintenance
}

func Login(in *LoginInput) error {
	return errs.ErrNotInMaintenance
}

func DetectInstall(rt *runtime.Runtime) []Folder {
	root := gh.MediaRoot(rt).Path()
	out := make([]Folder, 0, len(folders))
	for _, name := range folders {
		f := Folder{Folder: name}
		entries, _ := os.ReadDir(filepath.Join(root, name))
		for _, e := range entries {
			if e.Name() == ".immich" {
				continue
			}
			if name == "backups" && !ValidBackupName(e.Name()) {
				continue
			}
			f.Files++
		}
		out = append(out, f)
	}

	return out
}
