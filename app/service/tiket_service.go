package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/helper"
)

// TiketService berisi logika bisnis sekaligus handler Fiber untuk tiket.
// Setiap perubahan status tiket WAJIB menulis event outbox di transaksi yang sama.
type TiketService struct {
	tiketRepo   *repository.TiketRepository
	layananRepo *repository.LayananRepository
	outboxRepo  *repository.OutboxRepository
	txManager   *repository.TxManager
}

func NewTiketService(
	tiketRepo *repository.TiketRepository,
	layananRepo *repository.LayananRepository,
	outboxRepo *repository.OutboxRepository,
	txManager *repository.TxManager,
) *TiketService {
	return &TiketService{
		tiketRepo:   tiketRepo,
		layananRepo: layananRepo,
		outboxRepo:  outboxRepo,
		txManager:   txManager,
	}
}

// AmbilNomor menangani POST /api/layanan/:id/tiket
// Seluruh proses berjalan dalam satu transaksi:
// cek layanan & tiket aktif -> naikkan nomor harian -> cek kuota -> insert tiket -> outbox.
func (s *TiketService) AmbilNomor(c *fiber.Ctx) error {
	ctx := c.UserContext()

	layananID, err := parseID(c)
	if err != nil {
		return err
	}

	userID := currentUserID(c)
	if userID <= 0 {
		return helper.Unauthorized("silakan login terlebih dahulu")
	}

	layanan, err := s.layananRepo.GetByID(ctx, layananID)
	if err != nil {
		return mapLayananError(err)
	}
	if !layanan.IsActive {
		return helper.BadRequest("layanan sedang tidak aktif")
	}

	tanggal := time.Now()
	tiket := &model.Tiket{
		LayananID: layananID,
		UserID:    userID,
		Tanggal:   tanggal,
		Status:    model.StatusMenunggu,
	}

	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		// 1. Satu user hanya boleh punya satu tiket aktif per layanan per hari.
		aktif, err := s.tiketRepo.CekTiketAktif(ctx, tx, userID, layananID, tanggal)
		if err != nil {
			return err
		}
		if aktif {
			return helper.Conflict("kamu masih punya tiket aktif untuk layanan ini")
		}

		// 2. Naikkan nomor harian secara atomik.
		nomor, err := s.tiketRepo.AmbilNomor(ctx, tx, layananID, tanggal)
		if err != nil {
			return err
		}

		// 3. Cek kuota. Error di sini membuat transaksi rollback,
		//    sehingga nomor yang sudah naik tidak terpakai.
		if layanan.KuotaHarian != nil && nomor > *layanan.KuotaHarian {
			return helper.NewError(409, "KUOTA_HABIS", "Kuota layanan hari ini sudah habis")
		}

		// 4. Simpan tiket.
		tiket.Nomor = nomor
		tiket.Kode = fmt.Sprintf("%s-%03d", layanan.Kode, nomor)
		if err := s.tiketRepo.Insert(ctx, tx, tiket); err != nil {
			return err
		}

		// 5. Catat event outbox di transaksi yang sama.
		return s.outboxRepo.Insert(ctx, tx, model.EventTiketDiambil,
			newTiketEvent(tiket, "", model.StatusMenunggu))
	})
	if err != nil {
		return mapTiketError(err)
	}

	return helper.Created(c, "tiket berhasil diambil", tiket)
}

// Saya menangani GET /api/tiket/saya
func (s *TiketService) Saya(c *fiber.Ctx) error {
	ctx := c.UserContext()

	userID := currentUserID(c)
	if userID <= 0 {
		return helper.Unauthorized("silakan login terlebih dahulu")
	}

	list, err := s.tiketRepo.ListByUser(ctx, userID)
	if err != nil {
		return err
	}

	return helper.Success(c, "daftar tiket saya", list)
}

// Detail menangani GET /api/tiket/:id (termasuk posisi di antrean).
func (s *TiketService) Detail(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	tiket, err := s.tiketRepo.GetByID(ctx, id)
	if err != nil {
		return mapTiketError(err)
	}

	// Pengunjung hanya boleh melihat tiketnya sendiri; admin/petugas boleh semua.
	role := currentRole(c)
	if role != "admin" && role != "petugas" && tiket.UserID != currentUserID(c) {
		return helper.Forbidden("tiket ini bukan milik kamu")
	}

	posisi := 0
	if tiket.Status == model.StatusMenunggu {
		posisi, err = s.tiketRepo.Posisi(ctx, tiket.LayananID, tiket.Tanggal, tiket.Nomor)
		if err != nil {
			return err
		}
	}

	return helper.Success(c, "detail tiket", fiber.Map{
		"tiket":  tiket,
		"posisi": posisi,
	})
}

// Batal menangani DELETE /api/tiket/:id (hanya tiket berstatus menunggu).
func (s *TiketService) Batal(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	userID := currentUserID(c)
	if userID <= 0 {
		return helper.Unauthorized("silakan login terlebih dahulu")
	}

	tiket, err := s.tiketRepo.GetByID(ctx, id)
	if err != nil {
		return mapTiketError(err)
	}
	if tiket.UserID != userID {
		return helper.Forbidden("tiket ini bukan milik kamu")
	}
	if tiket.Status != model.StatusMenunggu {
		return helper.Conflict("tiket hanya bisa dibatalkan saat masih menunggu")
	}

	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		ok, err := s.tiketRepo.Batal(ctx, id, userID)
		if err != nil {
			return err
		}
		if !ok {
			return helper.Conflict("tiket tidak bisa dibatalkan lagi")
		}
		return s.outboxRepo.Insert(ctx, tx, model.EventTiketDibatalkan,
			newTiketEvent(tiket, model.StatusMenunggu, model.StatusBatal))
	})
	if err != nil {
		return mapTiketError(err)
	}

	return helper.Success(c, "tiket berhasil dibatalkan", fiber.Map{"id": id})
}

