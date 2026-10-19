package jobs

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
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
		for _, del := range []func(string) (int, error){
			inspector.DeleteAllPendingTasks,
			inspector.DeleteAllArchivedTasks,
			inspector.DeleteAllCompletedTasks,
			inspector.DeleteAllScheduledTasks,
			inspector.DeleteAllRetryTasks,
		} {
			if _, err := del(queueDefault); err != nil && !errors.Is(err, asynq.ErrQueueNotFound) {
				t.Errorf("clear: %v", err)
			}
		}
	}

	queues, err := inspector.Queues()
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range queues {
		if q == queueDefault {
			clear()
		}
	}

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

	j, err := New(disp)
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

	pending, err := inspector.ListPendingTasks(queueDefault)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != taskID {
		t.Fatalf("pending = %v, want the one task %s", pending, taskID)
	}

	if err = inspector.ArchiveTask(queueDefault, taskID); err != nil {
		t.Fatal(err)
	}
	if err = j.DeleteUser(id, false); err != nil {
		t.Fatal(err)
	}

	info, err := inspector.GetTaskInfo(queueDefault, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if info.State != asynq.TaskStatePending {
		t.Errorf("after the archived task the state is %v, want pending", info.State)
	}
}
