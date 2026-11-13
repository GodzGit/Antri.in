package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/model"
	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/helper"
)

type fakeNotifikasiStore struct {
	items     []model.Notifikasi
	next      string
	listErr   error
	found     bool
	markErr   error
	gotUserID int64
	gotCursor string
	gotLimit  int
	gotMarkID string
}

func (f *fakeNotifikasiStore) ListByUser(_ context.Context, userID int64, cursor string, limit int) ([]model.Notifikasi, string, error) {
	f.gotUserID, f.gotCursor, f.gotLimit = userID, cursor, limit
	return f.items, f.next, f.listErr
}

func (f *fakeNotifikasiStore) MarkRead(_ context.Context, id string, userID int64) (bool, error) {
	f.gotMarkID, f.gotUserID = id, userID
	return f.found, f.markErr
}

func newNotifikasiApp(store *fakeNotifikasiStore, userID any) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: helper.Error})
	svc := NewNotifikasiService(store)
	login := func(c *fiber.Ctx) error {
		if userID != nil {
			c.Locals("user_id", userID)
		}
		return c.Next()
	}
	app.Get("/notifikasi/saya", login, svc.ListSaya)
	app.Patch("/notifikasi/:id/baca", login, svc.TandaiDibaca)
	return app
}

func readBody(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		t.Fatalf("body bukan JSON: %v", err)
	}
	return out
}

func TestNotifikasiListSaya(t *testing.T) {
	tests := []struct {
		name       string
		userID     any
		query      string
		store      fakeNotifikasiStore
		wantStatus int
		wantCode   string
		wantLimit  int
	}{
		{name: "belum login", userID: nil, wantStatus: 401, wantCode: "UNAUTHORIZED"},
		{name: "user_id bukan int64", userID: "7", wantStatus: 401, wantCode: "UNAUTHORIZED"},
		{name: "limit nol memakai default", userID: int64(7), query: "?limit=0", wantStatus: 200, wantLimit: 10},
		{name: "limit terlalu besar dibatasi", userID: int64(7), query: "?limit=500", wantStatus: 200, wantLimit: 100},
		{name: "cursor rusak", userID: int64(7), query: "?cursor=abc", store: fakeNotifikasiStore{listErr: repository.ErrIDTidakValid}, wantStatus: 400, wantCode: "BAD_REQUEST"},
		{name: "error database", userID: int64(7), store: fakeNotifikasiStore{listErr: errors.New("mongo mati")}, wantStatus: 500, wantCode: "INTERNAL_ERROR"},
		{name: "berhasil dengan limit default", userID: int64(7), store: fakeNotifikasiStore{items: []model.Notifikasi{{ID: "a", UserID: 7}}, next: "a"}, wantStatus: 200, wantLimit: 10},
		{name: "berhasil dengan limit dan cursor", userID: int64(7), query: "?limit=5&cursor=665f1c2b9d3e4a0012345678", wantStatus: 200, wantLimit: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.store
			app := newNotifikasiApp(&store, tt.userID)

			resp, err := app.Test(httptest.NewRequest("GET", "/notifikasi/saya"+tt.query, nil))
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, mau %d", resp.StatusCode, tt.wantStatus)
			}
			body := readBody(t, resp.Body)

			if tt.wantCode != "" {
				code := body["error"].(map[string]any)["code"]
				if code != tt.wantCode {
					t.Fatalf("code = %v, mau %s", code, tt.wantCode)
				}
				return
			}

			if store.gotUserID != 7 || store.gotLimit != tt.wantLimit {
				t.Fatalf("store dipanggil dengan user %d limit %d", store.gotUserID, store.gotLimit)
			}
			meta := body["meta"].(map[string]any)
			if meta["next_cursor"] != store.next || int(meta["limit"].(float64)) != tt.wantLimit {
				t.Fatalf("meta = %v", meta)
			}
		})
	}
}

func TestNotifikasiTandaiDibaca(t *testing.T) {
	tests := []struct {
		name       string
		userID     any
		store      fakeNotifikasiStore
		wantStatus int
		wantCode   string
	}{
		{name: "belum login", userID: nil, wantStatus: 401, wantCode: "UNAUTHORIZED"},
		{name: "id rusak", userID: int64(7), store: fakeNotifikasiStore{markErr: repository.ErrIDTidakValid}, wantStatus: 400, wantCode: "BAD_REQUEST"},
		{name: "milik orang lain atau tidak ada", userID: int64(7), store: fakeNotifikasiStore{found: false}, wantStatus: 404, wantCode: "NOT_FOUND"},
		{name: "error database", userID: int64(7), store: fakeNotifikasiStore{markErr: errors.New("mongo mati")}, wantStatus: 500, wantCode: "INTERNAL_ERROR"},
		{name: "berhasil", userID: int64(7), store: fakeNotifikasiStore{found: true}, wantStatus: 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.store
			app := newNotifikasiApp(&store, tt.userID)

			resp, err := app.Test(httptest.NewRequest("PATCH", "/notifikasi/665f1c2b9d3e4a0012345678/baca", nil))
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, mau %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantCode != "" {
				code := readBody(t, resp.Body)["error"].(map[string]any)["code"]
				if code != tt.wantCode {
					t.Fatalf("code = %v, mau %s", code, tt.wantCode)
				}
				return
			}
			if store.gotMarkID != "665f1c2b9d3e4a0012345678" || store.gotUserID != 7 {
				t.Fatalf("store dipanggil dengan id %q user %d", store.gotMarkID, store.gotUserID)
			}
		})
	}
}
