package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/helper"
)

const MinSampelDurasi = 5

type tiketPemilikStore interface {
	TiketByID(ctx context.Context, id int64) (*model.InfoTiket, error)
}

type estimasiStore interface {
	tiketPemilikStore
	OrangDiDepan(ctx context.Context, layananID int64, tanggal time.Time, nomor int) (int, error)
	LoketBuka(ctx context.Context, layananID int64) (int, error)
	DurasiLayanan(ctx context.Context, layananID int64, minSampel int) (float64, bool, error)
}

type EstimasiService struct {
	store estimasiStore
	now   func() time.Time
}

func NewEstimasiService(store estimasiStore) *EstimasiService {
	return &EstimasiService{store: store, now: time.Now}
}

func idDariParam(c *fiber.Ctx, nama string) (int64, error) {
	id, err := strconv.ParseInt(c.Params(nama), 10, 64)
	if err != nil || id <= 0 {
		return 0, helper.BadRequest(nama + " harus berupa angka positif")
	}
	return id, nil
}

func tiketMilik(ctx context.Context, store tiketPemilikStore, c *fiber.Ctx) (*model.InfoTiket, error) {
	userID, err := penggunaLogin(c)
	if err != nil {
		return nil, err
	}
	id, err := idDariParam(c, "id")
	if err != nil {
		return nil, err
	}

	tiket, err := store.TiketByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tiket == nil || tiket.UserID != userID {
		return nil, helper.NotFound("tiket tidak ditemukan")
	}
	return tiket, nil
}

// Estimasi godoc
//
// @Summary      Estimasi waktu tunggu tiket
// @Description  Orang di depan x rata-rata durasi layanan / jumlah loket buka. Durasi memakai rata-rata 7 hari terakhir bila ada minimal 5 tiket selesai, selain itu memakai durasi bawaan layanan.
// @Tags         tiket
// @Produce      json
// @Param        id   path      int  true  "ID tiket"
// @Success      200  {object}  model.WebResponse{data=model.EstimasiTunggu}
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse  "Tidak ada, atau milik pengguna lain"
// @Failure      409  {object}  model.ErrorResponse  "Tiket sudah selesai, dilewati, atau batal"
// @Security     BearerAuth
// @Router       /tiket/{id}/estimasi [get]
func (s *EstimasiService) Estimasi(c *fiber.Ctx) error {
	ctx := c.UserContext()
	tiket, err := tiketMilik(ctx, s.store, c)
	if err != nil {
		return err
	}

	hasil := model.EstimasiTunggu{
		TiketID:            tiket.ID,
		Kode:               tiket.Kode,
		Status:             tiket.Status,
		PerkiraanDipanggil: s.now(),
	}

	switch tiket.Status {
	case "dipanggil":
		return helper.Success(c, "tiket sedang dipanggil", hasil)
	case "menunggu":
	default:
		return helper.Conflict("tiket sudah tidak berada di antrean")
	}

	depan, err := s.store.OrangDiDepan(ctx, tiket.LayananID, tiket.Tanggal, tiket.Nomor)
	if err != nil {
		return err
	}
	loket, err := s.store.LoketBuka(ctx, tiket.LayananID)
	if err != nil {
		return err
	}
	durasi, historis, err := s.store.DurasiLayanan(ctx, tiket.LayananID, MinSampelDurasi)
	if errors.Is(err, repository.ErrDataTidakAda) {
		return helper.NotFound("layanan tiket tidak ditemukan")
	}
	if err != nil {
		return err
	}

	pembagi := loket
	if pembagi < 1 {
		pembagi = 1
	}
	estimasi := int(math.Ceil(float64(depan) * durasi / float64(pembagi)))

	hasil.Posisi = depan + 1
	hasil.OrangDiDepan = depan
	hasil.LoketBuka = loket
	hasil.RataDurasiMenit = math.Round(durasi*10) / 10
	hasil.SumberDurasi = "bawaan layanan"
	if historis {
		hasil.SumberDurasi = "rata-rata 7 hari terakhir"
	}
	hasil.EstimasiMenit = estimasi
	hasil.PerkiraanDipanggil = s.now().Add(time.Duration(estimasi) * time.Minute)
	return helper.Success(c, "estimasi waktu tunggu", hasil)
}
