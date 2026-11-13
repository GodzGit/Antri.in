package model

type StatistikJam struct {
	Jam    int `json:"jam"`
	Jumlah int `json:"jumlah"`
}

type StatistikLayanan struct {
	LayananID        int64          `json:"layanan_id"`
	Tanggal          string         `json:"tanggal"`
	TotalTiket       int            `json:"total_tiket"`
	JumlahPerStatus  map[string]int `json:"jumlah_per_status"`
	PerJam           []StatistikJam `json:"per_jam"`
	JamTersibuk      *int           `json:"jam_tersibuk"`
	RataTungguMenit  float64        `json:"rata_tunggu_menit"`
	MaksTungguMenit  float64        `json:"maks_tunggu_menit"`
	RataLayananMenit float64        `json:"rata_layanan_menit"`
}
