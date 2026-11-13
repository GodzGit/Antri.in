package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// Logger mengembalikan middleware Fiber untuk mencatat request HTTP.
// Format default mencatat: waktu, status, method, path, dan durasi.
func Logger() fiber.Handler {
	return logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} | ${method} ${path}\n",
	})
}