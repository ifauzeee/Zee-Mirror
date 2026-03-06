package service

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"zee-mirror/internal/metrics"
	"zee-mirror/internal/organizer"
	"zee-mirror/internal/uploader"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) UploadWithRclone(task *Task) error {
	task.SetStatus(StatusUploading)
	task.Mu.Lock()
	task.Progress = 0
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	err := s.RcloneUploader.Upload(task.Ctx, &task.Task, func(up uploader.ProgressUpdate) {
		task.Mu.Lock()
		if up.UploadedSize > 0 {
			task.UploadedSize = up.UploadedSize
		}
		if up.TotalSize > 0 {
			task.TotalSize = up.TotalSize
		}
		if up.Progress > 0 {
			task.Progress = up.Progress
		}
		if up.Speed > 0 {
			task.Speed = up.Speed
		}
		if up.ETA > 0 {
			task.ETA = up.ETA
		}
		task.Mu.Unlock()

		s.updateTaskStatus(task)
	})

	if err != nil {
		return err
	}

	task.Mu.Lock()
	task.Progress = 100
	task.Mu.Unlock()
	return nil
}

func (s *BotService) UploadToTelegram(task *Task) error {
	task.SetStatus(StatusUploading)
	task.Mu.Lock()
	task.Progress = 0
	task.Mu.Unlock()
	s.updateTaskStatus(task)
	startTime := time.Now()

	filePath := task.LocalPath
	if filePath == "" {
		return fmt.Errorf("no file to upload")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	if info.IsDir() {
		return fmt.Errorf("cannot upload directory to telegram directly, please zip it first")
	}

	if info.Size() > 2*1024*1024*1024 {
		return fmt.Errorf("file too large for telegram (max 2GB)")
	}

	var msg tgbotapi.Chattable
	if organizer.IsVideoFile(filePath) {
		video := tgbotapi.NewVideo(task.ChatID, tgbotapi.FilePath(filePath))
		video.Caption = fmt.Sprintf("📄 %s", task.FileName)

		if thumb, errThumb := GenerateThumbnail(filePath, s.TaskManager.DownloadDir); errThumb == nil {
			video.Thumb = tgbotapi.FilePath(thumb)
			defer func() {
				if errRemove := os.Remove(thumb); errRemove != nil {
					slog.Warn("Failed to remove thumbnail", "error", errRemove, "path", thumb)
				}
			}()
		}

		msg = video
	} else {
		doc := tgbotapi.NewDocument(task.ChatID, tgbotapi.FilePath(filePath))
		doc.Caption = fmt.Sprintf("📄 %s", task.FileName)
		msg = doc
	}

	task.Mu.Lock()
	task.Progress = 50
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	sentMsg, err := s.Bot.Send(msg)
	if err != nil {
		metrics.UploadDuration.WithLabelValues("telegram", "failed").Observe(time.Since(startTime).Seconds())
		return fmt.Errorf("telegram upload failed: %v", err)
	}
	metrics.UploadDuration.WithLabelValues("telegram", "success").Observe(time.Since(startTime).Seconds())

	task.Mu.Lock()
	task.ResultMessageID = sentMsg.MessageID
	task.Progress = 100
	task.UploadedSize = info.Size()
	task.RemotePath = "telegram"
	task.Mu.Unlock()

	return nil
}
