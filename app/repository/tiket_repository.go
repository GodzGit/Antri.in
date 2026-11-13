package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
)

// TiketRepository menangani semua query ke tabel tiket, antrean_harian,
// dan pembacaan ringan tabel loket.
//
// Method yang ikut dalam transaksi menerima parameter `db DBTX` eksplisit;
// kalau nil, dipakai koneksi default repository ini.
type TiketRepository struct {
	db DBTX
}

func NewTiketRepository(db DBTX) *TiketRepository {
	return &TiketRepository{db: db}
}

// pickDB memilih koneksi transaksi kalau diberikan, kalau tidak pakai default.
func (r *TiketRepository) pickDB(db DBTX) DBTX {
	if db != nil {
		return db
	}
	return r.db
}

const tiketColumns = `
	id, layanan_id, user_id, loket_id, tanggal, nomor, kode, status,
	diambil_at, dipanggil_at, selesai_at, version`

func scanTiket(row pgx.Row, t *model.Tiket) error {
	return row.Scan(
		&t.ID, &t.LayananID, &t.UserID, &t.LoketID, &t.Tanggal, &t.Nomor,
		&t.Kode, &t.Status, &t.DiambilAt, &t.DipanggilAt, &t.SelesaiAt, &t.Version,
	)
}

// CekTiketAktif memeriksa apakah user masih punya tiket aktif
// (menunggu/dipanggil) untuk layanan tersebut pada tanggal itu.
// Ini mencegah pelanggaran partial unique index idx_tiket_aktif_per_user_layanan.
func (r *TiketRepository) CekTiketAktif(ctx context.Context, db DBTX, userID, layananID int64, tanggal time.Time) (bool, error) {
	var ada bool
	err := r.pickDB(db).QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM tiket
			WHERE user_id = $1
			  AND layanan_id = $2
			  AND tanggal = $3::date
			  AND status IN ('menunggu', 'dipanggil')
		)`, userID, layananID, tanggal).Scan(&ada)
	if err != nil {
		return false, err
	}
	return ada, nil
}

// AmbilNomor menaikkan nomor antrean harian secara atomik dan mengembalikan
// nomor baru. Aman untuk request bersamaan karena memakai ON CONFLICT DO UPDATE.
func (r *TiketRepository) AmbilNomor(ctx context.Context, db DBTX, layananID int64, tanggal time.Time) (int, error) {
	var nomor int
	err := r.pickDB(db).QueryRow(ctx, `
		INSERT INTO antrean_harian (layanan_id, tanggal, nomor_terakhir)
		VALUES ($1, $2::date, 1)
		ON CONFLICT (layanan_id, tanggal)
		DO UPDATE SET nomor_terakhir = antrean_harian.nomor_terakhir + 1
		RETURNING nomor_terakhir`, layananID, tanggal).Scan(&nomor)
	if err != nil {
		return 0, err
	}
	return nomor, nil
}

// Insert menyimpan tiket baru dan mengisi ID + timestamp pada struct.
func (r *TiketRepository) Insert(ctx context.Context, db DBTX, t *model.Tiket) error {
	return r.pickDB(db).QueryRow(ctx, `
		INSERT INTO tiket (layanan_id, user_id, loket_id, tanggal, nomor, kode, status, diambil_at)
		VALUES ($1, $2, $3, $4::date, $5, $6, $7, now())
		RETURNING id, diambil_at, version`,
		t.LayananID, t.UserID, t.LoketID, t.Tanggal, t.Nomor, t.Kode, t.Status,
	).Scan(&t.ID, &t.DiambilAt, &t.Version)
}

// GetByID mengambil satu tiket. Mengembalikan pgx.ErrNoRows kalau tidak ada.
func (r *TiketRepository) GetByID(ctx context.Context, id int64) (*model.Tiket, error) {
	var t model.Tiket
	if err := scanTiket(r.db.QueryRow(ctx, `
		SELECT `+tiketColumns+` FROM tiket WHERE id = $1`, id), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// ListByUser mengambil semua tiket milik satu user (terbaru dulu)
// lengkap dengan nama layanan, nama loket, dan posisi antrean.
func (r *TiketRepository) ListByUser(ctx context.Context, userID int64) ([]model.TiketSaya, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+tiketColumns+`,
			ly.nama,
			COALESCE(lo.nama, '') AS loket,
			CASE WHEN tiket.status = 'menunggu' THEN (
				SELECT count(*) FROM tiket t2
				WHERE t2.layanan_id = tiket.layanan_id
				  AND t2.tanggal = tiket.tanggal
				  AND t2.status = 'menunggu'
				  AND t2.nomor < tiket.nomor
			) ELSE 0 END AS posisi
		FROM tiket
		JOIN layanan ly ON ly.id = tiket.layanan_id
		LEFT JOIN loket lo ON lo.id = tiket.loket_id
		WHERE tiket.user_id = $1
		ORDER BY tiket.diambil_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.TiketSaya, 0)
	for rows.Next() {
		var t model.TiketSaya
		if err := rows.Scan(
			&t.ID, &t.LayananID, &t.UserID, &t.LoketID, &t.Tanggal, &t.Nomor,
			&t.Kode, &t.Status, &t.DiambilAt, &t.DipanggilAt, &t.SelesaiAt, &t.Version,
			&t.Layanan, &t.Loket, &t.Posisi,
		); err != nil {
			return nil, err
		}
		list = append(list, t)
	}

	return list, rows.Err()
}

