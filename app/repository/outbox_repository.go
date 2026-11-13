package repository

import (
	"context"
	"encoding/json"
	"fmt"
)

type OutboxRepository struct {
	db DBTX
}

func NewOutboxRepository(db DBTX) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Insert(ctx context.Context, db DBTX, eventType string, payload any) error {
	if db == nil {
		db = r.db
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("menyusun payload outbox: %w", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO outbox (event_type, payload) VALUES ($1, $2)`,
		eventType, data)
	if err != nil {
		return fmt.Errorf("mencatat outbox: %w", err)
	}
	return nil
}
