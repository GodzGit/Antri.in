package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/helper"
)

// ErrorHandler mengubah semua error jadi response JSON standar.
func ErrorHandler() fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		return helper.Error(ctx, err)
	}
}