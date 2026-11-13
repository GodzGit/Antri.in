package helper

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate adalah instance global validator v10.
var validate = validator.New()

// ValidateStruct memvalidasi struct menggunakan tag `validate:"..."`.
// Jika gagal, dia mengembalikan *AppError dengan status 400 (Bad Request)
// dan pesan error yang rapi, siap dikirim ke user.
func ValidateStruct(s any) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	// Ubah error bawaan validator jadi pesan yang lebih manusiawi.
	if errs, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range errs {
			messages = append(messages, fmt.Sprintf("field '%s' gagal pada aturan '%s'", e.Field(), e.Tag()))
		}
		return BadRequest(strings.Join(messages, ", "))
	}

	return BadRequest("validasi gagal")
}