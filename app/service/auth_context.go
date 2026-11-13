package service

import (
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// currentUserID mengambil ID user yang login.
//
// TODO(auth): setelah middleware Auth dari Fase 2 merge, hapus fallback header
// di bawah dan cukup pakai c.Locals("user_id").
func currentUserID(c *fiber.Ctx) int64 {
	if v, ok := c.Locals("user_id").(int64); ok && v > 0 {
		return v
	}

	// Fallback sementara untuk pengujian manual selama auth belum ada.
	// Sengaja dimatikan di production.
	if os.Getenv("APP_ENV") == "production" {
		return 0
	}

	id, err := strconv.ParseInt(c.Get("X-User-ID"), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// currentRole mengambil role user yang login (kosong kalau middleware belum ada).
func currentRole(c *fiber.Ctx) string {
	if v, ok := c.Locals("role").(string); ok {
		return v
	}
	return ""
}
