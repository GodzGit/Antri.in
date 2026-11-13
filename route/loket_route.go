package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
)

// RegisterLoketRoutes mendaftarkan endpoint loket.
//
// TODO(auth): setelah middleware Auth + RequirePermission dari Fase 2 masuk:
//
//	loket:manage     -> GET/POST/PUT/DELETE /loket
//	loket:operate    -> PATCH /loket/:id/buka dan /loket/:id/tutup
func RegisterLoketRoutes(api fiber.Router, svc *service.LoketService) {
	loket := api.Group("/loket")

	loket.Get("/", svc.List)
	loket.Get("/:id", svc.Get)
	loket.Post("/", svc.Create)
	loket.Put("/:id", svc.Update)
	loket.Delete("/:id", svc.Delete)

	loket.Patch("/:id/buka", svc.Buka)
	loket.Patch("/:id/tutup", svc.Tutup)
}
