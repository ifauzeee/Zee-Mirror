package service_test

import (
	"errors"
	"testing"

	"zee-mirror/internal/config"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/repository/mocks"
	"zee-mirror/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestIsAuthorized(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	cfg := &config.Config{OwnerID: 12345}
	svc := &service.BotService{
		UserRepo: mockRepo,
		Config:   cfg,
	}

	t.Run("Owner", func(t *testing.T) {
		assert.True(t, svc.IsAuthorized(12345))
	})

	t.Run("Admin User", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, int64(67890)).Return(&domain.User{
			ID:       67890,
			Role:     service.RoleAdmin,
			IsActive: true,
		}, nil).Once()

		assert.True(t, svc.IsAuthorized(67890))
	})

	t.Run("Authorized User", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, int64(11111)).Return(&domain.User{
			ID:       11111,
			Role:     service.RoleAuthorized,
			IsActive: true,
		}, nil).Once()

		assert.True(t, svc.IsAuthorized(11111))
	})

	t.Run("Inactive User", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, int64(22222)).Return(&domain.User{
			ID:       22222,
			Role:     service.RoleAuthorized,
			IsActive: false,
		}, nil).Once()

		assert.False(t, svc.IsAuthorized(22222))
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, int64(33333)).Return(nil, errors.New("not found")).Once()

		assert.False(t, svc.IsAuthorized(33333))
	})
}

func TestCheckQuota(t *testing.T) {
	cfg := &config.Config{OwnerID: 12345}

	t.Run("Owner", func(t *testing.T) {
		mockRepo := new(mocks.MockRepository)
		svc := &service.BotService{UserRepo: mockRepo, DB: mockRepo, Config: cfg}

		assert.NoError(t, svc.CheckQuota(12345))
		mockRepo.AssertExpectations(t)
	})

	t.Run("User Not Found", func(t *testing.T) {
		mockRepo := new(mocks.MockRepository)
		svc := &service.BotService{UserRepo: mockRepo, DB: mockRepo, Config: cfg}

		mockRepo.On("GetByID", mock.Anything, int64(999)).Return(nil, errors.New("db error")).Once()

		err := svc.CheckQuota(999)
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("Inactive User", func(t *testing.T) {
		mockRepo := new(mocks.MockRepository)
		svc := &service.BotService{UserRepo: mockRepo, DB: mockRepo, Config: cfg}

		mockRepo.On("GetByID", mock.Anything, int64(888)).Return(&domain.User{
			ID:       888,
			IsActive: false,
		}, nil).Once()

		err := svc.CheckQuota(888)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "akun anda tidak aktif")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Unlimited Quota", func(t *testing.T) {
		mockRepo := new(mocks.MockRepository)
		svc := &service.BotService{UserRepo: mockRepo, DB: mockRepo, Config: cfg}

		mockRepo.On("GetByID", mock.Anything, int64(777)).Return(&domain.User{
			ID:                777,
			IsActive:          true,
			MaxDailyTasks:     -1,
			MaxDailyBandwidth: -1,
		}, nil).Once()

		assert.NoError(t, svc.CheckQuota(777))
		mockRepo.AssertExpectations(t)
	})

	t.Run("Task Limit Exceeded", func(t *testing.T) {
		mockRepo := new(mocks.MockRepository)
		svc := &service.BotService{UserRepo: mockRepo, DB: mockRepo, Config: cfg}

		mockRepo.On("GetByID", mock.Anything, int64(666)).Return(&domain.User{
			ID:                666,
			IsActive:          true,
			MaxDailyTasks:     5,
			MaxDailyBandwidth: -1,
		}, nil).Once()

		mockRepo.On("GetUserTodayStats", mock.Anything, int64(666)).Return(&domain.DailyStats{
			TotalTasks: 5,
		}, nil).Once()

		err := svc.CheckQuota(666)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kuota task harian habis")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Bandwidth Limit Exceeded", func(t *testing.T) {
		mockRepo := new(mocks.MockRepository)
		svc := &service.BotService{UserRepo: mockRepo, DB: mockRepo, Config: cfg}

		mockRepo.On("GetByID", mock.Anything, int64(555)).Return(&domain.User{
			ID:                555,
			IsActive:          true,
			MaxDailyTasks:     -1,
			MaxDailyBandwidth: 1000,
		}, nil).Once()

		mockRepo.On("GetUserTodayStats", mock.Anything, int64(555)).Return(&domain.DailyStats{
			TotalTasks:     1,
			TotalBandwidth: 1000,
		}, nil).Once()

		err := svc.CheckQuota(555)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kuota bandwidth harian habis")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Within Limits", func(t *testing.T) {
		mockRepo := new(mocks.MockRepository)
		svc := &service.BotService{UserRepo: mockRepo, DB: mockRepo, Config: cfg}

		mockRepo.On("GetByID", mock.Anything, int64(444)).Return(&domain.User{
			ID:                444,
			IsActive:          true,
			MaxDailyTasks:     5,
			MaxDailyBandwidth: 1000,
		}, nil).Once()

		mockRepo.On("GetUserTodayStats", mock.Anything, int64(444)).Return(&domain.DailyStats{
			TotalTasks:     4,
			TotalBandwidth: 500,
		}, nil).Once()

		assert.NoError(t, svc.CheckQuota(444))
		mockRepo.AssertExpectations(t)
	})
}
