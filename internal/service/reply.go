package service

import (
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) Reply(message *tgbotapi.Message, text string) {
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.ReplyToMessageID = message.MessageID

	if _, err := s.Bot.Send(msg); err != nil {
		slog.Error("Failed to send reply", "error", err)
	}
}

func (s *BotService) SendOrEditMessage(chatID int64, messageID int, text string, keyboard interface{}) {
	if messageID != 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdownV2

		if kb, ok := keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
			editMsg.ReplyMarkup = &kb
		} else if kb, ok := keyboard.(*tgbotapi.InlineKeyboardMarkup); ok {
			editMsg.ReplyMarkup = kb
		}

		if _, err := s.Bot.Send(editMsg); err == nil {
			return
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2

	if kb, ok := keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
		msg.ReplyMarkup = kb
	} else if kb, ok := keyboard.(*tgbotapi.InlineKeyboardMarkup); ok {
		msg.ReplyMarkup = kb
	}

	if _, err := s.Bot.Send(msg); err != nil {
		slog.Error("Failed to send message", "error", err)
	}
}
