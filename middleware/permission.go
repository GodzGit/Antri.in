package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/helper"
)

// rolePermissionsMap memetakan setiap role ke himpunan permission resmi
// berdasarkan matriks hak akses pada panduan dokumen proyek.
var rolePermissionsMap = map[string]map[string]bool{
	model.RoleAdmin: {
		model.PermissionUserManage:        true,
		model.PermissionLayananManage:     true,
		model.PermissionLoketManage:       true,
		model.PermissionLoketOperate:      true,
		model.PermissionAntreanRead:       true,
		model.PermissionStatistikRead:     true,
		model.PermissionNotifikasiReadOwn: true,
	},
	model.RolePetugas: {
		model.PermissionLoketOperate:      true,
		model.PermissionAntreanRead:       true,
		model.PermissionStatistikRead:     true,
		model.PermissionNotifikasiReadOwn: true,
	},
	model.RolePengunjung: {
		model.PermissionTiketCreate:       true,
		model.PermissionTiketReadOwn:      true,
		model.PermissionAntreanRead:       true,
		model.PermissionNotifikasiReadOwn: true,
	},
}

// RequirePermission adalah middleware untuk memeriksa apakah role pengguna memiliki hak akses (permission) yang diminta.
// Middleware ini mengandalkan data role yang telah disimpan ke ctx.Locals("role") oleh middleware Auth sebelumnya.
func RequirePermission(permission string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		roleVal := ctx.Locals("role")
		if roleVal == nil {
			return helper.Forbidden("akses ditolak: role tidak ditemukan")
		}

		role, ok := roleVal.(string)
		if !ok || role == "" {
			return helper.Forbidden("akses ditolak: format role tidak valid")
		}

		permissions, exists := rolePermissionsMap[role]
		if !exists {
			return helper.Forbidden("akses ditolak: role tidak dikenali")
		}

		if !permissions[permission] {
			return helper.Forbidden("akses ditolak: Anda tidak memiliki izin untuk tindakan ini")
		}

		return ctx.Next()
	}
}

