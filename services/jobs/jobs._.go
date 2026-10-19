package jobs

import (
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"xn--gckvb8fzb.com/glides/models/asyncjob"
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
)

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

type Enqueuer interface {
	JobUnique(j *asyncjob.AsyncJob, id string, opts ...asynq.Option) error
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

	return j.queue.JobUnique(job, string(jobType)+":"+id, opts...)
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

func (j *Jobs) DeleteAsset(assetID uuid.UUID, deleteOnDisk bool) error {
	return j.enqueue(DeleteAsset, assetID.String(), DeleteAssetPayload{AssetID: assetID, DeleteOnDisk: deleteOnDisk})
}

func (j *Jobs) EmptyTrash() error {
	return j.enqueue(EmptyTrash, "all", EmptyTrashPayload{})
}
