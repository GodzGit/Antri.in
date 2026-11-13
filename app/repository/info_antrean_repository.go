package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
)

var ErrDataTidakAda = errors.New("data tidak ditemukan")

type InfoAntreanRepository struct {
	db DBTX
}

func NewInfoAntreanRepository(db DBTX) *InfoAntreanRepository {
	return &InfoAntreanRepository{db: db}
}

func scanInfoTiket(row pgx.Row) (*model.InfoTiket, error) {
	var t model.InfoTiket
	err := row.Scan(&t.ID, &t.UserID, &t.LayananID, &t.Kode, &t.Nomor, &t.Tanggal, &t.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *InfoAntreanRepository) TiketByID(ctx context.Context, id int64) (*model.InfoTiket, error) {
	t, err := scanInfoTiket(r.db.QueryRow(ctx, `
		SELECT id, user_id, layanan_id, kode, nomor, tanggal, status
		FROM tiket
		WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("mengambil tiket %d: %w", id, err)
	}
	return t, nil
}

func (r *InfoAntreanRepository) TiketMenungguKe(ctx context.Context, layananID int64, tanggal time.Time, urutan int) (*model.InfoTiket, error) {
	t, err := scanInfoTiket(r.db.QueryRow(ctx, `
		SELECT id, user_id, layanan_id, kode, nomor, tanggal, status
		FROM tiket
		WHERE layanan_id = $1 AND tanggal = $2 AND status = 'menunggu'
		ORDER BY nomor
		OFFSET $3 LIMIT 1`, layananID, tanggal, urutan-1))
	if err != nil {
		return nil, fmt.Errorf("mengambil antrean ke-%d: %w", urutan, err)
	}
	return t, nil
}

func (r *InfoAntreanRepository) NamaLoket(ctx context.Context, id int64) (string, error) {
	var nama string
	err := r.db.QueryRow(ctx, `SELECT nama FROM loket WHERE id = $1`, id).Scan(&nama)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("mengambil loket %d: %w", id, err)
	}
	return nama, nil
}

func (r *InfoAntreanRepository) OrangDiDepan(ctx context.Context, layananID int64, tanggal time.Time, nomor int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM tiket
		WHERE layanan_id = $1 AND tanggal = $2 AND status = 'menunggu' AND nomor < $3`,
		layananID, tanggal, nomor).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("menghitung antrean di depan: %w", err)
	}
	return n, nil
}

func (r *InfoAntreanRepository) LoketBuka(ctx context.Context, layananID int64) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM loket WHERE layanan_id = $1 AND status = 'buka'`, layananID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("menghitung loket buka: %w", err)
	}
	return n, nil
}

func (r *InfoAntreanRepository) DurasiLayanan(ctx context.Context, layananID int64, minSampel int) (float64, bool, error) {
	var bawaan int
	var rata *float64
	var sampel int
	err := r.db.QueryRow(ctx, `
		SELECT l.rata_durasi_menit, h.rata, h.sampel
		FROM layanan l
		CROSS JOIN LATERAL (
			SELECT AVG(EXTRACT(EPOCH FROM (t.selesai_at - t.dipanggil_at)) / 60)::float8 AS rata,
			       COUNT(*)::int AS sampel
			FROM tiket t
			WHERE t.layanan_id = l.id
			  AND t.status = 'selesai'
			  AND t.dipanggil_at IS NOT NULL
			  AND t.selesai_at IS NOT NULL
			  AND t.selesai_at > NOW() - INTERVAL '7 days'
		) h
		WHERE l.id = $1`, layananID).Scan(&bawaan, &rata, &sampel)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, ErrDataTidakAda
	}
	if err != nil {
		return 0, false, fmt.Errorf("menghitung durasi layanan: %w", err)
	}
	if rata != nil && sampel >= minSampel {
		return *rata, true, nil
	}
	return float64(bawaan), false, nil
}

func (r *InfoAntreanRepository) LayananAda(ctx context.Context, id int64) (bool, error) {
	var ada bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM layanan WHERE id = $1)`, id).Scan(&ada)
	if err != nil {
		return false, fmt.Errorf("memeriksa layanan %d: %w", id, err)
	}
	return ada, nil
}
