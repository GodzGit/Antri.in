package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
)

// UserRepository mengelola akses data ke tabel users di PostgreSQL.
type UserRepository struct {
	db DBTX
}

// NewUserRepository membuat instance baru UserRepository dengan dependensi DBTX.
func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{db: db}
}

// WithTx mengembalikan salinan UserRepository dengan koneksi transaksi tertentu.
func (r *UserRepository) WithTx(tx DBTX) *UserRepository {
	return &UserRepository{db: tx}
}

// Create menambahkan pengguna baru ke tabel users dan mengembalikan id, created_at, dan updated_at.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (nama, email, password_hash, role_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		user.Nama,
		user.Email,
		user.PasswordHash,
		user.RoleID,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("gagal membuat user: %w", err)
	}

	return nil
}

// FindByEmail mencari user berdasarkan alamat email (digunakan pada alur login dan verifikasi duplikasi).
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, nama, email, password_hash, role_id, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Nama,
		&user.Email,
		&user.PasswordHash,
		&user.RoleID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari user berdasarkan email: %w", err)
	}

	return &user, nil
}

// FindByID mencari user berdasarkan ID.
func (r *UserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, nama, email, password_hash, role_id, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Nama,
		&user.Email,
		&user.PasswordHash,
		&user.RoleID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari user berdasarkan id: %w", err)
	}

	return &user, nil
}

// FindDetailByID mengambil detail profil user lengkap dengan nama rolenya (JOIN dengan tabel roles).
func (r *UserRepository) FindDetailByID(ctx context.Context, id int64) (*model.UserResponse, error) {
	query := `
		SELECT u.id, u.nama, u.email, u.role_id, r.nama AS role_name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`

	var resp model.UserResponse
	err := r.db.QueryRow(ctx, query, id).Scan(
		&resp.ID,
		&resp.Nama,
		&resp.Email,
		&resp.RoleID,
		&resp.RoleName,
		&resp.IsActive,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mengambil detail user: %w", err)
	}

	return &resp, nil
}

// FindAll mengambil daftar pengguna menggunakan cursor-based pagination (JOIN dengan tabel roles).
// Method ini mengambil hingga limit + 1 data (WHERE u.id > $1 ORDER BY u.id ASC LIMIT $2)
// untuk memungkinkan Service Layer mendeteksi ketersediaan halaman berikutnya secara efisien.
//
// Kontrak Service Layer:
// - Jika jumlah data yang dikembalikan > limit:
//   - Ada halaman berikutnya (has_more = true).
//   - Service memotong elemen ekstra (users = users[:limit]).
//   - next_cursor dibuat dari ID data terakhir yang ditampilkan (users[len-1].ID).
//
// - Jika jumlah data <= limit:
//   - Tidak ada halaman berikutnya (next_cursor = "").
func (r *UserRepository) FindAll(ctx context.Context, cursorID int64, limit int) ([]model.UserResponse, error) {
	query := `
		SELECT u.id, u.nama, u.email, u.role_id, r.nama AS role_name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id > $1
		ORDER BY u.id ASC
		LIMIT $2
	`

	fetchLimit := limit + 1
	rows, err := r.db.Query(ctx, query, cursorID, fetchLimit)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar user: %w", err)
	}
	defer rows.Close()

	var users []model.UserResponse
	for rows.Next() {
		var u model.UserResponse
		if err := rows.Scan(
			&u.ID,
			&u.Nama,
			&u.Email,
			&u.RoleID,
			&u.RoleName,
			&u.IsActive,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan data user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan saat iterasi user: %w", err)
	}

	return users, nil
}

// UpdateRole memperbarui role seorang pengguna (PATCH /api/users/:id/role).
func (r *UserRepository) UpdateRole(ctx context.Context, id int64, roleID int64) error {
	query := `
		UPDATE users
		SET role_id = $1, updated_at = NOW()
		WHERE id = $2
	`

	tag, err := r.db.Exec(ctx, query, roleID, id)
	if err != nil {
		return fmt.Errorf("gagal memperbarui role user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// UpdateStatus mengaktifkan atau menonaktifkan akun pengguna (PATCH /api/users/:id/status).
func (r *UserRepository) UpdateStatus(ctx context.Context, id int64, isActive bool) error {
	query := `
		UPDATE users
		SET is_active = $1, updated_at = NOW()
		WHERE id = $2
	`

	tag, err := r.db.Exec(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("gagal memperbarui status user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// ExistsByEmail memeriksa secara efisien apakah alamat email sudah terdaftar.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("gagal memeriksa keberadaan email: %w", err)
	}

	return exists, nil
}
