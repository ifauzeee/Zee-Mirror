package handlers

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	ActionBack = "back"
)

func (s *BotService) HandleStart(message *tgbotapi.Message) {
	userName := message.From.FirstName
	if userName == "" {
		userName = "User"
	}

	welcomeText := GetWelcomeMessage(userName)
	keyboard := GetStartKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard

	if sentMsg, err := s.Bot.Send(msg); err != nil {
		fmt.Printf("❌ Error sending welcome message: %v\n", err)
	} else {
		s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 60*time.Second)
	}
}

func (s *BotService) HandleHelp(message *tgbotapi.Message) {
	helpText := GetHelpMainText()
	keyboard := GetHelpKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard

	if sentMsg, err := s.Bot.Send(msg); err != nil {
		fmt.Printf("❌ Error sending help message: %v\n", err)
		msg.ParseMode = ""
		msg.Text = "📖 Panduan Bantuan\n\nSilakan pilih kategori di bawah:"
		_, _ = s.Bot.Send(msg)
	} else {
		s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 60*time.Second)
	}
}

func (s *BotService) HandleDashboardCallback(callback *tgbotapi.CallbackQuery) {
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		return
	}
	action := parts[1]

	if parts[0] == "help" {
		s.handleHelpCallback(callback, action)
		return
	}

	if s.handleDashboardAction(callback, action, parts) {
		return
	}

	text := s.getDashboardModeText(action)
	keyboard := s.getModeKeyboard(action)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

func (s *BotService) handleDashboardAction(callback *tgbotapi.CallbackQuery, action string, parts []string) bool {
	switch action {
	case "page":
		s.handlePagination(callback, parts)
	case "status":
		s.HandleStatusFromCallback(callback)
	case "settings":
		s.HandleSettingsFromCallback(callback)
	case "ping":
		s.HandlePingFromCallback(callback)
	case "speed":
		s.HandleSpeedFromCallback(callback)
	case "stats":
		s.HandleStatsFromCallback(callback)
	case "storage":
		s.HandleStoragesFromCallback(callback)
	case "files":
		s.HandleDriveListFromCallback(callback)
	case "media":
		s.sendMediaMenu(callback)
	case "system":
		s.HandleSystemFromCallback(callback)
	case ActionBack:
		content := GetWelcomeMessage(callback.From.FirstName)
		kb := GetStartKeyboard()
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, content)
		editMsg.ParseMode = MarkdownV2
		editMsg.ReplyMarkup = &kb
		_, _ = s.Bot.Send(editMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	case CmdClose:
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, _ = s.Bot.Request(deleteMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Closed"))
	default:
		return false
	}
	return true
}

func (s *BotService) sendMediaMenu(callback *tgbotapi.CallbackQuery) {
	content := "Operasi media untuk video/audio:\\n\\n" +
		"• `/extractaudio` ─ Ambil audio dari video\\n" +
		"• `/compress` ─ Kompres ukuran video\\n" +
		"• `/thumbnail` ─ Generate thumbnail\\n" +
		"• `/screenshots` ─ Multi screenshot\\n" +
		"• `/subtitle` ─ Embed subtitle ke video\\n" +
		"• `/convert` ─ Konversi format file\\n" +
		"• `/mediainfo` ─ Info detail file\\n\\n" +
		"💡 *Cara pakai:* Reply ke file video dengan command"

	text := ProfessionalMessage("MEDIA PROCESSING", content)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Extract Audio", "media:extract"),
			tgbotapi.NewInlineKeyboardButtonData("🗜️ Compress", "media:compress"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🖼️ Thumbnail", "media:thumb"),
			tgbotapi.NewInlineKeyboardButtonData("📸 Screenshots", "media:screens"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Subtitle", "media:subtitle"),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Convert", "media:convert"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Media Info", "media:info"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Help", "help:main"),
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🎵 Media Menu"))
}

func (s *BotService) getModeKeyboard(action string) tgbotapi.InlineKeyboardMarkup {
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:download")

	switch action {
	case "mirror", "leech":
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊 Status", "dashboard:status"),
				backBtn,
			),
		)
	case "batch":
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Batch Status", "batch:status"),
				backBtn,
			),
		)
	default:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				backBtn,
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "dashboard:close"),
			),
		)
	}
}

