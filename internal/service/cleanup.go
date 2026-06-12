package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func (s *BotService) startDiskCleanupWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.TaskManager.ShutdownChan:
			return
		case <-ticker.C:
			s.performDiskCleanup()
		}
	}
}

func (s *BotService) performDiskCleanup() {
	cutoff := time.Now().Add(-24 * time.Hour)

	entries, err := os.ReadDir(s.Config.DownloadDir)
	if err != nil {
		slog.Error("Error reading download dir", "error", err, "path", s.Config.DownloadDir)
		return
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(s.Config.DownloadDir, entry.Name())

			if s.TaskManager.GetTask(entry.Name()) != nil {
				continue
			}

			slog.Info("Removing old entry", "name", entry.Name())
			_ = os.RemoveAll(path)
		}
	}

	if usage := s.GetDiskUsage(); usage > 90 {
		slog.Warn("Disk usage critical", "usage", usage)
	}
}

func (s *BotService) GetDiskUsage() float64 {
	return s.getDiskUsageOS()
}

func (s *BotService) startDatabaseCleanupWorker() {
	if s.Config.AutoCleanupDays <= 0 {
		slog.Info("Database auto-cleanup is disabled (AUTO_CLEANUP_DAYS <= 0)")
		return
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	slog.Info("Database auto-cleanup worker started", "days", s.Config.AutoCleanupDays)

	s.performDatabaseCleanup()

	for {
		select {
		case <-s.TaskManager.ShutdownChan:
			return
		case <-ticker.C:
			s.performDatabaseCleanup()
		}
	}
}

func (s *BotService) performDatabaseCleanup() {
	ctx := context.Background()
	threshold := time.Now().AddDate(0, 0, -s.Config.AutoCleanupDays).Format("2006-01-02 15:04:05")

	deleted, err := s.DB.DeleteOld(ctx, threshold)
	if err != nil {
		slog.Error("Failed to perform database auto-cleanup", "error", err)
		return
	}

	if deleted > 0 {
		slog.Info("Database auto-cleanup completed", "deleted_tasks", deleted, "older_than_days", s.Config.AutoCleanupDays)
	}
}
