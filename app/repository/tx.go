package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager adalah alat untuk menjalankan beberapa operasi database
// sekaligus dalam satu transaksi (BEGIN ... COMMIT / ROLLBACK).
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager membuat TxManager baru dari sebuah pgxpool.Pool.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithTx menjalankan fn di dalam satu transaksi.
// - Kalau fn mengembalikan nil, transaksi di-COMMIT.
// - Kalau fn mengembalikan error, transaksi di-ROLLBACK dan error dikembalikan.
// - Kalau fn panic, transaksi di-ROLLBACK lalu panic dilempar ulang.
func (m *TxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}

	// Pastikan rollback terjadi kalau ada panic atau error.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}