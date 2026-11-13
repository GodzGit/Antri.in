package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/helper"
)

// Auth adalah middleware untuk memeriksa dan memvalidasi access token JWT pada request.
// Middleware ini mengekstrak Bearer token dari header Authorization,
// memverifikasi keabsahannya menggunakan helper.ValidateAccessToken,
// dan menyimpan klaim pengguna (user_id, role, email) ke dalam context Locals Fiber.
func Auth(secret string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		authHeader := ctx.Get("Authorization")
		if authHeader == "" {
			return helper.Unauthorized("header Authorization tidak ditemukan")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return helper.Unauthorized("format header Authorization harus 'Bearer <token>'")
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			return helper.Unauthorized("token tidak boleh kosong")
		}

		claims, err := helper.ValidateAccessToken(tokenString, secret)
		if err != nil {
			return helper.Unauthorized("token tidak valid atau telah kedaluwarsa")
		}

		// Simpan informasi pengguna ke Locals sesuai kontrak proyek
		ctx.Locals("user_id", claims.UserID)
		ctx.Locals("role", claims.Role)
		ctx.Locals("email", claims.Email)

		return ctx.Next()
	}
}

