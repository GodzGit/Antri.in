package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
)

// RoleRepository mengelola akses data ke tabel roles, permissions, dan role_permissions di PostgreSQL.
type RoleRepository struct {
	db DBTX
}

// NewRoleRepository membuat instance baru RoleRepository dengan dependensi DBTX.
func NewRoleRepository(db DBTX) *RoleRepository {
	return &RoleRepository{db: db}
}

// WithTx mengembalikan salinan RoleRepository dengan koneksi transaksi tertentu.
func (r *RoleRepository) WithTx(tx DBTX) *RoleRepository {
	return &RoleRepository{db: tx}
}

// FindByID mencari role berdasarkan ID.
func (r *RoleRepository) FindByID(ctx context.Context, id int64) (*model.Role, error) {
	query := `SELECT id, nama FROM roles WHERE id = $1`

	var role model.Role
	err := r.db.QueryRow(ctx, query, id).Scan(&role.ID, &role.Nama)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari role berdasarkan id: %w", err)
	}

	return &role, nil
}

// FindByName mencari role berdasarkan nama (contoh: "admin", "petugas", "pengunjung").
func (r *RoleRepository) FindByName(ctx context.Context, nama string) (*model.Role, error) {
	query := `SELECT id, nama FROM roles WHERE nama = $1`

	var role model.Role
	err := r.db.QueryRow(ctx, query, nama).Scan(&role.ID, &role.Nama)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari role berdasarkan nama: %w", err)
	}

	return &role, nil
}

// FindAll mengambil seluruh daftar role yang tersedia.
func (r *RoleRepository) FindAll(ctx context.Context) ([]model.Role, error) {
	query := `SELECT id, nama FROM roles ORDER BY id ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar role: %w", err)
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.Nama); err != nil {
			return nil, fmt.Errorf("gagal scan data role: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan saat iterasi role: %w", err)
	}

	return roles, nil
}

// Create menambahkan data role baru (digunakan oleh seeder).
func (r *RoleRepository) Create(ctx context.Context, role *model.Role) error {
	query := `INSERT INTO roles (nama) VALUES ($1) RETURNING id`

	err := r.db.QueryRow(ctx, query, role.Nama).Scan(&role.ID)
	if err != nil {
		return fmt.Errorf("gagal membuat role: %w", err)
	}

	return nil
}

// FindPermissionByName mencari permission berdasarkan nama (contoh: "tiket:create").
func (r *RoleRepository) FindPermissionByName(ctx context.Context, nama string) (*model.Permission, error) {
	query := `SELECT id, nama FROM permissions WHERE nama = $1`

	var perm model.Permission
	err := r.db.QueryRow(ctx, query, nama).Scan(&perm.ID, &perm.Nama)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari permission berdasarkan nama: %w", err)
	}

	return &perm, nil
}

// CreatePermission menambahkan permission baru (digunakan oleh seeder).
func (r *RoleRepository) CreatePermission(ctx context.Context, perm *model.Permission) error {
	query := `INSERT INTO permissions (nama) VALUES ($1) RETURNING id`

	err := r.db.QueryRow(ctx, query, perm.Nama).Scan(&perm.ID)
	if err != nil {
		return fmt.Errorf("gagal membuat permission: %w", err)
	}

	return nil
}

// AssignPermission menghubungkan role dan permission pada tabel pivot role_permissions.
func (r *RoleRepository) AssignPermission(ctx context.Context, roleID, permissionID int64) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("gagal menghubungkan role dan permission: %w", err)
	}

	return nil
}

// GetPermissionsByRoleID mengambil semua nama permission yang dimiliki oleh suatu role.
func (r *RoleRepository) GetPermissionsByRoleID(ctx context.Context, roleID int64) ([]string, error) {
	query := `
		SELECT p.nama
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.nama ASC
	`

	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil permission untuk role: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var permName string
		if err := rows.Scan(&permName); err != nil {
			return nil, fmt.Errorf("gagal scan permission: %w", err)
		}
		permissions = append(permissions, permName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan saat iterasi permission: %w", err)
	}

	return permissions, nil
}

// HasPermission memeriksa apakah suatu role memiliki permission tertentu di database.
func (r *RoleRepository) HasPermission(ctx context.Context, roleID int64, permissionName string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM role_permissions rp
			JOIN permissions p ON rp.permission_id = p.id
			WHERE rp.role_id = $1 AND p.nama = $2
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, roleID, permissionName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("gagal memeriksa hak akses: %w", err)
	}

	return exists, nil
}

