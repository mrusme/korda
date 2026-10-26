package jobs

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"tty.fail/mrus/korda/models/queue"
	"xn--gckvb8fzb.com/glides/models/asyncjob"
	"xn--gckvb8fzb.com/glides/services/config"
	"xn--gckvb8fzb.com/glides/services/dispatch"
)

const (
	envRedis      = "KORDA_TEST_REDIS"
	testDatabase  = 13
	queueDefault  = dispatch.QueueDefault
	deleteUserKey = "delete-user:"
)

type recorder struct {
	jobs []*asyncjob.AsyncJob
	ids  []string
}

func (r *recorder) JobUnique(j *asyncjob.AsyncJob, id string, opts ...asynq.Option) error {
	r.jobs = append(r.jobs, j)
	r.ids = append(r.ids, id)
	return nil
}

func (r *recorder) Remove(queue string, id string) error {
	return nil
}

func (r *recorder) Stats(queue string) (Stats, error) {
	return Stats{}, nil
}

func (r *recorder) Pause(queue string) error {
	return nil
}

func (r *recorder) Resume(queue string) error {
	return nil
}

func (r *recorder) Empty(queue string) error {
	return nil
}

func (r *recorder) ClearFailed(queue string) error {
	return nil
}

func (r *recorder) Tasks(name string, statuses []queue.Status) ([]Task, error) {
	return nil, nil
}

func TestDeleteUserEnqueuesOneJobPerUser(t *testing.T) {
	r := &recorder{}
	j, err := New(r)
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	if err = j.DeleteUser(id, true); err != nil {
		t.Fatal(err)
	}

	if len(r.jobs) != 1 || r.ids[0] != deleteUserKey+id.String() {
		t.Fatalf("enqueued %v under %v", r.jobs, r.ids)
	}

	job := r.jobs[0]
	if job.TargetID != TargetID || job.Type != DeleteUser || job.SubType != Run {
		t.Errorf("job = %+v", job)
	}

	payloads, err := asyncjob.Payloads[DeleteUserPayload](*job)
	if err != nil || len(payloads) != 1 || payloads[0].UserID != id || !payloads[0].Force {
		t.Errorf("payloads = %+v, %v", payloads, err)
	}

	var raw map[string]any
	if err = json.Unmarshal(job.Payload, &raw); err != nil || raw["userId"] != id.String() || raw["force"] != true {
		t.Errorf("payload = %s", job.Payload)
	}
}

func cfgOf(t *testing.T) config.Redis {
	t.Helper()

	return config.Redis{Addrs: []string{os.Getenv(envRedis)}, Database: testDatabase}
}

func live(t *testing.T) (*dispatch.Dispatch, *asynq.Inspector) {
	t.Helper()

	addr := os.Getenv(envRedis)
	if addr == "" {
		t.Skipf("%s is not set", envRedis)
	}

	cfg := config.Redis{Addrs: []string{addr}, Database: testDatabase}
	disp, err := dispatch.New(cfg, config.Targets{})
	if err != nil {
		t.Fatal(err)
	}
	if err = disp.Startup(); err != nil {
		t.Fatal(err)
	}

	inspector := asynq.NewInspector(dispatch.RedisConnOpt(cfg))
	clear := func() {
		queues, err := inspector.Queues()
		if err != nil {
			t.Errorf("clear: %v", err)
		}
		for _, q := range []string{queueDefault, QueueBackground} {
			if !slices.Contains(queues, q) {
				continue
			}
			for _, del := range []func(string) (int, error){
				inspector.DeleteAllPendingTasks,
				inspector.DeleteAllArchivedTasks,
				inspector.DeleteAllCompletedTasks,
				inspector.DeleteAllScheduledTasks,
				inspector.DeleteAllRetryTasks,
			} {
				if _, err := del(q); err != nil {
					t.Errorf("clear: %v", err)
				}
			}
		}
	}

	clear()

	t.Cleanup(func() {
		clear()
		inspector.Close()
		if err := disp.Shutdown(); err != nil {
			t.Error(err)
		}
	})

	return disp, inspector
}

