package model

import "time"

// Status tiket sesuai CHECK constraint pada tabel tiket.
const (
	StatusMenunggu  = "menunggu"
	StatusDipanggil = "dipanggil"
	StatusSelesai   = "selesai"
	StatusDilewati  = "dilewati"
	StatusBatal     = "batal"
)

// Tiket adalah satu nomor antrean yang diambil pengunjung.
// Kolom nullable di database memakai pointer.
type Tiket struct {
	ID          int64      `json:"id"`
	LayananID   int64      `json:"layanan_id"`
	UserID      int64      `json:"user_id"`
	LoketID     *int64     `json:"loket_id,omitempty"`
	Tanggal     time.Time  `json:"tanggal"`
	Nomor       int        `json:"nomor"`
	Kode        string     `json:"kode"` // contoh "A-012"
	Status      string     `json:"status"`
	DiambilAt   time.Time  `json:"diambil_at"`
	DipanggilAt *time.Time `json:"dipanggil_at,omitempty"`
	SelesaiAt   *time.Time `json:"selesai_at,omitempty"`
	Version     int        `json:"version"`
}

// TiketSaya adalah tiket milik pengunjung + info layanan/loket/posisi.
type TiketSaya struct {
	Tiket
	Layanan string `json:"layanan"`
	Loket   string `json:"loket,omitempty"`
	Posisi  int    `json:"posisi"` // jumlah orang yang masih menunggu di depannya
}

// PapanAntrean adalah tampilan papan antrean sebuah layanan.
type PapanAntrean struct {
	SedangDipanggil []Tiket `json:"sedang_dipanggil"`
	Menunggu        []Tiket `json:"menunggu"`
}

// LoketInfo adalah data loket yang dibutuhkan tiket service
// (tanpa perlu mengimpor service loket).
type LoketInfo struct {
	ID        int64
	LayananID int64
	Nama      string
	PetugasID *int64
	Status    string // "buka" | "tutup"
}
