package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/helper"
)

// Deps menampung semua service yang dibutuhkan router.
// Setiap anggota cukup menambah SATU field di sini (lewat PR masing-masing),
// lalu mengisinya di main.go.
type Deps struct {
	Layanan *service.LayananService
	Loket   *service.LoketService
	Tiket   *service.TiketService
	// Auth, Notifikasi, ... menyusul.
}

// Setup mendaftarkan semua route aplikasi.
// Setiap anggota menambahkan SATU baris pemanggilan RegisterXxxRoutes di sini
// pada bagian "SLOT PENDAFTARAN ROUTE" di bawah.
func Setup(app *fiber.App, deps Deps) {
	app.Get("/health", func(ctx *fiber.Ctx) error {
		return helper.Success(ctx, "server sehat", fiber.Map{
			"status": "ok",
		})
	})

	api := app.Group("/api")

	// ===== SLOT PENDAFTARAN ROUTE =====
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
			"nama":   "ANTRI.IN",
			"versi":  "0.1.0",
			"docs":   "/swagger",
			"health": "/health",
		})
	})
}
