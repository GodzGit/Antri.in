package helper

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

const (
	DefaultLimit = 10
	MaxLimit     = 100
)

// GetPaginationParams mengambil parameter `limit` dan `cursor` dari query string.
// Mengembalikan limit (default 10) dan cursor dalam bentuk ID (int64).
// Kalau cursor kosong, kembalikan 0 (artinya mulai dari awal).
func GetPaginationParams(ctx *fiber.Ctx) (limit int, cursorID int64) {
	limit = DefaultLimit
	if l := ctx.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	if c := ctx.Query("cursor"); c != "" {
		if parsed, err := strconv.ParseInt(c, 10, 64); err == nil {
			cursorID = parsed
		}
	}

	return limit, cursorID
}

// EncodeCursor mengubah ID menjadi string base64 (bisa dipakai di URL).
func EncodeCursor(id int64) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", id)))
}

// DecodeCursor membalikkan string base64 menjadi ID.
func DecodeCursor(cursor string) int64 {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	id, _ := strconv.ParseInt(string(decoded), 10, 64)
	return id
}