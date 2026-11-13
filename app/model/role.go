package model

// Role merepresentasikan entitas tabel 'roles' di database.
type Role struct {
	ID   int64  `json:"id"`
	Nama string `json:"nama"`
}

// Permission merepresentasikan entitas tabel 'permissions' di database.
type Permission struct {
	ID   int64  `json:"id"`
	Nama string `json:"nama"`
}

// RolePermission merepresentasikan entitas tabel pivot 'role_permissions'.
type RolePermission struct {
	RoleID       int64 `json:"role_id"`
	PermissionID int64 `json:"permission_id"`
}

// Definisi konstanta nama role standar sesuai panduan dokumen.
const (
	RoleAdmin      = "admin"
	RolePetugas    = "petugas"
	RolePengunjung = "pengunjung"
)

// Definisi konstanta nama permission sesuai matriks hak akses pada panduan dokumen.
const (
	PermissionUserManage        = "user:manage"
	PermissionLayananManage     = "layanan:manage"
	PermissionLoketManage       = "loket:manage"
	PermissionLoketOperate      = "loket:operate"
	PermissionTiketCreate       = "tiket:create"
	PermissionTiketReadOwn      = "tiket:read:own"
	PermissionAntreanRead       = "antrean:read"
	PermissionStatistikRead     = "statistik:read"
	PermissionNotifikasiReadOwn = "notifikasi:read:own"
)

// RoleResponse DTO untuk response data role beserta daftar permission yang dimilikinya.
type RoleResponse struct {
	ID          int64    `json:"id"`
	Nama        string   `json:"nama"`
	Permissions []string `json:"permissions,omitempty"`
}

