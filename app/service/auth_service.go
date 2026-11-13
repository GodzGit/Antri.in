package service

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/config"
	"github.com/GodzGit/Antri.in/helper"
)

// AuthService menangani logika bisnis dan HTTP handler Fiber untuk seluruh alur autentikasi.
type AuthService struct {
	userRepo  *repository.UserRepository
	roleRepo  *repository.RoleRepository
	tokenRepo *repository.TokenRepository
	txManager *repository.TxManager
	jwtCfg    config.JWTConfig
}

// NewAuthService membuat instance baru AuthService dengan dependensi repository, txManager, dan konfigurasi JWT.
func NewAuthService(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	tokenRepo *repository.TokenRepository,
	txManager *repository.TxManager,
	jwtCfg config.JWTConfig,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		tokenRepo: tokenRepo,
		txManager: txManager,
		jwtCfg:    jwtCfg,
	}
}

// Register mendaftarkan akun baru untuk publik secara mandiri dengan role default 'pengunjung'.
func (s *AuthService) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request body tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// 1. Periksa ketersediaan email
	exists, err := s.userRepo.ExistsByEmail(c.UserContext(), req.Email)
	if err != nil {
		return helper.Internal("gagal memeriksa ketersediaan email")
	}
	if exists {
		return helper.Conflict("email sudah terdaftar")
	}

	// 2. Ambil role 'pengunjung' dari database
	role, err := s.roleRepo.FindByName(c.UserContext(), model.RolePengunjung)
	if err != nil {
		return helper.Internal("gagal mencari role pengunjung")
	}
	if role == nil {
		return helper.Internal("role pengunjung belum dikonfigurasi di sistem")
	}

	// 3. Hash password menggunakan bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return helper.Internal("gagal mengenkripsi password")
	}

	// 4. Buat entitas user baru (role default pengunjung, status aktif)
	newUser := &model.User{
		Nama:         req.Nama,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		RoleID:       role.ID,
		IsActive:     true,
	}

	if err := s.userRepo.Create(c.UserContext(), newUser); err != nil {
		return helper.Internal("gagal membuat user baru")
	}

	// 5. Kembalikan response profil tanpa password_hash
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

	return helper.Created(c, "registrasi berhasil", res)
}

// Login memverifikasi kredensial pengguna dan mengembalikan access token serta refresh token.
func (s *AuthService) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request body tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// 1. Cari user berdasarkan email
	user, err := s.userRepo.FindByEmail(c.UserContext(), req.Email)
	if err != nil {
		return helper.Internal("gagal mencari akun pengguna")
	}
	if user == nil {
		return helper.Unauthorized("email atau password salah")
	}

	// 2. Verifikasi password menggunakan bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return helper.Unauthorized("email atau password salah")
	}

	// 3. Pastikan akun dalam keadaan aktif
	if !user.IsActive {
		return helper.Forbidden("akun Anda dinonaktifkan, silakan hubungi admin")
	}

	// 4. Ambil nama role untuk payload klaim JWT
	role, err := s.roleRepo.FindByID(c.UserContext(), user.RoleID)
	if err != nil || role == nil {
		return helper.Internal("gagal memuat role pengguna")
	}

	// 5. Buat access token
	accessToken, err := helper.GenerateAccessToken(
		user.ID,
		user.Email,
		role.Nama,
		s.jwtCfg.Secret,
		s.jwtCfg.AccessTTL,
	)
	if err != nil {
		return helper.Internal("gagal membuat access token")
	}

	// 6. Buat refresh token mentah dan bentuk hash-nya
	rawRefreshToken, err := helper.GenerateRefreshToken()
	if err != nil {
		return helper.Internal("gagal membuat refresh token")
	}
	tokenHash := helper.HashToken(rawRefreshToken)

	// 7. Simpan hash refresh token ke database
	refreshTokenModel := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.jwtCfg.RefreshTTL),
	}
	if err := s.tokenRepo.Create(c.UserContext(), refreshTokenModel); err != nil {
		return helper.Internal("gagal menyimpan refresh token")
	}

	// 8. Bentuk response autentikasi
	tokenResp := model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtCfg.AccessTTL.Seconds()),
		User: model.UserResponse{
			ID:        user.ID,
			Nama:      user.Nama,
			Email:     user.Email,
			RoleID:    user.RoleID,
			RoleName:  role.Nama,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}

	return helper.Success(c, "login berhasil", tokenResp)
}

