package model

import (
	"encoding/json"
	"time"
)

const (
	EventTiketDiambil    = "tiket.diambil"
	EventTiketDipanggil  = "tiket.dipanggil"
	EventTiketSelesai    = "tiket.selesai"
	EventTiketDilewati   = "tiket.dilewati"
	EventTiketDibatalkan = "tiket.dibatalkan"
)

const (
	OutboxPending   = "pending"
	OutboxProcessed = "processed"
	OutboxFailed    = "failed"
)

type Outbox struct {
	ID          int64           `json:"id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	Attempts    int             `json:"attempts"`
	LastError   *string         `json:"last_error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
}

type TiketEvent struct {
	TiketID    int64     `json:"tiket_id"`
	LayananID  int64     `json:"layanan_id"`
	UserID     int64     `json:"user_id"`
	LoketID    *int64    `json:"loket_id"`
	StatusLama string    `json:"status_lama"`
	StatusBaru string    `json:"status_baru"`
	Waktu      time.Time `json:"waktu"`
}
