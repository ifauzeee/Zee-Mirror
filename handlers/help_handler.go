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
	var keyboard tgbotapi.InlineKeyboardMarkup

	switch action {
	case "main":
		lang := s.GetUserLanguage(callback.From.ID)
		text = service.GetHelpMainText(lang)
		keyboard = service.GetHelpKeyboard(lang)

	case "cat_download":
		lang := s.GetUserLanguage(callback.From.ID)
		text = service.ProfessionalMessage("📥 DOWNLOAD", "Pilih sub-kategori bantuan download:")
		keyboard = service.GetDownloadHelpKeyboard(lang)

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
		backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:cat_download")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📥 Mirror", "help:cmd_mirror"),
				tgbotapi.NewInlineKeyboardButtonData("📤 Leech", "help:cmd_leech"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚔️ Viking", "help:cmd_viking"),
				tgbotapi.NewInlineKeyboardButtonData("📋 Clone", "help:cmd_clone"),
			),
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)
	case "sub_dl_video":
		text = service.ProfessionalMessage("🎬 VIDEO DOWNLOAD", "Pilih perintah download video:")
		backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:cat_download")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎬 YT Mirror", "help:cmd_ytdlp"),
				tgbotapi.NewInlineKeyboardButtonData("🎬 YT Leech", "help:cmd_ytdlpleech"),
			),
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)
	case "sub_dl_torrent":
		text = service.ProfessionalMessage("🧲 TORRENT DOWNLOAD", "Pilih perintah torrent:")
		backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:cat_download")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🧲 Torrent", "help:cmd_torrent"),
				tgbotapi.NewInlineKeyboardButtonData("🔍 Search", "help:cmd_search"),
			),
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)
	case "sub_dl_adv":
		text = service.ProfessionalMessage("📋 ADVANCED DOWNLOAD", "Pilih perintah lanjutan:")
		backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:cat_download")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📦 Batch", "help:cmd_batch"),
			),
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)

	case "all":
		text = basic.GenerateAllCommandsHelp(basic.GetRegisteredCommands())
		backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)

	case ActionBack:
		lang := s.GetUserLanguage(callback.From.ID)
		content := service.GetWelcomeMessage(lang, callback.From.FirstName)
		kb := service.GetStartKeyboard(lang)
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, content)
		editMsg.ParseMode = tgbotapi.ModeMarkdownV2
		editMsg.ReplyMarkup = &kb
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

	s.sendHelpMessage(callback, text, keyboard)
}

