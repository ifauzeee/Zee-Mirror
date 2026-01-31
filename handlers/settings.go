package handlers

import (
	"fmt"
	"sync"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Settings struct {
	AutoDeleteMessages bool
	DefaultMode        string
	mu                 sync.RWMutex
}

func NewSettings() *Settings {
	return &Settings{
		AutoDeleteMessages: false,
		DefaultMode:        string(TypeMirror),
	}
}

func (s *BotService) HandleSettings(message *tgbotapi.Message) {
	text := s.formatSettingsMessage()
	keyboard := s.getSettingsKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) HandleSettingsCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[1]
	s.Settings.mu.Lock()
	if val, err := s.DB.GetSetting("auto_delete_messages"); err == nil {
		s.Settings.AutoDeleteMessages = (val == "true")
	}
	if val, err := s.DB.GetSetting("default_mode"); err == nil {
		s.Settings.DefaultMode = val
	}
	s.Settings.mu.Unlock()

	switch action {
	case "auto_delete":
		s.Settings.mu.Lock()
		s.Settings.AutoDeleteMessages = !s.Settings.AutoDeleteMessages
		status := s.Settings.AutoDeleteMessages
		s.Settings.mu.Unlock()

		_ = s.DB.SetSetting("auto_delete_messages", fmt.Sprintf("%v", status))

		if status {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Auto Delete: ON"))
		} else {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Auto Delete: OFF"))
		}

	case "default_mirror":
		s.Settings.mu.Lock()
		s.Settings.DefaultMode = string(TypeMirror)
		s.Settings.mu.Unlock()
		_ = s.DB.SetSetting("default_mode", string(TypeMirror))
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📥 Default: Mirror"))

	case "default_leech":
		s.Settings.mu.Lock()
		s.Settings.DefaultMode = string(TypeLeech)
		s.Settings.mu.Unlock()
		_ = s.DB.SetSetting("default_mode", string(TypeLeech))
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔗 Default: Leech"))

	default:
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}

	text := s.formatSettingsMessage()
	keyboard := s.getSettingsKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) formatSettingsMessage() string {
	s.Settings.mu.RLock()
	defer s.Settings.mu.RUnlock()

	autoDeleteStatus := "❌ OFF"
	if s.Settings.AutoDeleteMessages {
		autoDeleteStatus = "✅ ON"
	}

	defaultModeEmoji := "📥"
	if s.Settings.DefaultMode == ModeLeech {
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
		utils.EscapeMarkdownV2(s.Settings.DefaultMode),
	)
}

func (s *BotService) getSettingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	s.Settings.mu.RLock()
	defer s.Settings.mu.RUnlock()

	autoDeleteLabel := "🔕 Auto Delete: OFF"
	if s.Settings.AutoDeleteMessages {
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
			tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", "help:back"),
		),
	)
}
