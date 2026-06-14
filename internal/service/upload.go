package service

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"zee-mirror/internal/domain"
	"zee-mirror/internal/metrics"
	"zee-mirror/internal/organizer"
	"zee-mirror/internal/uploader"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) UploadWithRclone(task *Task) error {
	task.SetStatus(StatusUploading)
	task.SetProgress(0)
	s.updateTaskStatus(task)

	err := s.RcloneUploader.Upload(task.Ctx, &task.Task, func(up uploader.ProgressUpdate) {
		task.UpdateFromUploadProgress(up)
		s.updateTaskStatus(task)
	})

	if err != nil {
		return err
	}

	task.SetProgress(100)
	return nil
}

func (s *BotService) UploadToTelegram(task *Task) error {
	task.SetStatus(StatusUploading)
	task.SetProgress(0)
	s.updateTaskStatus(task)
	startTime := time.Now()

	filePath := task.LocalPath
	if filePath == "" {
		return fmt.Errorf("%w: no local file to upload", domain.ErrInvalidInput)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("%w: failed to stat file: %v", domain.ErrStorage, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%w: cannot upload directory to telegram directly, please zip it first", domain.ErrInvalidInput)
	}

	if info.Size() > domain.MaxTelegramUploadSize {
		return fmt.Errorf("%w: file too large for telegram (max 2GB)", domain.ErrInvalidInput)
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

	task.SetProgress(50)
	s.updateTaskStatus(task)

	sentMsg, err := s.Bot.Send(msg)
	if err != nil {
		metrics.UploadDuration.WithLabelValues("telegram", "failed").Observe(time.Since(startTime).Seconds())
		return fmt.Errorf("%w: telegram upload failed: %v", domain.ErrExternal, err)
	}
	metrics.UploadDuration.WithLabelValues("telegram", "success").Observe(time.Since(startTime).Seconds())

	var fileID, tgFilePath string
	switch {
	case sentMsg.Document != nil:
		fileID = sentMsg.Document.FileID
	case sentMsg.Video != nil:
		fileID = sentMsg.Video.FileID
	case sentMsg.Audio != nil:
		fileID = sentMsg.Audio.FileID
	case sentMsg.Animation != nil:
		fileID = sentMsg.Animation.FileID
	default:
		if len(sentMsg.Photo) > 0 {
			fileID = sentMsg.Photo[0].FileID
		}
	}
	if fileID != "" {
		if tgFile, err := s.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID}); err == nil {
			tgFilePath = tgFile.FilePath
		} else {
			slog.Warn("Failed to get Telegram file path", "error", err, "fileID", fileID)
		}
	}

	task.CompleteTelegramUpload(sentMsg.MessageID, info.Size(), fileID, tgFilePath)

	return nil
}
