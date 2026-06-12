package drive

import (
	"context"

	"zee-mirror/internal/config"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/uploader"
	"zee-mirror/plugins/registry"
)

func init() {
	registry.RegisterFileUploader("drive", func(cfg *config.Config) uploader.FileUploader {
		return NewUploader(cfg)
	})
}

type Uploader struct {
	Config *config.Config
}

func NewUploader(cfg *config.Config) *Uploader {
	return &Uploader{Config: cfg}
}

func (u *Uploader) Upload(ctx context.Context, task *domain.Task, onProgress func(uploader.ProgressUpdate)) error {
	cfgCopy := *u.Config

	cfgCopy.RcloneDest = "drive:/"

	rcloneUploader := uploader.NewRcloneUploader(&cfgCopy)
	return rcloneUploader.Upload(ctx, task, onProgress)
}
