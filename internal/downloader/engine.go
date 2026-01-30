package downloader

import (
	"context"
	"time"
	"zee-mirror/internal/domain"
)

type ProgressUpdate struct {
	Downloaded  int64
	Total       int64
	Speed       int64
	Progress    float64
	ETA         time.Duration
	Connections int
	FileName    string
	Error       string
}

type DownloadEngine interface {
	Download(ctx context.Context, task *domain.Task, outputDir string, onProgress func(ProgressUpdate)) error
}
