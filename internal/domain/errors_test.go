package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"zee-mirror/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestCategorizeError(t *testing.T) {
	t.Run("NilError", func(t *testing.T) {
		result := domain.CategorizeError(nil)
		assert.Nil(t, result)
	})

	t.Run("AlreadyTypedError", func(t *testing.T) {
		netErr := &domain.NetworkError{Err: errors.New("timeout"), URL: "http://example.com"}
		result := domain.CategorizeError(netErr)
		assert.Equal(t, netErr, result)
	})

	t.Run("LinkExpired", func(t *testing.T) {
		err := errors.New("status=618 link expired")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrLinkExpired)
	})

	t.Run("TokenExpired", func(t *testing.T) {
		err := errors.New("token expired")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrLinkExpired)
	})

	t.Run("NotFound404", func(t *testing.T) {
		err := errors.New("HTTP 404 not found")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrNotFound)
	})

	t.Run("Forbidden403", func(t *testing.T) {
		err := errors.New("HTTP 403 forbidden")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrAuth)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		err := errors.New("unauthorized access")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrAuth)
	})

	t.Run("QuotaExceeded", func(t *testing.T) {
		err := errors.New("quota exceeded for user")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrQuota)
	})

	t.Run("RateLimit", func(t *testing.T) {
		err := errors.New("rate limit exceeded")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrQuota)
	})

	t.Run("TooManyRequests", func(t *testing.T) {
		err := errors.New("too many requests")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrQuota)
	})

	t.Run("ConnectionError", func(t *testing.T) {
		err := errors.New("connection refused")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrNetwork)
	})

	t.Run("TimeoutError", func(t *testing.T) {
		err := errors.New("timeout waiting for response")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrNetwork)
	})

	t.Run("DiskSpaceError", func(t *testing.T) {
		err := errors.New("no space left on device")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrStorage)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		err := errors.New("permission denied")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrStorage)
	})

	t.Run("Aria2Error", func(t *testing.T) {
		err := errors.New("aria2 download failed")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrExternal)
	})

	t.Run("YTDLPError", func(t *testing.T) {
		err := errors.New("yt-dlp extraction failed")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrExternal)
	})

	t.Run("RcloneError", func(t *testing.T) {
		err := errors.New("rclone sync failed")
		result := domain.CategorizeError(err)
		assert.ErrorIs(t, result, domain.ErrExternal)
	})

	t.Run("UnknownError", func(t *testing.T) {
		err := errors.New("something weird happened")
		result := domain.CategorizeError(err)
		assert.Equal(t, err, result)
	})
}

func TestIsRetryable(t *testing.T) {
	t.Run("NilError", func(t *testing.T) {
		assert.False(t, domain.IsRetryable(nil))
	})

	t.Run("LinkExpired_NotRetryable", func(t *testing.T) {
		err := fmt.Errorf("%w: expired", domain.ErrLinkExpired)
		assert.False(t, domain.IsRetryable(err))
	})

	t.Run("NotFound_NotRetryable", func(t *testing.T) {
		err := fmt.Errorf("%w: missing", domain.ErrNotFound)
		assert.False(t, domain.IsRetryable(err))
	})

	t.Run("Auth_NotRetryable", func(t *testing.T) {
		err := fmt.Errorf("%w: bad creds", domain.ErrAuth)
		assert.False(t, domain.IsRetryable(err))
	})

	t.Run("InvalidInput_NotRetryable", func(t *testing.T) {
		err := fmt.Errorf("%w: bad url", domain.ErrInvalidInput)
		assert.False(t, domain.IsRetryable(err))
	})

	t.Run("Network_Retryable", func(t *testing.T) {
		err := fmt.Errorf("%w: timeout", domain.ErrNetwork)
		assert.True(t, domain.IsRetryable(err))
	})

	t.Run("Storage_Retryable", func(t *testing.T) {
		err := fmt.Errorf("%w: disk full", domain.ErrStorage)
		assert.True(t, domain.IsRetryable(err))
	})

	t.Run("External_Retryable", func(t *testing.T) {
		err := fmt.Errorf("%w: aria2 failed", domain.ErrExternal)
		assert.True(t, domain.IsRetryable(err))
	})

	t.Run("GenericError_Retryable", func(t *testing.T) {
		err := errors.New("some random error")
		assert.True(t, domain.IsRetryable(err))
	})
}

func TestNetworkError(t *testing.T) {
	inner := errors.New("connection reset")
	err := &domain.NetworkError{Err: inner, URL: "http://example.com"}

	assert.Contains(t, err.Error(), "http://example.com")
	assert.Contains(t, err.Error(), "connection reset")
	assert.ErrorIs(t, err, domain.ErrNetwork)
}

func TestStorageError(t *testing.T) {
	inner := errors.New("disk full")
	err := &domain.StorageError{Err: inner, Path: "/downloads"}

	assert.Contains(t, err.Error(), "/downloads")
	assert.Contains(t, err.Error(), "disk full")
	assert.ErrorIs(t, err, domain.ErrStorage)
}

func TestQuotaError(t *testing.T) {
	inner := errors.New("limit reached")
	err := &domain.QuotaError{Err: inner, UserID: 12345, Limit: 10}

	assert.Contains(t, err.Error(), "12345")
	assert.Contains(t, err.Error(), "10")
	assert.ErrorIs(t, err, domain.ErrQuota)
}
