package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"

	_ "github.com/GodzGit/Antri.in/docs"
)

// @title                       ANTRI.IN API
// @version                     0.1.0
// @description                 Backend sistem antrean digital ANTRI.IN: ambil nomor antrean, panggil tiket di loket, notifikasi giliran, dan statistik layanan.
// @BasePath                    /api
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Isi dengan "Bearer <access_token>" dari endpoint login.
func RegisterSwaggerRoutes(app *fiber.App) {
	app.Get("/swagger/*", swagger.HandlerDefault)
}
