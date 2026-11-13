package model

import "time"

// Layanan adalah jenis antrean yang bisa diambil pengunjung
// (contoh: "A" untuk Akademik, "B" untuk Klinik).
// Kolom yang nullable di database memakai pointer.
type Layanan struct {
	ID              int64     `json:"id"`
	Kode            string    `json:"kode"`
	Nama            string    `json:"nama"`
	Deskripsi       *string   `json:"deskripsi,omitempty"`
	JamBuka         *string   `json:"jam_buka,omitempty"`     // format "08:00"
	JamTutup        *string   `json:"jam_tutup,omitempty"`    // format "16:00"
	KuotaHarian     *int      `json:"kuota_harian,omitempty"` // nil = tanpa kuota
	RataDurasiMenit int       `json:"rata_durasi_menit"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateLayananRequest struct {
	Kode            string `json:"kode" validate:"required,max=10"`
	Nama            string `json:"nama" validate:"required,max=100"`
	Deskripsi       string `json:"deskripsi"`
	JamBuka         string `json:"jam_buka" validate:"required"`
	JamTutup        string `json:"jam_tutup" validate:"required"`
	KuotaHarian     *int   `json:"kuota_harian" validate:"omitempty,gte=0"`
	RataDurasiMenit int    `json:"rata_durasi_menit" validate:"required,gte=1"`
}

type UpdateLayananRequest struct {
	Kode            string `json:"kode" validate:"required,max=10"`
	Nama            string `json:"nama" validate:"required,max=100"`
	Deskripsi       string `json:"deskripsi"`
	JamBuka         string `json:"jam_buka" validate:"required"`
	JamTutup        string `json:"jam_tutup" validate:"required"`
	KuotaHarian     *int   `json:"kuota_harian" validate:"omitempty,gte=0"`
	RataDurasiMenit int    `json:"rata_durasi_menit" validate:"required,gte=1"`
	IsActive        bool   `json:"is_active"`
}