func (s *BotService) getDashboardModeText(action string) string {
	switch action {
	case "mirror":
		return ProfessionalMessage("MIRROR MODE",
			"Upload file langsung ke Google Drive\\.\n\n"+
				"📌 *CARA PAKAI*\n"+
				"• Reply ke file dengan `/mirror`\n"+
				"• Atau kirim: `/mirror URL`\n\n"+
				"🏷️ *FLAGS*\n"+
				"• `\\-z` ─ Zip sebelum upload\n"+
				"• `\\-uz` ─ Unzip setelah download\n"+
				"• `\\-p PASS` ─ Password zip")

	case "leech":
		return ProfessionalMessage("LEECH MODE",
			"Download file dari URL ke server\\.\n\n"+
				"✅ *SUPPORT*\n"+
				"• HTTP/HTTPS \\| FTP \\| Magnet\n\n"+
				"📌 *CARA PAKAI*\n"+
				"• `/leech <URL>`")

	case "ytdlp":
		return ProfessionalMessage("YT-DLP MODE",
			"Download video dari 1000+ situs\\.\n\n"+
				"✅ *SUPPORT*\n"+
				"• YouTube \\| Twitter \\| TikTok \\| etc.\n\n"+
				"📌 *CARA PAKAI*\n"+
				"• `/ytdlp <URL>`")

	case "torrent":
		return ProfessionalMessage("TORRENT MODE",
			"Download via magnet atau file torrent\\.\n\n"+
				"📌 *CARA PAKAI*\n"+
				"• `/torrent magnet_link`\n"+
				"• Reply ke file `.torrent` dengan `/torrent`")

	case "clone":
		return ProfessionalMessage("CLONE MODE",
			"Clone file/folder Google Drive\\.\n\n"+
				"📌 *CARA PAKAI*\n"+
				"• `/clone <GDRIVE_URL>`")

	case "batch":
		return ProfessionalMessage("BATCH MODE",
			"Download multiple URLs sekaligus\\.\n\n"+
				"📌 *CARA PAKAI*\n"+
				"`/batch`\n"+
				"`URL1`\n"+
				"`URL2`")

	case "search":
		return ProfessionalMessage("SEARCH MODE",
			"Cari torrent dari berbagai sumber\\.\n\n"+
				"📌 *CARA PAKAI*\n"+
				"• `/search <keyword>`")

	case "back":
		return ""

	default:
		return GetErrorMessage("UNKNOWN", "Aksi tidak dikenal.")
	}
}

func (s *BotService) HandleStatusFromCallback(callback *tgbotapi.CallbackQuery) {
	tasks := s.TaskManager.GetActiveTasks()

	if len(tasks) == 0 {
		content := GetErrorMessage("STATUS", "Tidak ada task aktif.")
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, content)
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📭 Empty"))
		return
	}

	text := GetStatusHeader()
	for _, task := range tasks {
		snapshot := task.GetSnapshot()
		text += FormatTaskProfessional(snapshot) + "\n"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "refresh_status"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:monitor"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "dashboard:close"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Updated"))
}

func (s *BotService) HandleSettingsFromCallback(callback *tgbotapi.CallbackQuery) {
	text := s.formatSettingsMessage()
	keyboard := s.getSettingsKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "⚙️ Settings"))
}

func (s *BotService) HandlePingFromCallback(callback *tgbotapi.CallbackQuery) {
	start := time.Now()
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, "🏓 *Pinging\\.\\.\\.*")
	editMsg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(editMsg)

	elapsed := time.Since(start)
	text := fmt.Sprintf("🏓 *Pong\\!* `%v`", elapsed.Round(time.Millisecond))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Re-Ping", "dashboard:ping"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:monitor"),
		),
	)

	finalEdit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	finalEdit.ParseMode = MarkdownV2
	finalEdit.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(finalEdit)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🏓 Pong!"))
}

