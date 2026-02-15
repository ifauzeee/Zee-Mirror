package service

import (
	"context"
	"fmt"

	"zee-mirror/internal/config"
	"zee-mirror/internal/errors"
	"zee-mirror/internal/repository"
	"zee-mirror/pkg/utils"
)

const (
	RoleAdmin      = "admin"
	RoleAuthorized = "authorized"
	RoleOwner      = "owner"
)

type AuthService struct {
	Config   *config.Config
	UserRepo repository.UserRepository
	DB       repository.FullRepository
}

func NewAuthService(cfg *config.Config, userRepo repository.UserRepository, db repository.FullRepository) *AuthService {
	return &AuthService{
		Config:   cfg,
		UserRepo: userRepo,
		DB:       db,
	}
}

func (s *AuthService) IsAuthorized(userID int64) bool {
	if userID == s.Config.OwnerID {
		return true
	}

	ctx := context.Background()
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return false
	}

	if !user.IsActive {
		return false
	}

	return user.Role == RoleAdmin || user.Role == RoleAuthorized || user.Role == RoleOwner
}

func (s *AuthService) IsOwner(userID int64) bool {
	return userID == s.Config.OwnerID
}

func (s *AuthService) IsAdmin(userID int64) bool {
	if userID == s.Config.OwnerID {
		return true
	}

	ctx := context.Background()
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return false
	}

	if !user.IsActive {
		return false
	}

	return user.Role == RoleAdmin || user.Role == RoleOwner
}

func (s *AuthService) CheckQuota(userID int64) error {
	if s.IsOwner(userID) {
		return nil
	}

	ctx := context.Background()
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.ErrUserNotFound
	}

	if !user.IsActive {
		if user.ExpiresAt.Valid {
			return fmt.Errorf("%w: akses anda telah berakhir pada %s", errors.ErrAccountInactive, user.ExpiresAt.Time.Format("02 Jan 2006"))
		}
		return errors.ErrAccountInactive
	}

	if user.MaxDailyTasks == -1 && user.MaxDailyBandwidth == -1 {
		return nil
	}

	stats, err := s.DB.GetUserTodayStats(ctx, userID)
	if err != nil {
		return nil
	}

	if user.MaxDailyTasks != -1 && stats.TotalTasks >= user.MaxDailyTasks {
		return fmt.Errorf("%w: (%d/%d)", errors.ErrQuotaExceeded, stats.TotalTasks, user.MaxDailyTasks)
	}

	if user.MaxDailyBandwidth != -1 && stats.TotalBandwidth >= user.MaxDailyBandwidth {
		return fmt.Errorf("%w: bandwidth (%s/%s)", errors.ErrQuotaExceeded, utils.FormatBytes(stats.TotalBandwidth), utils.FormatBytes(user.MaxDailyBandwidth))
	}

	return nil
}

func (s *BotService) IsAuthorized(userID int64) bool {
	return s.Auth.IsAuthorized(userID)
}

func (s *BotService) IsOwner(userID int64) bool {
	return s.Auth.IsOwner(userID)
}

func (s *BotService) IsAdmin(userID int64) bool {
	return s.Auth.IsAdmin(userID)
}

func (s *BotService) CheckQuota(userID int64) error {
	return s.Auth.CheckQuota(userID)
}
