package downloader

import (
	"context"
	"fmt"
	"path/filepath"
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

	path, err := ub.DownloadFile(task.URL, outputDir)
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
