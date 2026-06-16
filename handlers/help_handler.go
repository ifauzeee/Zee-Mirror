package handlers

import (
	"log/slog"
	"zee-mirror/handlers/basic"
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	ActionBack = "back"
	CmdClose   = "close"
)

func (s *BotService) handleHelpCallback(callback *tgbotapi.CallbackQuery, action string) {
	if s.handleDownloadHelp(callback, action) {
		return
	}
	if s.handleMonitorHelp(callback, action) {
		return
	}
	if s.handleFilesHelp(callback, action) {
		return
	}
	if s.handleMediaHelp(callback, action) {
		return
	}
	if s.handleTaskHelp(callback, action) {
		return
	}
	if s.handleStorageHelp(callback, action) {
		return
	}
	if s.handleAdminHelp(callback, action) {
		return
	}
	if s.handleRecoveryHelp(callback, action) {
		return
	}

	var text string

	switch action {
	case "main":
		lang := s.GetUserLanguage(callback.From.ID)
		text = service.GetHelpMainText(lang)

	case "cat_download":
		lang := s.GetUserLanguage(callback.From.ID)
		text = service.ProfessionalMessage("📥 DOWNLOAD", "Pilih sub-kategori bantuan download:")
		_ = lang

	case "cat_monitor":
		s.handleMonitorHelp(callback, "monitor")
		return
	case "cat_files":
		s.handleFilesHelp(callback, "files")
		return
	case "cat_media":
		s.handleMediaHelp(callback, "media")
		return
	case "cat_task":
		s.handleTaskHelp(callback, "task")
		return
	case "cat_storage":
		s.handleStorageHelp(callback, "storage")
		return
	case "cat_admin":
		s.handleAdminHelp(callback, "admin")
		return
	case "cat_recovery":
		s.handleRecoveryHelp(callback, "recovery")
		return
	case "cat_settings":
		s.HandleSettingsFromCallback(callback)
		return

	case "sub_dl_general":
		text = service.ProfessionalMessage("📥 GENERAL DOWNLOAD", "Pilih perintah download umum:")
	case "sub_dl_video":
		text = service.ProfessionalMessage("🎬 VIDEO DOWNLOAD", "Pilih perintah download video:")
	case "sub_dl_torrent":
		text = service.ProfessionalMessage("🧲 TORRENT DOWNLOAD", "Pilih perintah torrent:")
	case "sub_dl_adv":
		text = service.ProfessionalMessage("📋 ADVANCED DOWNLOAD", "Pilih perintah lanjutan:")

	case "all":
		text = basic.GenerateAllCommandsHelp(basic.GetRegisteredCommands())

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

	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
}

func (s *BotService) sendHelpMessage(callback *tgbotapi.CallbackQuery, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	if len(keyboard.InlineKeyboard) > 0 {
		editMsg.ReplyMarkup = &keyboard
	}
	_, err := s.Bot.Send(editMsg)
	if err != nil {
		slog.Error("Error editing help message", "error", err)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Error loading menu"))
	} else {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

func (s *BotService) handleDownloadHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "download":
		text = service.ProfessionalMessage("📥 DOWNLOAD", "Pilih command untuk melihat detail lengkap:")
	case "cmd_mirror":
		text = basic.GetHelpMirror()
	case "cmd_leech":
		text = basic.GetHelpLeech()
	case "cmd_ytdlp":
		text = basic.GetHelpYTDLP()
	case "cmd_ytdlpleech":
		text = basic.GetHelpYTDLPLeech()
	case "cmd_viking":
		text = basic.GetHelpViking()
	case "cmd_torrent":
		text = basic.GetHelpTorrent()
	case "cmd_clone":
		text = basic.GetHelpClone()
	case "cmd_batch":
		text = basic.GetHelpBatch()
	case "cmd_search":
		text = basic.GetHelpSearch()
	default:
		return false
	}
	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func (s *BotService) handleMonitorHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "monitor":
		text = service.ProfessionalMessage("📊 MONITOR", "Pilih command untuk melihat detail lengkap:")
	case "cmd_status":
		text = basic.GetHelpStatus()
	case "cmd_stats":
		text = basic.GetHelpStats()
	case "cmd_system":
		text = basic.GetHelpSystem()
	case "cmd_health":
		text = basic.GetHelpHealth()
	case "cmd_logs":
		text = basic.GetHelpLogs()
	case "cmd_ping":
		text = basic.GetHelpPing()
	case "cmd_speed":
		text = basic.GetHelpSpeed()
	default:
		return false
	}
	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func (s *BotService) handleFilesHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "files":
		text = service.ProfessionalMessage("📁 FILES", "Pilih command untuk melihat detail lengkap:")
	case "cmd_ls":
		text = basic.GetHelpLs()
	case "cmd_mkdir":
		text = basic.GetHelpMkdir()
	case "cmd_rm":
		text = basic.GetHelpRm()
	case "cmd_mv":
		text = basic.GetHelpMv()
	case "cmd_share":
		text = basic.GetHelpShare()
	case "cmd_find":
		text = basic.GetHelpFind()
	default:
		return false
	}
	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func (s *BotService) handleMediaHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "media":
		text = service.ProfessionalMessage("🎵 MEDIA", "Pilih command untuk melihat detail lengkap:")
	case "cmd_extractaudio":
		text = basic.GetHelpExtractAudio()
	case "cmd_compress":
		text = basic.GetHelpCompress()
	case "cmd_thumbnail":
		text = basic.GetHelpThumbnail()
	case "cmd_screenshots":
		text = basic.GetHelpScreenshots()
	case "cmd_subtitle":
		text = basic.GetHelpSubtitle()
	case "cmd_hardsub":
		text = basic.GetHelpHardsub()
	case "cmd_rescale":
		text = basic.GetHelpRescale()
	case "cmd_convert":
		text = basic.GetHelpConvert()
	case "cmd_mediainfo":
		text = basic.GetHelpMediaInfo()
	default:
		return false
	}
	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func (s *BotService) handleTaskHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "task":
		text = service.ProfessionalMessage("📋 TASK", "Pilih command untuk melihat detail lengkap:")
	case "cmd_cancel":
		text = basic.GetHelpCancel()
	case "cmd_cancelall":
		text = basic.GetHelpCancelAll()
	default:
		return false
	}
	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func (s *BotService) handleStorageHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "storage":
		text = service.ProfessionalMessage("💾 STORAGE", "Pilih command untuk melihat detail lengkap:")
	case "cmd_storages":
		text = basic.GetHelpStorages()
	case "cmd_setstorage":
		text = basic.GetHelpSetStorage()
	default:
		return false
	}
	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func (s *BotService) handleAdminHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "admin":
		text = service.ProfessionalMessage("👑 ADMIN", "Pilih command untuk melihat detail lengkap:")
	case "cmd_authorize":
		text = basic.GetHelpAuthorize()
	case "cmd_unauthorize":
		text = basic.GetHelpUnauthorize()
	case "cmd_users":
		text = basic.GetHelpUsers()
	case "cmd_setalertchannel":
		text = basic.GetHelpSetAlertChannel()
	case "cmd_setlogchannel":
		text = basic.GetHelpSetLogChannel()
	default:
		return false
	}
	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}

func (s *BotService) handleRecoveryHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string

	switch action {
	case "recovery":
		text = service.ProfessionalMessage("🔧 RECOVERY", "Pilih command untuk melihat detail lengkap:")
	case "cmd_recover":
		text = basic.GetHelpRecover()
	case "cmd_recoverystatus":
		text = basic.GetHelpRecoveryStatus()
	default:
		return false
	}
	s.sendHelpMessage(callback, text, tgbotapi.InlineKeyboardMarkup{})
	return true
}
