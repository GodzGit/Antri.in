package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/service"
)

// RegisterTiketRoutes mendaftarkan endpoint tiket & antrean.
//
// Akses (panduan dokumen bagian 4):
//
//	POST /layanan/:id/tiket    -> tiket:create
//	GET  /layanan/:id/antrean  -> antrean:read
//	GET  /tiket/saya           -> tiket:read:own
//	GET  /tiket/:id            -> tiket:read:own
//	DELETE /tiket/:id          -> tiket:read:own
//	POST /loket/:id/panggil, /tiket/:id/selesai, /lewati, /panggil-ulang -> loket:operate
//
// Middleware dipasang per-route (bukan api.Use) supaya tidak ikut
// melindungi route milik anggota lain yang didaftarkan setelah ini.
func RegisterTiketRoutes(api fiber.Router, svc *service.TiketService, auth fiber.Handler, can func(permission string) fiber.Handler) {
	// Pengunjung
	api.Post("/layanan/:id/tiket", auth, can(model.PermissionTiketCreate), svc.AmbilNomor)
	api.Get("/layanan/:id/antrean", auth, can(model.PermissionAntreanRead), svc.Papan)

	// /tiket/saya harus didaftarkan sebelum /tiket/:id
	api.Get("/tiket/saya", auth, can(model.PermissionTiketReadOwn), svc.Saya)
	api.Get("/tiket/:id", auth, can(model.PermissionTiketReadOwn), svc.Detail)
	api.Delete("/tiket/:id", auth, can(model.PermissionTiketReadOwn), svc.Batal)

	// Petugas
	api.Post("/loket/:id/panggil", auth, can(model.PermissionLoketOperate), svc.Panggil)
	api.Post("/tiket/:id/selesai", auth, can(model.PermissionLoketOperate), svc.Selesai)
	api.Post("/tiket/:id/lewati", auth, can(model.PermissionLoketOperate), svc.Lewati)
	api.Post("/tiket/:id/panggil-ulang", auth, can(model.PermissionLoketOperate), svc.PanggilUlang)
}
