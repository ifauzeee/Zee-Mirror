package handlers

import (
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) handleSettingsCallback(callback *tgbotapi.CallbackQuery, action string) {
	var text string

	switch action {
	case "main":
		text = s.formatSettingsMessage(callback.From.ID)

	case "cat_storage":
		text = service.ProfessionalMessage("💾 STORAGE SETTINGS", "Konfigurasi penyimpanan cloud:")

	case "cat_user":
		text = service.ProfessionalMessage("👤 USER SETTINGS", "Pengaturan profil dan limitasi Anda:")

	case "cat_global":
		text = service.ProfessionalMessage("⚙️ GLOBAL SETTINGS", "Pengaturan umum bot:")

	case "cat_admin":
		if callback.From.ID != s.Config.OwnerID {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🚫 Owner only!"))
			return
		}
		text = service.ProfessionalMessage("🛡️ ADMIN PANEL", "Kontrol administratif sistem:")

	default:
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "⚙️ Coming Soon"))
		return
	}

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

func (s *BotService) HandleSettingsFromCallback(callback *tgbotapi.CallbackQuery) {
	s.handleSettingsCallback(callback, "main")
}

func (s *BotService) formatSettingsMessage(_ int64) string {
	return service.ProfessionalMessage("SETTINGS", "Pilih kategori pengaturan di bawah untuk melanjutkan:")
}
