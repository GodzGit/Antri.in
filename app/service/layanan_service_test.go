package service_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/middleware"
)

// fakeLayananRepo adalah mock LayananStore untuk unit test service.
type fakeLayananRepo struct {
	list      []model.Layanan
	layanan   *model.Layanan
	err       error
	createErr error
	updateErr error
	deleteErr error

	hanyaAktif    bool
	created       *model.Layanan
	updated       *model.Layanan
	deletedID     int64
	getByIDCalled int
}

func (f *fakeLayananRepo) List(ctx context.Context, hanyaAktif bool) ([]model.Layanan, error) {
	f.hanyaAktif = hanyaAktif
	return f.list, f.err
}

func (f *fakeLayananRepo) GetByID(ctx context.Context, id int64) (*model.Layanan, error) {
	f.getByIDCalled++
	if f.layanan == nil {
		return nil, pgx.ErrNoRows
	}
	return f.layanan, f.err
}

func (f *fakeLayananRepo) Create(ctx context.Context, m *model.Layanan) error {
	f.created = m
	if f.createErr != nil {
		return f.createErr
	}
	if m.ID == 0 {
		m.ID = 42
	}
	return nil
}

func (f *fakeLayananRepo) Update(ctx context.Context, m *model.Layanan) error {
	f.updated = m
	return f.updateErr
}

func (f *fakeLayananRepo) SoftDelete(ctx context.Context, id int64) error {
	f.deletedID = id
	return f.deleteErr
}

func newLayananApp(svc *service.LayananService) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler()})
	api := app.Group("/api")

	api.Get("/layanan", svc.List)
	api.Get("/layanan/:id", svc.Get)
	api.Post("/layanan", svc.Create)
	api.Put("/layanan/:id", svc.Update)
	api.Delete("/layanan/:id", svc.Delete)

	return app
}

type layananResp struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func callLayanan(t *testing.T, app *fiber.App, method, path, body string) (int, layananResp) {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	defer resp.Body.Close()

	var out layananResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response gagal: %v", err)
	}
	return resp.StatusCode, out
}

