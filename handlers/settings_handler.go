package handlers

import (
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) formatSettingsMessage(_ int64) string {
	return service.ProfessionalMessage("SETTINGS", "Pengaturan bot belum diimplementasikan sepenuhnya.")
}

func (s *BotService) getSettingsKeyboard(_ int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:main"),
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "dashboard:close"),
		),
	)
}
