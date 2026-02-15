package handlers

import (
	"fmt"
	"log/slog"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) reply(message *tgbotapi.Message, text string) {
	s.Reply(message, text)
}

func (s *BotService) editStatusMessage(chatID int64, messageID int, text string) {
	s.SendOrEditMessage(chatID, messageID, text, nil)
}

func (s *BotService) handleCreateTaskError(chatID int64, messageID int, err error) {
	slog.Error("Failed to create task", "error", err)
	s.editStatusMessage(chatID, messageID, fmt.Sprintf("❌ *Gagal membuat task*\n\nError: `%s`", utils.EscapeMarkdownV2(err.Error())))
}