func TestCreateLayanan_Sukses(t *testing.T) {
	repo := &fakeLayananRepo{}
	svc := service.NewLayananService(repo)

	body := `{"kode":"A","nama":"Akademik","jam_buka":"08:00","jam_tutup":"16:00","kuota_harian":10,"rata_durasi_menit":5}`
	code, resp := callLayanan(t, newLayananApp(svc), "POST", "/api/layanan", body)

	if code != fiber.StatusCreated {
		t.Fatalf("status = %d, mau 201 (%+v)", code, resp.Error)
	}
	if repo.created == nil {
		t.Fatal("repository Create tidak dipanggil")
	}
	if repo.created.Kode != "A" || repo.created.Nama != "Akademik" {
		t.Errorf("data tersimpan = %+v", repo.created)
	}
	if !repo.created.IsActive {
		t.Error("layanan baru harus aktif")
	}
	if repo.created.Deskripsi != nil {
		t.Errorf("deskripsi kosong harus nil, dapat %q", *repo.created.Deskripsi)
	}
	if repo.created.KuotaHarian == nil || *repo.created.KuotaHarian != 10 {
		t.Errorf("kuota harian salah: %+v", repo.created.KuotaHarian)
	}

	var data struct {
		ID   int64  `json:"id"`
		Kode string `json:"kode"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decode data gagal: %v", err)
	}
	if data.ID != 42 || data.Kode != "A" {
		t.Errorf("payload = %+v", data)
	}
}

func TestCreateLayanan_BodyKosong(t *testing.T) {
	repo := &fakeLayananRepo{}
	svc := service.NewLayananService(repo)

	code, resp := callLayanan(t, newLayananApp(svc), "POST", "/api/layanan", `{}`)

	if code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", code)
	}
	if resp.Error == nil || resp.Error.Code != "BAD_REQUEST" {
		t.Errorf("error = %+v, mau BAD_REQUEST", resp.Error)
	}
	if repo.created != nil {
		t.Error("repository tidak boleh dipanggil saat validasi gagal")
	}
}

func TestCreateLayanan_KodeDuplikat(t *testing.T) {
	repo := &fakeLayananRepo{
		createErr: &pgconn.PgError{Code: "23505", ConstraintName: "layanan_kode_key"},
	}
	svc := service.NewLayananService(repo)

	body := `{"kode":"A","nama":"Akademik","jam_buka":"08:00","jam_tutup":"16:00","rata_durasi_menit":5}`
	code, resp := callLayanan(t, newLayananApp(svc), "POST", "/api/layanan", body)

	if code != fiber.StatusConflict {
		t.Fatalf("status = %d, mau 409", code)
	}
	if resp.Error == nil || resp.Error.Code != "CONFLICT" {
		t.Errorf("error = %+v, mau CONFLICT", resp.Error)
	}
}

func TestGetLayanan_TidakAda(t *testing.T) {
	repo := &fakeLayananRepo{}
	svc := service.NewLayananService(repo)

	code, resp := callLayanan(t, newLayananApp(svc), "GET", "/api/layanan/99", "")

	if code != fiber.StatusNotFound {
		t.Fatalf("status = %d, mau 404", code)
	}
	if resp.Error == nil || resp.Error.Code != "NOT_FOUND" {
		t.Errorf("error = %+v, mau NOT_FOUND", resp.Error)
	}
}

func TestGetLayanan_IDTidakValid(t *testing.T) {
	repo := &fakeLayananRepo{}
	svc := service.NewLayananService(repo)

	code, _ := callLayanan(t, newLayananApp(svc), "GET", "/api/layanan/abc", "")

	if code != fiber.StatusBadRequest {
		t.Fatalf("status = %d, mau 400", code)
	}
	if repo.getByIDCalled != 0 {
		t.Error("repository tidak boleh dipanggil kalau id tidak valid")
	}
}

func TestListLayanan_FilterAktif(t *testing.T) {
	repo := &fakeLayananRepo{list: []model.Layanan{{ID: 1, Kode: "A"}}}
	svc := service.NewLayananService(repo)

	code, _ := callLayanan(t, newLayananApp(svc), "GET", "/api/layanan?aktif=true", "")

	if code != fiber.StatusOK {
		t.Fatalf("status = %d, mau 200", code)
	}
	if !repo.hanyaAktif {
		t.Error("query ?aktif=true harus diteruskan ke repository")
	}
}

func TestUpdateLayanan_Sukses(t *testing.T) {
	repo := &fakeLayananRepo{
		layanan: &model.Layanan{ID: 3, Kode: "B", Nama: "Klinik", IsActive: true},
	}
	svc := service.NewLayananService(repo)

	body := `{"kode":"B","nama":"Klinik","jam_buka":"09:00","jam_tutup":"15:00","rata_durasi_menit":8,"is_active":true}`
	code, resp := callLayanan(t, newLayananApp(svc), "PUT", "/api/layanan/3", body)

	if code != fiber.StatusOK {
		t.Fatalf("status = %d, mau 200 (%+v)", code, resp.Error)
	}
	if repo.updated == nil || repo.updated.ID != 3 || repo.updated.Nama != "Klinik" {
		t.Errorf("data update = %+v", repo.updated)
	}
	// Response memakai data hasil GetByID (bukan struct request).
	if repo.getByIDCalled != 1 {
		t.Errorf("GetByID dipanggil %d kali, mau 1", repo.getByIDCalled)
	}

	var data struct {
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decode data gagal: %v", err)
	}
	if data.Kode != "B" || data.Nama != "Klinik" {
		t.Errorf("payload = %+v", data)
	}
}

func TestDeleteLayanan_SoftDelete(t *testing.T) {
	repo := &fakeLayananRepo{}
	svc := service.NewLayananService(repo)

	code, _ := callLayanan(t, newLayananApp(svc), "DELETE", "/api/layanan/7", "")

	if code != fiber.StatusOK {
		t.Fatalf("status = %d, mau 200", code)
	}
	if repo.deletedID != 7 {
		t.Errorf("SoftDelete dipanggil dengan id %d, mau 7", repo.deletedID)
	}
}

func TestDeleteLayanan_TidakAda(t *testing.T) {
	repo := &fakeLayananRepo{deleteErr: pgx.ErrNoRows}
	svc := service.NewLayananService(repo)

	code, resp := callLayanan(t, newLayananApp(svc), "DELETE", "/api/layanan/7", "")

	if code != fiber.StatusNotFound {
		t.Fatalf("status = %d, mau 404", code)
	}
	if resp.Error == nil || resp.Error.Code != "NOT_FOUND" {
		t.Errorf("error = %+v, mau NOT_FOUND", resp.Error)
	}
}
