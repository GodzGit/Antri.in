package helper

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError adalah error yang sudah "siap" dikirim ke client.
// Semua error dari service sebaiknya dikembalikan dalam bentuk ini,
// supaya error handler global bisa mengubahnya langsung jadi JSON.
type AppError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error membuat AppError memenuhi interface error bawaan Go.
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewError membuat AppError dengan status, code, dan message bebas.
func NewError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// ----- Constructor untuk kasus yang sering muncul -----

func BadRequest(message string) *AppError {
	return NewError(http.StatusBadRequest, "BAD_REQUEST", message)
}

func Unauthorized(message string) *AppError {
	return NewError(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(message string) *AppError {
	return NewError(http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(message string) *AppError {
	return NewError(http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(message string) *AppError {
	return NewError(http.StatusConflict, "CONFLICT", message)
}

func Internal(message string) *AppError {
	return NewError(http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// AsAppError mencoba meng-cast sebuah error menjadi *AppError.
// Kalau bukan *AppError, kembalikan nil. Ini dipakai error handler global.
func AsAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}