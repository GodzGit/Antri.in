package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/GodzGit/Antri.in/app/model"
)

const minSecretLength = 32

// GenerateAccessToken menghasilkan JWT access token yang ditandatangani menggunakan algoritma HS256.
// Fungsi ini menerima parameter secret dan duration secara eksplisit agar fleksibel terhubung dengan konfigurasi.
func GenerateAccessToken(userID int64, email, role, secret string, duration time.Duration) (string, error) {
	if len([]byte(secret)) < minSecretLength {
		return "", fmt.Errorf("jwt secret harus memiliki panjang minimal %d byte", minSecretLength)
	}

	if duration <= 0 {
		return "", errors.New("durasi token harus lebih dari 0")
	}

	now := time.Now()
	claims := model.JWTCustomClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("gagal menandatangani access token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken menghasilkan string acak aman sepanjang 32 byte (64 karakter heksadesimal)
// menggunakan crypto/rand sebagai sumber keacakan kriptografis yang aman.
// Token mentah ini dikembalikan untuk diserahkan kepada client.
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("gagal membuat refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken melakukan hashing pada token mentah menggunakan algoritma SHA-256.
// Nilai hash inilah yang disimpan pada tabel database 'refresh_tokens.token_hash',
// sehingga token mentah tidak pernah tersimpan di database dan hash tidak pernah diberikan ke client.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ValidateAccessToken memvalidasi JWT access token:
// 1. Memastikan panjang secret memenuhi standar minimal 32 byte.
// 2. Memastikan algoritma penandatanganan adalah HMAC-SHA256 (HS256).
// 3. Memverifikasi keabsahan signature menggunakan secret key.
// 4. Memverifikasi klaim exp wajib ada (WithExpirationRequired) dan token belum kedaluwarsa.
// Jika valid, fungsi mengembalikan pointer ke model.JWTCustomClaims.
func ValidateAccessToken(tokenString, secret string) (*model.JWTCustomClaims, error) {
	if len([]byte(secret)) < minSecretLength {
		return nil, fmt.Errorf("jwt secret harus memiliki panjang minimal %d byte", minSecretLength)
	}

	token, err := jwt.ParseWithClaims(tokenString, &model.JWTCustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Validasi tipe metode penandatanganan harus HMAC
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode penandatanganan token tidak valid: %v", t.Header["alg"])
		}

		// Validasi algoritma spesifik harus HS256
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("algoritma token harus HS256, ditemukan: %v", t.Method.Alg())
		}

		return []byte(secret), nil
	}, jwt.WithExpirationRequired())

	if err != nil {
		return nil, fmt.Errorf("token tidak valid atau telah kedaluwarsa: %w", err)
	}

	claims, ok := token.Claims.(*model.JWTCustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("klaim token tidak valid")
	}

	return claims, nil
}