// RefreshToken memperbarui access token dan refresh token (rotasi token) menggunakan transaksi atomik.
// Menggunakan SELECT FOR UPDATE untuk mencegah race condition saat dua request bersamaan mencoba
// menggunakan refresh token yang sama — hanya satu yang dapat berhasil melakukan rotasi.
func (s *AuthService) RefreshToken(c *fiber.Ctx) error {
	var req model.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request body tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	tokenHash := helper.HashToken(req.RefreshToken)

	// 1. Periksa cepat sebelum transaksi — memberikan pesan error yang spesifik
	//    jika token tidak ada sama sekali di database (tanpa perlu membuka transaksi).
	existingToken, err := s.tokenRepo.FindByHash(c.UserContext(), tokenHash)
	if err != nil {
		return helper.Internal("gagal memverifikasi refresh token")
	}
	if existingToken == nil {
		return helper.Unauthorized("refresh token tidak ditemukan")
	}
	if existingToken.RevokedAt != nil {
		return helper.Unauthorized("refresh token telah dicabut")
	}
	if existingToken.ExpiresAt.Before(time.Now()) {
		return helper.Unauthorized("refresh token telah kedaluwarsa")
	}

	// 2. Ambil data user pemilik token
	user, err := s.userRepo.FindByID(c.UserContext(), existingToken.UserID)
	if err != nil {
		return helper.Internal("gagal memverifikasi data pengguna")
	}
	if user == nil {
		return helper.NotFound("pengguna tidak ditemukan")
	}
	if !user.IsActive {
		return helper.Forbidden("akun Anda dinonaktifkan")
	}

	// 3. Ambil data role pengguna untuk payload JWT
	role, err := s.roleRepo.FindByID(c.UserContext(), user.RoleID)
	if err != nil || role == nil {
		return helper.Internal("gagal memuat role pengguna")
	}

	// 4. Buat access token baru (di luar transaksi — tidak memerlukan atomisitas DB)
	newAccessToken, err := helper.GenerateAccessToken(
		user.ID,
		user.Email,
		role.Nama,
		s.jwtCfg.Secret,
		s.jwtCfg.AccessTTL,
	)
	if err != nil {
		return helper.Internal("gagal membuat access token baru")
	}

	// 5. Buat refresh token baru
	newRawRefreshToken, err := helper.GenerateRefreshToken()
	if err != nil {
		return helper.Internal("gagal membuat refresh token baru")
	}
	newTokenHash := helper.HashToken(newRawRefreshToken)

	// 6. Rotasi token dalam satu transaksi atomik dengan SELECT FOR UPDATE.
	//
	//    Alur keamanan terhadap dua request bersamaan:
	//    - Request A dan Request B sama-sama melewati pengecekan cepat di atas (token masih aktif).
	//    - Keduanya masuk ke transaksi dan memanggil FindByHashForUpdate.
	//    - PostgreSQL memberi row-level lock pada baris token: hanya satu yang mendapatkan lock,
	//      request lainnya menunggu.
	//    - Request A berhasil: Revoke() mengubah revoked_at, Create() menyimpan token baru, COMMIT.
	//    - Request B mendapat lock: FindByHashForUpdate menemukan token (revoked_at != nil),
	//      Revoke() mengembalikan pgx.ErrNoRows (RowsAffected == 0), transaksi rollback, 401 dikembalikan.
	err = s.txManager.WithTx(c.UserContext(), func(tx pgx.Tx) error {
		txTokenRepo := s.tokenRepo.WithTx(tx)

		// Kunci baris token menggunakan SELECT FOR UPDATE untuk mencegah race condition.
		lockedToken, err := txTokenRepo.FindByHashForUpdate(c.UserContext(), tokenHash)
		if err != nil {
			return err
		}
		if lockedToken == nil {
			return helper.Unauthorized("refresh token tidak ditemukan")
		}
		// Periksa ulang status setelah lock diperoleh (request lain mungkin sudah mencabut token ini).
		if lockedToken.RevokedAt != nil {
			return helper.Unauthorized("refresh token sudah dicabut oleh permintaan lain")
		}
		if lockedToken.ExpiresAt.Before(time.Now()) {
			return helper.Unauthorized("refresh token telah kedaluwarsa")
		}

		// A. Cabut token lama — RowsAffected == 0 berarti token sudah dicabut (kembalikan error).
		if err := txTokenRepo.Revoke(c.UserContext(), tokenHash); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return helper.Unauthorized("refresh token sudah tidak aktif atau telah digunakan")
			}
			return err
		}

		// B. Simpan hash token baru — jika gagal, seluruh transaksi otomatis rollback.
		newToken := &model.RefreshToken{
			UserID:    user.ID,
			TokenHash: newTokenHash,
			ExpiresAt: time.Now().Add(s.jwtCfg.RefreshTTL),
		}
		return txTokenRepo.Create(c.UserContext(), newToken)
	})

	if err != nil {
		if appErr := helper.AsAppError(err); appErr != nil {
			return appErr
		}
		return helper.Internal("gagal memproses transaksi rotasi token")
	}

	// 7. Kembalikan token baru kepada client
	tokenResp := model.TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtCfg.AccessTTL.Seconds()),
		User: model.UserResponse{
			ID:        user.ID,
			Nama:      user.Nama,
			Email:     user.Email,
			RoleID:    user.RoleID,
			RoleName:  role.Nama,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}

	return helper.Success(c, "token berhasil diperbarui", tokenResp)
}

