package queue

type Name string

const (
	ThumbnailGeneration      Name = "thumbnailGeneration"
	MetadataExtraction       Name = "metadataExtraction"
	VideoConversion          Name = "videoConversion"
	FaceDetection            Name = "faceDetection"
	FacialRecognition        Name = "facialRecognition"
	SmartSearch              Name = "smartSearch"
	DuplicateDetection       Name = "duplicateDetection"
	BackgroundTask           Name = "backgroundTask"
	StorageTemplateMigration Name = "storageTemplateMigration"
	Migration                Name = "migration"
	Search                   Name = "search"
	Sidecar                  Name = "sidecar"
	Library                  Name = "library"
	Notifications            Name = "notifications"
	BackupDatabase           Name = "backupDatabase"
	Ocr                      Name = "ocr"
	Workflow                 Name = "workflow"
	IntegrityCheck           Name = "integrityCheck"
	Editor                   Name = "editor"
)

var Names = []Name{
	ThumbnailGeneration, MetadataExtraction, VideoConversion, FaceDetection, FacialRecognition, SmartSearch,
	DuplicateDetection, BackgroundTask, StorageTemplateMigration, Migration, Search, Sidecar, Library, Notifications,
	BackupDatabase, Ocr, Workflow, IntegrityCheck, Editor,
}

func (Name) Enum() []string {
	out := make([]string, 0, len(Names))
	for _, n := range Names {
		out = append(out, string(n))
	}

	return out
}

type Status string

const (
	StatusActive    Status = "active"
	StatusFailed    Status = "failed"
	StatusCompleted Status = "completed"
	StatusDelayed   Status = "delayed"
	StatusWaiting   Status = "waiting"
	StatusPaused    Status = "paused"
)

var Statuses = []Status{StatusActive, StatusFailed, StatusCompleted, StatusDelayed, StatusWaiting, StatusPaused}

func (Status) Enum() []string {
	out := make([]string, 0, len(Statuses))
	for _, s := range Statuses {
		out = append(out, string(s))
	}

	return out
}

type Command string

const (
	CommandStart       Command = "start"
	CommandPause       Command = "pause"
	CommandResume      Command = "resume"
	CommandEmpty       Command = "empty"
	CommandClearFailed Command = "clear-failed"
)

func (Command) Enum() []string {
	return []string{string(CommandStart), string(CommandPause), string(CommandResume), string(CommandEmpty),
		string(CommandClearFailed)}
}

type ManualJob string

const (
	ManualPersonCleanup                    ManualJob = "person-cleanup"
	ManualTagCleanup                       ManualJob = "tag-cleanup"
	ManualUserCleanup                      ManualJob = "user-cleanup"
	ManualMemoryCleanup                    ManualJob = "memory-cleanup"
	ManualMemoryCreate                     ManualJob = "memory-create"
	ManualBackupDatabase                   ManualJob = "backup-database"
	ManualIntegrityMissingFiles            ManualJob = "integrity-missing-files"
	ManualIntegrityUntrackedFiles          ManualJob = "integrity-untracked-files"
	ManualIntegrityChecksumFiles           ManualJob = "integrity-checksum-mismatch"
	ManualIntegrityMissingFilesRefresh     ManualJob = "integrity-missing-files-refresh"
	ManualIntegrityUntrackedFilesRefresh   ManualJob = "integrity-untracked-files-refresh"
	ManualIntegrityChecksumFilesRefresh    ManualJob = "integrity-checksum-mismatch-refresh"
	ManualIntegrityMissingFilesDeleteAll   ManualJob = "integrity-missing-files-delete-all"
	ManualIntegrityUntrackedFilesDeleteAll ManualJob = "integrity-untracked-files-delete-all"
	ManualIntegrityChecksumFilesDeleteAll  ManualJob = "integrity-checksum-mismatch-delete-all"
)

func (ManualJob) Enum() []string {
	return []string{string(ManualPersonCleanup), string(ManualTagCleanup), string(ManualUserCleanup),
		string(ManualMemoryCleanup), string(ManualMemoryCreate), string(ManualBackupDatabase),
		string(ManualIntegrityMissingFiles), string(ManualIntegrityUntrackedFiles), string(ManualIntegrityChecksumFiles),
		string(ManualIntegrityMissingFilesRefresh), string(ManualIntegrityUntrackedFilesRefresh),
		string(ManualIntegrityChecksumFilesRefresh), string(ManualIntegrityMissingFilesDeleteAll),
		string(ManualIntegrityUntrackedFilesDeleteAll), string(ManualIntegrityChecksumFilesDeleteAll)}
}

type Statistics struct {
	Active    int
	Completed int
	Failed    int
	Delayed   int
	Waiting   int
	Paused    int
}

type Queue struct {
	Name       Name
	IsPaused   bool
	Statistics Statistics
}
