package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
)

func RegisterNotifikasiRoutes(api fiber.Router, svc *service.NotifikasiService, auth fiber.Handler, can func(permission string) fiber.Handler) {
	notifikasi := api.Group("/notifikasi", auth, can("notifikasi:read:own"))
	notifikasi.Get("/saya", svc.ListSaya)
	notifikasi.Patch("/:id/baca", svc.TandaiDibaca)
}
