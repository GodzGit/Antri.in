package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/service"
)

// RegisterLoketRoutes mendaftarkan endpoint loket.
//
// Akses (panduan dokumen bagian 4):
//
//	GET/POST/PUT/DELETE -> loket:manage
//	PATCH /:id/buka     -> loket:operate
//	PATCH /:id/tutup    -> loket:operate
func RegisterLoketRoutes(api fiber.Router, svc *service.LoketService, auth fiber.Handler, can func(permission string) fiber.Handler) {
	loket := api.Group("/loket", auth)

	loket.Get("/", can(model.PermissionLoketManage), svc.List)
	loket.Get("/:id", can(model.PermissionLoketManage), svc.Get)
	loket.Post("/", can(model.PermissionLoketManage), svc.Create)
	loket.Put("/:id", can(model.PermissionLoketManage), svc.Update)
	loket.Delete("/:id", can(model.PermissionLoketManage), svc.Delete)

	loket.Patch("/:id/buka", can(model.PermissionLoketOperate), svc.Buka)
	loket.Patch("/:id/tutup", can(model.PermissionLoketOperate), svc.Tutup)
}
