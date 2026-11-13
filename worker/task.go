package worker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/GodzGit/Antri.in/app/model"
)

const (
	TaskOutboxTiket = "outbox:tiket"
	QueueTiket      = "tiket"
)

type OutboxJob struct {
	OutboxID  int64           `json:"outbox_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

func NewOutboxTask(e model.Outbox) (*asynq.Task, []asynq.Option, error) {
	job, err := json.Marshal(OutboxJob{OutboxID: e.ID, EventType: e.EventType, Payload: e.Payload})
	if err != nil {
		return nil, nil, fmt.Errorf("menyusun task outbox %d: %w", e.ID, err)
	}

	opts := []asynq.Option{
		asynq.TaskID(fmt.Sprintf("outbox-%d", e.ID)),
		asynq.Queue(QueueTiket),
		asynq.MaxRetry(5),
		asynq.Timeout(30 * time.Second),
		asynq.Retention(24 * time.Hour),
	}
	return asynq.NewTask(TaskOutboxTiket, job), opts, nil
}
