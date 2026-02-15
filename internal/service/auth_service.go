package service

import (
	"context"
	"fmt"

	"zee-mirror/pkg/utils"
)

const (
	RoleAdmin      = "admin"
	RoleAuthorized = "authorized"
	RoleOwner      = "owner"
)

func (s *BotService) IsAuthorized(userID int64) bool {
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

func (s *BotService) IsOwner(userID int64) bool {
	return userID == s.Config.OwnerID
}

func (s *BotService) IsAdmin(userID int64) bool {
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

func (s *BotService) CheckQuota(userID int64) error {
	if s.IsOwner(userID) {
		return nil
	}

	ctx := context.Background()
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if !user.IsActive {
		if user.ExpiresAt.Valid {
			return fmt.Errorf("akses anda telah berakhir pada %s", user.ExpiresAt.Time.Format("02 Jan 2006"))
		}
		return fmt.Errorf("akun anda tidak aktif")
	}

	if user.MaxDailyTasks == -1 && user.MaxDailyBandwidth == -1 {
		return nil
	}

	stats, err := s.DB.GetUserTodayStats(ctx, userID)
	if err != nil {
		return nil
	}

	if user.MaxDailyTasks != -1 && stats.TotalTasks >= user.MaxDailyTasks {
		return fmt.Errorf("kuota task harian habis (%d/%d)", stats.TotalTasks, user.MaxDailyTasks)
	}

	if user.MaxDailyBandwidth != -1 && stats.TotalBandwidth >= user.MaxDailyBandwidth {
		return fmt.Errorf("kuota bandwidth harian habis (%s/%s)", utils.FormatBytes(stats.TotalBandwidth), utils.FormatBytes(user.MaxDailyBandwidth))
	}

	return nil
}