// Papan menangani GET /api/layanan/:id/antrean
func (s *TiketService) Papan(c *fiber.Ctx) error {
	ctx := c.UserContext()

	layananID, err := parseID(c)
	if err != nil {
		return err
	}

	papan, err := s.tiketRepo.Papan(ctx, layananID, time.Now())
	if err != nil {
		return err
	}

	return helper.Success(c, "papan antrean", papan)
}

// Panggil menangani POST /api/loket/:id/panggil
func (s *TiketService) Panggil(c *fiber.Ctx) error {
	ctx := c.UserContext()

	loketID, err := parseID(c)
	if err != nil {
		return err
	}

	loket, err := s.tiketRepo.GetLoketInfo(ctx, loketID)
	if err != nil {
		return mapTiketError(err)
	}
	if loket.Status != "buka" {
		return helper.Conflict("loket sedang tutup")
	}

	// TODO(auth): pastikan yang memanggil adalah petugas loket ini atau admin.

	tanggal := time.Now()
	var tiket *model.Tiket

	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		t, err := s.tiketRepo.PanggilBerikutnya(ctx, tx, loket.LayananID, loketID, tanggal)
		if err != nil {
			return err
		}
		tiket = t

		return s.outboxRepo.Insert(ctx, tx, model.EventTiketDipanggil,
			newTiketEvent(tiket, model.StatusMenunggu, model.StatusDipanggil))
	})
	if err != nil {
		return mapTiketError(err)
	}

	return helper.Success(c, "tiket dipanggil", tiket)
}

// Selesai menangani POST /api/tiket/:id/selesai (dipanggil -> selesai).
func (s *TiketService) Selesai(c *fiber.Ctx) error {
	return s.ubahStatus(c, model.StatusDipanggil, model.StatusSelesai, model.EventTiketSelesai)
}

// Lewati menangani POST /api/tiket/:id/lewati (dipanggil -> dilewati).
func (s *TiketService) Lewati(c *fiber.Ctx) error {
	return s.ubahStatus(c, model.StatusDipanggil, model.StatusDilewati, model.EventTiketDilewati)
}

// PanggilUlang menangani POST /api/tiket/:id/panggil-ulang
// (dipanggil/dilewati -> dipanggil).
func (s *TiketService) PanggilUlang(c *fiber.Ctx) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	tiket, err := s.tiketRepo.GetByID(ctx, id)
	if err != nil {
		return mapTiketError(err)
	}
	if tiket.Status != model.StatusDipanggil && tiket.Status != model.StatusDilewati {
		return helper.Conflict("hanya tiket yang dipanggil/dilewati bisa dipanggil ulang")
	}

	// TODO(auth): pastikan yang memanggil ulang adalah petugas loket atau admin.

	statusLama := tiket.Status
	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		ok, err := s.tiketRepo.PanggilUlang(ctx, tx, id, tiket.LoketID)
		if err != nil {
			return err
		}
		if !ok {
			return helper.Conflict("tiket tidak bisa dipanggil ulang")
		}

		return s.outboxRepo.Insert(ctx, tx, model.EventTiketDipanggil,
			newTiketEvent(tiket, statusLama, model.StatusDipanggil))
	})
	if err != nil {
		return mapTiketError(err)
	}

	tiket.Status = model.StatusDipanggil
	return helper.Success(c, "tiket dipanggil ulang", tiket)
}

// ubahStatus adalah pola umum untuk aksi petugas: cek transisi lalu update + outbox.
func (s *TiketService) ubahStatus(c *fiber.Ctx, dari, ke, event string) error {
	ctx := c.UserContext()

	id, err := parseID(c)
	if err != nil {
		return err
	}

	tiket, err := s.tiketRepo.GetByID(ctx, id)
	if err != nil {
		return mapTiketError(err)
	}
	if tiket.Status != dari {
		return helper.Conflict(fmt.Sprintf("status tiket harus %q, sekarang %q", dari, tiket.Status))
	}

	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		ok, err := s.tiketRepo.UbahStatus(ctx, tx, id, dari, ke)
		if err != nil {
			return err
		}
		if !ok {
			return helper.Conflict("status tiket sudah berubah, coba lagi")
		}

		return s.outboxRepo.Insert(ctx, tx, event, newTiketEvent(tiket, dari, ke))
	})
	if err != nil {
		return mapTiketError(err)
	}

	tiket.Status = ke
	return helper.Success(c, "status tiket diperbarui", tiket)
}

// newTiketEvent menyusun payload outbox untuk perubahan status tiket.
func newTiketEvent(t *model.Tiket, lama, baru string) model.TiketEvent {
	return model.TiketEvent{
		TiketID:    t.ID,
		LayananID:  t.LayananID,
		UserID:     t.UserID,
		LoketID:    t.LoketID,
		StatusLama: lama,
		StatusBaru: baru,
		Waktu:      time.Now(),
	}
}

// mapTiketError menerjemahkan error database menjadi AppError yang rapi.
func mapTiketError(err error) error {
	if err == nil {
		return nil
	}

	// Error yang sudah berbentuk *AppError diteruskan apa adanya.
	if appErr := helper.AsAppError(err); appErr != nil {
		return appErr
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return helper.NotFound("data tidak ditemukan")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "idx_tiket_aktif_per_user_layanan" {
			return helper.Conflict("kamu masih punya tiket aktif untuk layanan ini")
		}
		return helper.Conflict("terjadi duplikasi data")
	}

	return err
}
