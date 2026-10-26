package view

import (
	"bytes"
	"encoding/json"

	"tty.fail/mrus/korda/models/queue"
	"tty.fail/mrus/korda/services/jobs"
)

type QueueStatistics struct {
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Delayed   int `json:"delayed"`
	Waiting   int `json:"waiting"`
	Paused    int `json:"paused"`
}

type Queue struct {
	Name       queue.Name      `json:"name"`
	IsPaused   bool            `json:"isPaused"`
	Statistics QueueStatistics `json:"statistics"`
}

type QueueJob struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

type QueueStatusLegacy struct {
	IsPaused bool `json:"isPaused"`
	IsActive bool `json:"isActive"`
}

type QueueLegacy struct {
	QueueStatus QueueStatusLegacy `json:"queueStatus"`
	JobCounts   QueueStatistics   `json:"jobCounts"`
}

type QueuesLegacy struct {
	names  []queue.Name
	queues []QueueLegacy
}

func (q QueuesLegacy) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, name := range q.names {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(name)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(q.queues[i])
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')

	return buf.Bytes(), nil
}

func newQueueStatistics(s queue.Statistics) QueueStatistics {
	return QueueStatistics{
		Active: s.Active, Completed: s.Completed, Failed: s.Failed,
		Delayed: s.Delayed, Waiting: s.Waiting, Paused: s.Paused,
	}
}

func NewQueue(q queue.Queue) Queue {
	return Queue{Name: q.Name, IsPaused: q.IsPaused, Statistics: newQueueStatistics(q.Statistics)}
}

func NewQueues(list []queue.Queue) []Queue {
	out := make([]Queue, 0, len(list))
	for _, q := range list {
		out = append(out, NewQueue(q))
	}

	return out
}

func NewQueueLegacy(q queue.Queue) QueueLegacy {
	return QueueLegacy{
		QueueStatus: QueueStatusLegacy{IsPaused: q.IsPaused, IsActive: q.Statistics.Active > 0},
		JobCounts:   newQueueStatistics(q.Statistics),
	}
}

func NewQueuesLegacy(list []queue.Queue) QueuesLegacy {
	out := QueuesLegacy{}
	for _, q := range list {
		out.names = append(out.names, q.Name)
		out.queues = append(out.queues, NewQueueLegacy(q))
	}

	return out
}

func NewQueueJobs(tasks []jobs.Task) []QueueJob {
	out := make([]QueueJob, 0, len(tasks))
	for _, t := range tasks {
		data := json.RawMessage(`{}`)
		if len(t.Job.Payload) > 0 && !t.Job.Batch {
			data = t.Job.Payload
		}
		out = append(out, QueueJob{ID: t.ID, Name: jobs.JobName(t.Job.Type), Timestamp: jobs.Created(t.Job), Data: data})
	}

	return out
}
