package service

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/helper"
)

const ZonaWIB = "+07:00"

var wib = time.FixedZone("WIB", 7*60*60)

type layananCek interface {
	LayananAda(ctx context.Context, id int64) (bool, error)
}

type statistikStore interface {
	Statistik(ctx context.Context, layananID int64, dari, sampai time.Time, zona string) (model.StatistikLayanan, error)
}

type StatistikService struct {
	layanan layananCek
	riwayat statistikStore
	now     func() time.Time
}

func NewStatistikService(layanan layananCek, riwayat statistikStore) *StatistikService {
	return &StatistikService{layanan: layanan, riwayat: riwayat, now: time.Now}
}

// Layanan godoc
//
// @Summary      Statistik harian sebuah layanan
// @Description  Jumlah tiket per jam (WIB), jam tersibuk, jumlah per status, serta rata-rata waktu tunggu dan waktu layanan. Dihitung dengan aggregation $facet di MongoDB.
// @Tags         statistik
// @Produce      json
// @Param        id       path      int     true   "ID layanan"
// @Param        tanggal  query     string  false  "Tanggal YYYY-MM-DD, default hari ini (WIB)"
// @Success      200      {object}  model.WebResponse{data=model.StatistikLayanan}
// @Failure      400      {object}  model.ErrorResponse
// @Failure      401      {object}  model.ErrorResponse
// @Failure      403      {object}  model.ErrorResponse
// @Failure      404      {object}  model.ErrorResponse  "Layanan tidak ditemukan"
// @Security     BearerAuth
// @Router       /statistik/layanan/{id} [get]
func (s *StatistikService) Layanan(c *fiber.Ctx) error {
	ctx := c.UserContext()
	id, err := idDariParam(c, "id")
	if err != nil {
		return err
	}

	tanggal := c.Query("tanggal", s.now().In(wib).Format(time.DateOnly))
	dari, err := time.ParseInLocation(time.DateOnly, tanggal, wib)
	if err != nil {
		return helper.BadRequest("tanggal harus berformat YYYY-MM-DD")
	}

	ada, err := s.layanan.LayananAda(ctx, id)
	if err != nil {
		return err
	}
	if !ada {
		return helper.NotFound("layanan tidak ditemukan")
	}

	stat, err := s.riwayat.Statistik(ctx, id, dari, dari.AddDate(0, 0, 1), ZonaWIB)
	if err != nil {
		return err
	}
	stat.Tanggal = tanggal
	return helper.Success(c, "statistik layanan", stat)
}
