// internal/api/handler/error_response.go
package handler

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}
