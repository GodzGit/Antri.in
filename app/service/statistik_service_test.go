package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/helper"
)

type fakeEstimasiStore struct {
	tiket     *model.InfoTiket
	depan     int
	loket     int
	durasi    float64
	historis  bool
	durasiErr error
}

func (f *fakeEstimasiStore) TiketByID(_ context.Context, _ int64) (*model.InfoTiket, error) {
	return f.tiket, nil
}

func (f *fakeEstimasiStore) OrangDiDepan(_ context.Context, _ int64, _ time.Time, _ int) (int, error) {
	return f.depan, nil
}

func (f *fakeEstimasiStore) LoketBuka(_ context.Context, _ int64) (int, error) {
	return f.loket, nil
}

func (f *fakeEstimasiStore) DurasiLayanan(_ context.Context, _ int64, _ int) (float64, bool, error) {
	return f.durasi, f.historis, f.durasiErr
}

type fakeRiwayatLister struct {
	items []model.RiwayatTiket
}

func (f *fakeRiwayatLister) ListByTiket(_ context.Context, _ int64) ([]model.RiwayatTiket, error) {
	return f.items, nil
}

type fakeStatistik struct {
	ada        bool
	gotDari    time.Time
	gotSampai  time.Time
	gotZona    string
	gotLayanan int64
}

func (f *fakeStatistik) LayananAda(_ context.Context, _ int64) (bool, error) {
	return f.ada, nil
}

func (f *fakeStatistik) Statistik(_ context.Context, id int64, dari, sampai time.Time, zona string) (model.StatistikLayanan, error) {
	f.gotLayanan, f.gotDari, f.gotSampai, f.gotZona = id, dari, sampai, zona
	return model.StatistikLayanan{LayananID: id, TotalTiket: 4}, nil
}

var jamUji = time.Date(2026, 11, 13, 9, 0, 0, 0, wib)

func appUji(userID any, daftar func(app *fiber.App)) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: helper.Error})
	app.Use(func(c *fiber.Ctx) error {
		if userID != nil {
			c.Locals("user_id", userID)
		}
		return c.Next()
	})
	daftar(app)
	return app
}

func panggil(t *testing.T, app *fiber.App, path string) (int, map[string]any) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, readBody(t, resp.Body)
}

func TestEstimasi(t *testing.T) {
	milik := func(status string) *model.InfoTiket {
		return &model.InfoTiket{ID: 3, UserID: 7, LayananID: 1, Kode: "A-003", Nomor: 3, Status: status}
	}

	tests := []struct {
		name         string
		userID       any
		path         string
		store        fakeEstimasiStore
		wantStatus   int
		wantCode     string
		wantEstimasi float64
		wantPosisi   float64
		wantSumber   string
	}{
		{name: "belum login", path: "/tiket/3/estimasi", wantStatus: 401, wantCode: "UNAUTHORIZED"},
		{name: "id bukan angka", userID: int64(7), path: "/tiket/abc/estimasi", wantStatus: 400, wantCode: "BAD_REQUEST"},
		{name: "tiket tidak ada", userID: int64(7), path: "/tiket/3/estimasi", wantStatus: 404, wantCode: "NOT_FOUND"},
		{name: "tiket milik orang lain", userID: int64(8), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("menunggu")}, wantStatus: 404, wantCode: "NOT_FOUND"},
		{name: "tiket sudah selesai", userID: int64(7), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("selesai")}, wantStatus: 409, wantCode: "CONFLICT"},
		{name: "sedang dipanggil", userID: int64(7), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("dipanggil")}, wantStatus: 200, wantEstimasi: 0, wantPosisi: 0},
		{name: "dua loket membagi antrean", userID: int64(7), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("menunggu"), depan: 5, loket: 2, durasi: 4}, wantStatus: 200, wantEstimasi: 10, wantPosisi: 6, wantSumber: "bawaan layanan"},
		{name: "loket tutup dihitung satu", userID: int64(7), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("menunggu"), depan: 3, loket: 0, durasi: 5}, wantStatus: 200, wantEstimasi: 15, wantPosisi: 4, wantSumber: "bawaan layanan"},
		{name: "dibulatkan ke atas dengan data historis", userID: int64(7), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("menunggu"), depan: 1, loket: 3, durasi: 4.2, historis: true}, wantStatus: 200, wantEstimasi: 2, wantPosisi: 2, wantSumber: "rata-rata 7 hari terakhir"},
		{name: "terdepan", userID: int64(7), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("menunggu"), loket: 1, durasi: 5}, wantStatus: 200, wantEstimasi: 0, wantPosisi: 1, wantSumber: "bawaan layanan"},
		{name: "layanan terhapus", userID: int64(7), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("menunggu"), durasiErr: repository.ErrDataTidakAda}, wantStatus: 404, wantCode: "NOT_FOUND"},
		{name: "database error", userID: int64(7), path: "/tiket/3/estimasi", store: fakeEstimasiStore{tiket: milik("menunggu"), durasiErr: errors.New("db mati")}, wantStatus: 500, wantCode: "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.store
			svc := NewEstimasiService(&store)
			svc.now = func() time.Time { return jamUji }
			app := appUji(tt.userID, func(app *fiber.App) { app.Get("/tiket/:id/estimasi", svc.Estimasi) })

			status, body := panggil(t, app, tt.path)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, mau %d (%v)", status, tt.wantStatus, body)
			}
			if tt.wantCode != "" {
				if code := body["error"].(map[string]any)["code"]; code != tt.wantCode {
					t.Fatalf("code = %v, mau %s", code, tt.wantCode)
				}
				return
			}
			data := body["data"].(map[string]any)
			if data["estimasi_menit"] != tt.wantEstimasi || data["posisi"] != tt.wantPosisi {
				t.Fatalf("estimasi = %v posisi = %v, mau %v dan %v", data["estimasi_menit"], data["posisi"], tt.wantEstimasi, tt.wantPosisi)
			}
			if tt.wantSumber != "" && data["sumber_durasi"] != tt.wantSumber {
				t.Fatalf("sumber = %v, mau %s", data["sumber_durasi"], tt.wantSumber)
			}
			want := jamUji.Add(time.Duration(tt.wantEstimasi) * time.Minute)
			got, _ := time.Parse(time.RFC3339, data["perkiraan_dipanggil"].(string))
			if !got.Equal(want) {
				t.Fatalf("perkiraan = %v, mau %v", got, want)
			}
		})
	}
}

