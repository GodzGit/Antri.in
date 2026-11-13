package worker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/GodzGit/Antri.in/app/model"
)

type fakeRiwayat struct {
	dicatat []model.RiwayatTiket
	err     error
}

func (f *fakeRiwayat) CatatSekali(_ context.Context, _ int64, rw model.RiwayatTiket) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	f.dicatat = append(f.dicatat, rw)
	return true, nil
}

type fakeNotifikasi struct {
	keys   map[string]bool
	dibuat []model.Notifikasi
}

func (f *fakeNotifikasi) CreateSekali(_ context.Context, key string, n *model.Notifikasi) (bool, error) {
	if f.keys == nil {
		f.keys = map[string]bool{}
	}
	if f.keys[key] {
		return false, nil
	}
	f.keys[key] = true
	f.dibuat = append(f.dibuat, *n)
	return true, nil
}

type fakeAntrean struct {
	tiket  map[int64]*model.InfoTiket
	ketiga *model.InfoTiket
	loket  map[int64]string
	urutan int
}

func (f *fakeAntrean) TiketByID(_ context.Context, id int64) (*model.InfoTiket, error) {
	return f.tiket[id], nil
}

func (f *fakeAntrean) TiketMenungguKe(_ context.Context, _ int64, _ time.Time, urutan int) (*model.InfoTiket, error) {
	f.urutan = urutan
	return f.ketiga, nil
}

func (f *fakeAntrean) NamaLoket(_ context.Context, id int64) (string, error) {
	return f.loket[id], nil
}

func taskDari(t *testing.T, outboxID int64, eventType string, ev any) *asynq.Task {
	t.Helper()
	payload, _ := json.Marshal(ev)
	job, _ := json.Marshal(OutboxJob{OutboxID: outboxID, EventType: eventType, Payload: payload})
	return asynq.NewTask(TaskOutboxTiket, job)
}

func handlerUji(r *fakeRiwayat, n *fakeNotifikasi, a *fakeAntrean) *Handler {
	return NewHandler(r, n, a, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestProcessTaskPayloadRusakTidakDiulang(t *testing.T) {
	h := handlerUji(&fakeRiwayat{}, &fakeNotifikasi{}, &fakeAntrean{})

	err := h.ProcessTask(context.Background(), asynq.NewTask(TaskOutboxTiket, []byte("bukan json")))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("err = %v, mau SkipRetry", err)
	}

	job, _ := json.Marshal(OutboxJob{OutboxID: 1, EventType: model.EventTiketDiambil, Payload: json.RawMessage(`"teks"`)})
	err = h.ProcessTask(context.Background(), asynq.NewTask(TaskOutboxTiket, job))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("err = %v, mau SkipRetry", err)
	}
}

