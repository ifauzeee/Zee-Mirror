package handlers

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

func (s *BotService) HandleMirror(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, service.GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}
	url, zip, unzip, password, quality, name, _, _ := utils.ParseFlags(args)

	var fileName string

	if name != "" {
		fileName = name
	}

	if url == "" && message.ReplyToMessage == nil {
		s.HandleMirrorWizard(message)
		return
	}

	if message.ReplyToMessage != nil {
		fileID, replyName, fileSize := s.extractFileFromReply(message.ReplyToMessage)
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
			s.HandleYTDLP(message, args)
			return
		}
		replyID := 0
		if message.ReplyToMessage != nil {
			replyID = message.ReplyToMessage.MessageID
		}
		task, err := s.TaskManager.CreateTask(service.TypeMirror, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0, "", false)
		if err != nil {
			s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
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
		slog.Info("Mirror task created", "taskID", task.ID, "url", url)

		confirmMsg := tgbotapi.NewMessage(message.Chat.ID,
			service.GetSuccessMessage("TASK QUEUED",
				fmt.Sprintf("Task: `%s`\nFile: `%s`",
					task.ID, utils.EscapeMarkdownV2(fileName))))
		confirmMsg.ParseMode = tgbotapi.ModeMarkdownV2
		if _, err := s.Bot.Send(confirmMsg); err != nil {
			slog.Error("Failed to send mirror confirmation", "error", err, "taskID", task.ID)
		}

		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.HandleAutoDelete(task)
		return
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "reply_required"))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}
}

func (s *BotService) extractFileFromReply(reply *tgbotapi.Message) (string, string, int64) {
	return s.BotService.ExtractFileFromReply(reply)
}

func (s *BotService) HandleLeech(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, service.GetErrorMessage("QUOTA EXCEEDED", err.Error()))
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
		fileID, replyName, fileSize := s.extractFileFromReply(message.ReplyToMessage)
		if fileID != "" {
			go s.HandleTelegramFileDownload(message, fileID, replyName, fileSize, service.TypeLeech, zip, unzip, password, quality)
			return
		}
	}

	if url != "" {
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
		task, err := s.TaskManager.CreateTask(service.TypeLeech, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0, "", false)
		if err != nil {
			s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
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
		slog.Info("Leech task created", "taskID", task.ID, "url", url)

		confirmMsg := tgbotapi.NewMessage(message.Chat.ID,
			service.GetSuccessMessage("TASK QUEUED",
				fmt.Sprintf("Task: `%s`\nFile: `%s`",
					task.ID, utils.EscapeMarkdownV2(fileName))))
		confirmMsg.ParseMode = tgbotapi.ModeMarkdownV2
		if _, err := s.Bot.Send(confirmMsg); err != nil {
			slog.Error("Failed to send leech confirmation", "error", err, "taskID", task.ID)
		}

		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.HandleAutoDelete(task)
	}
}
