package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
)

// RegisterLayananRoutes mendaftarkan endpoint layanan.
//
// TODO(auth): setelah middleware Auth + RequirePermission dari Fase 2 masuk,
// tambahkan:
//
//	GET    -> middleware.RequirePermission("antrean:read")
//	POST/PUT/DELETE -> middleware.RequirePermission("layanan:manage")
func RegisterLayananRoutes(api fiber.Router, svc *service.LayananService) {
	layanan := api.Group("/layanan")

	layanan.Get("/", svc.List)
	layanan.Get("/:id", svc.Get)
	layanan.Post("/", svc.Create)
	layanan.Put("/:id", svc.Update)
	layanan.Delete("/:id", svc.Delete)
}
