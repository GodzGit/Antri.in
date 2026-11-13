package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
)

// SeedRolesAndPermissions menanam data awal roles, permissions, dan relasi role_permissions
// ke dalam database. Fungsi ini bersifat idempotent: aman dijalankan berulang kali
// tanpa menyebabkan duplikasi data.
//
// Parameter:
//   - ctx  : context untuk batas waktu dan pembatalan operasi
//   - pool : koneksi pgxpool yang sudah aktif (tidak membuat koneksi baru)
//
// Cara menjalankan (dari main.go atau perintah terpisah):
//
//	if err := database.SeedRolesAndPermissions(ctx, pgPool); err != nil {
//	    log.Fatalf("seeder gagal: %v", err)
//	}
func SeedRolesAndPermissions(ctx context.Context, pool *pgxpool.Pool) error {
	roleRepo := repository.NewRoleRepository(pool)

	// ----------------------------------------------------------------
	// TAHAP 1: Seed Roles
	// ----------------------------------------------------------------
	log.Println("[Seeder] Memulai seed roles...")

	roleNames := []string{
		model.RoleAdmin,
		model.RolePetugas,
		model.RolePengunjung,
	}

	// roleMap menyimpan nama -> *model.Role agar bisa digunakan saat assign permission.
	roleMap := make(map[string]*model.Role, len(roleNames))

	for _, nama := range roleNames {
		role, err := roleRepo.FindByName(ctx, nama)
		if err != nil {
			return fmt.Errorf("[Seeder] gagal memeriksa role %q: %w", nama, err)
		}

		if role != nil {
			// Role sudah ada — lewati, simpan referensinya.
			log.Printf("[Seeder] Role %q sudah ada (id=%d), dilewati.", nama, role.ID)
			roleMap[nama] = role
			continue
		}

		// Role belum ada — buat baru.
		newRole := &model.Role{Nama: nama}
		if err := roleRepo.Create(ctx, newRole); err != nil {
			return fmt.Errorf("[Seeder] gagal membuat role %q: %w", nama, err)
		}
		log.Printf("[Seeder] Role %q dibuat (id=%d).", nama, newRole.ID)
		roleMap[nama] = newRole
	}

	// ----------------------------------------------------------------
	// TAHAP 2: Seed Permissions
	// ----------------------------------------------------------------
	log.Println("[Seeder] Memulai seed permissions...")

	allPermissions := []string{
		model.PermissionUserManage,
		model.PermissionLayananManage,
		model.PermissionLoketManage,
		model.PermissionLoketOperate,
		model.PermissionTiketCreate,
		model.PermissionTiketReadOwn,
		model.PermissionAntreanRead,
		model.PermissionStatistikRead,
		model.PermissionNotifikasiReadOwn,
	}

	// permMap menyimpan nama -> *model.Permission untuk digunakan saat assign.
	permMap := make(map[string]*model.Permission, len(allPermissions))

	for _, nama := range allPermissions {
		perm, err := roleRepo.FindPermissionByName(ctx, nama)
		if err != nil {
			return fmt.Errorf("[Seeder] gagal memeriksa permission %q: %w", nama, err)
		}

		if perm != nil {
			// Permission sudah ada — lewati, simpan referensinya.
			log.Printf("[Seeder] Permission %q sudah ada (id=%d), dilewati.", nama, perm.ID)
			permMap[nama] = perm
			continue
		}

		// Permission belum ada — buat baru.
		newPerm := &model.Permission{Nama: nama}
		if err := roleRepo.CreatePermission(ctx, newPerm); err != nil {
			return fmt.Errorf("[Seeder] gagal membuat permission %q: %w", nama, err)
		}
		log.Printf("[Seeder] Permission %q dibuat (id=%d).", nama, newPerm.ID)
		permMap[nama] = newPerm
	}

	// ----------------------------------------------------------------
	// TAHAP 3: Seed Relasi Role–Permission
	//
	// Matriks hak akses sesuai dokumen panduan:
	//
	//  Admin     : user:manage, layanan:manage, loket:manage, loket:operate,
	//              antrean:read, statistik:read, notifikasi:read:own
	//  Petugas   : loket:operate, antrean:read, statistik:read, notifikasi:read:own
	//  Pengunjung: tiket:create, tiket:read:own, antrean:read, notifikasi:read:own
	// ----------------------------------------------------------------
	log.Println("[Seeder] Memulai seed relasi role_permissions...")

	rolePermissions := map[string][]string{
		model.RoleAdmin: {
			model.PermissionUserManage,
			model.PermissionLayananManage,
			model.PermissionLoketManage,
			model.PermissionLoketOperate,
			model.PermissionAntreanRead,
			model.PermissionStatistikRead,
			model.PermissionNotifikasiReadOwn,
		},
		model.RolePetugas: {
			model.PermissionLoketOperate,
			model.PermissionAntreanRead,
			model.PermissionStatistikRead,
			model.PermissionNotifikasiReadOwn,
		},
		model.RolePengunjung: {
			model.PermissionTiketCreate,
			model.PermissionTiketReadOwn,
			model.PermissionAntreanRead,
			model.PermissionNotifikasiReadOwn,
		},
	}

	for roleName, permissions := range rolePermissions {
		role, ok := roleMap[roleName]
		if !ok {
			// Seharusnya tidak terjadi karena roleMap diisi di tahap 1.
			return fmt.Errorf("[Seeder] role %q tidak ditemukan di roleMap", roleName)
		}

		for _, permName := range permissions {
			perm, ok := permMap[permName]
			if !ok {
				// Seharusnya tidak terjadi karena permMap diisi di tahap 2.
				return fmt.Errorf("[Seeder] permission %q tidak ditemukan di permMap", permName)
			}

			// AssignPermission sudah menggunakan ON CONFLICT DO NOTHING di dalam repository,
			// sehingga idempotent untuk relasi yang sudah ada.
			if err := roleRepo.AssignPermission(ctx, role.ID, perm.ID); err != nil {
				return fmt.Errorf(
					"[Seeder] gagal menghubungkan role %q dengan permission %q: %w",
					roleName, permName, err,
				)
			}
			log.Printf("[Seeder] Relasi %q -> %q berhasil (atau sudah ada).", roleName, permName)
		}
	}

	log.Println("[Seeder] Seed roles dan permissions selesai.")
	return nil
}

