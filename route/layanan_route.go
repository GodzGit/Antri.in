package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/service"
)

// RegisterLayananRoutes mendaftarkan endpoint layanan.
//
// Akses (panduan dokumen bagian 4):
//
//	GET /           -> antrean:read
//	GET /:id        -> antrean:read
//	POST/PUT/DELETE -> layanan:manage
func RegisterLayananRoutes(api fiber.Router, svc *service.LayananService, auth fiber.Handler, can func(permission string) fiber.Handler) {
	layanan := api.Group("/layanan", auth)

	layanan.Get("/", can(model.PermissionAntreanRead), svc.List)
	layanan.Get("/:id", can(model.PermissionAntreanRead), svc.Get)

	layanan.Post("/", can(model.PermissionLayananManage), svc.Create)
	layanan.Put("/:id", can(model.PermissionLayananManage), svc.Update)
	layanan.Delete("/:id", can(model.PermissionLayananManage), svc.Delete)
}
