package model

import "time"

// Status loket sesuai CHECK constraint pada tabel loket.
const (
	LoketBuka  = "buka"
	LoketTutup = "tutup"
)

// Loket adalah tempat petugas memanggil antrean untuk satu layanan.
// PetugasID nullable: loket boleh belum punya petugas.
type Loket struct {
	ID        int64     `json:"id"`
	LayananID int64     `json:"layanan_id"`
	Layanan   string    `json:"layanan,omitempty"` // hasil JOIN, hanya untuk response
	Nama      string    `json:"nama"`
	PetugasID *int64    `json:"petugas_id,omitempty"`
	Petugas   string    `json:"petugas,omitempty"` // hasil JOIN, hanya untuk response
	Status    string    `json:"status"`            // "buka" | "tutup"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateLoketRequest struct {
	LayananID int64  `json:"layanan_id" validate:"required,gt=0"`
	Nama      string `json:"nama" validate:"required,max=50"`
}

type UpdateLoketRequest struct {
	LayananID int64  `json:"layanan_id" validate:"required,gt=0"`
	Nama      string `json:"nama" validate:"required,max=50"`
	PetugasID *int64 `json:"petugas_id" validate:"omitempty,gt=0"`
}
