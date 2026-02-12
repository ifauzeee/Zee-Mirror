package handlers

import (
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		text = GetHelpMainText(lang)
		keyboard = GetHelpKeyboard(lang)

	case "settings":
		text = getHelpSettings()
		backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚙️ Open Settings", "dashboard:settings"),
			),
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)

	case "all":
		text = getHelpAllCommands()
		backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
			),
		)

	case ActionBack:
		lang := s.GetUserLanguage(callback.From.ID)
		content := GetWelcomeMessage(lang, callback.From.FirstName)
		kb := GetStartKeyboard(lang)
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
		text = ProfessionalMessage("📥 DOWNLOAD", "Pilih command untuk melihat detail lengkap:")
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
		text = getHelpMirror()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_leech":
		text = getHelpLeech()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ytdlp":
		text = getHelpYTDLP()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ytdlpleech":
		text = getHelpYTDLPLeech()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_viking":
		text = getHelpViking()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_torrent":
		text = getHelpTorrent()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_clone":
		text = getHelpClone()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_batch":
		text = getHelpBatch()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_search":
		text = getHelpSearch()
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
		text = ProfessionalMessage("📊 MONITOR", "Pilih command untuk melihat detail lengkap:")
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
		text = getHelpStatus()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_stats":
		text = getHelpStats()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_system":
		text = getHelpSystem()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_health":
		text = getHelpHealth()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_logs":
		text = getHelpLogs()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ping":
		text = getHelpPing()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_speed":
		text = getHelpSpeed()
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
		text = ProfessionalMessage("📁 FILES", "Pilih command untuk melihat detail lengkap:")
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
		text = getHelpLs()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mkdir":
		text = getHelpMkdir()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_rm":
		text = getHelpRm()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mv":
		text = getHelpMv()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_share":
		text = getHelpShare()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_find":
		text = getHelpFind()
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
		text = ProfessionalMessage("🎵 MEDIA", "Pilih command untuk melihat detail lengkap:")
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
		text = getHelpExtractAudio()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_compress":
		text = getHelpCompress()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_thumbnail":
		text = getHelpThumbnail()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_screenshots":
		text = getHelpScreenshots()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_subtitle":
		text = getHelpSubtitle()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_hardsub":
		text = getHelpHardsub()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_rescale":
		text = getHelpRescale()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_convert":
		text = getHelpConvert()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mediainfo":
		text = getHelpMediaInfo()
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
		text = ProfessionalMessage("📋 TASK", "Pilih command untuk melihat detail lengkap:")
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
		text = getHelpCancel()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_cancelall":
		text = getHelpCancelAll()
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
		text = ProfessionalMessage("💾 STORAGE", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Storages", "help:cmd_storages"),
				tgbotapi.NewInlineKeyboardButtonData("⚙️ Set Storage", "help:cmd_setstorage"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_storages":
		text = getHelpStorages()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setstorage":
		text = getHelpSetStorage()
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
		text = ProfessionalMessage("👑 ADMIN", "Pilih command untuk melihat detail lengkap:")
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
		text = getHelpAuthorize()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_unauthorize":
		text = getHelpUnauthorize()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_users":
		text = getHelpUsers()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setalertchannel":
		text = getHelpSetAlertChannel()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setlogchannel":
		text = getHelpSetLogChannel()
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
		text = ProfessionalMessage("🔧 RECOVERY", "Pilih command untuk melihat detail lengkap:")
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Recover", "help:cmd_recover"),
				tgbotapi.NewInlineKeyboardButtonData("📊 Status", "help:cmd_recoverystatus"),
			),
			tgbotapi.NewInlineKeyboardRow(backBtn),
		)
	case "cmd_recover":
		text = getHelpRecover()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_recoverystatus":
		text = getHelpRecoveryStatus()
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
