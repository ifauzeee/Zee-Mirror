package handlers

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

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
		fileID, replyName := s.extractFileFromReply(message.ReplyToMessage)
		if fileID != "" {
			go s.HandleTelegramFileDownload(message, fileID, replyName, zip, unzip, password, quality)
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
		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.HandleAutoDelete(task)
		slog.Info("Mirror task created", "taskID", task.ID, "url", url)
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

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_url"))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
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
	task, err := s.TaskManager.CreateTask(service.TypeLeech, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0, "", false)
	if err != nil {
		s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.HandleAutoDelete(task)
	slog.Info("Leech task created", "taskID", task.ID, "url", url)
}
