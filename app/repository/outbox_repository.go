package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GodzGit/Antri.in/app/model"
)

type OutboxRepository struct {
	db DBTX
}

func NewOutboxRepository(db DBTX) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) conn(db DBTX) DBTX {
	if db == nil {
		return r.db
	}
	return db
}

func (r *OutboxRepository) Insert(ctx context.Context, db DBTX, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("menyusun payload outbox: %w", err)
	}

	_, err = r.conn(db).Exec(ctx,
		`INSERT INTO outbox (event_type, payload) VALUES ($1, $2)`,
		eventType, data)
	if err != nil {
		return fmt.Errorf("mencatat outbox: %w", err)
	}
	return nil
}

func (r *OutboxRepository) FetchPending(ctx context.Context, db DBTX, limit int) ([]model.Outbox, error) {
	rows, err := r.conn(db).Query(ctx, `
		SELECT id, event_type, payload, status, attempts, last_error, created_at, processed_at
		FROM outbox
		WHERE status = 'pending'
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("mengambil outbox pending: %w", err)
	}
	defer rows.Close()

	events := []model.Outbox{}
	for rows.Next() {
		var e model.Outbox
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.Status, &e.Attempts,
			&e.LastError, &e.CreatedAt, &e.ProcessedAt); err != nil {
			return nil, fmt.Errorf("membaca outbox: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, db DBTX, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.conn(db).Exec(ctx, `
		UPDATE outbox
		SET status = 'processed', processed_at = NOW()
		WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("menandai outbox processed: %w", err)
	}
	return nil
}

func (r *OutboxRepository) MarkRetry(ctx context.Context, db DBTX, id int64, cause string) error {
	_, err := r.conn(db).Exec(ctx, `
		UPDATE outbox
		SET attempts = attempts + 1, last_error = $2
		WHERE id = $1`, id, cause)
	if err != nil {
		return fmt.Errorf("mencatat kegagalan outbox %d: %w", id, err)
	}
	return nil
}
