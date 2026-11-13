package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RefreshToken merepresentasikan entitas tabel 'refresh_tokens' di database PostgreSQL.
type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	TokenHash string     `json:"-"` // Disembunyikan agar hash token tidak terekspos melalui JSON
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// RegisterRequest DTO untuk request pendaftaran akun baru oleh publik (POST /api/auth/register).
// Otomatis mendapatkan role 'pengunjung'.
type RegisterRequest struct {
	Nama     string `json:"nama" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginRequest DTO untuk request login user (POST /api/auth/login).
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshTokenRequest DTO untuk request pembaruan access token (POST /api/auth/refresh).
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest DTO untuk request pembatalan refresh token saat logout (POST /api/auth/logout).
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenResponse DTO untuk response autentikasi yang memuat token dan ringkasan profil user.
type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"` // Durasi masa berlaku token dalam detik
	User         UserResponse `json:"user"`
}

// JWTCustomClaims struct untuk memuat data identitas pengguna di dalam payload JWT,
// dilengkapi dengan jwt.RegisteredClaims untuk klaim standar JWT (seperti exp, iat, nbf, iss).
type JWTCustomClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

