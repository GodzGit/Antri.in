package service

import (
	"github.com/gofiber/fiber/v2"
)

// currentUserID mengambil ID user yang login dari middleware Auth.
func currentUserID(c *fiber.Ctx) int64 {
	if v, ok := c.Locals("user_id").(int64); ok && v > 0 {
		return v
	}
	return 0
}

// currentRole mengambil role user yang login dari middleware Auth.
func currentRole(c *fiber.Ctx) string {
	if v, ok := c.Locals("role").(string); ok {
		return v
	}
	return ""
}
