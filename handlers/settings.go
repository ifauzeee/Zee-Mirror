package handlers

import (
	"fmt"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Settings struct {
	AutoDeleteMessages bool
	DefaultMode        string
	mu                 sync.RWMutex
}

func InitSettings() {
	settings = &Settings{
		AutoDeleteMessages: false,
		DefaultMode:        "mirror",
	}
}

func HandleSettings(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	text := formatSettingsMessage()
	keyboard := getSettingsKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "MarkdownV2"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func HandleSettingsCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[1]
	settings.mu.Lock()

	switch action {
	case "auto_delete":
		settings.AutoDeleteMessages = !settings.AutoDeleteMessages
		if settings.AutoDeleteMessages {
			bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Auto Delete: ON"))
		} else {
			bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Auto Delete: OFF"))
		}

	case "default_mirror":
		settings.DefaultMode = "mirror"
		bot.Request(tgbotapi.NewCallback(callback.ID, "📥 Default: Mirror"))

	case "default_leech":
		settings.DefaultMode = "leech"
		bot.Request(tgbotapi.NewCallback(callback.ID, "🔗 Default: Leech"))

	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}

	settings.mu.Unlock()

	text := formatSettingsMessage()
	keyboard := getSettingsKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
}

func formatSettingsMessage() string {
	settings.mu.RLock()
	defer settings.mu.RUnlock()

	autoDeleteStatus := "❌ OFF"
	if settings.AutoDeleteMessages {
		autoDeleteStatus = "✅ ON"
	}

	defaultModeEmoji := "📥"
	if settings.DefaultMode == "leech" {
		defaultModeEmoji = "🔗"
	}

	return fmt.Sprintf(`⚙️ *Pengaturan Bot*
 
*Auto Delete Messages:* %s
_Hapus pesan status setelah task selesai \(60 detik\)_
 
*Default Mode:* %s %s
_Mode yang digunakan saat tidak ada flag_
 
Klik tombol di bawah untuk mengubah pengaturan\.`,
		autoDeleteStatus,
		defaultModeEmoji,
		EscapeMarkdownV2(settings.DefaultMode),
	)
}

func getSettingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	settings.mu.RLock()
	defer settings.mu.RUnlock()

	autoDeleteLabel := "🔕 Auto Delete: OFF"
	if settings.AutoDeleteMessages {
		autoDeleteLabel = "🔔 Auto Delete: ON"
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(autoDeleteLabel, "settings:auto_delete"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📥 Set Mirror", "settings:default_mirror"),
			tgbotapi.NewInlineKeyboardButtonData("🔗 Set Leech", "settings:default_leech"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", "dashboard:start"),
		),
	)
}

func GetSettings() *Settings {
	return settings
}

func IsAutoDeleteEnabled() bool {
	if settings == nil {
		return false
	}
	settings.mu.RLock()
	defer settings.mu.RUnlock()
	return settings.AutoDeleteMessages
}

func GetDefaultMode() string {
	if settings == nil {
		return "mirror"
	}
	settings.mu.RLock()
	defer settings.mu.RUnlock()
	return settings.DefaultMode
}
