package model

import "time"

type Notifikasi struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	TiketID   int64     `json:"tiket_id"`
	Judul     string    `json:"judul"`
	Pesan     string    `json:"pesan"`
	Dibaca    bool      `json:"dibaca"`
	CreatedAt time.Time `json:"created_at"`
}
