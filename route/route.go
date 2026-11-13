
package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/helper"
	"github.com/GodzGit/Antri.in/middleware"
)

// Deps menampung semua service yang dibutuhkan oleh route aplikasi.
type Deps struct {
	Auth    *service.AuthService
	User    *service.UserService
	Layanan *service.LayananService
	Loket   *service.LoketService
	Tiket   *service.TiketService

	Notifikasi *service.NotifikasiService
	Estimasi   *service.EstimasiService
	Riwayat    *service.RiwayatService
	Statistik  *service.StatistikService
}

// Setup mendaftarkan semua route aplikasi.
func Setup(app *fiber.App, deps Deps, jwtSecret string) {
	app.Get("/health", func(ctx *fiber.Ctx) error {
		return helper.Success(ctx, "server sehat", fiber.Map{
			"status": "ok",
		})
	})

	api := app.Group("/api")

	// ===== SLOT PENDAFTARAN ROUTE =====
	RegisterAuthRoutes(api, deps.Auth, jwtSecret)
	RegisterUserRoutes(api, deps.User, jwtSecret)

	if deps.Layanan != nil {
		RegisterLayananRoutes(api, deps.Layanan)
	}

	if deps.Loket != nil {
		RegisterLoketRoutes(api, deps.Loket)
	}
	if deps.Tiket != nil {
		RegisterTiketRoutes(api, deps.Tiket)
	}

	auth := middleware.Auth(jwtSecret)
	if deps.Notifikasi != nil {
		RegisterNotifikasiRoutes(api, deps.Notifikasi, auth, middleware.RequirePermission)
	}
	if deps.Estimasi != nil && deps.Riwayat != nil && deps.Statistik != nil {
		RegisterStatistikRoutes(api, deps.Estimasi, deps.Riwayat, deps.Statistik, auth, middleware.RequirePermission)
	}
	RegisterSwaggerRoutes(app)
	// =================================

	app.Get("/", func(ctx *fiber.Ctx) error {
		return helper.Success(ctx, "ANTRI.IN API", fiber.Map{
			"health": "/health",
		})
	})
}
