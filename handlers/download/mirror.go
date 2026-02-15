package download

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"zee-mirror/internal/service"
	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BtnTextCloudLink = "☁️ Cloud Link"
	BtnTextIndexURL  = "🌐 Index URL"
)

func HandleMirror(s *service.BotService, message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.Reply(message, service.GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}
	url, zip, unzip, password, quality, name, _, _ := utils.ParseFlags(args)
	var fileName string

	if name != "" {
		fileName = name
	}

	if message.ReplyToMessage != nil {
		fileID, replyName := s.ExtractFileFromReply(message.ReplyToMessage)
		if fileID != "" {
			go processTelegramFile(s, message, fileID, replyName, zip, unzip, password, quality)
			return
		}
	}

	if url != "" {
		if fileName == "" {
			fileName = utils.GetFileNameFromURL(url)
			if service.IsGenericName(fileName) {
				resolvedName := utils.ResolveFileName(url)
				if resolvedName != "" {
					fileName = resolvedName
					slog.Debug("Resolved filename from header", "filename", fileName)
				}
			}
		}

		if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
			HandleYTDLP(s, message, args)
			return
		}
		replyID := 0
		if message.ReplyToMessage != nil {
			replyID = message.ReplyToMessage.MessageID
		}
		task, err := s.TaskManager.CreateTask(service.TypeMirror, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0, "", false)
		if err != nil {
			s.HandleCreateTaskError(message.Chat.ID, message.MessageID, err)
			return
		}
		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.HandleAutoDelete(task)
		slog.Info("Mirror task created", "taskID", task.ID, "url", url)
		return
	}

	lang := s.GetUserLanguage(message.From.ID)
	msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "reply_required"))
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, _ = s.Bot.Send(msg)
}

func HandleLeech(s *service.BotService, message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.Reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
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
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		HandleYTDLPLeech(s, message, args)
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
	task, err := s.TaskManager.CreateTask(service.TypeLeech, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0, "", false)
	if err != nil {
		s.HandleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.HandleAutoDelete(task)
	slog.Info("Leech task created", "taskID", task.ID, "url", url)
}

func processTelegramFile(s *service.BotService, message *tgbotapi.Message, fileID, fileName string, zip, unzip bool, password, quality string) {
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
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}
	if fileName == "cookies.txt" && message.From.ID == s.Config.OwnerID {
		slog.Info("Cookies.txt upload detected from owner")
		destPath := filepath.Join(s.Config.ConfigDir, "cookies.txt")

		if filepath.IsAbs(tgFile.FilePath) {
			translatedPath := strings.Replace(tgFile.FilePath, "/var/lib/telegram-bot-api", s.Config.DownloadDir, 1)
			if copyErr := utils.CopyFile(translatedPath, destPath); copyErr == nil {
				s.Reply(message, "✅ *Cookies.txt Berhasil Diperbarui*")
				return
			}
		}

		if dlErr := s.DownloadFile(tgFile.Link(s.Bot.Token), destPath); dlErr == nil {
			s.Reply(message, "✅ *Cookies.txt Berhasil Diperbarui*")
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

	taskType := service.TypeMirror
	if strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
		taskType = service.TypeTorrent
	}

	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}

	if taskType == service.TypeTorrent {
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

func GetErrorMessage(title, content string) string {
	return fmt.Sprintf("❌ *%s*\n\n%s", title, content)
}
