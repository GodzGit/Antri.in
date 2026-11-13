package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/helper"
)

// LayananService berisi logika bisnis sekaligus handler Fiber untuk layanan.
// Proyek ini tidak memakai layer controller/handler terpisah.
type LayananService struct {
	repo *repository.LayananRepository
}

func NewLayananService(repo *repository.LayananRepository) *LayananService {
	return &LayananService{repo: repo}
}

// List menangani GET /api/layanan
// Query opsional: ?aktif=true untuk hanya menampilkan layanan aktif.
func (s *LayananService) List(c *fiber.Ctx) error {
	ctx := c.UserContext()

	hanyaAktif := c.Query("aktif") == "true"

	list, err := s.repo.List(ctx, hanyaAktif)
	if err != nil {
		return err
	}

	return helper.Success(c, "daftar layanan", list)
}

// Get menangani GET /api/layanan/:id
func (s *LayananService) Get(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	layanan, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return mapLayananError(err)
	}

	return helper.Success(c, "detail layanan", layanan)
}

// Create menangani POST /api/layanan
func (s *LayananService) Create(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req model.CreateLayananRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	layanan := &model.Layanan{
		Kode:            req.Kode,
		Nama:            req.Nama,
		Deskripsi:       optionalString(req.Deskripsi),
		JamBuka:         &req.JamBuka,
		JamTutup:        &req.JamTutup,
		KuotaHarian:     req.KuotaHarian,
		RataDurasiMenit: req.RataDurasiMenit,
		IsActive:        true,
	}

	if err := s.repo.Create(ctx, layanan); err != nil {
		return mapLayananError(err)
	}

	return helper.Created(c, "layanan berhasil dibuat", layanan)
}

// Update menangani PUT /api/layanan/:id
func (s *LayananService) Update(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	var req model.UpdateLayananRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	layanan := &model.Layanan{
		ID:              id,
		Kode:            req.Kode,
		Nama:            req.Nama,
		Deskripsi:       optionalString(req.Deskripsi),
		JamBuka:         &req.JamBuka,
		JamTutup:        &req.JamTutup,
		KuotaHarian:     req.KuotaHarian,
		RataDurasiMenit: req.RataDurasiMenit,
		IsActive:        req.IsActive,
	}

	if err := s.repo.Update(ctx, layanan); err != nil {
		return mapLayananError(err)
	}

	return helper.Success(c, "layanan berhasil diperbarui", layanan)
}

// Delete menangani DELETE /api/layanan/:id (soft delete).
func (s *LayananService) Delete(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return mapLayananError(err)
	}

	return helper.Success(c, "layanan berhasil dinonaktifkan", fiber.Map{"id": id})
}

// parseID mengambil :id dari path dan memastikan valid.
func parseID(c *fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, helper.BadRequest("id tidak valid")
	}
	return id, nil
}

// optionalString mengubah string kosong menjadi nil supaya kolom nullable tetap NULL.
func optionalString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// mapLayananError menerjemahkan error database menjadi AppError yang rapi.
func mapLayananError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return helper.NotFound("layanan tidak ditemukan")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return helper.Conflict("kode layanan sudah dipakai")
	}

	return err
}
