package service

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/helper"
)

type riwayatLister interface {
	ListByTiket(ctx context.Context, tiketID int64) ([]model.RiwayatTiket, error)
}

type RiwayatService struct {
	tiket   tiketPemilikStore
	riwayat riwayatLister
}

func NewRiwayatService(tiket tiketPemilikStore, riwayat riwayatLister) *RiwayatService {
	return &RiwayatService{tiket: tiket, riwayat: riwayat}
}

// Riwayat godoc
//
// @Summary      Riwayat perubahan status tiket
// @Description  Dicatat oleh worker dari event outbox, diurutkan dari yang paling awal.
// @Tags         tiket
// @Produce      json
// @Param        id   path      int  true  "ID tiket"
// @Success      200  {object}  model.WebResponse{data=[]model.RiwayatTiket}
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse  "Tidak ada, atau milik pengguna lain"
// @Security     BearerAuth
// @Router       /tiket/{id}/riwayat [get]
func (s *RiwayatService) Riwayat(c *fiber.Ctx) error {
	ctx := c.UserContext()
	tiket, err := tiketMilik(ctx, s.tiket, c)
	if err != nil {
		return err
	}

	items, err := s.riwayat.ListByTiket(ctx, tiket.ID)
	if err != nil {
		return err
	}
	return helper.Success(c, "riwayat tiket", items)
}