func TestDeleteUserAgainstRedis(t *testing.T) {
	disp, inspector := live(t)

	j, err := New(NewDispatcher(disp, cfgOf(t)))
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.New()
	taskID := deleteUserKey + id.String()
	for range 2 {
		if err = j.DeleteUser(id, false); err != nil {
			t.Fatal(err)
		}
	}

	pending, err := inspector.ListPendingTasks(QueueBackground)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != taskID {
		t.Fatalf("pending = %v, want the one task %s", pending, taskID)
	}

	if err = inspector.ArchiveTask(QueueBackground, taskID); err != nil {
		t.Fatal(err)
	}
	if err = j.DeleteUser(id, false); err != nil {
		t.Fatal(err)
	}

	info, err := inspector.GetTaskInfo(QueueBackground, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if info.State != asynq.TaskStatePending {
		t.Errorf("after the archived task the state is %v, want pending", info.State)
	}
}

func TestQueueInspectionAgainstRedis(t *testing.T) {
	disp, inspector := live(t)

	unused := string(queue.Ocr)
	reset := func() {
		for _, q := range []string{QueueVideo, unused} {
			if err := inspector.UnpauseQueue(q); err != nil && !strings.Contains(err.Error(), "not paused") {
				t.Errorf("unpause %s: %v", q, err)
			}
			if err := inspector.DeleteQueue(q, true); err != nil && !errors.Is(err, asynq.ErrQueueNotFound) {
				t.Errorf("delete %s: %v", q, err)
			}
		}
	}
	reset()
	t.Cleanup(reset)

	d := NewDispatcher(disp, cfgOf(t))
	t.Cleanup(func() { d.Close() })
	j, err := New(d)
	if err != nil {
		t.Fatal(err)
	}

	stats := func(q string) Stats {
		t.Helper()
		st, err := j.Stats(q)
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	tasks := func(q string, statuses ...queue.Status) []Task {
		t.Helper()
		list, err := j.Tasks(q, statuses)
		if err != nil {
			t.Fatal(err)
		}
		return list
	}

	if st := stats(unused); st != (Stats{}) {
		t.Errorf("stats of an unused queue = %+v", st)
	}
	for range 2 {
		if err = j.Pause(unused); err != nil {
			t.Fatal(err)
		}
	}
	if st := stats(unused); st != (Stats{IsPaused: true}) {
		t.Errorf("stats of the paused unused queue = %+v", st)
	}
	for range 2 {
		if err = j.Resume(unused); err != nil {
			t.Fatal(err)
		}
	}
	if st := stats(unused); st.IsPaused {
		t.Errorf("stats after resuming = %+v", st)
	}

	id := uuid.New()
	before := time.Now().UnixMilli()
	if err = j.Transcode(AssetPayload{AssetID: id}); err != nil {
		t.Fatal(err)
	}
	if st := stats(QueueVideo); st != (Stats{Waiting: 1}) {
		t.Errorf("stats with one task = %+v", st)
	}
	list := tasks(QueueVideo, queue.StatusWaiting, queue.StatusPaused)
	if len(list) != 1 || list[0].ID != "transcode:"+id.String() || list[0].Job.Type != Transcode {
		t.Fatalf("tasks = %+v", list)
	}
	if created := Created(list[0].Job); created < before || created > time.Now().UnixMilli() {
		t.Errorf("created %d, enqueued after %d", created, before)
	}
	if string(list[0].Job.Payload) != `{"assetId":"`+id.String()+`"}` {
		t.Errorf("payload = %s", list[0].Job.Payload)
	}

	if err = j.Pause(QueueVideo); err != nil {
		t.Fatal(err)
	}
	if st := stats(QueueVideo); st != (Stats{Paused: 1, IsPaused: true}) {
		t.Errorf("stats while paused = %+v", st)
	}
	if got := len(tasks(QueueVideo, queue.StatusWaiting)); got != 0 {
		t.Errorf("waiting while paused = %d", got)
	}
	if got := len(tasks(QueueVideo, queue.StatusPaused)); got != 1 {
		t.Errorf("paused while paused = %d", got)
	}
	if err = j.Resume(QueueVideo); err != nil {
		t.Fatal(err)
	}

	if err = inspector.ArchiveTask(QueueVideo, list[0].ID); err != nil {
		t.Fatal(err)
	}
	other := uuid.New()
	if err = j.Transcode(AssetPayload{AssetID: other}); err != nil {
		t.Fatal(err)
	}
	if st := stats(QueueVideo); st != (Stats{Failed: 1, Waiting: 1}) {
		t.Errorf("stats with a failed and a waiting task = %+v", st)
	}
	if got := tasks(QueueVideo, queue.StatusFailed); len(got) != 1 || got[0].ID != list[0].ID {
		t.Errorf("failed tasks = %+v", got)
	}

	if err = j.Empty(QueueVideo); err != nil {
		t.Fatal(err)
	}
	if st := stats(QueueVideo); st != (Stats{Failed: 1}) {
		t.Errorf("stats after emptying = %+v", st)
	}
	if err = j.ClearFailed(QueueVideo); err != nil {
		t.Fatal(err)
	}
	if st := stats(QueueVideo); st != (Stats{}) {
		t.Errorf("stats after clearing the failed tasks = %+v", st)
	}
}
