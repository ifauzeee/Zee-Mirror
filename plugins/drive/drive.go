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
		return NewDriveUploader(cfg)
	})
}

type DriveUploader struct {
	Config *config.Config
}

func NewDriveUploader(cfg *config.Config) *DriveUploader {
	return &DriveUploader{Config: cfg}
}

func (u *DriveUploader) Upload(ctx context.Context, task *domain.Task, onProgress func(uploader.ProgressUpdate)) error {
	cfgCopy := *u.Config
	
	cfgCopy.RcloneDest = "drive:/"
	
	rcloneUploader := uploader.NewRcloneUploader(&cfgCopy)
	return rcloneUploader.Upload(ctx, task, onProgress)
}
