package service

import (
	"log/slog"
	"os"

	"zee-mirror/internal/organizer"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BtnTextCloudLink = "☁️ Cloud Link"
	BtnTextIndexURL  = "🌐 Index URL"
)

func (s *BotService) updateTaskStatus(task *Task) {
	snapshot := task.GetSnapshot()

	if snapshot.Status != StatusCompleted && snapshot.Status != StatusFailed && snapshot.Status != StatusCancelled {
		s.UpdateSharedDashboard(snapshot.ChatID, false)
		return
	}

	s.UpdateSharedDashboard(snapshot.ChatID, false)

	lang := s.GetUserLanguage(snapshot.UserID)
	text := buildTaskStatusText(lang, snapshot)

	if snapshot.Status == StatusCompleted && organizer.IsVideoFile(snapshot.FileName) && snapshot.LocalPath != "" {
		task.Mu.RLock()
		existingID := task.ResultMessageID
		task.Mu.RUnlock()

		if existingID == 0 {
			if s.sendVideoWithThumbnail(task, text) {
				return
			}
		}
	}

	slog.Debug("Sending final task message", "task_id", task.ID, "status", snapshot.Status)
	s.sendFinalMessage(task, text)
}

func (s *BotService) sendVideoWithThumbnail(task *Task, text string) bool {
	snapshot := task.GetSnapshot()
	if thumb, err := GenerateThumbnail(snapshot.LocalPath, s.TaskManager.DownloadDir); err == nil {
		photo := tgbotapi.NewPhoto(snapshot.ChatID, tgbotapi.FilePath(thumb))
		photo.Caption = text
		photo.ParseMode = MarkdownV2
		if snapshot.RemoteURL != "" {
			btnText := BtnTextCloudLink
			if s.Config.IndexURL != "" {
				btnText = BtnTextIndexURL
			}
			photo.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(btnText, snapshot.RemoteURL),
				),
			)
		}
		sentMsg, sendErr := s.Bot.Send(photo)
		if sendErr == nil {
			task.Mu.Lock()
			task.ResultMessageID = sentMsg.MessageID
			task.Mu.Unlock()
			slog.Info("Captured result video message ID", "message_id", sentMsg.MessageID, "task_id", task.ID)
			_ = os.Remove(thumb)
			return true
		}
		slog.Error("Failed to send video with thumbnail", "error", sendErr, "task_id", task.ID)
		_ = os.Remove(thumb)
	}
	return false
}

func (s *BotService) sendFinalMessage(task *Task, text string) {
	snapshot := task.GetSnapshot()

	task.Mu.RLock()
	msgID := task.ResultMessageID
	task.Mu.RUnlock()

	if msgID != 0 {
		editCaption := tgbotapi.NewEditMessageCaption(snapshot.ChatID, msgID, text)
		editCaption.ParseMode = MarkdownV2
		if snapshot.RemoteURL != "" {
			btnText := BtnTextCloudLink
			if s.Config.IndexURL != "" {
				btnText = BtnTextIndexURL
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(btnText, snapshot.RemoteURL),
				),
			)
			editCaption.ReplyMarkup = &keyboard
		}

		if _, err := s.Bot.Send(editCaption); err == nil {
			return
		}

		editText := tgbotapi.NewEditMessageText(snapshot.ChatID, msgID, text)
		editText.ParseMode = MarkdownV2
		if snapshot.RemoteURL != "" {
			btnText := BtnTextCloudLink
			if s.Config.IndexURL != "" {
				btnText = BtnTextIndexURL
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(btnText, snapshot.RemoteURL),
				),
			)
			editText.ReplyMarkup = &keyboard
		}

		if _, err := s.Bot.Send(editText); err == nil {
			return
		}

		slog.Warn("Failed to edit existing message, sending new one", "taskID", task.ID, "msgID", msgID)
	}

	msg := tgbotapi.NewMessage(snapshot.ChatID, text)
	msg.ParseMode = MarkdownV2

	if snapshot.Status == StatusCompleted && snapshot.RemoteURL != "" {
		btnText := BtnTextCloudLink
		if s.Config.IndexURL != "" {
			btnText = BtnTextIndexURL
		}
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(btnText, snapshot.RemoteURL),
			),
		)
		msg.ReplyMarkup = keyboard
	}

	if sentMsg, err := s.Bot.Send(msg); err == nil {
		task.Mu.Lock()
		task.ResultMessageID = sentMsg.MessageID
		task.Mu.Unlock()
		slog.Info("Captured result final message ID", "message_id", sentMsg.MessageID, "task_id", task.ID)
	} else {
		slog.Error("Failed to send final task message", "error", err, "task_id", task.ID, "chatID", snapshot.ChatID)
	}
}
