package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
)

type Enqueuer interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type outboxStore interface {
	FetchPending(ctx context.Context, db repository.DBTX, limit int) ([]model.Outbox, error)
	MarkProcessed(ctx context.Context, db repository.DBTX, ids []int64) error
	MarkRetry(ctx context.Context, db repository.DBTX, id int64, cause string) error
}

type txRunner interface {
	WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error
}

type Relay struct {
	outbox   outboxStore
	tx       txRunner
	queue    Enqueuer
	logger   *slog.Logger
	interval time.Duration
	batch    int
}

func NewRelay(outbox outboxStore, tx txRunner, queue Enqueuer, logger *slog.Logger, interval time.Duration, batch int) *Relay {
	return &Relay{outbox: outbox, tx: tx, queue: queue, logger: logger, interval: interval, batch: batch}
}

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		n, err := r.RelayOnce(ctx)
		if err != nil {
			r.logger.Warn("relay tertunda", slog.Int("terkirim", n), slog.String("error", err.Error()))
		} else if n > 0 {
			r.logger.Info("relay selesai", slog.Int("terkirim", n))
		}

		if err == nil && n == r.batch && ctx.Err() == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Relay) RelayOnce(ctx context.Context) (int, error) {
	var sent int
	var enqueueErr error

	err := r.tx.WithTx(ctx, func(tx pgx.Tx) error {
		events, err := r.outbox.FetchPending(ctx, tx, r.batch)
		if err != nil {
			return err
		}

		ids := make([]int64, 0, len(events))
		for _, e := range events {
			if err := r.publish(ctx, e); err != nil {
				enqueueErr = err
				if err := r.outbox.MarkRetry(ctx, tx, e.ID, err.Error()); err != nil {
					return err
				}
				break
			}
			ids = append(ids, e.ID)
		}

		if err := r.outbox.MarkProcessed(ctx, tx, ids); err != nil {
			return err
		}
		sent = len(ids)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return sent, enqueueErr
}

func (r *Relay) publish(ctx context.Context, e model.Outbox) error {
	task, opts, err := NewOutboxTask(e)
	if err != nil {
		return err
	}

	_, err = r.queue.EnqueueContext(ctx, task, opts...)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue outbox %d: %w", e.ID, err)
	}
	return nil
}
