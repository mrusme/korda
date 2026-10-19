package jobs

import (
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"xn--gckvb8fzb.com/glides/models/asyncjob"
)

const TargetID string = "jobs"

const Run asyncjob.JobSubType = "run"

const (
	DeleteUser asyncjob.JobType = "delete-user"
)

type DeleteUserPayload struct {
	UserID uuid.UUID `json:"userId"`
	Force  bool      `json:"force"`
}

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

func (j *Jobs) enqueue(jobType asyncjob.JobType, id string, payload any) error {
	job := asyncjob.New(TargetID, jobType, Run)
	if err := job.SetPayload(payload); err != nil {
		return err
	}

	return j.queue.JobUnique(job, string(jobType)+":"+id)
}

func (j *Jobs) DeleteUser(userID uuid.UUID, force bool) error {
	return j.enqueue(DeleteUser, userID.String(), DeleteUserPayload{UserID: userID, Force: force})
}
