package drive

import (
	"zee-mirror/internal/config"
)

func init() {
}
type DriveUploader struct {
	Config *config.Config
}

func NewDriveUploader(cfg *config.Config) *DriveUploader {
	return &DriveUploader{Config: cfg}
}
