package repository

import (
	"context"
	"zee-mirror/internal/domain"
)

type TaskRepository interface {
	Save(ctx context.Context, task domain.TaskRecord) error
	GetTaskByID(ctx context.Context, id string) (*domain.TaskRecord, error)
	GetActive(ctx context.Context) ([]domain.TaskRecord, error)
	GetRecoverable(ctx context.Context) ([]domain.TaskRecord, error)
	UpdateStatus(ctx context.Context, id, status, err string) error
	DeleteOld(ctx context.Context, before string) (int, error)

	// Extended methods for stats and recovery
	GetBotStats(ctx context.Context) (map[string]interface{}, error)
	GetUserStats(ctx context.Context, userID int64) (*domain.UserStats, error)
	GetTodayStats(ctx context.Context) (*domain.DailyStats, error)
	GetUserTodayStats(ctx context.Context, userID int64) (*domain.DailyStats, error)
	GetWeeklyStats(ctx context.Context) ([]domain.DailyStats, error)
	GetMonthlyStats(ctx context.Context) ([]domain.DailyStats, error)
}

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (string, string, error)
	Upsert(ctx context.Context, id int64, username, role string) error
	SetRole(ctx context.Context, id int64, role string) error
	GetAll(ctx context.Context) ([]map[string]interface{}, error)
	GetCount(ctx context.Context) (int, error)
}

type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

type FullRepository interface {
	TaskRepository
	UserRepository
	SettingsRepository
	Ping(ctx context.Context) error
}
