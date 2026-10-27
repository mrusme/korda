package jobs

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"tty.fail/mrus/korda/models/queue"
	"xn--gckvb8fzb.com/glides/models/asyncjob"
	"xn--gckvb8fzb.com/glides/services/config"
	"xn--gckvb8fzb.com/glides/services/dispatch"
)

const TargetID string = "jobs"

const Run asyncjob.JobSubType = "run"

const (
	DeleteUser  asyncjob.JobType = "delete-user"
	Metadata    asyncjob.JobType = "metadata"
	Thumbnails  asyncjob.JobType = "thumbnails"
	Transcode   asyncjob.JobType = "transcode"
	DeleteAsset asyncjob.JobType = "delete-asset"
	EmptyTrash  asyncjob.JobType = "empty-trash"

	NotifyAlbumInvite asyncjob.JobType = "notify-album-invite"
	NotifyAlbumUpdate asyncjob.JobType = "notify-album-update"

	Edit asyncjob.JobType = "edit"

	ThumbnailsAll asyncjob.JobType = "thumbnails-all"
	MetadataAll   asyncjob.JobType = "metadata-all"
	TranscodeAll  asyncjob.JobType = "transcode-all"

	UserDeleteCheck asyncjob.JobType = "user-delete-check"
	MemoryCleanup   asyncjob.JobType = "memory-cleanup"
	MemoryGenerate  asyncjob.JobType = "memory-generate"
	TagCleanup      asyncjob.JobType = "tag-cleanup"
	BackupDatabase  asyncjob.JobType = "backup-database"

	registerQueue asyncjob.JobType = "register-queue"
)

const (
	QueueThumbnails    = "thumbnailGeneration"
	QueueMetadata      = "metadataExtraction"
	QueueVideo         = "videoConversion"
	QueueBackground    = "backgroundTask"
	QueueEditor        = "editor"
	QueueNotifications = "notifications"
	QueueBackup        = "backupDatabase"
)

var queueOf = map[asyncjob.JobType]string{
	DeleteUser:        QueueBackground,
	Metadata:          QueueMetadata,
	Thumbnails:        QueueThumbnails,
	Transcode:         QueueVideo,
	DeleteAsset:       QueueBackground,
	EmptyTrash:        QueueBackground,
	NotifyAlbumInvite: QueueNotifications,
	NotifyAlbumUpdate: QueueNotifications,
	Edit:              QueueEditor,
	ThumbnailsAll:     QueueThumbnails,
	MetadataAll:       QueueMetadata,
	TranscodeAll:      QueueVideo,
	UserDeleteCheck:   QueueBackground,
	MemoryCleanup:     QueueBackground,
	MemoryGenerate:    QueueBackground,
	TagCleanup:        QueueBackground,
	BackupDatabase:    QueueBackup,
}

var nameOf = map[asyncjob.JobType]string{
	DeleteUser:        "UserDelete",
	Metadata:          "AssetExtractMetadata",
	Thumbnails:        "AssetGenerateThumbnails",
	Transcode:         "AssetEncodeVideo",
	DeleteAsset:       "AssetDelete",
	EmptyTrash:        "AssetEmptyTrash",
	NotifyAlbumInvite: "NotifyAlbumInvite",
	NotifyAlbumUpdate: "NotifyAlbumUpdate",
	Edit:              "AssetEditThumbnailGeneration",
	ThumbnailsAll:     "AssetGenerateThumbnailsQueueAll",
	MetadataAll:       "AssetExtractMetadataQueueAll",
	TranscodeAll:      "AssetEncodeVideoQueueAll",
	UserDeleteCheck:   "UserDeleteCheck",
	MemoryCleanup:     "MemoryCleanup",
	MemoryGenerate:    "MemoryGenerate",
	TagCleanup:        "TagCleanup",
	BackupDatabase:    "DatabaseBackup",
}

const AlbumUpdateDelay = 5 * time.Minute

