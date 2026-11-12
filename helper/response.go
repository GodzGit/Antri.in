package helper

import (
	"github.com/gofiber/fiber/v2"
)

// Response standar sukses (dengan data).
type successResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Response standar gagal.
type errorResponse struct {
	Success bool      `json:"success"`
	Error   *AppError `json:"error"`
}

// Meta untuk list pagination.
type PaginationMeta struct {
	NextCursor string `json:"next_cursor"`
	Limit      int    `json:"limit"`
}

// Response standar list paginated.
type paginatedResponse struct {
	Success bool           `json:"success"`
	Data    any            `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

// Success mengirim response 200 OK dengan data.
func Success(ctx *fiber.Ctx, message string, data any) error {
	return ctx.Status(fiber.StatusOK).JSON(successResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created mengirim response 201 Created dengan data.
func Created(ctx *fiber.Ctx, message string, data any) error {
	return ctx.Status(fiber.StatusCreated).JSON(successResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Paginated mengirim response list dengan meta cursor pagination.
func Paginated(ctx *fiber.Ctx, data any, nextCursor string, limit int) error {
	return ctx.Status(fiber.StatusOK).JSON(paginatedResponse{
		Success: true,
		Data:    data,
		Meta: PaginationMeta{
			NextCursor: nextCursor,
			Limit:      limit,
		},
	})
}

// Error mengubah error menjadi response gagal standar.
// Kalau err bukan *AppError, otomatis dianggap error internal (500).
func Error(ctx *fiber.Ctx, err error) error {
	appErr := AsAppError(err)
	if appErr == nil {
		appErr = Internal("terjadi kesalahan pada server")
	}
	return ctx.Status(appErr.Status).JSON(errorResponse{
		Success: false,
		Error:   appErr,
	})
}