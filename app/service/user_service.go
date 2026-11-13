package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/helper"
)

// UserService menangani logika bisnis dan HTTP handler Fiber untuk manajemen data user.
// Seluruh endpoint di service ini hanya boleh diakses oleh role admin
// (dilindungi oleh middleware.RequirePermission(model.PermissionUserManage)).
type UserService struct {
	userRepo *repository.UserRepository
	roleRepo *repository.RoleRepository
}

// NewUserService membuat instance baru UserService dengan dependensi repository yang diperlukan.
func NewUserService(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
) *UserService {
	return &UserService{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

// List mengembalikan daftar seluruh user menggunakan cursor-based pagination.
//
// Query params:
//   - limit  : jumlah data per halaman (default 10, maksimum 100)
//   - cursor : nilai cursor dari respons halaman sebelumnya (base64-encoded ID)
//
// Response: { success, data: []UserResponse, meta: { next_cursor, limit } }
func (s *UserService) List(c *fiber.Ctx) error {
	limit, cursorID := helper.GetPaginationParams(c)

	// Ambil limit+1 data — satu data ekstra digunakan untuk mendeteksi ketersediaan halaman berikutnya.
	users, err := s.userRepo.FindAll(c.UserContext(), cursorID, limit)
	if err != nil {
		return helper.Internal("gagal mengambil daftar pengguna")
	}

	// Tentukan next_cursor berdasarkan apakah data ekstra tersedia.
	var nextCursor string
	if len(users) > limit {
		// Ada halaman berikutnya: potong data ekstra dan encode ID terakhir sebagai cursor.
		users = users[:limit]
		nextCursor = helper.EncodeCursor(users[len(users)-1].ID)
	}

	return helper.Paginated(c, users, nextCursor, limit)
}

// Detail mengembalikan profil lengkap seorang user berdasarkan ID pada path param `:id`.
//
// Response: { success, message, data: UserResponse }
func (s *UserService) Detail(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return helper.BadRequest("parameter id tidak valid")
	}

	user, err := s.userRepo.FindDetailByID(c.UserContext(), int64(id))
	if err != nil {
		return helper.Internal("gagal mengambil data pengguna")
	}
	if user == nil {
		return helper.NotFound("pengguna tidak ditemukan")
	}

	return helper.Success(c, "berhasil mengambil data pengguna", user)
}

// Create membuat user baru dengan role yang ditentukan oleh admin (POST /api/users).
// Berbeda dengan Register, endpoint ini memperbolehkan admin menentukan role secara bebas.
//
// Request body: { nama, email, password, role_id }
// Response: { success, message, data: UserResponse }
func (s *UserService) Create(c *fiber.Ctx) error {
	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request body tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// 1. Periksa apakah email sudah terdaftar.
	exists, err := s.userRepo.ExistsByEmail(c.UserContext(), req.Email)
	if err != nil {
		return helper.Internal("gagal memeriksa ketersediaan email")
	}
	if exists {
		return helper.Conflict("email sudah terdaftar")
	}

	// 2. Validasi role yang diminta ada di database.
	role, err := s.roleRepo.FindByID(c.UserContext(), req.RoleID)
	if err != nil {
		return helper.Internal("gagal memverifikasi role")
	}
	if role == nil {
		return helper.BadRequest("role tidak ditemukan")
	}

	// 3. Hash password menggunakan bcrypt.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return helper.Internal("gagal mengenkripsi password")
	}

	// 4. Buat entitas user baru (is_active = true secara default).
	newUser := &model.User{
		Nama:         req.Nama,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		RoleID:       req.RoleID,
		IsActive:     true,
	}

	if err := s.userRepo.Create(c.UserContext(), newUser); err != nil {
		return helper.Internal("gagal membuat pengguna baru")
	}

	// 5. Kembalikan data user tanpa password_hash.
	res := model.UserResponse{
		ID:        newUser.ID,
		Nama:      newUser.Nama,
		Email:     newUser.Email,
		RoleID:    newUser.RoleID,
		RoleName:  role.Nama,
		IsActive:  newUser.IsActive,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
	}

	return helper.Created(c, "pengguna berhasil dibuat", res)
}

// UpdateRole memperbarui role seorang user (PATCH /api/users/:id/role).
//
// Request body: { role_id }
// Response: { success, message, data: UserResponse }
func (s *UserService) UpdateRole(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return helper.BadRequest("parameter id tidak valid")
	}

	var req model.UpdateUserRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request body tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// Validasi role yang diminta ada di database.
	role, err := s.roleRepo.FindByID(c.UserContext(), req.RoleID)
	if err != nil {
		return helper.Internal("gagal memverifikasi role")
	}
	if role == nil {
		return helper.BadRequest("role tidak ditemukan")
	}

	// Lakukan update di database.
	if err := s.userRepo.UpdateRole(c.UserContext(), int64(id), req.RoleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		return helper.Internal("gagal memperbarui role pengguna")
	}

	// Ambil data terbaru untuk response.
	updatedUser, err := s.userRepo.FindDetailByID(c.UserContext(), int64(id))
	if err != nil {
		return helper.Internal("gagal mengambil data pengguna setelah diperbarui")
	}

	return helper.Success(c, "role pengguna berhasil diperbarui", updatedUser)
}

// UpdateStatus mengaktifkan atau menonaktifkan akun user (PATCH /api/users/:id/status).
//
// Request body: { is_active: true|false }
// Response: { success, message, data: UserResponse }
func (s *UserService) UpdateStatus(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return helper.BadRequest("parameter id tidak valid")
	}

	var req model.UpdateUserStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request body tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// Lakukan update status di database.
	if err := s.userRepo.UpdateStatus(c.UserContext(), int64(id), *req.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		return helper.Internal("gagal memperbarui status pengguna")
	}

	// Ambil data terbaru untuk response.
	updatedUser, err := s.userRepo.FindDetailByID(c.UserContext(), int64(id))
	if err != nil {
		return helper.Internal("gagal mengambil data pengguna setelah diperbarui")
	}

	statusMsg := "pengguna berhasil diaktifkan"
	if !*req.IsActive {
		statusMsg = "pengguna berhasil dinonaktifkan"
	}

	return helper.Success(c, statusMsg, updatedUser)
}

