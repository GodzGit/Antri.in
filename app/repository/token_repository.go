package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/GodzGit/Antri.in/app/model"
)

// TokenRepository mengelola persistensi dan siklus hidup refresh token pada tabel refresh_tokens di PostgreSQL.
type TokenRepository struct {
	db DBTX
}

// NewTokenRepository membuat instance baru TokenRepository dengan dependensi DBTX.
func NewTokenRepository(db DBTX) *TokenRepository {
	return &TokenRepository{db: db}
}

// WithTx mengembalikan salinan TokenRepository dengan koneksi transaksi tertentu.
func (r *TokenRepository) WithTx(tx DBTX) *TokenRepository {
	return &TokenRepository{db: tx}
}

// Create menyimpan refresh token baru ke database dan mengisi ID serta CreatedAt yang dihasilkan PostgreSQL.
// Nilai yang disimpan adalah hash token (SHA-256), bukan string token mentah.
func (r *TokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, revoked_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.RevokedAt,
	).Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		return fmt.Errorf("gagal menyimpan refresh token: %w", err)
	}

	return nil
}

// FindByHash mencari refresh token berdasarkan nilai hash uniknya.
// Mengembalikan nil, nil secara konsisten jika data token tidak ditemukan di database.
func (r *TokenRepository) FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var token model.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari refresh token berdasarkan hash: %w", err)
	}

	return &token, nil
}

// FindActiveByHash mencari refresh token yang masih aktif (belum dicabut dan belum kedaluwarsa).
// Mengembalikan nil, nil jika token tidak ditemukan, sudah dicabut, atau sudah melewati waktu kedaluwarsa.
func (r *TokenRepository) FindActiveByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`

	var token model.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari refresh token aktif: %w", err)
	}

	return &token, nil
}

// FindByHashForUpdate mencari refresh token dan mengunci barisnya menggunakan SELECT FOR UPDATE.
// Harus dipanggil di dalam transaksi aktif (pgx.Tx via WithTx).
// Pola ini memastikan hanya satu transaksi yang dapat memproses token yang sama secara bersamaan.
// Transaksi kedua akan menunggu sampai transaksi pertama selesai (commit atau rollback).
func (r *TokenRepository) FindByHashForUpdate(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		FOR UPDATE
	`

	var token model.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mengunci refresh token untuk diproses: %w", err)
	}

	return &token, nil
}

// Revoke mencabut sebuah refresh token dengan mengisi revoked_at = NOW().
// Klausa 'revoked_at IS NULL' memastikan waktu pencabutan awal tidak tertimpa jika token sudah dicabut sebelumnya.
// Mengembalikan pgx.ErrNoRows jika token sudah dicabut sebelumnya (RowsAffected == 0),
// sehingga pemanggil dapat membedakan antara "berhasil dicabut" dan "sudah dicabut lebih dulu".
func (r *TokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`

	tag, err := r.db.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("gagal mencabut refresh token: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// RevokeAllByUserID mencabut seluruh refresh token aktif milik user tertentu (misal saat logout semua sesi atau ganti password).
// Hanya memperbarui baris token yang saat ini belum dicabut (revoked_at IS NULL).
func (r *TokenRepository) RevokeAllByUserID(ctx context.Context, userID int64) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("gagal mencabut seluruh refresh token user: %w", err)
	}

	return nil
}

