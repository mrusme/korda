package queues

import (
	"fmt"
	"slices"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/queue"
	"tty.fail/mrus/korda/services/jobs"
	"xn--gckvb8fzb.com/glides/models/asyncjob"
	"xn--gckvb8fzb.com/glides/runtime"
)

type NameParams struct {
	Name queue.Name `json:"name"`
}

type UpdateInput struct {
	IsPaused *bool `json:"isPaused"`
}

type EmptyInput struct {
	Failed *bool `json:"failed"`
}

type JobsInput struct {
	Status []queue.Status `json:"status" v:"optional"`
}

type CommandInput struct {
	Command queue.Command `json:"command"`
	Force   *bool         `json:"force"`
}

type CreateJobInput struct {
	Name queue.ManualJob `json:"name"`
}

var sweeps = map[queue.Name]asyncjob.JobType{
	queue.ThumbnailGeneration: jobs.ThumbnailsAll,
	queue.MetadataExtraction:  jobs.MetadataAll,
	queue.VideoConversion:     jobs.TranscodeAll,
}

var idle = []queue.Name{
	queue.FaceDetection, queue.FacialRecognition, queue.SmartSearch, queue.DuplicateDetection, queue.Ocr,
	queue.Sidecar, queue.Library, queue.StorageTemplateMigration, queue.Migration,
}

var manual = map[queue.ManualJob]asyncjob.JobType{
	queue.ManualTagCleanup:    jobs.TagCleanup,
	queue.ManualUserCleanup:   jobs.UserDeleteCheck,
	queue.ManualMemoryCleanup: jobs.MemoryCleanup,
	queue.ManualMemoryCreate:  jobs.MemoryGenerate,
}

func Get(rt *runtime.Runtime, name queue.Name) (queue.Queue, error) {
	st, err := gh.Jobs(rt).Stats(string(name))
	if err != nil {
		return queue.Queue{}, err
	}

	return queue.Queue{
		Name:     name,
		IsPaused: st.IsPaused,
		Statistics: queue.Statistics{
			Active: st.Active, Completed: st.Completed, Failed: st.Failed,
			Delayed: st.Delayed, Waiting: st.Waiting, Paused: st.Paused,
		},
	}, nil
}

func List(rt *runtime.Runtime) ([]queue.Queue, error) {
	out := make([]queue.Queue, 0, len(queue.Names))
	for _, name := range queue.Names {
		q, err := Get(rt, name)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}

	return out, nil
}

func Update(rt *runtime.Runtime, name queue.Name, in *UpdateInput) (queue.Queue, error) {
	switch {
	case in.IsPaused == nil:
	case *in.IsPaused && name == queue.BackgroundTask:
		return queue.Queue{}, errs.ErrQueueNotPausable
	case *in.IsPaused:
		if err := gh.Jobs(rt).Pause(string(name)); err != nil {
			return queue.Queue{}, err
		}
	default:
		if err := gh.Jobs(rt).Resume(string(name)); err != nil {
			return queue.Queue{}, err
		}
	}

	return Get(rt, name)
}

func Empty(rt *runtime.Runtime, name queue.Name, in *EmptyInput) error {
	if err := gh.Jobs(rt).Empty(string(name)); err != nil {
		return err
	}
	if in.Failed != nil && *in.Failed {
		return gh.Jobs(rt).ClearFailed(string(name))
	}

	return nil
}

func Jobs(rt *runtime.Runtime, name queue.Name, in *JobsInput) ([]jobs.Task, error) {
	statuses := queue.Statuses
	if in.Status != nil {
		statuses = []queue.Status{}
		for _, s := range in.Status {
			if !slices.Contains(statuses, s) {
				statuses = append(statuses, s)
			}
		}
		if slices.Contains(statuses, queue.StatusWaiting) && !slices.Contains(statuses, queue.StatusPaused) {
			statuses = append(statuses, queue.StatusPaused)
		}
	}

	return gh.Jobs(rt).Tasks(string(name), statuses)
}

func Command(rt *runtime.Runtime, name queue.Name, in *CommandInput) (queue.Queue, error) {
	j := gh.Jobs(rt)

	var err error
	switch in.Command {
	case queue.CommandStart:
		err = start(rt, name, in.Force)
	case queue.CommandPause:
		err = j.Pause(string(name))
	case queue.CommandResume:
		err = j.Resume(string(name))
	case queue.CommandEmpty:
		err = j.Empty(string(name))
	case queue.CommandClearFailed:
		err = j.ClearFailed(string(name))
	}
	if err != nil {
		return queue.Queue{}, err
	}

	return Get(rt, name)
}

func start(rt *runtime.Runtime, name queue.Name, force *bool) error {
	q, err := Get(rt, name)
	if err != nil {
		return err
	}
	if q.Statistics.Active > 0 {
		return errs.ErrJobRunning
	}

	switch {
	case sweeps[name] != "":
		return gh.Jobs(rt).All(sweeps[name], force)
	case name == queue.BackupDatabase:
		return errs.UnsupportedBackups
	case slices.Contains(idle, name):
		return nil
	}

	return fmt.Errorf("%w: %s", errs.ErrInvalidJobName, name)
}

func CreateJob(rt *runtime.Runtime, in *CreateJobInput) error {
	switch {
	case manual[in.Name] != "":
		return gh.Jobs(rt).Manual(manual[in.Name])
	case in.Name == queue.ManualPersonCleanup:
		return nil
	case in.Name == queue.ManualBackupDatabase:
		return errs.UnsupportedBackups
	}

	return errs.UnsupportedIntegrity
}