func TestRiwayatHanyaPemilik(t *testing.T) {
	tiket := &model.InfoTiket{ID: 3, UserID: 7}
	lister := &fakeRiwayatLister{items: []model.RiwayatTiket{{TiketID: 3, StatusBaru: "menunggu"}, {TiketID: 3, StatusBaru: "dipanggil"}}}

	for _, tt := range []struct {
		name       string
		userID     int64
		wantStatus int
	}{
		{name: "pemilik", userID: 7, wantStatus: 200},
		{name: "orang lain", userID: 8, wantStatus: 404},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewRiwayatService(&fakeEstimasiStore{tiket: tiket}, lister)
			app := appUji(tt.userID, func(app *fiber.App) { app.Get("/tiket/:id/riwayat", svc.Riwayat) })
			status, body := panggil(t, app, "/tiket/3/riwayat")
			if status != tt.wantStatus {
				t.Fatalf("status = %d, mau %d", status, tt.wantStatus)
			}
			if status == 200 && len(body["data"].([]any)) != 2 {
				t.Fatalf("data = %v", body["data"])
			}
		})
	}
}

func TestStatistikLayanan(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		ada         bool
		wantStatus  int
		wantTanggal string
		wantDari    time.Time
	}{
		{name: "id bukan angka", path: "/statistik/layanan/x", wantStatus: 400},
		{name: "tanggal salah format", path: "/statistik/layanan/1?tanggal=13-11-2026", ada: true, wantStatus: 400},
		{name: "layanan tidak ada", path: "/statistik/layanan/1", wantStatus: 404},
		{name: "default hari ini WIB", path: "/statistik/layanan/1", ada: true, wantStatus: 200, wantTanggal: "2026-11-13", wantDari: time.Date(2026, 11, 13, 0, 0, 0, 0, wib)},
		{name: "tanggal dipilih", path: "/statistik/layanan/1?tanggal=2026-11-10", ada: true, wantStatus: 200, wantTanggal: "2026-11-10", wantDari: time.Date(2026, 11, 10, 0, 0, 0, 0, wib)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStatistik{ada: tt.ada}
			svc := NewStatistikService(store, store)
			svc.now = func() time.Time { return jamUji }
			app := appUji(int64(1), func(app *fiber.App) { app.Get("/statistik/layanan/:id", svc.Layanan) })

			status, body := panggil(t, app, tt.path)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, mau %d (%v)", status, tt.wantStatus, body)
			}
			if status != 200 {
				return
			}
			if body["data"].(map[string]any)["tanggal"] != tt.wantTanggal {
				t.Fatalf("tanggal = %v", body["data"])
			}
			if !store.gotDari.Equal(tt.wantDari) || !store.gotSampai.Equal(tt.wantDari.Add(24*time.Hour)) || store.gotZona != ZonaWIB {
				t.Fatalf("rentang = %v - %v zona %s", store.gotDari, store.gotSampai, store.gotZona)
			}
		})
	}
}
