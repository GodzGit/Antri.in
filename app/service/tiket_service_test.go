package service_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/middleware"
)

// ===================== Mock =====================

type fakeTiketStore struct {
	aktif    bool
	nomor    int
	inserted bool
	tiket    *model.Tiket
	list     []model.TiketSaya
	err      error

	ubahOK       bool
	batalOK      bool
	panggilOK    bool
	panggilTiket *model.Tiket
	panggilErr   error

	loket *model.LoketInfo
}

func (f *fakeTiketStore) CekTiketAktif(ctx context.Context, db repository.DBTX, userID, layananID int64, tanggal time.Time) (bool, error) {
	return f.aktif, f.err
}

func (f *fakeTiketStore) AmbilNomor(ctx context.Context, db repository.DBTX, layananID int64, tanggal time.Time) (int, error) {
	return f.nomor, f.err
}

func (f *fakeTiketStore) Insert(ctx context.Context, db repository.DBTX, t *model.Tiket) error {
	f.inserted = true
	if t.ID == 0 {
		t.ID = 99
	}
	return f.err
}

func (f *fakeTiketStore) GetByID(ctx context.Context, id int64) (*model.Tiket, error) {
	if f.tiket == nil {
		return nil, pgx.ErrNoRows
	}
	return f.tiket, f.err
}

func (f *fakeTiketStore) ListByUser(ctx context.Context, userID int64) ([]model.TiketSaya, error) {
	return f.list, f.err
}

func (f *fakeTiketStore) Posisi(ctx context.Context, layananID int64, tanggal time.Time, nomor int) (int, error) {
	return 0, f.err
}

func (f *fakeTiketStore) Papan(ctx context.Context, layananID int64, tanggal time.Time) (*model.PapanAntrean, error) {
	return &model.PapanAntrean{}, f.err
}

func (f *fakeTiketStore) PanggilBerikutnya(ctx context.Context, db repository.DBTX, layananID, loketID int64, tanggal time.Time) (*model.Tiket, error) {
	if f.panggilErr != nil {
		return nil, f.panggilErr
	}
	return f.panggilTiket, f.err
}

func (f *fakeTiketStore) UbahStatus(ctx context.Context, db repository.DBTX, id int64, dari, ke string) (bool, error) {
	return f.ubahOK, f.err
}

func (f *fakeTiketStore) PanggilUlang(ctx context.Context, db repository.DBTX, id int64, loketID *int64) (bool, error) {
	return f.panggilOK, f.err
}

func (f *fakeTiketStore) Batal(ctx context.Context, id, userID int64) (bool, error) {
	return f.batalOK, f.err
}

func (f *fakeTiketStore) GetLoketInfo(ctx context.Context, loketID int64) (*model.LoketInfo, error) {
	if f.loket == nil {
		return nil, pgx.ErrNoRows
	}
	return f.loket, f.err
}

type fakeLayananStore struct {
	layanan *model.Layanan
	err     error
}

func (f *fakeLayananStore) GetByID(ctx context.Context, id int64) (*model.Layanan, error) {
	if f.layanan == nil {
		return nil, pgx.ErrNoRows
	}
	return f.layanan, f.err
}

type fakeOutbox struct {
	types []string
	err   error
}

func (f *fakeOutbox) Insert(ctx context.Context, db repository.DBTX, eventType string, payload any) error {
	f.types = append(f.types, eventType)
	return f.err
}

// fakeTx menjalankan callback transaksi tanpa database sungguhan.
type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

// ===================== Helper =====================

type apiResp struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newTiketApp(svc *service.TiketService) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler()})
	api := app.Group("/api")

	api.Post("/layanan/:id/tiket", svc.AmbilNomor)
	api.Delete("/tiket/:id", svc.Batal)
	api.Get("/tiket/:id", svc.Detail)
	api.Post("/loket/:id/panggil", svc.Panggil)
	api.Post("/tiket/:id/selesai", svc.Selesai)
	api.Post("/tiket/:id/lewati", svc.Lewati)
	api.Post("/tiket/:id/panggil-ulang", svc.PanggilUlang)

	return app
}

func doRequest(t *testing.T, app *fiber.App, method, path, userID string) (int, apiResp) {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	defer resp.Body.Close()

	var out apiResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response gagal: %v", err)
	}
	return resp.StatusCode, out
}

func intPtr(v int) *int { return &v }

// ===================== Test: ambil nomor =====================

