package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
)

// LayananRepository menangani semua query ke tabel layanan.
// Menerima DBTX supaya bisa dipakai dengan pool maupun di dalam transaksi.
type LayananRepository struct {
	db DBTX
}

func NewLayananRepository(db DBTX) *LayananRepository {
	return &LayananRepository{db: db}
}

// layananColumns dipakai di semua SELECT supaya kolom dan scan selalu konsisten.
// jam_buka/jam_tutup di-cast ke teks "HH24:MI" agar hasilnya bersih ("08:00").
const layananColumns = `
	id, kode, nama, deskripsi,
	to_char(jam_buka, 'HH24:MI'),
	to_char(jam_tutup, 'HH24:MI'),
	kuota_harian, rata_durasi_menit, is_active, created_at, updated_at`

func scanLayanan(row pgx.Row, m *model.Layanan) error {
	return row.Scan(
		&m.ID, &m.Kode, &m.Nama, &m.Deskripsi,
		&m.JamBuka, &m.JamTutup,
		&m.KuotaHarian, &m.RataDurasiMenit, &m.IsActive,
		&m.CreatedAt, &m.UpdatedAt,
	)
}

// List mengambil semua layanan. Kalau hanyaAktif true, hanya layanan aktif.
func (r *LayananRepository) List(ctx context.Context, hanyaAktif bool) ([]model.Layanan, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+layananColumns+`
		FROM layanan
		WHERE ($1 = false OR is_active = true)
		ORDER BY kode`, hanyaAktif)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.Layanan, 0)
	for rows.Next() {
		var m model.Layanan
		if err := scanLayanan(rows, &m); err != nil {
			return nil, err
		}
		list = append(list, m)
	}

	return list, rows.Err()
}

// GetByID mengambil satu layanan. Mengembalikan pgx.ErrNoRows kalau tidak ada.
func (r *LayananRepository) GetByID(ctx context.Context, id int64) (*model.Layanan, error) {
	var m model.Layanan
	err := scanLayanan(r.db.QueryRow(ctx, `
		SELECT `+layananColumns+`
		FROM layanan
		WHERE id = $1`, id), &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create menyimpan layanan baru dan mengisi ID + timestamp pada struct.
func (r *LayananRepository) Create(ctx context.Context, m *model.Layanan) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO layanan (kode, nama, deskripsi, jam_buka, jam_tutup, kuota_harian, rata_durasi_menit, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`,
		m.Kode, m.Nama, m.Deskripsi, m.JamBuka, m.JamTutup,
		m.KuotaHarian, m.RataDurasiMenit, m.IsActive,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// Update mengubah data layanan. Mengembalikan pgx.ErrNoRows kalau ID tidak ada.
func (r *LayananRepository) Update(ctx context.Context, m *model.Layanan) error {
	return r.db.QueryRow(ctx, `
		UPDATE layanan SET
			kode = $2,
			nama = $3,
			deskripsi = $4,
			jam_buka = $5,
			jam_tutup = $6,
			kuota_harian = $7,
			rata_durasi_menit = $8,
			is_active = $9,
			updated_at = now()
		WHERE id = $1
		RETURNING updated_at`,
		m.ID, m.Kode, m.Nama, m.Deskripsi, m.JamBuka, m.JamTutup,
		m.KuotaHarian, m.RataDurasiMenit, m.IsActive,
	).Scan(&m.UpdatedAt)
}

// SoftDelete menonaktifkan layanan (is_active = false), bukan menghapus baris.
func (r *LayananRepository) SoftDelete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE layanan SET is_active = false, updated_at = now()
		WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
