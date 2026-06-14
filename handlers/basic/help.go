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
	var keyboard tgbotapi.InlineKeyboardMarkup

	switch action {
	case "main":
		lang := s.GetUserLanguage(callback.From.ID)
		text = service.GetHelpMainText(lang)
		keyboard = service.GetHelpKeyboard(lang)

	case "settings":
		text = GetHelpSettings()
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

	sendHelpMessage(s, callback, text, keyboard)
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

func sendHelpMessage(s *service.BotService, callback *tgbotapi.CallbackQuery, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
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

func handleDownloadHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
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
		text = GetHelpMirror()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_leech":
		text = GetHelpLeech()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ytdlp":
		text = GetHelpYTDLP()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ytdlpleech":
		text = GetHelpYTDLPLeech()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_viking":
		text = GetHelpViking()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_torrent":
		text = GetHelpTorrent()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_clone":
		text = GetHelpClone()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_batch":
		text = GetHelpBatch()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_search":
		text = GetHelpSearch()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	sendHelpMessage(s, callback, text, keyboard)
	return true
}

func handleMonitorHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
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
		text = GetHelpStatus()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_stats":
		text = GetHelpStats()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_system":
		text = GetHelpSystem()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_health":
		text = GetHelpHealth()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_logs":
		text = GetHelpLogs()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_ping":
		text = GetHelpPing()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_speed":
		text = GetHelpSpeed()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	sendHelpMessage(s, callback, text, keyboard)
	return true
}

func handleFilesHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
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
		text = GetHelpLs()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mkdir":
		text = GetHelpMkdir()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_rm":
		text = GetHelpRm()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mv":
		text = GetHelpMv()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_share":
		text = GetHelpShare()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_find":
		text = GetHelpFind()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	sendHelpMessage(s, callback, text, keyboard)
	return true
}

func handleMediaHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
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
		text = GetHelpExtractAudio()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_compress":
		text = GetHelpCompress()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_thumbnail":
		text = GetHelpThumbnail()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_screenshots":
		text = GetHelpScreenshots()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_subtitle":
		text = GetHelpSubtitle()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_hardsub":
		text = GetHelpHardsub()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_rescale":
		text = GetHelpRescale()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_convert":
		text = GetHelpConvert()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_mediainfo":
		text = GetHelpMediaInfo()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	sendHelpMessage(s, callback, text, keyboard)
	return true
}

func handleTaskHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
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
		text = GetHelpCancel()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_cancelall":
		text = GetHelpCancelAll()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	sendHelpMessage(s, callback, text, keyboard)
	return true
}

func handleStorageHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
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
		text = GetHelpStorages()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setstorage":
		text = GetHelpSetStorage()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	sendHelpMessage(s, callback, text, keyboard)
	return true
}

func handleAdminHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
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
		text = GetHelpAuthorize()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_unauthorize":
		text = GetHelpUnauthorize()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_users":
		text = GetHelpUsers()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setalertchannel":
		text = GetHelpSetAlertChannel()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_setlogchannel":
		text = GetHelpSetLogChannel()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	sendHelpMessage(s, callback, text, keyboard)
	return true
}

func handleRecoveryHelp(s *service.BotService, callback *tgbotapi.CallbackQuery, action string) bool {
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
		text = GetHelpRecover()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	case "cmd_recoverystatus":
		text = GetHelpRecoveryStatus()
		keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			backToCat,
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		))
	default:
		return false
	}
	sendHelpMessage(s, callback, text, keyboard)
	return true
}
