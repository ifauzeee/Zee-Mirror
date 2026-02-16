package handlers

import (
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) handleSettingsCallback(callback *tgbotapi.CallbackQuery, action string) {
	lang := s.GetUserLanguage(callback.From.ID)
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup

	switch action {
	case "main":
		text = s.formatSettingsMessage(callback.From.ID)
		keyboard = service.GetSettingsKeyboard(lang)

	case "cat_storage":
		text = service.ProfessionalMessage("💾 STORAGE SETTINGS", "Konfigurasi penyimpanan cloud:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 List Remotes", "dashboard:storage"),
				tgbotapi.NewInlineKeyboardButtonData("⚙️ Set Default", "help:cmd_setstorage"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "settings:main"),
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)

	case "cat_user":
		text = service.ProfessionalMessage("👤 USER SETTINGS", "Pengaturan profil dan limitasi Anda:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊 My Stats", "dashboard:stats"),
				tgbotapi.NewInlineKeyboardButtonData("🌐 Language", "help:cmd_lang"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "settings:main"),
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)

	case "cat_global":
		text = service.ProfessionalMessage("⚙️ GLOBAL SETTINGS", "Pengaturan umum bot:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗑️ Auto Delete", "settings:toggle_autodel"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "settings:main"),
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)

	case "cat_admin":
		if callback.From.ID != s.Config.OwnerID {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🚫 Owner only!"))
			return
		}
		text = service.ProfessionalMessage("🛡️ ADMIN PANEL", "Kontrol administratif sistem:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👥 Manage Users", "dashboard:users"),
				tgbotapi.NewInlineKeyboardButtonData("📜 Logs", "dashboard:system"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "settings:main"),
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)

	default:
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "⚙️ Coming Soon"))
		return
	}

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

func (s *BotService) HandleSettingsFromCallback(callback *tgbotapi.CallbackQuery) {
	s.handleSettingsCallback(callback, "main")
}

func (s *BotService) formatSettingsMessage(_ int64) string {
	return service.ProfessionalMessage("SETTINGS", "Pilih kategori pengaturan di bawah untuk melanjutkan:")
}

func (s *BotService) getSettingsKeyboard(id int64) tgbotapi.InlineKeyboardMarkup {
	lang := s.GetUserLanguage(id)
	return service.GetSettingsKeyboard(lang)
}