func (s *BotService) sendHelpMessage(callback *tgbotapi.CallbackQuery, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	editMsg.ReplyMarkup = &keyboard
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
	var keyboard tgbotapi.InlineKeyboardMarkup
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
	backToCat := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:download")

	switch action {
	case "download":
		text = service.ProfessionalMessage("📥 DOWNLOAD", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📥 Mirror", "help:cmd_mirror"),
				tgbotapi.NewInlineKeyboardButtonData("📤 Leech", "help:cmd_leech"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎬 YT Mirror", "help:cmd_ytdlp"),
				tgbotapi.NewInlineKeyboardButtonData("🎬 YT Leech", "help:cmd_ytdlpleech"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚔️ Viking", "help:cmd_viking"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🧲 Torrent", "help:cmd_torrent"),
				tgbotapi.NewInlineKeyboardButtonData("📋 Clone", "help:cmd_clone"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📦 Batch", "help:cmd_batch"),
				tgbotapi.NewInlineKeyboardButtonData("🔍 Search", "help:cmd_search"),
			),
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)
	case "cmd_mirror":
		text = basic.GetHelpMirror()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_leech":
		text = basic.GetHelpLeech()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ytdlp":
		text = basic.GetHelpYTDLP()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ytdlpleech":
		text = basic.GetHelpYTDLPLeech()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_viking":
		text = basic.GetHelpViking()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_torrent":
		text = basic.GetHelpTorrent()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_clone":
		text = basic.GetHelpClone()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_batch":
		text = basic.GetHelpBatch()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_search":
		text = basic.GetHelpSearch()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	s.sendHelpMessage(callback, text, keyboard)
	return true
}

func (s *BotService) handleMonitorHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
	backToCat := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:monitor")

	switch action {
	case "monitor":
		text = service.ProfessionalMessage("📊 MONITOR", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊 Status", "help:cmd_status"),
				tgbotapi.NewInlineKeyboardButtonData("📈 Stats", "help:cmd_stats"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🖥️ System", "help:cmd_system"),
				tgbotapi.NewInlineKeyboardButtonData("🏥 Health", "help:cmd_health"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📜 Logs", "help:cmd_logs"),
				tgbotapi.NewInlineKeyboardButtonData("🏓 Ping", "help:cmd_ping"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚀 Speed", "help:cmd_speed"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_status":
		text = basic.GetHelpStatus()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_stats":
		text = basic.GetHelpStats()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_system":
		text = basic.GetHelpSystem()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_health":
		text = basic.GetHelpHealth()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_logs":
		text = basic.GetHelpLogs()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ping":
		text = basic.GetHelpPing()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_speed":
		text = basic.GetHelpSpeed()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	s.sendHelpMessage(callback, text, keyboard)
	return true
}

func (s *BotService) handleFilesHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
	backToCat := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:files")

	switch action {
	case "files":
		text = service.ProfessionalMessage("📁 FILES", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📂 List (ls)", "help:cmd_ls"),
				tgbotapi.NewInlineKeyboardButtonData("📁 Mkdir", "help:cmd_mkdir"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗑️ Remove", "help:cmd_rm"),
				tgbotapi.NewInlineKeyboardButtonData("📦 Move", "help:cmd_mv"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔗 Share", "help:cmd_share"),
				tgbotapi.NewInlineKeyboardButtonData("🔍 Find", "help:cmd_find"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_ls":
		text = basic.GetHelpLs()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mkdir":
		text = basic.GetHelpMkdir()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_rm":
		text = basic.GetHelpRm()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mv":
		text = basic.GetHelpMv()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_share":
		text = basic.GetHelpShare()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_find":
		text = basic.GetHelpFind()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	s.sendHelpMessage(callback, text, keyboard)
	return true
}

func (s *BotService) handleMediaHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
	backToCat := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:media")

	switch action {
	case "media":
		text = service.ProfessionalMessage("🎵 MEDIA", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎵 Extract Audio", "help:cmd_extractaudio"),
				tgbotapi.NewInlineKeyboardButtonData("🗜️ Compress", "help:cmd_compress"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🖼️ Thumbnail", "help:cmd_thumbnail"),
				tgbotapi.NewInlineKeyboardButtonData("📸 Screenshots", "help:cmd_screenshots"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Soft-sub", "help:cmd_subtitle"),
				tgbotapi.NewInlineKeyboardButtonData("🔥 Hard-sub", "help:cmd_hardsub"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📐 Rescale", "help:cmd_rescale"),
				tgbotapi.NewInlineKeyboardButtonData("🔄 Convert", "help:cmd_convert"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("ℹ️ Media Info", "help:cmd_mediainfo"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_extractaudio":
		text = basic.GetHelpExtractAudio()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_compress":
		text = basic.GetHelpCompress()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_thumbnail":
		text = basic.GetHelpThumbnail()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_screenshots":
		text = basic.GetHelpScreenshots()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_subtitle":
		text = basic.GetHelpSubtitle()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_hardsub":
		text = basic.GetHelpHardsub()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_rescale":
		text = basic.GetHelpRescale()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_convert":
		text = basic.GetHelpConvert()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mediainfo":
		text = basic.GetHelpMediaInfo()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	s.sendHelpMessage(callback, text, keyboard)
	return true
}

func (s *BotService) handleTaskHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
	backToCat := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:task")

	switch action {
	case "task":
		text = service.ProfessionalMessage("📋 TASK", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊 Status", "help:cmd_status"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "help:cmd_cancel"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚫 Cancel All", "help:cmd_cancelall"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_cancel":
		text = basic.GetHelpCancel()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_cancelall":
		text = basic.GetHelpCancelAll()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	s.sendHelpMessage(callback, text, keyboard)
	return true
}

func (s *BotService) handleStorageHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
	backToCat := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:storage")

	switch action {
	case "storage":
		text = service.ProfessionalMessage("💾 STORAGE", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Storages", "help:cmd_storages"),
				tgbotapi.NewInlineKeyboardButtonData("⚙️ Set Storage", "help:cmd_setstorage"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_storages":
		text = basic.GetHelpStorages()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setstorage":
		text = basic.GetHelpSetStorage()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	s.sendHelpMessage(callback, text, keyboard)
	return true
}

func (s *BotService) handleAdminHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
	backToCat := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:admin")

	switch action {
	case "admin":
		text = service.ProfessionalMessage("👑 ADMIN", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Authorize", "help:cmd_authorize"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Unauthorize", "help:cmd_unauthorize"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👥 Users", "help:cmd_users"),
				tgbotapi.NewInlineKeyboardButtonData("📜 Log Channel", "help:cmd_setlogchannel"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚨 Alert Channel", "help:cmd_setalertchannel"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_authorize":
		text = basic.GetHelpAuthorize()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_unauthorize":
		text = basic.GetHelpUnauthorize()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_users":
		text = basic.GetHelpUsers()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setalertchannel":
		text = basic.GetHelpSetAlertChannel()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setlogchannel":
		text = basic.GetHelpSetLogChannel()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	s.sendHelpMessage(callback, text, keyboard)
	return true
}

func (s *BotService) handleRecoveryHelp(callback *tgbotapi.CallbackQuery, action string) bool {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
	backToCat := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:recovery")

	switch action {
	case "recovery":
		text = service.ProfessionalMessage("🔧 RECOVERY", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Recover", "help:cmd_recover"),
				tgbotapi.NewInlineKeyboardButtonData("📊 Status", "help:cmd_recoverystatus"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_recover":
		text = basic.GetHelpRecover()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_recoverystatus":
		text = basic.GetHelpRecoveryStatus()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	s.sendHelpMessage(callback, text, keyboard)
	return true
}
