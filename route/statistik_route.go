package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
)

func RegisterStatistikRoutes(api fiber.Router, estimasi *service.EstimasiService, riwayat *service.RiwayatService, statistik *service.StatistikService, auth fiber.Handler, can func(permission string) fiber.Handler) {
	api.Get("/tiket/:id/estimasi", auth, can("tiket:read:own"), estimasi.Estimasi)
	api.Get("/tiket/:id/riwayat", auth, can("tiket:read:own"), riwayat.Riwayat)
	api.Get("/statistik/layanan/:id", auth, can("statistik:read"), statistik.Layanan)
}