// Logout mencabut refresh token user yang sedang login.
func (s *AuthService) Logout(c *fiber.Ctx) error {
	var req model.LogoutRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format request body tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// Ambil user_id dari Locals (disuntikkan oleh Auth Middleware)
	userIDVal := c.Locals("user_id")
	userID, ok := userIDVal.(int64)
	if !ok || userID == 0 {
		return helper.Unauthorized("otentikasi diperlukan")
	}

	tokenHash := helper.HashToken(req.RefreshToken)

	// Cari token berdasarkan hash
	token, err := s.tokenRepo.FindByHash(c.UserContext(), tokenHash)
	if err != nil {
		return helper.Internal("gagal memverifikasi token")
	}
	if token == nil {
		return helper.NotFound("refresh token tidak ditemukan")
	}

	// Pastikan token adalah milik user yang sedang login (mencegah pencabutan token user lain)
	if token.UserID != userID {
		return helper.Forbidden("akses ditolak: token bukan milik pengguna yang sedang login")
	}

	// Jika token belum dicabut, lakukan pencabutan
	if token.RevokedAt == nil {
		if err := s.tokenRepo.Revoke(c.UserContext(), tokenHash); err != nil {
			return helper.Internal("gagal mencabut refresh token")
		}
	}

	return helper.Success(c, "logout berhasil", nil)
}

// Me mengembalikan profil detail pengguna yang sedang login.
func (s *AuthService) Me(c *fiber.Ctx) error {
	userIDVal := c.Locals("user_id")
	userID, ok := userIDVal.(int64)
	if !ok || userID == 0 {
		return helper.Unauthorized("otentikasi diperlukan")
	}

	user, err := s.userRepo.FindDetailByID(c.UserContext(), userID)
	if err != nil {
		return helper.Internal("gagal mengambil profil pengguna")
	}
	if user == nil {
		return helper.NotFound("pengguna tidak ditemukan")
	}

	return helper.Success(c, "berhasil mengambil profil pengguna", user)
}