func TestProcessTaskTiketDiambilHanyaCatatRiwayat(t *testing.T) {
	r, n := &fakeRiwayat{}, &fakeNotifikasi{}
	h := handlerUji(r, n, &fakeAntrean{})

	err := h.ProcessTask(context.Background(), taskDari(t, 1, model.EventTiketDiambil,
		model.TiketEvent{TiketID: 10, LayananID: 1, UserID: 7, StatusBaru: "menunggu"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.dicatat) != 1 || r.dicatat[0].StatusBaru != "menunggu" || r.dicatat[0].Waktu.IsZero() {
		t.Fatalf("riwayat = %+v", r.dicatat)
	}
	if len(n.dibuat) != 0 {
		t.Fatalf("tidak boleh ada notifikasi, dapat %d", len(n.dibuat))
	}
}

func TestProcessTaskTiketDipanggil(t *testing.T) {
	loket := int64(2)
	tiket := &model.InfoTiket{ID: 10, UserID: 7, LayananID: 1, Kode: "A-001"}
	ketiga := &model.InfoTiket{ID: 14, UserID: 9, LayananID: 1, Kode: "A-005"}
	ev := model.TiketEvent{TiketID: 10, LayananID: 1, UserID: 7, LoketID: &loket, StatusLama: "menunggu", StatusBaru: "dipanggil"}

	tests := []struct {
		name       string
		antrean    *fakeAntrean
		wantNotif  int
		wantPesan0 string
	}{
		{
			name:       "ada antrean ketiga",
			antrean:    &fakeAntrean{tiket: map[int64]*model.InfoTiket{10: tiket}, ketiga: ketiga, loket: map[int64]string{2: "Loket 2"}},
			wantNotif:  2,
			wantPesan0: "Nomor A-001 dipanggil. Silakan menuju Loket 2.",
		},
		{
			name:       "antrean kurang dari tiga",
			antrean:    &fakeAntrean{tiket: map[int64]*model.InfoTiket{10: tiket}, loket: map[int64]string{2: "Loket 2"}},
			wantNotif:  1,
			wantPesan0: "Nomor A-001 dipanggil. Silakan menuju Loket 2.",
		},
		{
			name:       "nama loket tidak ditemukan",
			antrean:    &fakeAntrean{tiket: map[int64]*model.InfoTiket{10: tiket}},
			wantNotif:  1,
			wantPesan0: "Nomor A-001 dipanggil. Silakan menuju loket.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &fakeNotifikasi{}
			h := handlerUji(&fakeRiwayat{}, n, tt.antrean)

			if err := h.ProcessTask(context.Background(), taskDari(t, 5, model.EventTiketDipanggil, ev)); err != nil {
				t.Fatal(err)
			}
			if len(n.dibuat) != tt.wantNotif {
				t.Fatalf("notifikasi = %d, mau %d", len(n.dibuat), tt.wantNotif)
			}
			if n.dibuat[0].UserID != 7 || n.dibuat[0].Pesan != tt.wantPesan0 {
				t.Fatalf("notifikasi pertama = %+v", n.dibuat[0])
			}
			if tt.antrean.urutan != UrutanPengingat {
				t.Fatalf("urutan yang dicari = %d, mau %d", tt.antrean.urutan, UrutanPengingat)
			}
			if tt.wantNotif == 2 {
				if n.dibuat[1].UserID != 9 || !strings.Contains(n.dibuat[1].Pesan, "A-005") {
					t.Fatalf("notifikasi kedua = %+v", n.dibuat[1])
				}
			}
		})
	}
}

func TestProcessTaskDiulangTidakMembuatNotifikasiGanda(t *testing.T) {
	loket := int64(2)
	n := &fakeNotifikasi{}
	a := &fakeAntrean{
		tiket:  map[int64]*model.InfoTiket{10: {ID: 10, UserID: 7, LayananID: 1, Kode: "A-001"}},
		ketiga: &model.InfoTiket{ID: 14, UserID: 9, LayananID: 1, Kode: "A-005"},
	}
	h := handlerUji(&fakeRiwayat{}, n, a)
	task := taskDari(t, 5, model.EventTiketDipanggil, model.TiketEvent{TiketID: 10, LoketID: &loket})

	for i := 0; i < 3; i++ {
		if err := h.ProcessTask(context.Background(), task); err != nil {
			t.Fatal(err)
		}
	}
	if len(n.dibuat) != 2 {
		t.Fatalf("notifikasi = %d setelah 3 kali proses, mau 2", len(n.dibuat))
	}
}

func TestProcessTaskTiketTidakAdaTidakDiulang(t *testing.T) {
	h := handlerUji(&fakeRiwayat{}, &fakeNotifikasi{}, &fakeAntrean{})
	err := h.ProcessTask(context.Background(), taskDari(t, 5, model.EventTiketDipanggil, model.TiketEvent{TiketID: 99}))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("err = %v, mau SkipRetry", err)
	}
}

func TestProcessTaskMongoGagalDiulang(t *testing.T) {
	h := handlerUji(&fakeRiwayat{err: errors.New("mongo mati")}, &fakeNotifikasi{}, &fakeAntrean{})
	err := h.ProcessTask(context.Background(), taskDari(t, 1, model.EventTiketDiambil, model.TiketEvent{TiketID: 10}))
	if err == nil || errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("err = %v, mau error biasa supaya di-retry", err)
	}
}
