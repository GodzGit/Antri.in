package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/helper"
)

// LoketService berisi logika bisnis sekaligus handler Fiber untuk loket.
type LoketService struct {
	repo *repository.LoketRepository
}

func NewLoketService(repo *repository.LoketRepository) *LoketService {
	return &LoketService{repo: repo}
}

// List menangani GET /api/loket
func (s *LoketService) List(c *fiber.Ctx) error {
	list, err := s.repo.List(c.UserContext())
	if err != nil {
		return err
	}

	return helper.Success(c, "daftar loket", list)
}

// Get menangani GET /api/loket/:id
func (s *LoketService) Get(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	loket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return mapLoketError(err)
	}

	return helper.Success(c, "detail loket", loket)
}

// Create menangani POST /api/loket
func (s *LoketService) Create(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req model.CreateLoketRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	loket := &model.Loket{
		LayananID: req.LayananID,
		Nama:      req.Nama,
	}

	if err := s.repo.Create(ctx, loket); err != nil {
		return mapLoketError(err)
	}

	return helper.Created(c, "loket berhasil dibuat", loket)
}

// Update menangani PUT /api/loket/:id (termasuk set petugas).
func (s *LoketService) Update(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	var req model.UpdateLoketRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	loket := &model.Loket{
		ID:        id,
		LayananID: req.LayananID,
		Nama:      req.Nama,
		PetugasID: req.PetugasID,
	}

	if err := s.repo.Update(ctx, loket); err != nil {
		return mapLoketError(err)
	}

	// Ambil ulang supaya response berisi data lengkap (nama layanan/petugas).
	lengkap, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return mapLoketError(err)
	}

	return helper.Success(c, "loket berhasil diperbarui", lengkap)
}

// Delete menangani DELETE /api/loket/:id
func (s *LoketService) Delete(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return mapLoketError(err)
	}

	return helper.Success(c, "loket berhasil dihapus", fiber.Map{"id": id})
}

// Buka menangani PATCH /api/loket/:id/buka
func (s *LoketService) Buka(c *fiber.Ctx) error {
	return s.setStatus(c, model.LoketBuka)
}

// Tutup menangani PATCH /api/loket/:id/tutup
func (s *LoketService) Tutup(c *fiber.Ctx) error {
	return s.setStatus(c, model.LoketTutup)
}

// setStatus membuka/menutup loket sekaligus mengecek siapa yang berhak.
func (s *LoketService) setStatus(c *fiber.Ctx, status string) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	loket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return mapLoketError(err)
	}
	if loket.Status == status {
		return helper.Conflict("status loket sudah " + status)
	}

	// Hanya admin atau petugas yang ditugaskan di loket ini yang boleh.
	// TODO(auth): aturan ini otomatis berlaku setelah RequirePermission("loket:operate") dipasang.
	role := currentRole(c)
	if role != "admin" && loket.PetugasID != nil && *loket.PetugasID != currentUserID(c) {
		return helper.Forbidden("kamu bukan petugas loket ini")
	}

	if err := s.repo.SetStatus(ctx, id, status); err != nil {
		return mapLoketError(err)
	}

	lengkap, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return mapLoketError(err)
	}

	return helper.Success(c, "status loket sekarang "+status, lengkap)
}

// mapLoketError menerjemahkan error database menjadi AppError yang rapi.
func mapLoketError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return helper.NotFound("loket tidak ditemukan")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign key: layanan_id / petugas_id tidak ada
			return helper.BadRequest("layanan atau petugas tidak ditemukan")
		case "23514": // check constraint: status tidak valid
			return helper.BadRequest("status loket tidak valid")
		}
	}

	return err
}
