package model

import "time"

type InfoTiket struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	LayananID int64     `json:"layanan_id"`
	Kode      string    `json:"kode"`
	Nomor     int       `json:"nomor"`
	Tanggal   time.Time `json:"tanggal"`
	Status    string    `json:"status"`
}

type EstimasiTunggu struct {
	TiketID            int64     `json:"tiket_id"`
	Kode               string    `json:"kode"`
	Status             string    `json:"status"`
	Posisi             int       `json:"posisi"`
	OrangDiDepan       int       `json:"orang_di_depan"`
	LoketBuka          int       `json:"loket_buka"`
	RataDurasiMenit    float64   `json:"rata_durasi_menit"`
	SumberDurasi       string    `json:"sumber_durasi"`
	EstimasiMenit      int       `json:"estimasi_menit"`
	PerkiraanDipanggil time.Time `json:"perkiraan_dipanggil"`
}
