package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/middleware"
)

// RegisterUserRoutes mendaftarkan seluruh endpoint manajemen user pada group /api/users.
// Seluruh endpoint memerlukan autentikasi (JWT) dan permission "user:manage" (khusus admin).
//
// Endpoint yang tersedia:
//   - GET    /api/users          → daftar semua user (cursor pagination: ?limit=&cursor=)
//   - GET    /api/users/:id      → detail satu user berdasarkan ID
//   - POST   /api/users          → buat user baru dengan role yang ditentukan admin
//   - PATCH  /api/users/:id/role → ubah role seorang user
//   - PATCH  /api/users/:id/status → aktifkan atau nonaktifkan akun user
//
// Cara memanggil dari route/route.go (koordinasikan dengan Bagus):
//
//	route.RegisterUserRoutes(api, userSvc, cfg.JWT.Secret)
func RegisterUserRoutes(api fiber.Router, svc *service.UserService, jwtSecret string) {
	// Middleware chain yang berlaku untuk semua endpoint user:
	// 1. Auth(jwtSecret)                                   — validasi Bearer token, isi Locals
	// 2. RequirePermission(PermissionUserManage)            — pastikan role adalah admin
	authMiddleware := middleware.Auth(jwtSecret)
	permMiddleware := middleware.RequirePermission(model.PermissionUserManage)

	users := api.Group("/users", authMiddleware, permMiddleware)

	users.Get("/", svc.List)
	users.Get("/:id", svc.Detail)
	users.Post("/", svc.Create)
	users.Patch("/:id/role", svc.UpdateRole)
	users.Patch("/:id/status", svc.UpdateStatus)
}