func TestAmbilNomor_Sukses(t *testing.T) {
	store := &fakeTiketStore{nomor: 1}
	outbox := &fakeOutbox{}
	layanan := &fakeLayananStore{layanan: &model.Layanan{
		ID: 1, Kode: "A", Nama: "Akademik", IsActive: true, KuotaHarian: intPtr(10),
	}}

	svc := service.NewTiketService(store, layanan, outbox, fakeTx{})
	code, resp := doRequest(t, newTiketApp(svc), "POST", "/api/layanan/1/tiket", "7")

	if code != fiber.StatusCreated {
		t.Fatalf("status = %d, mau 201 (%+v)", code, resp.Error)
	}

	var data struct {
		Kode   string `json:"kode"`
		Nomor  int    `json:"nomor"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decode data gagal: %v", err)
	}

	if data.Kode != "A-001" {
		t.Errorf("kode = %q, mau %q", data.Kode, "A-001")
	}
	if data.Status != model.StatusMenunggu {
		t.Errorf("status = %q, mau %q", data.Status, model.StatusMenunggu)
	}
	if !store.inserted {
		t.Error("tiket seharusnya disimpan")
	}
	if len(outbox.types) != 1 || outbox.types[0] != model.EventTiketDiambil {
		t.Errorf("outbox = %v, mau [%s]", outbox.types, model.EventTiketDiambil)
	}
}

func TestAmbilNomor_TiketAktif(t *testing.T) {
	store := &fakeTiketStore{aktif: true, nomor: 1}
	outbox := &fakeOutbox{}
	layanan := &fakeLayananStore{layanan: &model.Layanan{ID: 1, Kode: "A", IsActive: true}}

	svc := service.NewTiketService(store, layanan, outbox, fakeTx{})
	code, resp := doRequest(t, newTiketApp(svc), "POST", "/api/layanan/1/tiket", "7")

	if code != fiber.StatusConflict {
		t.Fatalf("status = %d, mau 409", code)
	}
	if store.inserted {
		t.Error("tiket tidak boleh disimpan kalau masih ada tiket aktif")
	}
	if len(outbox.types) != 0 {
		t.Errorf("outbox tidak boleh terisi, dapat %v", outbox.types)
	}
	if resp.Error == nil || resp.Error.Code != "CONFLICT" {
		t.Errorf("error code = %+v, mau CONFLICT", resp.Error)
	}
}

func TestAmbilNomor_KuotaHabis(t *testing.T) {
	store := &fakeTiketStore{nomor: 3}
	outbox := &fakeOutbox{}
	layanan := &fakeLayananStore{layanan: &model.Layanan{
		ID: 1, Kode: "A", IsActive: true, KuotaHarian: intPtr(2),
	}}

	svc := service.NewTiketService(store, layanan, outbox, fakeTx{})
	code, resp := doRequest(t, newTiketApp(svc), "POST", "/api/layanan/1/tiket", "7")

	if code != fiber.StatusConflict {
		t.Fatalf("status = %d, mau 409", code)
	}
	if resp.Error == nil || resp.Error.Code != "KUOTA_HABIS" {
		t.Errorf("error = %+v, mau KUOTA_HABIS", resp.Error)
	}
	if store.inserted {
		t.Error("tiket tidak boleh disimpan saat kuota habis")
	}
	if len(outbox.types) != 0 {
		t.Errorf("outbox tidak boleh terisi, dapat %v", outbox.types)
	}
}

func TestAmbilNomor_LayananTidakAktif(t *testing.T) {
	store := &fakeTiketStore{nomor: 1}
	outbox := &fakeOutbox{}
	layanan := &fakeLayananStore{layanan: &model.Layanan{ID: 1, Kode: "A", IsActive: false}}

	svc := service.NewTiketService(store, layanan, outbox, fakeTx{})
	code, _ := doRequest(t, newTiketApp(svc), "POST", "/api/layanan/1/tiket", "7")

	if code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", code)
	}
	if store.inserted || len(outbox.types) != 0 {
		t.Error("layanan tidak aktif tidak boleh membuat tiket/outbox")
	}
}

func TestAmbilNomor_TanpaLogin(t *testing.T) {
	store := &fakeTiketStore{nomor: 1}
	outbox := &fakeOutbox{}
	layanan := &fakeLayananStore{layanan: &model.Layanan{ID: 1, Kode: "A", IsActive: true}}

	svc := service.NewTiketService(store, layanan, outbox, fakeTx{})
	code, _ := doRequest(t, newTiketApp(svc), "POST", "/api/layanan/1/tiket", "")

	if code != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", code)
	}
	if store.inserted {
		t.Error("tanpa login tidak boleh membuat tiket")
	}
}

// ===================== Test: batal =====================

func TestBatal_BukanMilik(t *testing.T) {
	tiket := &model.Tiket{ID: 5, UserID: 8, Status: model.StatusMenunggu}
	store := &fakeTiketStore{tiket: tiket, batalOK: true}
	svc := service.NewTiketService(store, &fakeLayananStore{}, &fakeOutbox{}, fakeTx{})

	code, _ := doRequest(t, newTiketApp(svc), "DELETE", "/api/tiket/5", "7")

	if code != fiber.StatusForbidden {
		t.Fatalf("status = %d, mau 403", code)
	}
}

func TestBatal_Sukses(t *testing.T) {
	tiket := &model.Tiket{ID: 5, UserID: 7, LayananID: 1, Status: model.StatusMenunggu}
	store := &fakeTiketStore{tiket: tiket, batalOK: true}
	outbox := &fakeOutbox{}
	svc := service.NewTiketService(store, &fakeLayananStore{}, outbox, fakeTx{})

	code, _ := doRequest(t, newTiketApp(svc), "DELETE", "/api/tiket/5", "7")

	if code != fiber.StatusOK {
		t.Fatalf("status = %d, mau 200", code)
	}
	if len(outbox.types) != 1 || outbox.types[0] != model.EventTiketDibatalkan {
		t.Errorf("outbox = %v, mau [%s]", outbox.types, model.EventTiketDibatalkan)
	}
}

// ===================== Test: selesai =====================

func TestSelesai_StatusSalah(t *testing.T) {
	tiket := &model.Tiket{ID: 5, UserID: 7, Status: model.StatusMenunggu}
	store := &fakeTiketStore{tiket: tiket, ubahOK: true}
	outbox := &fakeOutbox{}
	svc := service.NewTiketService(store, &fakeLayananStore{}, outbox, fakeTx{})

	code, _ := doRequest(t, newTiketApp(svc), "POST", "/api/tiket/5/selesai", "9")

	if code != fiber.StatusConflict {
		t.Fatalf("status = %d, mau 409", code)
	}
	if len(outbox.types) != 0 {
		t.Errorf("outbox tidak boleh terisi, dapat %v", outbox.types)
	}
}

func TestSelesai_Sukses(t *testing.T) {
	tiket := &model.Tiket{ID: 5, UserID: 7, LayananID: 1, Status: model.StatusDipanggil}
	store := &fakeTiketStore{tiket: tiket, ubahOK: true}
	outbox := &fakeOutbox{}
	svc := service.NewTiketService(store, &fakeLayananStore{}, outbox, fakeTx{})

	code, _ := doRequest(t, newTiketApp(svc), "POST", "/api/tiket/5/selesai", "9")

	if code != fiber.StatusOK {
		t.Fatalf("status = %d, mau 200", code)
	}
	if len(outbox.types) != 1 || outbox.types[0] != model.EventTiketSelesai {
		t.Errorf("outbox = %v, mau [%s]", outbox.types, model.EventTiketSelesai)
	}
}

// ===================== Test: panggil =====================

func TestPanggil_LoketTutup(t *testing.T) {
	store := &fakeTiketStore{loket: &model.LoketInfo{ID: 3, LayananID: 1, Status: model.LoketTutup}}
	outbox := &fakeOutbox{}
	svc := service.NewTiketService(store, &fakeLayananStore{}, outbox, fakeTx{})

	code, _ := doRequest(t, newTiketApp(svc), "POST", "/api/loket/3/panggil", "12")

	if code != fiber.StatusConflict {
		t.Fatalf("status = %d, mau 409", code)
	}
	if len(outbox.types) != 0 {
		t.Errorf("outbox tidak boleh terisi, dapat %v", outbox.types)
	}
}

func TestPanggil_AntreanKosong(t *testing.T) {
	store := &fakeTiketStore{
		loket:      &model.LoketInfo{ID: 3, LayananID: 1, Status: model.LoketBuka},
		panggilErr: pgx.ErrNoRows,
	}
	svc := service.NewTiketService(store, &fakeLayananStore{}, &fakeOutbox{}, fakeTx{})

	code, resp := doRequest(t, newTiketApp(svc), "POST", "/api/loket/3/panggil", "12")

	if code != fiber.StatusNotFound {
		t.Fatalf("status = %d, mau 404", code)
	}
	if resp.Error == nil || resp.Error.Code != "NOT_FOUND" {
		t.Errorf("error = %+v, mau NOT_FOUND", resp.Error)
	}
}

func TestPanggil_Sukses(t *testing.T) {
	store := &fakeTiketStore{
		loket: &model.LoketInfo{ID: 3, LayananID: 1, Status: model.LoketBuka},
		panggilTiket: &model.Tiket{
			ID: 5, LayananID: 1, UserID: 7, Status: model.StatusMenunggu, Nomor: 1, Kode: "A-001",
		},
	}
	outbox := &fakeOutbox{}
	svc := service.NewTiketService(store, &fakeLayananStore{}, outbox, fakeTx{})

	code, _ := doRequest(t, newTiketApp(svc), "POST", "/api/loket/3/panggil", "12")

	if code != fiber.StatusOK {
		t.Fatalf("status = %d, mau 200", code)
	}
	if len(outbox.types) != 1 || outbox.types[0] != model.EventTiketDipanggil {
		t.Errorf("outbox = %v, mau [%s]", outbox.types, model.EventTiketDipanggil)
	}
}
