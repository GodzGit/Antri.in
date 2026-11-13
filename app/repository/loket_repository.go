package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
)

// LoketRepository menangani semua query ke tabel loket.
type LoketRepository struct {
	db DBTX
}

func NewLoketRepository(db DBTX) *LoketRepository {
	return &LoketRepository{db: db}
}

// loketColumns dipakai di semua SELECT. Kolom diberi prefix supaya tidak
// ambigu saat JOIN dengan layanan/users.
const loketColumns = `
	loket.id, loket.layanan_id, loket.nama, loket.petugas_id, loket.status,
	loket.created_at, loket.updated_at,
	ly.nama AS layanan,
	COALESCE(u.nama, '') AS petugas`

const loketJoins = `
	FROM loket
	JOIN layanan ly ON ly.id = loket.layanan_id
	LEFT JOIN users u ON u.id = loket.petugas_id`

func scanLoket(row pgx.Row, l *model.Loket) error {
	return row.Scan(
		&l.ID, &l.LayananID, &l.Nama, &l.PetugasID, &l.Status,
		&l.CreatedAt, &l.UpdatedAt, &l.Layanan, &l.Petugas,
	)
}

// List mengambil semua loket (dengan nama layanan & petugas).
func (r *LoketRepository) List(ctx context.Context) ([]model.Loket, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+loketColumns+` `+loketJoins+`
		ORDER BY loket.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.Loket, 0)
	for rows.Next() {
		var l model.Loket
		if err := scanLoket(rows, &l); err != nil {
			return nil, err
		}
		list = append(list, l)
	}

	return list, rows.Err()
}

// GetByID mengambil satu loket. Mengembalikan pgx.ErrNoRows kalau tidak ada.
func (r *LoketRepository) GetByID(ctx context.Context, id int64) (*model.Loket, error) {
	var l model.Loket
	err := scanLoket(r.db.QueryRow(ctx, `
		SELECT `+loketColumns+` `+loketJoins+`
		WHERE loket.id = $1`, id), &l)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// Create menyimpan loket baru; status awal selalu "tutup".
func (r *LoketRepository) Create(ctx context.Context, l *model.Loket) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO loket (layanan_id, nama, status)
		VALUES ($1, $2, 'tutup')
		RETURNING id, status, created_at, updated_at`,
		l.LayananID, l.Nama,
	).Scan(&l.ID, &l.Status, &l.CreatedAt, &l.UpdatedAt)
}

// Update mengubah layanan, nama, dan petugas loket.
// Mengembalikan pgx.ErrNoRows kalau ID tidak ada.
func (r *LoketRepository) Update(ctx context.Context, l *model.Loket) error {
	return r.db.QueryRow(ctx, `
		UPDATE loket SET
			layanan_id = $2,
			nama = $3,
			petugas_id = $4,
			updated_at = now()
		WHERE id = $1
		RETURNING updated_at`,
		l.ID, l.LayananID, l.Nama, l.PetugasID,
	).Scan(&l.UpdatedAt)
}

// Delete menghapus loket. Mengembalikan pgx.ErrNoRows kalau ID tidak ada.
func (r *LoketRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM loket WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// SetStatus membuka atau menutup loket.
func (r *LoketRepository) SetStatus(ctx context.Context, id int64, status string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE loket SET status = $2, updated_at = now()
		WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
