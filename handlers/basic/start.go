package basic

import (
	"log/slog"
	"time"
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	ActionBack = "back"
	CmdClose   = "close"
)

func StartHandler(s *service.BotService, message *tgbotapi.Message) {
	userName := message.From.FirstName
	if userName == "" {
		userName = "User"
	}

	lang := s.GetUserLanguage(message.From.ID)
	welcomeText := service.GetWelcomeMessage(lang, userName)
	keyboard := service.GetStartKeyboard(lang)

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	if len(keyboard.InlineKeyboard) > 0 {
		msg.ReplyMarkup = keyboard
	}

	if sentMsg, err := s.Bot.Send(msg); err != nil {
		slog.Error("Error sending welcome message", "error", err)
	} else {
		s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 60*time.Second)
	}
}

func HelpHandler(s *service.BotService, message *tgbotapi.Message) {
	lang := s.GetUserLanguage(message.From.ID)
	helpText := service.GetHelpMainText(lang)
	keyboard := service.GetHelpKeyboard(lang)

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	if len(keyboard.InlineKeyboard) > 0 {
		msg.ReplyMarkup = keyboard
	}

	if sentMsg, err := s.Bot.Send(msg); err != nil {
		slog.Error("Error sending help message", "error", err)
		msg.ParseMode = ""
		msg.Text = "📖 Panduan Bantuan\n\nSilakan pilih kategori di bawah:"
		_, _ = s.Bot.Send(msg)
	} else {
		s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 60*time.Second)
	}
}
