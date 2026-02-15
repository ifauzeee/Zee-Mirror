package errors

import "errors"

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrCtxCancelled    = errors.New("context cancelled")
	ErrQuotaExceeded   = errors.New("quota limit exceeded")
	ErrAccountInactive = errors.New("account is inactive")
	ErrInvalidURL      = errors.New("invalid URL")
	ErrInternal        = errors.New("internal error")
)
