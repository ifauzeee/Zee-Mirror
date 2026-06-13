package download

import (
	"fmt"
	"log/slog"
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
		fileID, replyName, fileSize := s.ExtractFileFromReply(message.ReplyToMessage)
		if fileID != "" {
			go s.HandleTelegramFileDownload(message, fileID, replyName, fileSize, service.TypeMirror, zip, unzip, password, quality)
			return
		}
	}

	if url != "" {
		if fileName == "" {
			fileName = utils.GetFileNameFromURL(url)
			if utils.IsGenericName(fileName) {
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
		destStr := utils.ExtractDest(args)
		if destStr != "" {
			parts := strings.SplitN(destStr, "|", 2)
			task.Dest = parts[0]
			if len(parts) > 1 {
				task.Dest2 = parts[1]
			}
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

	if url == "" && message.ReplyToMessage == nil {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_url"))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if message.ReplyToMessage != nil {
		fileID, replyName, fileSize := s.ExtractFileFromReply(message.ReplyToMessage)
		if fileID != "" {
			go s.HandleTelegramFileDownload(message, fileID, replyName, fileSize, service.TypeLeech, zip, unzip, password, quality)
			return
		}
	}

	if url != "" {
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
		destStr := utils.ExtractDest(args)
		if destStr != "" {
			parts := strings.SplitN(destStr, "|", 2)
			task.Dest = parts[0]
			if len(parts) > 1 {
				task.Dest2 = parts[1]
			}
		}
		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.HandleAutoDelete(task)
		slog.Info("Leech task created", "taskID", task.ID, "url", url)
	}
}

func GetErrorMessage(title, content string) string {
	return fmt.Sprintf("❌ *%s*\n\n%s", title, content)
}
