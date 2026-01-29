package handlers

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func GenerateThumbnail(videoPath, downloadDir string) (string, error) {
	videoPath = filepath.Clean(videoPath)
	if !filepath.IsAbs(videoPath) {
		return "", fmt.Errorf("video path must be absolute")
	}

	thumbnailPath := videoPath + ".jpg"

	allowedBaseDir := filepath.Clean(downloadDir)
	videoDir := filepath.Dir(videoPath)

	if !strings.HasPrefix(videoDir, allowedBaseDir) {
		return "", fmt.Errorf("video path is not within allowed directory")
	}

	cmd := exec.Command("ffmpeg", "-i", videoPath, "-ss", "00:00:05", "-vframes", "1", "-q:v", "2", thumbnailPath, "-y")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("ffmpeg", "-i", videoPath, "-ss", "00:00:00", "-vframes", "1", "-q:v", "2", thumbnailPath, "-y")
		if err := cmd.Run(); err != nil {
			return "", err
		}
	}
	return thumbnailPath, nil
}
