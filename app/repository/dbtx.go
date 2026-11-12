package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX adalah abstraksi minimal dari koneksi PostgreSQL.
// Tujuannya agar sebuah repository bisa menerima entah itu
// *pgxpool.Pool (koneksi biasa) ATAU pgx.Tx (di dalam transaksi),
// tanpa harus mengubah signature function-nya.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}