func (s *BotService) HandleSpeedFromCallback(callback *tgbotapi.CallbackQuery) {
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, "🚀 *Running Speedtest\\.\\.\\.*")
	editMsg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(editMsg)

	go func() {
		cmd := exec.Command("speedtest-cli", "--simple")
		output, err := cmd.CombinedOutput()

		var text string
		if err != nil {
			text = fmt.Sprintf("❌ *Speedtest Error*\n\n`%s`", utils.EscapeMarkdownV2(err.Error()))
		} else {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			var result strings.Builder
			result.WriteString("🚀 *Speedtest Result*\n\n")
			for _, line := range lines {
				result.WriteString(fmt.Sprintf("• `%s`\n", utils.EscapeMarkdownV2(line)))
			}
			text = result.String()
		}

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Retest", "dashboard:speed"),
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:monitor"),
			),
		)

		finalEdit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		finalEdit.ParseMode = MarkdownV2
		finalEdit.ReplyMarkup = &keyboard
		_, _ = s.Bot.Send(finalEdit)
	}()
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🚀 Testing speed..."))
}

func (s *BotService) HandleStatsFromCallback(callback *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	stats, err := s.DB.GetBotStats(ctx)
	if err != nil {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Error"))
		return
	}

	userStats, _ := s.DB.GetUserStats(ctx, callback.From.ID)
	dailyStats, _ := s.DB.GetTodayStats(ctx)
	userDailyStats, _ := s.DB.GetUserTodayStats(ctx, callback.From.ID)

	text := s.formatStatsMessage(stats, userStats, dailyStats, userDailyStats)
	keyboard := s.getStatsKeyboard()

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📊 Statistics"))
}

func (s *BotService) HandleStoragesFromCallback(callback *tgbotapi.CallbackQuery) {
	providers, err := s.GetAvailableStorages()
	if err != nil {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Error"))
		return
	}

	var text strings.Builder
	text.WriteString("💾 *Available Storage Providers*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range providers {
		text.WriteString(fmt.Sprintf("%s *%s* \\(%s\\)\n", p.Icon, utils.EscapeMarkdownV2(p.Name), utils.EscapeMarkdownV2(p.Type)))

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %s", p.Icon, p.Name),
				fmt.Sprintf("storage:select:%s", p.Name),
			),
		))
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━\n")
	text.WriteString("*Current:* `" + utils.EscapeMarkdownV2(s.TaskManager.RcloneDest) + "`\n\n")

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:storage"),
		tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "storage:close:none"),
	))
	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text.String())
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "💾 Storages"))
}

func (s *BotService) HandleSystemFromCallback(callback *tgbotapi.CallbackQuery) {
	stats := s.getSystemStats()
	text := s.formatSystemStats(stats)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "system:refresh"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Detailed", "system:detailed"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Logs", "system:logs"),
			tgbotapi.NewInlineKeyboardButtonData("🧹 Cleanup", "system:cleanup"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:monitor"),
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "system:close"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🖥️ System"))
}

func (s *BotService) HandleDriveListFromCallback(callback *tgbotapi.CallbackQuery) {
	basePath := strings.TrimSuffix(s.TaskManager.RcloneDest, "/")
	path := basePath

	loadingMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, "🔍 *Memuat daftar file\\.\\.\\.*")
	loadingMsg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(loadingMsg)

	files, err := s.listDriveFiles(path)
	if err != nil {
		text := fmt.Sprintf("❌ *Gagal memuat daftar file*\n\nError: %s", utils.EscapeMarkdownV2(err.Error()))
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)
		return
	}

	relPath := ""
	if strings.Contains(path, ":") {
		parts := strings.SplitN(path, ":", 2)
		if len(parts) > 1 {
			relPath = "/"
		}
	}

	text := s.formatDriveFileList("/", files)
	keyboard := s.buildDriveNavigationKeyboard(files, relPath)

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "help:files"),
	))

	finalEdit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	finalEdit.ParseMode = MarkdownV2
	finalEdit.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(finalEdit)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📁 Drive List"))
}

func (s *BotService) handlePagination(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) >= 3 {
		page, _ := strconv.Atoi(parts[2])
		s.TaskManager.Mu.Lock()
		s.TaskManager.StatusPages[callback.Message.Chat.ID] = page
		s.TaskManager.Mu.Unlock()
		s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Halaman %d", page+1)))
	}
}
