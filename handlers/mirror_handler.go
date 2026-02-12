package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BtnTextCloudLink = "☁️ Cloud Link"
	BtnTextIndexURL  = "🌐 Index URL"
)

func (s *BotService) HandleMirror(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}
	url, zip, unzip, password, quality, name, _, _ := utils.ParseFlags(args)
	var fileName string

	if name != "" {
		fileName = name
	}

	if message.ReplyToMessage != nil {
		fileID, replyName := s.extractFileFromReply(message.ReplyToMessage)
		if fileID != "" {
			go s.handleTelegramFileDownload(message, fileID, replyName, zip, unzip, password, quality)
			return
		}
	}

	if url != "" {
		if fileName == "" {
			fileName = utils.GetFileNameFromURL(url)
			if isGenericName(fileName) {
				resolvedName := utils.ResolveFileName(url)
				if resolvedName != "" {
					fileName = resolvedName
					slog.Debug("Resolved filename from header", "filename", fileName)
				}
			}
		}

		if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
			s.HandleYTDLP(message, args)
			return
		}
		replyID := 0
		if message.ReplyToMessage != nil {
			replyID = message.ReplyToMessage.MessageID
		}
		task, err := s.TaskManager.CreateTask(TypeMirror, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0, "", false)
		if err != nil {
			s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
			return
		}
		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.handleAutoDelete(task)
		slog.Info("Mirror task created", "taskID", task.ID, "url", url)
		return
	}

	lang := s.GetUserLanguage(message.From.ID)
	msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "reply_required"))
	msg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) extractFileFromReply(reply *tgbotapi.Message) (string, string) {
	var fileID, fileName string

	switch {
	case reply.Document != nil:
		fileID = reply.Document.FileID
		fileName = reply.Document.FileName
	case reply.Video != nil:
		fileID = reply.Video.FileID
		fileName = reply.Video.FileName
		if fileName == "" {
			fileName = fmt.Sprintf("video_%d.mp4", time.Now().Unix())
		}
	case reply.Audio != nil:
		fileID = reply.Audio.FileID
		fileName = reply.Audio.FileName
		if fileName == "" {
			fileName = fmt.Sprintf("audio_%d.mp3", time.Now().Unix())
		}
	case reply.Voice != nil:
		fileID = reply.Voice.FileID
		fileName = fmt.Sprintf("voice_%d.ogg", time.Now().Unix())
	case reply.VideoNote != nil:
		fileID = reply.VideoNote.FileID
		fileName = fmt.Sprintf("video_note_%d.mp4", time.Now().Unix())
	case reply.Animation != nil:
		fileID = reply.Animation.FileID
		fileName = reply.Animation.FileName
		if fileName == "" {
			fileName = fmt.Sprintf("animation_%d.mp4", time.Now().Unix())
		}
	case len(reply.Photo) > 0:
		photo := reply.Photo[len(reply.Photo)-1]
		fileID = photo.FileID
		fileName = fmt.Sprintf("photo_%d.jpg", time.Now().Unix())
	}

	return fileID, fileName
}

func (s *BotService) HandleLeech(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name, _, _ := utils.ParseFlags(args)
	if url == "" {
		url = utils.ExtractMagnetFromText(args)
	}
	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_url"))
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		s.HandleYTDLPLeech(message, args)
		return
	}

	fileName := name
	if fileName == "" {
		fileName = utils.GetFileNameFromURL(url)
	}
	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}
	task, err := s.TaskManager.CreateTask(TypeLeech, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0, "", false)
	if err != nil {
		s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.handleAutoDelete(task)
	slog.Info("Leech task created", "taskID", task.ID, "url", url)
}

func (s *BotService) handleTelegramFileDownload(message *tgbotapi.Message, fileID, fileName string, zip, unzip bool, password, quality string) {
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
		translatedPath := strings.Replace(tgFile.FilePath, "/var/lib/telegram-bot-api", s.Config.DownloadDir, 1)
		if _, errStat := os.Stat(translatedPath); errStat == nil {
			slog.Info("Local TG file detected", "path", translatedPath)
			fileURL = "file://" + translatedPath
		}
	}

	if fileURL == "" {
		if s.Config.TelegramAPI != "" && !isOfficial {
			fileEndpoint := strings.Replace(s.Config.TelegramAPI, "/bot%s/%s", "/file/bot%s/%s", 1)
			fileURL = fmt.Sprintf(fileEndpoint, s.Bot.Token, tgFile.FilePath)
		} else {
			fileURL = tgFile.Link(s.Bot.Token)
		}
	}

	slog.Debug("Telegram download initiated", "fileID", fileID, "filePath", tgFile.FilePath, "url", fileURL)

	taskType := TypeMirror
	if strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
		taskType = TypeTorrent
	}

	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}

	if taskType == TypeTorrent {
		s.showTorrentSelectionMenu(message, fileURL, fileName, zip, unzip, password, replyID)
		return
	}

	task, err := s.TaskManager.CreateTask(taskType, fileURL, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, int64(tgFile.FileSize), "", false)
	if err != nil {
		s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.handleAutoDelete(task)
	s.UpdateSharedDashboard(message.Chat.ID, true)
	slog.Info("Telegram download task created", "taskID", task.ID, "type", taskType)
}

func (s *BotService) handleLocalFileDownload(task *Task, outputDir string) {
	sourcePath := strings.TrimPrefix(task.URL, "file://")

	task.Mu.RLock()
	expectedSize := task.TotalSize
	task.Mu.RUnlock()

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

			if time.Since(lastUpdate) >= 3*time.Second {
				task.Mu.Lock()
				task.DownloadedSize = currentSize
				task.Progress = float64(currentSize) / float64(expectedSize) * 100
				task.Mu.Unlock()
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

		time.Sleep(1 * time.Second)
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

	task.Mu.Lock()
	task.TotalSize = info.Size()
	task.DownloadedSize = 0
	task.Progress = 0
	task.Mu.Unlock()
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

	task.Mu.Lock()
	task.DownloadedSize = 0
	task.Progress = 0
	task.Mu.Unlock()
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
			task.Mu.Lock()
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
			task.Mu.Unlock()
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

	task.Mu.Lock()
	task.DownloadedSize = task.TotalSize
	task.Progress = 100
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	s.handlePostDownload(task, outputDir)
}

func isGenericName(name string) bool {
	uuidRegex := regexp.MustCompile(`^[a-fA-F0-9]{8}(-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}$`)
	if uuidRegex.MatchString(name) {
		return true
	}

	hexRegex := regexp.MustCompile(`^[a-fA-F0-9]{16,}$`)
	return hexRegex.MatchString(name)
}
