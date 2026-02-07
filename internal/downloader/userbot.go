package downloader

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
	"zee-mirror/internal/config"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/userbot"
)

type UserbotEngine struct {
	Config *config.Config
}

func NewUserbotEngine(cfg *config.Config) *UserbotEngine {
	return &UserbotEngine{Config: cfg}
}

func (e *UserbotEngine) Download(_ context.Context, task *domain.Task, outputDir string, onProgress func(ProgressUpdate)) error {
	ub := userbot.GetInstance(e.Config)
	if ub == nil || !ub.Started {
		return fmt.Errorf("userbot not started")
	}

	onProgress(ProgressUpdate{
		FileName: task.FileName,
		Progress: 0,
		Speed:    0,
	})

	startTime := time.Now()

	path, err := ub.DownloadFile(task.URL, outputDir, func(current, total int64) {
		duration := time.Since(startTime).Seconds()
		speed := int64(0)
		if duration > 0 {
			speed = int64(float64(current) / duration)
		}

		progress := float64(0)
		var eta time.Duration

		if total > 0 {
			progress = (float64(current) / float64(total)) * 100
			if speed > 0 {
				remaining := total - current
				eta = time.Duration(remaining/speed) * time.Second
			}
		}

		onProgress(ProgressUpdate{
			FileName:   task.FileName,
			Progress:   progress,
			Downloaded: current,
			Total:      total,
			Speed:      speed,
			ETA:        eta,
		})
	})

	if err != nil {
		return err
	}

	onProgress(ProgressUpdate{
		FileName:   filepath.Base(path),
		Progress:   100,
		Downloaded: 100,
		Total:      100,
	})

	return nil
}
