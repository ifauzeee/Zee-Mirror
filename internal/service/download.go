package service

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) HandleTelegramFileDownload(message *tgbotapi.Message, fileID, fileName string, fileSize int64, taskType TaskType, zip, unzip bool, password, quality string) {
	if strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
		taskType = TypeTorrent
	}

	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}

	if taskType != TypeTorrent {
		taskURL := "tgfileid://" + fileID
		task, err := s.TaskManager.CreateTask(taskType, taskURL, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, fileSize, "", false)
		if err != nil {
			s.HandleCreateTaskError(message.Chat.ID, message.MessageID, err)
			return
		}
		s.HandleAutoDelete(task)
		s.UpdateSharedDashboard(message.Chat.ID, true)
		slog.Info("Telegram download task created (Async)", "taskID", task.ID, "type", taskType)
		return
	}

	tgFile, isOfficial, err := s.GetFileWithFallback(fileID)
	if err != nil {
		slog.Error("Failed to get file from Telegram", "error", err, "fileID", fileID)
		errText := err.Error()
		lang := s.GetUserLanguage(message.From.ID)
		msgText := fmt.Sprintf("❌ *Error:* %s", utils.EscapeMarkdownV2(errText))

		if strings.Contains(errText, "file is too big") {
			msgText += i18n.T(lang, "telegram_file_limit")
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if fileName == "cookies.txt" && message.From.ID == s.Config.OwnerID {
		slog.Info("Cookies.txt upload detected from owner")
		destPath := filepath.Join(s.Config.ConfigDir, "cookies.txt")

		if filepath.IsAbs(tgFile.FilePath) {
			translatedPath := strings.Replace(tgFile.FilePath, "/var/lib/telegram-bot-api", s.Config.DownloadDir, 1)
			if copyErr := utils.CopyFile(translatedPath, destPath); copyErr == nil {
				s.reply(message, "✅ *Cookies.txt Berhasil Diperbarui*")
				return
			}
		}

		if dlErr := utils.DownloadFile(tgFile.Link(s.Bot.Token), destPath); dlErr == nil {
			s.reply(message, "✅ *Cookies.txt Berhasil Diperbarui*")
			return
		}
	}

	var fileURL string
	if filepath.IsAbs(tgFile.FilePath) {
		if _, errStat := os.Stat(tgFile.FilePath); errStat == nil {
			slog.Info("Local TG file detected (Direct path)", "path", tgFile.FilePath)
			fileURL = "file://" + tgFile.FilePath
		} else {
			slog.Warn("Local TG file path exists in metadata but not on disk", "path", tgFile.FilePath, "error", errStat)

			translatedPath := strings.Replace(tgFile.FilePath, "/var/lib/telegram-bot-api", s.Config.DownloadDir, 1)
			if _, errStat := os.Stat(translatedPath); errStat == nil {
				slog.Info("Local TG file detected (Translated path)", "path", translatedPath)
				fileURL = "file://" + translatedPath
			} else {
				slog.Warn("Translated path also not found", "path", translatedPath, "error", errStat)
			}
		}
	}

	if fileURL == "" {
		fileURL = s.GetFileLink(tgFile, isOfficial)
		if filepath.IsAbs(tgFile.FilePath) {
			slog.Debug("Local TG file failed disk checks, falling back to HTTP",
				"path", tgFile.FilePath,
				"url", fileURL)
		}
	}

	slog.Info("Telegram download initiated", "fileID", fileID, "filePath", tgFile.FilePath, "url", fileURL)

	if taskType == TypeTorrent {
		s.ShowTorrentSelectionMenu(message, fileURL, fileName, zip, unzip, password, replyID)
		return
	}

	task, err := s.TaskManager.CreateTask(taskType, fileURL, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, int64(tgFile.FileSize), "", false)
	if err != nil {
		s.HandleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.HandleAutoDelete(task)
	s.UpdateSharedDashboard(message.Chat.ID, true)
	slog.Info("Telegram download task created", "taskID", task.ID, "type", taskType)
}

func (s *BotService) HandleLocalFileDownload(task *Task, outputDir string) {
	sourcePath := strings.TrimPrefix(task.URL, "file://")

	var expectedSize int64
	task.Read(func() {
		expectedSize = task.TotalSize
	})

	s.updateTaskStatus(task)

	lastUpdate := time.Now()
	var sameSizeCount int
	var lastSize int64

	for {
		info, err := os.Stat(sourcePath)
		if err != nil {
			task.SetError(fmt.Sprintf("Local file not found: %v", err))
			s.updateTaskStatus(task)
			return
		}

		currentSize := info.Size()

		if expectedSize > 0 {
			if currentSize >= expectedSize {
				break
			}

			if time.Since(lastUpdate) >= 1*time.Second {
				task.Update(func() {
					task.DownloadedSize = currentSize
					if expectedSize > 0 {
						task.Progress = float64(currentSize) / float64(expectedSize) * 100
					}
				})
				s.updateTaskStatus(task)
				lastUpdate = time.Now()
			}
		} else {
			break
		}

		if currentSize == lastSize {
			sameSizeCount++
		} else {
			sameSizeCount = 0
		}
		lastSize = currentSize

		select {
		case <-task.Ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		task.SetError(fmt.Sprintf("Local file not found: %v", err))
		s.updateTaskStatus(task)
		return
	}

	fileName := task.FileName
	if fileName == "" || fileName == UnknownFile {
		fileName = filepath.Base(sourcePath)
	}
	destPath := filepath.Join(outputDir, fileName)

	task.Update(func() {
		task.TotalSize = info.Size()
		task.DownloadedSize = 0
		task.Progress = 0
	})
	s.updateTaskStatus(task)

	cleanedSource := filepath.Clean(sourcePath)
	source, err := os.Open(cleanedSource)
	if err != nil {
		task.SetError(fmt.Sprintf("Failed to open source file: %v", err))
		s.updateTaskStatus(task)
		return
	}
	defer func() { _ = source.Close() }()

	cleanedDest := filepath.Clean(destPath)
	dest, err := os.Create(cleanedDest)
	if err != nil {
		task.SetError(fmt.Sprintf("Failed to create destination file: %v", err))
		s.updateTaskStatus(task)
		return
	}
	defer func() { _ = dest.Close() }()

	buf := make([]byte, 32*1024)
	var copied int64
	startTime := time.Now()
	lastUpdate = time.Now()

	task.Update(func() {
		task.DownloadedSize = 0
		task.Progress = 0
	})
	s.updateTaskStatus(task)

	for {
		nr, readErr := source.Read(buf)
		if nr > 0 {
			nw, writeErr := dest.Write(buf[0:nr])
			if nw > 0 {
				copied += int64(nw)
			}
			if writeErr != nil {
				task.SetError(fmt.Sprintf("Failed to copy file: %v", writeErr))
				s.updateTaskStatus(task)
				return
			}
			if nr != nw {
				task.SetError("Failed to copy file: short write")
				s.updateTaskStatus(task)
				return
			}
		}

		if time.Since(lastUpdate) >= 1*time.Second {
			task.Update(func() {
				task.DownloadedSize = copied
				if task.TotalSize > 0 {
					task.Progress = float64(copied) / float64(task.TotalSize) * 100
				}
				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					task.Speed = int64(float64(copied) / elapsed)
					if task.Speed > 0 && task.TotalSize > 0 {
						remaining := task.TotalSize - copied
						task.ETA = time.Duration(remaining/task.Speed) * time.Second
					}
				}
			})
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			task.SetError(fmt.Sprintf("Failed to copy file: %v", readErr))
			s.updateTaskStatus(task)
			return
		}
	}

	task.Update(func() {
		task.DownloadedSize = task.TotalSize
		task.Progress = 100
	})
	s.updateTaskStatus(task)

	s.HandlePostDownload(task, outputDir)
}
