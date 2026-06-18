package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Settings struct {
	DefaultMode        string
	YTDLPQuality       string
	Mu                 sync.RWMutex
	AutoDeleteMessages bool
}

func NewSettings() *Settings {
	return &Settings{
		AutoDeleteMessages: true,
		DefaultMode:        string(TypeMirror),
		YTDLPQuality:       "1080p",
	}
}

func (s *BotService) HandleSettings(message *tgbotapi.Message) {
	text := s.formatSettingsMessage(message.From.ID)
	keyboard := s.getSettingsKeyboard(message.From.ID)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	sentMsg, err := s.Bot.Send(msg)
	if err == nil {
		s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 60*time.Second)
	}
}

func (s *BotService) HandleLanguage(message *tgbotapi.Message) {
	lang := s.GetUserLanguage(message.From.ID)
	text := i18n.T(lang, "settings_select_language")
	keyboard := s.getSettingsKeyboard(message.From.ID)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	msg.ReplyToMessageID = message.MessageID
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) HandleSettingsCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	ctx := context.Background()
	action := parts[1]
	s.Settings.Mu.Lock()
	if val, err := s.DB.Get(ctx, "auto_delete_messages"); err == nil {
		s.Settings.AutoDeleteMessages = (val == "true")
	}
	if val, err := s.DB.Get(ctx, "default_mode"); err == nil {
		s.Settings.DefaultMode = val
	}
	s.Settings.Mu.Unlock()

	switch action {
	case "auto_delete":
		s.Settings.Mu.Lock()
		s.Settings.AutoDeleteMessages = !s.Settings.AutoDeleteMessages
		status := s.Settings.AutoDeleteMessages
		s.Settings.Mu.Unlock()

		_ = s.DB.Set(ctx, "auto_delete_messages", fmt.Sprintf("%v", status))

		if status {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Auto Delete: ON"))
		} else {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Auto Delete: OFF"))
		}

	case "default_mirror":
		s.Settings.Mu.Lock()
		s.Settings.DefaultMode = string(TypeMirror)
		s.Settings.Mu.Unlock()
		_ = s.DB.Set(ctx, "default_mode", string(TypeMirror))
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📥 Default: Mirror"))

	case "default_leech":
		s.Settings.Mu.Lock()
		s.Settings.DefaultMode = string(TypeLeech)
		s.Settings.Mu.Unlock()
		_ = s.DB.Set(ctx, "default_mode", string(TypeLeech))
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔗 Default: Leech"))

	case "quality_720":
		s.Settings.Mu.Lock()
		s.Settings.YTDLPQuality = "720p"
		s.Settings.Mu.Unlock()
		_ = s.DB.Set(ctx, "ytdlp_quality", "720p")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📺 Quality: 720p"))

	case "quality_1080":
		s.Settings.Mu.Lock()
		s.Settings.YTDLPQuality = "1080p"
		s.Settings.Mu.Unlock()
		_ = s.DB.Set(ctx, "ytdlp_quality", "1080p")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📺 Quality: 1080p"))

	case "quality_2160":
		s.Settings.Mu.Lock()
		s.Settings.YTDLPQuality = "2160p"
		s.Settings.Mu.Unlock()
		_ = s.DB.Set(ctx, "ytdlp_quality", "2160p")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📺 Quality: 4K (2160p)"))

	case "lang_id":
		_ = s.DB.SetLanguage(ctx, callback.From.ID, "id")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🇮🇩 Bahasa: Indonesia"))

	case "lang_en":
		_ = s.DB.SetLanguage(ctx, callback.From.ID, "en")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🇺🇸 Language: English"))

	case "lang_ja":
		_ = s.DB.SetLanguage(ctx, callback.From.ID, "ja")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🇯🇵 言語: 日本語"))

	case "lang_zh":
		_ = s.DB.SetLanguage(ctx, callback.From.ID, "zh")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🇨🇳 语言: 简体中文"))

	default:
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}

	text := s.formatSettingsMessage(callback.From.ID)
	keyboard := s.getSettingsKeyboard(callback.From.ID)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) formatSettingsMessage(userID int64) string {
	s.Settings.Mu.RLock()
	defer s.Settings.Mu.RUnlock()

	lang := s.GetUserLanguage(userID)

	autoDeleteStatus := "❌ OFF"
	if s.Settings.AutoDeleteMessages {
		autoDeleteStatus = "✅ ON"
	}

	defaultModeEmoji := "📥"
	if s.Settings.DefaultMode == ModeLeech {
		defaultModeEmoji = "🔗"
	}

	langFlag := "🇮🇩"
	langName := "Indonesia"
	switch lang {
	case "en":
		langFlag = "🇺🇸"
		langName = "English"
	case "ja":
		langFlag = "🇯🇵"
		langName = "日本語"
	case "zh":
		langFlag = "🇨🇳"
		langName = "简体中文"
	}

	cookiesStatus := i18n.T(lang, "settings_cookies_missing")
	if _, err := os.Stat(filepath.Join(s.Config.ConfigDir, "cookies.txt")); err == nil {
		cookiesStatus = i18n.T(lang, "settings_cookies_installed")
	}

	return fmt.Sprintf("⚙️ *%s*\n\n"+
		"*%s:* %s\n"+
		"_%s_\n\n"+
		"*%s:* %s %s\n"+
		"_%s_\n\n"+
		"*%s:* 📺 %s\n"+
		"_%s_\n\n"+
		"*%s:* %s\n"+
		"_%s_\n\n"+
		"*%s:* %s %s\n\n"+
		"%s",
		i18n.T(lang, "settings_title"),
		i18n.T(lang, "settings_auto_delete"), autoDeleteStatus,
		i18n.T(lang, "settings_auto_delete_desc"),
		i18n.T(lang, "settings_default_mode"), defaultModeEmoji, utils.EscapeMarkdownV2(s.Settings.DefaultMode),
		i18n.T(lang, "settings_default_mode_desc"),
		i18n.T(lang, "settings_ytdlp_quality"), utils.EscapeMarkdownV2(s.Settings.YTDLPQuality),
		i18n.T(lang, "settings_ytdlp_quality_desc"),
		i18n.T(lang, "settings_cookies"), cookiesStatus,
		i18n.T(lang, "settings_cookies_desc"),
		i18n.T(lang, "settings_language"), langFlag, langName,
		i18n.T(lang, "settings_footer"),
	)
}

func (s *BotService) getSettingsKeyboard(_ int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.InlineKeyboardMarkup{}
}
