package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/GodzGit/Antri.in/app/model"
)

const UrutanPengingat = 3

type riwayatStore interface {
	CatatSekali(ctx context.Context, outboxID int64, rw model.RiwayatTiket) (bool, error)
}

type notifikasiStore interface {
	CreateSekali(ctx context.Context, eventKey string, n *model.Notifikasi) (bool, error)
}

type antreanStore interface {
	TiketByID(ctx context.Context, id int64) (*model.InfoTiket, error)
	TiketMenungguKe(ctx context.Context, layananID int64, tanggal time.Time, urutan int) (*model.InfoTiket, error)
	NamaLoket(ctx context.Context, id int64) (string, error)
}

type Handler struct {
	riwayat    riwayatStore
	notifikasi notifikasiStore
	antrean    antreanStore
	logger     *slog.Logger
}

func NewHandler(riwayat riwayatStore, notifikasi notifikasiStore, antrean antreanStore, logger *slog.Logger) *Handler {
	return &Handler{riwayat: riwayat, notifikasi: notifikasi, antrean: antrean, logger: logger}
}

func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var job OutboxJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}

	var ev model.TiketEvent
	if err := json.Unmarshal(job.Payload, &ev); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}
	if ev.Waktu.IsZero() {
		ev.Waktu = time.Now()
	}

	if _, err := h.riwayat.CatatSekali(ctx, job.OutboxID, model.RiwayatTiket{
		TiketID:    ev.TiketID,
		LayananID:  ev.LayananID,
		UserID:     ev.UserID,
		LoketID:    ev.LoketID,
		StatusLama: ev.StatusLama,
		StatusBaru: ev.StatusBaru,
		Waktu:      ev.Waktu,
	}); err != nil {
		return err
	}

	if job.EventType == model.EventTiketDipanggil {
		return h.notifDipanggil(ctx, job.OutboxID, ev)
	}
	return nil
}

func (h *Handler) notifDipanggil(ctx context.Context, outboxID int64, ev model.TiketEvent) error {
	tiket, err := h.antrean.TiketByID(ctx, ev.TiketID)
	if err != nil {
		return err
	}
	if tiket == nil {
		return fmt.Errorf("%w: tiket %d tidak ditemukan", asynq.SkipRetry, ev.TiketID)
	}

	loket := "loket"
	if ev.LoketID != nil {
		nama, err := h.antrean.NamaLoket(ctx, *ev.LoketID)
		if err != nil {
			return err
		}
		if nama != "" {
			loket = nama
		}
	}

	if _, err := h.notifikasi.CreateSekali(ctx, fmt.Sprintf("outbox-%d-dipanggil", outboxID), &model.Notifikasi{
		UserID:  tiket.UserID,
		TiketID: tiket.ID,
		Judul:   "Giliran kamu!",
		Pesan:   fmt.Sprintf("Nomor %s dipanggil. Silakan menuju %s.", tiket.Kode, loket),
	}); err != nil {
		return err
	}

	dekat, err := h.antrean.TiketMenungguKe(ctx, tiket.LayananID, tiket.Tanggal, UrutanPengingat)
	if err != nil {
		return err
	}
	if dekat == nil {
		return nil
	}

	_, err = h.notifikasi.CreateSekali(ctx, fmt.Sprintf("outbox-%d-dekat-%d", outboxID, dekat.ID), &model.Notifikasi{
		UserID:  dekat.UserID,
		TiketID: dekat.ID,
		Judul:   "Antrean kamu sudah dekat",
		Pesan:   fmt.Sprintf("Nomor %s: tinggal %d antrean lagi sebelum giliranmu. Bersiaplah menuju loket.", dekat.Kode, UrutanPengingat-1),
	})
	return err
}
