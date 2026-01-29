package handlers

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	fmt.Println("👉 HandleStart dipanggil")
	welcomeText := "🚀 *Selamat Datang di Zee\\-Mirror Bot\\!*\n\n" +
		"Bot ini membantu Anda melakukan mirror/leech file ke Google Drive\\.\n\n" +
		"*Fitur Utama:*\n" +
		"📥 Mirror \\- Download \\& upload ke Drive\n" +
		"🔗 Leech \\- Download dari URL\n" +
		"🎬 YT\\-DLP \\- Download video streaming\n" +
		"🧲 Torrent \\- Download via magnet/torrent\n" +
		"📊 Status \\- Pantau progress task\n" +
		"⚙️ Settings \\- Konfigurasi bot\n\n" +
		"Gunakan tombol di bawah atau ketik /help untuk bantuan\\."

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📥 Mirror", "dashboard:mirror"),
			tgbotapi.NewInlineKeyboardButtonData("🔗 Leech", "dashboard:leech"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎬 YT-DLP", "dashboard:ytdlp"),
			tgbotapi.NewInlineKeyboardButtonData("🧲 Torrent", "dashboard:torrent"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Status", "dashboard:status"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Settings", "dashboard:settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel Task", "dashboard:cancel"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		fmt.Printf("❌ Error sending welcome message: %v\n", err)
	} else {
		fmt.Println("✅ Welcome message sent")
	}
}

func HandleHelp(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	helpText := "📚 *Panduan Penggunaan Zee\\-Mirror Bot*\n\n" +
		"*Perintah Utama:*\n\n" +
		"/start \\- Tampilkan dashboard utama\n" +
		"/help \\- Tampilkan bantuan ini\n" +
		"/mirror URL \\- Mirror file ke Google Drive\n" +
		"/leech URL \\- Leech dari URL\n" +
		"/ytdlp URL \\- Download video via yt\\-dlp\n" +
		"/torrent magnet \\- Download torrent\n" +
		"/status \\- Lihat status task aktif\n" +
		"/cancel ID \\- Cancel task tertentu\n" +
		"/search keyword \\- Cari torrent via Jackett\n" +
		"/settings \\- Pengaturan bot\n\n" +
		"*Flag Opsional:*\n\n" +
		"\\-z : Zip file setelah download\n" +
		"\\-uz : Unzip/extract setelah download\n" +
		"\\-p PASSWORD : Password untuk zip\n\n" +
		"*Tips:*\n" +
		"\\- Reply ke file/media untuk mirror langsung\n" +
		"\\- Magnet link otomatis terdeteksi\n" +
		"\\- Progress diupdate setiap 5 detik\n" +
		"\\- Cancel task kapan saja dengan /cancel"

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = MarkdownV2

	_, _ = bot.Send(msg)
}
func HandleDashboardCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		return
	}
	action := parts[1]

	var text string
	switch action {
	case "page":
		if len(parts) >= 3 {
			page, _ := strconv.Atoi(parts[2])
			taskManager.Mu.Lock()
			taskManager.StatusPages[callback.Message.Chat.ID] = page
			taskManager.Mu.Unlock()
			UpdateSharedDashboard(bot, callback.Message.Chat.ID, false)
			_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Halaman %d", page+1)))
		}
		return
	case "mirror":
		text = "📥 *Mirror Mode*\n\n" +
			"Kirim file/media atau reply ke file dengan /mirror untuk memulai\\.\n\n" +
			"Atau kirim URL dengan format:\n" +
			"/mirror URL\n\n" +
			"*Flags:*\n" +
			"\\-z : Zip sebelum upload\n" +
			"\\-uz : Extract setelah download\n" +
			"\\-p PASSWORD : Password zip"

	case "leech":
		text = "🔗 *Leech Mode*\n\n" +
			"Kirim URL untuk download:\n" +
			"/leech URL\n\n" +
			"Support:\n" +
			"\\- HTTP/HTTPS links\n" +
			"\\- FTP links\n" +
			"\\- Magnet links\n" +
			"\\- Direct download links\n\n" +
			"*Flags:*\n" +
			"\\-uz : Extract setelah download\n" +
			"\\-z : Zip hasil download"

	case "ytdlp":
		text = "🎬 *YouTube\\-DLP Mode*\n\n" +
			"Download video dari YouTube dan 1000\\+ situs lainnya:\n" +
			"/ytdlp URL\n\n" +
			"Support:\n" +
			"\\- YouTube \\(video \\& playlist\\)\n" +
			"\\- Twitter/X\n" +
			"\\- Instagram\n" +
			"\\- TikTok\n" +
			"\\- Dan banyak lagi\\!"

	case "torrent":
		text = "🧲 *Torrent Mode*\n\n" +
			"Download via magnet atau file torrent:\n" +
			"/torrent magnet:xt=urn:btih:xxxxx\n\n" +
			"Atau reply ke file \\.torrent dengan /torrent"

	case "status":
		HandleStatusFromCallback(bot, callback)
		return

	case "settings":
		HandleSettingsFromCallback(bot, callback)
		return

	case "cancel":
		text = "❌ *Cancel Task*\n\n" +
			"Gunakan perintah:\n" +
			"/cancel TaskID\n\n" +
			"Lihat daftar task aktif dengan /status"

	default:
		text = "❓ Aksi tidak dikenal"
	}

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	_, _ = bot.Send(msg)
}

func HandleStatusFromCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	tasks := taskManager.GetActiveTasks()

	if len(tasks) == 0 {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "📭 *Tidak ada task aktif*")
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	text := StatusHeaderText
	for _, task := range tasks {
		snapshot := task.GetSnapshot()
		text += formatTaskLine(snapshot) + "\n\n"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "refresh_status"),
		),
	)

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, _ = bot.Send(msg)
}

func HandleSettingsFromCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	text := formatSettingsMessage()
	keyboard := getSettingsKeyboard()

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, _ = bot.Send(msg)
}

func formatTaskLine(task TaskSnapshot) string {
	emoji := StatusEmoji(string(task.Status))
	bar := ProgressBar(task.Progress, 10)

	return fmt.Sprintf(
		"━━━━━━━━━━━━━━━━━━━━━━━━\n"+
			"%s *ID:* `%s` \\| *%s\\.\\.\\.*\n"+
			"%s\n"+
			"📄 *File:* %s\n"+
			"📦 *Size:* %s\n"+
			"⚡ *Speed:* %s \\| *CN:* %d \\| ⏱️ *ETA:* %s\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━",
		emoji,
		task.ID,
		EscapeMarkdownV2(FormatStatus(string(task.Status))),
		EscapeMarkdownV2(bar),
		EscapeMarkdownV2(TruncateString(task.FileName, 40)),
		EscapeMarkdownV2(FormatBytes(task.TotalSize)),
		EscapeMarkdownV2(FormatSpeed(task.Speed)),
		task.Connections,
		EscapeMarkdownV2(FormatDuration(task.ETA)),
	)
}
