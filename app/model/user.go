package model

import "time"

// User merepresentasikan entitas tabel 'users' di database.
type User struct {
	ID           int64     `json:"id"`
	Nama         string    `json:"nama"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Disembunyikan agar hash password tidak pernah bocor ke JSON response
	RoleID       int64     `json:"role_id"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateUserRequest DTO untuk request pembuatan user baru oleh Admin (POST /api/users).
type CreateUserRequest struct {
	Nama     string `json:"nama" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	RoleID   int64  `json:"role_id" validate:"required"`
}

// UpdateUserRoleRequest DTO untuk request perubahan role user oleh Admin (PATCH /api/users/:id/role).
type UpdateUserRoleRequest struct {
	RoleID int64 `json:"role_id" validate:"required"`
}

// UpdateUserStatusRequest DTO untuk request perubahan status aktif user oleh Admin (PATCH /api/users/:id/status).
// Menggunakan pointer *bool agar nilai false tidak dianggap kosong oleh validator saat request dikirim.
type UpdateUserStatusRequest struct {
	IsActive *bool `json:"is_active" validate:"required"`
}

// UserResponse DTO untuk response data user yang aman dikembalikan ke client.
type UserResponse struct {
	ID        int64     `json:"id"`
	Nama      string    `json:"nama"`
	Email     string    `json:"email"`
	RoleID    int64     `json:"role_id"`
	RoleName  string    `json:"role_name,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

