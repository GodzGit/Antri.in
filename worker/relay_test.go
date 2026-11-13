package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
)

type fakeOutbox struct {
	pending   []model.Outbox
	processed []int64
	retry     map[int64]string
}

func (f *fakeOutbox) FetchPending(_ context.Context, _ repository.DBTX, limit int) ([]model.Outbox, error) {
	if len(f.pending) > limit {
		return f.pending[:limit], nil
	}
	return f.pending, nil
}

func (f *fakeOutbox) MarkProcessed(_ context.Context, _ repository.DBTX, ids []int64) error {
	f.processed = append(f.processed, ids...)
	return nil
}

func (f *fakeOutbox) MarkRetry(_ context.Context, _ repository.DBTX, id int64, cause string) error {
	if f.retry == nil {
		f.retry = map[int64]string{}
	}
	f.retry[id] = cause
	return nil
}

type fakeTx struct{}

func (fakeTx) WithTx(_ context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

type fakeQueue struct {
	ids     []string
	gagalDi string
	konflik string
}

func (f *fakeQueue) EnqueueContext(_ context.Context, _ *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	id := ""
	for _, o := range opts {
		if o.Type() == asynq.TaskIDOpt {
			id = o.Value().(string)
		}
	}
	f.ids = append(f.ids, id)
	if id == f.gagalDi {
		return nil, errors.New("redis mati")
	}
	if id == f.konflik {
		return nil, asynq.ErrTaskIDConflict
	}
	return &asynq.TaskInfo{ID: id}, nil
}

func outboxUji(ids ...int64) []model.Outbox {
	out := []model.Outbox{}
	for _, id := range ids {
		out = append(out, model.Outbox{ID: id, EventType: model.EventTiketDiambil, Payload: []byte(`{"tiket_id":1}`)})
	}
	return out
}

func relayUji(o *fakeOutbox, q *fakeQueue, batch int) *Relay {
	return NewRelay(o, fakeTx{}, q, slog.New(slog.NewTextHandler(io.Discard, nil)), 0, batch)
}

func TestRelayOnce(t *testing.T) {
	tests := []struct {
		name          string
		pending       []model.Outbox
		queue         fakeQueue
		batch         int
		wantSent      int
		wantErr       bool
		wantProcessed []int64
		wantRetry     int64
	}{
		{name: "tidak ada event", batch: 10},
		{name: "semua terkirim", pending: outboxUji(1, 2, 3), batch: 10, wantSent: 3, wantProcessed: []int64{1, 2, 3}},
		{name: "dibatasi batch", pending: outboxUji(1, 2, 3), batch: 2, wantSent: 2, wantProcessed: []int64{1, 2}},
		{name: "sudah pernah di-enqueue dianggap terkirim", pending: outboxUji(1, 2), queue: fakeQueue{konflik: "outbox-1"}, batch: 10, wantSent: 2, wantProcessed: []int64{1, 2}},
		{name: "redis gagal di tengah", pending: outboxUji(1, 2, 3), queue: fakeQueue{gagalDi: "outbox-2"}, batch: 10, wantSent: 1, wantErr: true, wantProcessed: []int64{1}, wantRetry: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &fakeOutbox{pending: tt.pending}
			q := tt.queue
			sent, err := relayUji(o, &q, tt.batch).RelayOnce(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if sent != tt.wantSent {
				t.Fatalf("terkirim = %d, mau %d", sent, tt.wantSent)
			}
			if len(o.processed) != len(tt.wantProcessed) {
				t.Fatalf("processed = %v, mau %v", o.processed, tt.wantProcessed)
			}
			for i := range o.processed {
				if o.processed[i] != tt.wantProcessed[i] {
					t.Fatalf("processed = %v, mau %v", o.processed, tt.wantProcessed)
				}
			}
			if tt.wantRetry != 0 {
				if _, ok := o.retry[tt.wantRetry]; !ok || len(o.retry) != 1 {
					t.Fatalf("retry = %v, mau hanya outbox %d", o.retry, tt.wantRetry)
				}
			} else if len(o.retry) != 0 {
				t.Fatalf("retry = %v, mau kosong", o.retry)
			}
		})
	}
}

func TestNewOutboxTaskMemakaiTaskIDDariOutbox(t *testing.T) {
	q := &fakeQueue{}
	o := &fakeOutbox{pending: outboxUji(42)}
	if _, err := relayUji(o, q, 10).RelayOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(q.ids) != 1 || q.ids[0] != "outbox-42" {
		t.Fatalf("task id = %v, mau [outbox-42]", q.ids)
	}
}