func Queue(jobType asyncjob.JobType) string {
	if q, ok := queueOf[jobType]; ok {
		return q
	}

	return dispatch.QueueDefault
}

func JobName(jobType asyncjob.JobType) string {
	if name, ok := nameOf[jobType]; ok {
		return name
	}

	return string(jobType)
}

func Queues() map[string]int {
	out := map[string]int{dispatch.QueueDefault: 1}
	for _, q := range queueOf {
		out[q] = 1
	}

	return out
}

const (
	mediaTimeout     = 30 * time.Minute
	transcodeTimeout = 2 * time.Hour
)

type DeleteUserPayload struct {
	UserID uuid.UUID `json:"userId"`
	Force  bool      `json:"force"`
}

type Source string

const (
	SourceUpload       Source = "upload"
	SourceCopy         Source = "copy"
	SourceSidecarWrite Source = "sidecar-write"
)

type AssetPayload struct {
	AssetID uuid.UUID `json:"assetId"`
	Force   bool      `json:"force,omitempty"`
	Source  Source    `json:"source,omitempty"`
	Notify  bool      `json:"notify,omitempty"`
}

type DeleteAssetPayload struct {
	AssetID      uuid.UUID `json:"assetId"`
	DeleteOnDisk bool      `json:"deleteOnDisk"`
}

type EmptyTrashPayload struct{}

type AlbumInvitePayload struct {
	AlbumID     uuid.UUID `json:"albumId"`
	RecipientID uuid.UUID `json:"recipientId"`
	SenderName  string    `json:"senderName"`
}

type AlbumUpdatePayload struct {
	AlbumID     uuid.UUID `json:"albumId"`
	RecipientID uuid.UUID `json:"recipientId"`
}

type ForcePayload struct {
	Force *bool `json:"force,omitempty"`
}

type ManualPayload struct{}

type Task struct {
	ID  string
	Job asyncjob.AsyncJob
}

type Enqueuer interface {
	JobUnique(j *asyncjob.AsyncJob, id string, opts ...asynq.Option) error
	Remove(queue string, id string) error
	Stats(queue string) (Stats, error)
	Pause(queue string) error
	Resume(queue string) error
	Empty(queue string) error
	ClearFailed(queue string) error
	Tasks(name string, statuses []queue.Status) ([]Task, error)
}

type Stats struct {
	Active    int
	Completed int
	Failed    int
	Delayed   int
	Waiting   int
	Paused    int
	IsPaused  bool
}

type Dispatcher struct {
	dispatch  *dispatch.Dispatch
	inspector *asynq.Inspector
}

func NewDispatcher(d *dispatch.Dispatch, cfg config.Redis) *Dispatcher {
	return &Dispatcher{dispatch: d, inspector: asynq.NewInspector(dispatch.RedisConnOpt(cfg))}
}

func (d *Dispatcher) JobUnique(j *asyncjob.AsyncJob, id string, opts ...asynq.Option) error {
	return d.dispatch.JobUnique(j, id, opts...)
}

func (d *Dispatcher) Remove(queue string, id string) error {
	info, err := d.inspector.GetTaskInfo(queue, id)
	switch {
	case errors.Is(err, asynq.ErrTaskNotFound), errors.Is(err, asynq.ErrQueueNotFound):
		return nil
	case err != nil:
		return err
	case info.State == asynq.TaskStateActive:
		return nil
	}

	err = d.inspector.DeleteTask(queue, id)
	if errors.Is(err, asynq.ErrTaskNotFound) || errors.Is(err, asynq.ErrQueueNotFound) {
		return nil
	}

	return err
}

func (d *Dispatcher) exists(queue string) (bool, error) {
	queues, err := d.inspector.Queues()
	if err != nil {
		return false, err
	}

	return slices.Contains(queues, queue), nil
}

