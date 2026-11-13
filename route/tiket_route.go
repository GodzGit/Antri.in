package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
)

// RegisterTiketRoutes mendaftarkan endpoint tiket & antrean.
//
// TODO(auth): setelah middleware Auth + RequirePermission dari Fase 2 masuk:
//
//	tiket:create    -> POST /layanan/:id/tiket
//	tiket:read:own  -> GET /tiket/saya, GET /tiket/:id, DELETE /tiket/:id
//	antrean:read    -> GET /layanan/:id/antrean
//	loket:operate   -> POST /loket/:id/panggil, selesai, lewati, panggil-ulang
func RegisterTiketRoutes(api fiber.Router, svc *service.TiketService) {
	// Pengunjung
	api.Post("/layanan/:id/tiket", svc.AmbilNomor)
	api.Get("/layanan/:id/antrean", svc.Papan)

	// /tiket/saya harus didaftarkan sebelum /tiket/:id
	api.Get("/tiket/saya", svc.Saya)
	api.Get("/tiket/:id", svc.Detail)
	api.Delete("/tiket/:id", svc.Batal)

	// Petugas
	api.Post("/loket/:id/panggil", svc.Panggil)
	api.Post("/tiket/:id/selesai", svc.Selesai)
	api.Post("/tiket/:id/lewati", svc.Lewati)
	api.Post("/tiket/:id/panggil-ulang", svc.PanggilUlang)
}
