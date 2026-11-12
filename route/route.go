package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/helper"
)

// Setup mendaftarkan semua route aplikasi.
// Setiap anggota menambahkan SATU baris pemanggilan RegisterXxxRoutes di sini
// pada bagian "SLOT PENDAFTARAN ROUTE" di bawah.
//
// Contoh (nanti, setelah service-nya ada):
//   route.RegisterAuthRoutes(api, authService)
//   route.RegisterLayananRoutes(api, layananService, authMiddleware)
//   route.RegisterNotifikasiRoutes(api, notifikasiService, authMiddleware)
func Setup(app *fiber.App) {
	app.Get("/health", func(ctx *fiber.Ctx) error {
		return helper.Success(ctx, "server sehat", fiber.Map{
			"status": "ok",
		})
	})

	api := app.Group("/api")
	_ = api // sengaja belum dipakai; anggota akan menambah RegisterXxxRoutes di bawah.

	// ===== SLOT PENDAFTARAN ROUTE =====
	// ...
	// =================================

	app.Get("/", func(ctx *fiber.Ctx) error {
		return helper.Success(ctx, "ANTRI.IN API", fiber.Map{
			"nama":   "ANTRI.IN",
			"versi":  "0.1.0",
			"docs":   "/swagger",
			"health": "/health",
		})
	})
}