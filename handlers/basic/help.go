package basic

import (
	"log/slog"
	"strings"
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleHelpCallback(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) {
	if action == "" {
		parts := strings.SplitN(callback.Data, ":", 2)
		if len(parts) == 2 {
			action = parts[1]
		} else if callback.Data == "help" {
			action = "main"
		}
	}
	action = normalizeHelpAction(action)

	if handleDownloadHelp(s, callback, action) {
		return
	}
	if handleMonitorHelp(s, callback, action) {
		return
	}
	if handleFilesHelp(s, callback, action) {
		return
	}
	if handleMediaHelp(s, callback, action) {
		return
	}
	if handleTaskHelp(s, callback, action) {
		return
	}
	if handleStorageHelp(s, callback, action) {
		return
	}
	if handleAdminHelp(s, callback, action) {
		return
	}
	if handleRecoveryHelp(s, callback, action) {
		return
	}

	var text string

	switch action {
	case "main":
		lang := s.GetUserLanguage(callback.From.ID)
		text = service.GetHelpMainText(lang)

	case "settings":
		text = GetHelpSettings()

	case "all":
		text = getHelpAllCommands()

	case ActionBack:
		lang := s.GetUserLanguage(callback.From.ID)
		content := service.GetWelcomeMessage(lang, callback.From.FirstName)
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, content)
		editMsg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(editMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return

	case CmdClose:
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, _ = s.Bot.Request(deleteMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Closed"))
		return

	default:
		return
	}

	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
}

func normalizeHelpAction(action string) string {
	switch action {
	case "cat_download", "sub_dl_general", "sub_dl_video", "sub_dl_torrent", "sub_dl_adv":
		return "download"
	case "cat_monitor":
		return "monitor"
	case "cat_files":
		return "files"
	case "cat_media":
		return "media"
	case "cat_task":
		return "task"
	case "cat_storage":
		return "storage"
	case "cat_admin":
		return "admin"
	case "cat_recovery":
		return "recovery"
	case "cat_settings", "cmd_lang":
		return "settings"
	default:
		return action
	}
}

func sendHelpMessage(s *service.BotService, callback *tgbotapi.CallbackQuery, text string, _ tgbotapi.InlineKeyboardMarkup) {
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err := s.Bot.Send(editMsg)
	if err != nil {
		slog.Error("Error editing help message", "error", err)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Error loading menu"))
	} else {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

func handleDownloadHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "download":
		text = service.ProfessionalMessage("📥 DOWNLOAD", "Pilih command untuk melihat detail lengkap:")
	case "cmd_mirror":
		text = GetHelpMirror()
	case "cmd_leech":
		text = GetHelpLeech()
	case "cmd_ytdlp":
		text = GetHelpYTDLP()
	case "cmd_ytdlpleech":
		text = GetHelpYTDLPLeech()
	case "cmd_viking":
		text = GetHelpViking()
	case "cmd_torrent":
		text = GetHelpTorrent()
	case "cmd_clone":
		text = GetHelpClone()
	case "cmd_batch":
		text = GetHelpBatch()
	case "cmd_search":
		text = GetHelpSearch()
	default:
		return false
	}
	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func handleMonitorHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "monitor":
		text = service.ProfessionalMessage("📊 MONITOR", "Pilih command untuk melihat detail lengkap:")
	case "cmd_status":
		text = GetHelpStatus()
	case "cmd_stats":
		text = GetHelpStats()
	case "cmd_system":
		text = GetHelpSystem()
	case "cmd_health":
		text = GetHelpHealth()
	case "cmd_logs":
		text = GetHelpLogs()
	case "cmd_ping":
		text = GetHelpPing()
	case "cmd_speed":
		text = GetHelpSpeed()
	default:
		return false
	}
	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func handleFilesHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "files":
		text = service.ProfessionalMessage("📁 FILES", "Pilih command untuk melihat detail lengkap:")
	case "cmd_ls":
		text = GetHelpLs()
	case "cmd_mkdir":
		text = GetHelpMkdir()
	case "cmd_rm":
		text = GetHelpRm()
	case "cmd_mv":
		text = GetHelpMv()
	case "cmd_share":
		text = GetHelpShare()
	case "cmd_find":
		text = GetHelpFind()
	default:
		return false
	}
	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func handleMediaHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "media":
		text = service.ProfessionalMessage("🎵 MEDIA", "Pilih command untuk melihat detail lengkap:")
	case "cmd_extractaudio":
		text = GetHelpExtractAudio()
	case "cmd_compress":
		text = GetHelpCompress()
	case "cmd_thumbnail":
		text = GetHelpThumbnail()
	case "cmd_screenshots":
		text = GetHelpScreenshots()
	case "cmd_subtitle":
		text = GetHelpSubtitle()
	case "cmd_hardsub":
		text = GetHelpHardsub()
	case "cmd_rescale":
		text = GetHelpRescale()
	case "cmd_convert":
		text = GetHelpConvert()
	case "cmd_mediainfo":
		text = GetHelpMediaInfo()
	default:
		return false
	}
	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func handleTaskHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "task":
		text = service.ProfessionalMessage("📋 TASK", "Pilih command untuk melihat detail lengkap:")
	case "cmd_cancel":
		text = GetHelpCancel()
	case "cmd_cancelall":
		text = GetHelpCancelAll()
	default:
		return false
	}
	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func handleStorageHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "storage":
		text = service.ProfessionalMessage("💾 STORAGE", "Pilih command untuk melihat detail lengkap:")
	case "cmd_storages":
		text = GetHelpStorages()
	case "cmd_setstorage":
		text = GetHelpSetStorage()
	default:
		return false
	}
	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func handleAdminHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "admin":
		text = service.ProfessionalMessage("👑 ADMIN", "Pilih command untuk melihat detail lengkap:")
	case "cmd_authorize":
		text = GetHelpAuthorize()
	case "cmd_unauthorize":
		text = GetHelpUnauthorize()
	case "cmd_users":
		text = GetHelpUsers()
	case "cmd_setalertchannel":
		text = GetHelpSetAlertChannel()
	case "cmd_setlogchannel":
		text = GetHelpSetLogChannel()
	default:
		return false
	}
	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func handleRecoveryHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "recovery":
		text = service.ProfessionalMessage("🔧 RECOVERY", "Pilih command untuk melihat detail lengkap:")
	case "cmd_recover":
		text = GetHelpRecover()
	case "cmd_recoverystatus":
		text = GetHelpRecoveryStatus()
	default:
		return false
	}
	sendHelpMessage(s, callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}
