
package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/helper"
)

// Deps menampung semua service yang dibutuhkan oleh route aplikasi.
type Deps struct {
	Auth    *service.AuthService
	User    *service.UserService
	Layanan *service.LayananService
	Loket   *service.LoketService
	Tiket   *service.TiketService
	// Auth, Notifikasi, ... menyusul.
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
	// =================================

	app.Get("/", func(ctx *fiber.Ctx) error {
		return helper.Success(ctx, "ANTRI.IN API", fiber.Map{
			"health": "/health",
		})
	})
}