func (d *Dispatcher) Stats(queue string) (Stats, error) {
	if ok, err := d.exists(queue); err != nil || !ok {
		return Stats{}, err
	}

	info, err := d.inspector.GetQueueInfo(queue)
	if err != nil {
		return Stats{}, err
	}

	st := Stats{
		Active:    info.Active,
		Completed: info.Completed,
		Failed:    info.Archived,
		Delayed:   info.Scheduled + info.Retry,
		IsPaused:  info.Paused,
	}
	if info.Paused {
		st.Paused = info.Pending
	} else {
		st.Waiting = info.Pending
	}

	return st, nil
}

func (d *Dispatcher) register(queue string) error {
	if ok, err := d.exists(queue); err != nil || ok {
		return err
	}

	const id = "register"
	if err := d.dispatch.Job(asyncjob.New(TargetID, registerQueue, Run),
		asynq.Queue(queue), asynq.TaskID(id), asynq.ProcessIn(24*time.Hour)); err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
		return err
	}

	return d.Remove(queue, id)
}

func (d *Dispatcher) Pause(queue string) error {
	if err := d.register(queue); err != nil {
		return err
	}
	if st, err := d.Stats(queue); err != nil || st.IsPaused {
		return err
	}

	return d.inspector.PauseQueue(queue)
}

func (d *Dispatcher) Resume(queue string) error {
	if st, err := d.Stats(queue); err != nil || !st.IsPaused {
		return err
	}

	return d.inspector.UnpauseQueue(queue)
}

func (d *Dispatcher) Empty(queue string) error {
	if ok, err := d.exists(queue); err != nil || !ok {
		return err
	}

	_, err := d.inspector.DeleteAllPendingTasks(queue)
	return err
}

func (d *Dispatcher) ClearFailed(queue string) error {
	if ok, err := d.exists(queue); err != nil || !ok {
		return err
	}

	_, err := d.inspector.DeleteAllArchivedTasks(queue)
	return err
}

const listLimit = 1001

