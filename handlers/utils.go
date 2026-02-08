package handlers

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func IsMediaMessage(msg *tgbotapi.Message) bool {
	if msg == nil {
		return false
	}
	return msg.Document != nil ||
		msg.Video != nil ||
		msg.Audio != nil ||
		msg.Voice != nil ||
		msg.VideoNote != nil ||
		msg.Animation != nil ||
		len(msg.Photo) > 0 ||
		msg.Sticker != nil
}
