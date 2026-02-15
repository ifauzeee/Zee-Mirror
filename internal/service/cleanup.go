package service

import (
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
