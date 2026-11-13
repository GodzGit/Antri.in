package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/middleware"
)

// RegisterAuthRoutes mendaftarkan seluruh endpoint autentikasi pada group /api/auth.
//
// Endpoint yang tersedia:
//   - POST /api/auth/register  → publik, mendaftarkan akun baru (role default: pengunjung)
//   - POST /api/auth/login     → publik, mengautentikasi user dan mengembalikan token
//   - POST /api/auth/refresh   → publik, merotasi refresh token dan access token
//   - POST /api/auth/logout    → memerlukan auth, mencabut refresh token
//   - GET  /api/auth/me        → memerlukan auth, mengembalikan profil user yang sedang login
//
// Cara memanggil dari route/route.go (koordinasikan dengan Bagus):
//
//	route.RegisterAuthRoutes(api, authSvc, cfg.JWT.Secret)
func RegisterAuthRoutes(api fiber.Router, svc *service.AuthService, jwtSecret string) {
	auth := api.Group("/auth")

	// Endpoint publik — tidak memerlukan token.
	auth.Post("/register", svc.Register)
	auth.Post("/login", svc.Login)
	auth.Post("/refresh", svc.RefreshToken)

	// Endpoint terproteksi — memerlukan access token yang valid.
	auth.Post("/logout", middleware.Auth(jwtSecret), svc.Logout)
	auth.Get("/me", middleware.Auth(jwtSecret), svc.Me)
}