// Posisi menghitung berapa tiket menunggu yang nomornya lebih kecil
// (artinya berapa orang lagi di depan kita).
func (r *TiketRepository) Posisi(ctx context.Context, layananID int64, tanggal time.Time, nomor int) (int, error) {
	var posisi int
	err := r.db.QueryRow(ctx, `
		SELECT count(*) FROM tiket
		WHERE layanan_id = $1
		  AND tanggal = $2::date
		  AND status = 'menunggu'
		  AND nomor < $3`, layananID, tanggal, nomor).Scan(&posisi)
	if err != nil {
		return 0, err
	}
	return posisi, nil
}

// Papan mengambil tiket yang sedang dipanggil dan daftar yang masih menunggu.
func (r *TiketRepository) Papan(ctx context.Context, layananID int64, tanggal time.Time) (*model.PapanAntrean, error) {
	papan := &model.PapanAntrean{
		SedangDipanggil: make([]model.Tiket, 0),
		Menunggu:        make([]model.Tiket, 0),
	}

	rows, err := r.db.Query(ctx, `
		SELECT `+tiketColumns+`
		FROM tiket
		WHERE layanan_id = $1 AND tanggal = $2::date AND status IN ('dipanggil', 'menunggu')
		ORDER BY nomor`, layananID, tanggal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t model.Tiket
		if err := scanTiket(rows, &t); err != nil {
			return nil, err
		}
		if t.Status == model.StatusDipanggil {
			papan.SedangDipanggil = append(papan.SedangDipanggil, t)
		} else {
			papan.Menunggu = append(papan.Menunggu, t)
		}
	}

	return papan, rows.Err()
}

// PanggilBerikutnya mengambil tiket menunggu dengan nomor terkecil dan
// menandainya dipanggil. Memakai FOR UPDATE SKIP LOCKED supaya dua loket
// yang memanggil bersamaan tidak mendapat tiket yang sama.
func (r *TiketRepository) PanggilBerikutnya(ctx context.Context, db DBTX, layananID, loketID int64, tanggal time.Time) (*model.Tiket, error) {
	conn := r.pickDB(db)

	var t model.Tiket
	err := scanTiket(conn.QueryRow(ctx, `
		SELECT `+tiketColumns+`
		FROM tiket
		WHERE layanan_id = $1 AND tanggal = $2::date AND status = 'menunggu'
		ORDER BY nomor
		FOR UPDATE SKIP LOCKED
		LIMIT 1`, layananID, tanggal), &t)
	if err != nil {
		return nil, err
	}

	var dipanggilAt time.Time
	err = conn.QueryRow(ctx, `
		UPDATE tiket
		SET status = 'dipanggil', loket_id = $2, dipanggil_at = now(), version = version + 1
		WHERE id = $1
		RETURNING dipanggil_at, version`, t.ID, loketID).Scan(&dipanggilAt, &t.Version)
	if err != nil {
		return nil, err
	}

	t.Status = model.StatusDipanggil
	t.LoketID = &loketID
	t.DipanggilAt = &dipanggilAt

	return &t, nil
}

// UbahStatus mengubah status tiket dengan syarat status lamanya cocok,
// sekaligus mengisi timestamp yang relevan. Mengembalikan false kalau
// tiket tidak ada atau statusnya sudah berubah (transisi tidak valid).
func (r *TiketRepository) UbahStatus(ctx context.Context, db DBTX, id int64, dari, ke string) (bool, error) {
	tag, err := r.pickDB(db).Exec(ctx, `
		UPDATE tiket SET
			status = $3,
			dipanggil_at = CASE WHEN $3 = 'dipanggil' THEN now() ELSE dipanggil_at END,
			selesai_at  = CASE WHEN $3 = 'selesai'  THEN now() ELSE selesai_at  END,
			version = version + 1
		WHERE id = $1 AND status = $2`, id, dari, ke)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// PanggilUlang memanggil kembali tiket yang sebelumnya dipanggil/dilewati.
// loketID boleh nil: kalau nil, loket lama dipertahankan.
func (r *TiketRepository) PanggilUlang(ctx context.Context, db DBTX, id int64, loketID *int64) (bool, error) {
	tag, err := r.pickDB(db).Exec(ctx, `
		UPDATE tiket SET
			status = 'dipanggil',
			loket_id = COALESCE($2, loket_id),
			dipanggil_at = now(),
			version = version + 1
		WHERE id = $1 AND status IN ('dipanggil', 'dilewati')`, id, loketID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// Batal membatalkan tiket milik user, hanya kalau statusnya masih menunggu.
func (r *TiketRepository) Batal(ctx context.Context, id, userID int64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE tiket SET status = 'batal', version = version + 1
		WHERE id = $1 AND user_id = $2 AND status = 'menunggu'`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// GetLoketInfo mengambil data loket yang dibutuhkan saat memanggil tiket.
func (r *TiketRepository) GetLoketInfo(ctx context.Context, loketID int64) (*model.LoketInfo, error) {
	var l model.LoketInfo
	err := r.db.QueryRow(ctx, `
		SELECT id, layanan_id, nama, petugas_id, status
		FROM loket WHERE id = $1`, loketID).
		Scan(&l.ID, &l.LayananID, &l.Nama, &l.PetugasID, &l.Status)
	if err != nil {
		return nil, err
	}
	return &l, nil
}
