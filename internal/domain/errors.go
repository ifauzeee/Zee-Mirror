package domain

import (
	"errors"
	"fmt"
)

var (
	ErrUnauthorized   = errors.New("unauthorized: access denied")
	ErrNotFound       = errors.New("resource not found")
	ErrDuplicateTask  = errors.New("active task already exists")
	ErrLimitExceeded  = errors.New("daily usage limit exceeded")
	ErrInvalidInput   = errors.New("invalid input provided")
	ErrInternal       = errors.New("internal server error")
	ErrStorageFull    = errors.New("storage is full")
	ErrInvalidQuality = errors.New("invalid media quality")
	ErrTaskCancelled  = errors.New("task was cancelled")
	ErrTaskFailed     = errors.New("task execution failed")
)

type AppError struct {
	Err     error
	Code    string
	Message string
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func GetUserMessage(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, ErrUnauthorized) {
		return "⚠️ Akses ditolak. Silakan hubungi pemilik bot."
	}
	if errors.Is(err, ErrDuplicateTask) {
		return "🚫 Tugas yang sama sudah sedang berjalan."
	}
	if errors.Is(err, ErrLimitExceeded) {
		return "📉 Limit harian Anda telah habis."
	}
	if errors.Is(err, ErrNotFound) {
		return "🔍 File atau resource tidak ditemukan."
	}
	if errors.Is(err, ErrInvalidInput) {
		return "❓ Input tidak valid. Periksa kembali perintah Anda."
	}

	return "❌ Terjadi kesalahan internal. Silakan coba lagi nanti."
}