func (d *Dispatcher) Tasks(name string, statuses []queue.Status) ([]Task, error) {
	st, err := d.Stats(name)
	if err != nil {
		return nil, err
	}

	var out []Task
	for _, status := range statuses {
		var lists []func(string, ...asynq.ListOption) ([]*asynq.TaskInfo, error)
		switch status {
		case queue.StatusActive:
			lists = append(lists, d.inspector.ListActiveTasks)
		case queue.StatusWaiting:
			if !st.IsPaused {
				lists = append(lists, d.inspector.ListPendingTasks)
			}
		case queue.StatusPaused:
			if st.IsPaused {
				lists = append(lists, d.inspector.ListPendingTasks)
			}
		case queue.StatusDelayed:
			lists = append(lists, d.inspector.ListScheduledTasks, d.inspector.ListRetryTasks)
		case queue.StatusFailed:
			lists = append(lists, d.inspector.ListArchivedTasks)
		case queue.StatusCompleted:
			lists = append(lists, d.inspector.ListCompletedTasks)
		}

		var group []Task
		for _, list := range lists {
			infos, err := list(name, asynq.PageSize(listLimit))
			if errors.Is(err, asynq.ErrQueueNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			for _, info := range infos {
				var j asyncjob.AsyncJob
				if err = json.Unmarshal(info.Payload, &j); err != nil {
					return nil, err
				}
				group = append(group, Task{ID: info.ID, Job: j})
			}
		}
		out = append(out, Newest(group)...)
	}

	return out, nil
}

func Newest(tasks []Task) []Task {
	slices.SortStableFunc(tasks, func(a, b Task) int { return bytes.Compare(b.Job.ID[:], a.Job.ID[:]) })
	return tasks
}

func Created(j asyncjob.AsyncJob) int64 {
	sec, nsec := j.ID.Time().UnixTime()
	return sec*1000 + nsec/int64(time.Millisecond)
}

func (d *Dispatcher) Close() error {
	return d.inspector.Close()
}

type Jobs struct {
	queue Enqueuer
}

func New(queue Enqueuer) (*Jobs, error) {
	j := new(Jobs)
	j.queue = queue

	return j, nil
}

func (j *Jobs) Startup() error {
	return nil
}

func (j *Jobs) Shutdown() error {
	return nil
}

func (j *Jobs) enqueue(jobType asyncjob.JobType, id string, payload any, opts ...asynq.Option) error {
	job := asyncjob.New(TargetID, jobType, Run)
	if err := job.SetPayload(payload); err != nil {
		return err
	}

	return j.queue.JobUnique(job, string(jobType)+":"+id, append([]asynq.Option{asynq.Queue(Queue(jobType))}, opts...)...)
}

func (j *Jobs) NotifyAlbumInvite(albumID uuid.UUID, recipientID uuid.UUID, senderName string) error {
	return j.enqueue(NotifyAlbumInvite, uuid.Must(uuid.NewV7()).String(),
		AlbumInvitePayload{AlbumID: albumID, RecipientID: recipientID, SenderName: senderName})
}

func (j *Jobs) NotifyAlbumUpdate(albumID uuid.UUID, recipientID uuid.UUID) error {
	id := albumID.String() + "/" + recipientID.String()
	if err := j.queue.Remove(Queue(NotifyAlbumUpdate), string(NotifyAlbumUpdate)+":"+id); err != nil {
		return err
	}

	return j.enqueue(NotifyAlbumUpdate, id, AlbumUpdatePayload{AlbumID: albumID, RecipientID: recipientID},
		asynq.ProcessIn(AlbumUpdateDelay))
}

func (j *Jobs) DeleteUser(userID uuid.UUID, force bool) error {
	return j.enqueue(DeleteUser, userID.String(), DeleteUserPayload{UserID: userID, Force: force})
}

func (j *Jobs) Metadata(p AssetPayload) error {
	return j.enqueue(Metadata, p.AssetID.String(), p, asynq.Timeout(mediaTimeout))
}

func (j *Jobs) Thumbnails(p AssetPayload) error {
	return j.enqueue(Thumbnails, p.AssetID.String(), p, asynq.Timeout(mediaTimeout))
}

func (j *Jobs) Transcode(p AssetPayload) error {
	return j.enqueue(Transcode, p.AssetID.String(), p, asynq.Timeout(transcodeTimeout))
}

func (j *Jobs) Edit(assetID uuid.UUID) error {
	return j.enqueue(Edit, uuid.Must(uuid.NewV7()).String(), AssetPayload{AssetID: assetID}, asynq.Timeout(mediaTimeout))
}

func (j *Jobs) DeleteAsset(assetID uuid.UUID, deleteOnDisk bool) error {
	return j.enqueue(DeleteAsset, assetID.String(), DeleteAssetPayload{AssetID: assetID, DeleteOnDisk: deleteOnDisk})
}

func (j *Jobs) EmptyTrash() error {
	return j.enqueue(EmptyTrash, "all", EmptyTrashPayload{})
}

func (j *Jobs) All(jobType asyncjob.JobType, force *bool) error {
	return j.enqueue(jobType, uuid.Must(uuid.NewV7()).String(), ForcePayload{Force: force})
}

func (j *Jobs) Manual(jobType asyncjob.JobType) error {
	return j.enqueue(jobType, uuid.Must(uuid.NewV7()).String(), ManualPayload{})
}

func (j *Jobs) Backup() error {
	return j.enqueue(BackupDatabase, uuid.Must(uuid.NewV7()).String(), ManualPayload{}, asynq.MaxRetry(0))
}

func (j *Jobs) Stats(queue string) (Stats, error) {
	return j.queue.Stats(queue)
}

func (j *Jobs) Pause(queue string) error {
	return j.queue.Pause(queue)
}

func (j *Jobs) Resume(queue string) error {
	return j.queue.Resume(queue)
}

func (j *Jobs) Empty(queue string) error {
	return j.queue.Empty(queue)
}

func (j *Jobs) ClearFailed(queue string) error {
	return j.queue.ClearFailed(queue)
}

func (j *Jobs) Tasks(name string, statuses []queue.Status) ([]Task, error) {
	return j.queue.Tasks(name, statuses)
}
