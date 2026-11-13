package model

type WebResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"berhasil"`
	Data    any    `json:"data,omitempty"`
}

type ErrorDetail struct {
	Code    string `json:"code" example:"NOT_FOUND"`
	Message string `json:"message" example:"data tidak ditemukan"`
}

type ErrorResponse struct {
	Success bool        `json:"success" example:"false"`
	Error   ErrorDetail `json:"error"`
}

type CursorMeta struct {
	NextCursor string `json:"next_cursor" example:"665f1c2b9d3e4a0012345678"`
	Limit      int    `json:"limit" example:"10"`
}

type CursorResponse struct {
	Success bool       `json:"success" example:"true"`
	Data    any        `json:"data"`
	Meta    CursorMeta `json:"meta"`
}
