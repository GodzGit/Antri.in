package model

import "time"

type RiwayatTiket struct {
	ID         string    `json:"id"`
	TiketID    int64     `json:"tiket_id"`
	LayananID  int64     `json:"layanan_id"`
	UserID     int64     `json:"user_id"`
	LoketID    *int64    `json:"loket_id,omitempty"`
	StatusLama string    `json:"status_lama"`
	StatusBaru string    `json:"status_baru"`
	Waktu      time.Time `json:"waktu"`
}
