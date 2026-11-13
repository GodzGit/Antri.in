package service

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/helper"
)

type notifikasiStore interface {
	ListByUser(ctx context.Context, userID int64, cursor string, limit int) ([]model.Notifikasi, string, error)
	MarkRead(ctx context.Context, id string, userID int64) (bool, error)
}

type NotifikasiService struct {
	repo notifikasiStore
}

func NewNotifikasiService(repo notifikasiStore) *NotifikasiService {
	return &NotifikasiService{repo: repo}
}

// ListSaya godoc
//
// @Summary      Notifikasi milik pengguna yang login
// @Description  Diurutkan dari yang terbaru. Gunakan meta.next_cursor untuk halaman berikutnya.
// @Tags         notifikasi
// @Produce      json
// @Param        limit   query     int     false  "Jumlah per halaman (1-100)"  default(10)
// @Param        cursor  query     string  false  "Cursor dari meta.next_cursor"
// @Success      200     {object}  model.CursorResponse{data=[]model.Notifikasi}
// @Failure      400     {object}  model.ErrorResponse  "Cursor tidak valid"
// @Failure      401     {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /notifikasi/saya [get]
func (s *NotifikasiService) ListSaya(c *fiber.Ctx) error {
	userID, err := penggunaLogin(c)
	if err != nil {
		return err
	}

	limit := c.QueryInt("limit", helper.DefaultLimit)
	if limit < 1 {
		limit = helper.DefaultLimit
	}
	if limit > helper.MaxLimit {
		limit = helper.MaxLimit
	}

	items, next, err := s.repo.ListByUser(c.UserContext(), userID, c.Query("cursor"), limit)
	if errors.Is(err, repository.ErrIDTidakValid) {
		return helper.BadRequest("cursor tidak valid")
	}
	if err != nil {
		return err
	}
	return helper.Paginated(c, items, next, limit)
}

// TandaiDibaca godoc
//
// @Summary   Menandai notifikasi sudah dibaca
// @Tags      notifikasi
// @Produce   json
// @Param     id   path      string  true  "ID notifikasi (ObjectID)"
// @Success   200  {object}  model.WebResponse
// @Failure   400  {object}  model.ErrorResponse  "ID bukan ObjectID"
// @Failure   401  {object}  model.ErrorResponse
// @Failure   404  {object}  model.ErrorResponse  "Tidak ada, atau milik pengguna lain"
// @Security  BearerAuth
// @Router    /notifikasi/{id}/baca [patch]
func (s *NotifikasiService) TandaiDibaca(c *fiber.Ctx) error {
	userID, err := penggunaLogin(c)
	if err != nil {
		return err
	}

	found, err := s.repo.MarkRead(c.UserContext(), c.Params("id"), userID)
	if errors.Is(err, repository.ErrIDTidakValid) {
		return helper.BadRequest("id notifikasi tidak valid")
	}
	if err != nil {
		return err
	}
	if !found {
		return helper.NotFound("notifikasi tidak ditemukan")
	}
	return helper.Success(c, "notifikasi ditandai sudah dibaca", nil)
}

func penggunaLogin(c *fiber.Ctx) (int64, error) {
	id, ok := c.Locals("user_id").(int64)
	if !ok || id == 0 {
		return 0, helper.Unauthorized("silakan login terlebih dahulu")
	}
	return id, nil
}
