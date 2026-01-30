package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"zee-mirror/internal/domain"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) HandleStart(message *tgbotapi.Message) {
	fmt.Println("👉 HandleStart dipanggil")
	welcomeText := "🚀 *Selamat Datang di Zee\\-Mirror Bot\\!*\n\n" +
		"Bot ini membantu Anda melakukan mirror/leech file ke Google Drive\\.\n\n" +
		"*Fitur Utama:*\n" +
		"📥 Mirror \\- Download \\& upload ke Drive\n" +
		"🔗 Leech \\- Download dari URL\n" +
		"🎬 YT\\-DLP \\- Download video streaming\n" +
		"🧲 Torrent \\- Download via magnet/torrent\n" +
		"📂 Clone \\- Clone file/folder GDrive\n" +
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
			tgbotapi.NewInlineKeyboardButtonData("📂 Clone", "dashboard:clone"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Batch", "dashboard:batch"),
			tgbotapi.NewInlineKeyboardButtonData("🔍 Search", "dashboard:search"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Status", "dashboard:status"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Settings", "dashboard:settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏓 Ping", "dashboard:ping"),
			tgbotapi.NewInlineKeyboardButtonData("🚀 Speedtest", "dashboard:speed"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel Task", "dashboard:cancel"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard

	if _, err := s.Bot.Send(msg); err != nil {
		fmt.Printf("❌ Error sending welcome message: %v\n", err)
	} else {
		fmt.Println("✅ Welcome message sent")
	}
}

func (s *BotService) HandleHelp(message *tgbotapi.Message) {
	helpText := "📚 *Panduan Penggunaan Zee\\-Mirror Bot*\n\n" +
		"*Perintah Utama:*\n\n" +
		"/start \\- Tampilkan dashboard utama\n" +
		"/help \\- Tampilkan bantuan ini\n" +
		"/mirror URL \\- Mirror file ke Google Drive\n" +
		"/leech URL \\- Leech dari URL\n" +
		"/ytdlp URL \\- Download video via yt\\-dlp\n" +
		"/torrent magnet \\- Download torrent\n" +
		"/clone URL \\- Clone Google Drive file/folder\n" +
		"/status \\- Lihat status task aktif\n" +
		"/cancel ID \\- Cancel task tertentu\n" +
		"/search keyword \\- Cari torrent via Jackett\n" +
		"/ping \\- Cek latency bot\n" +
		"/speed \\- Test kecepatan internet server\n" +
		"/settings \\- Pengaturan bot\n\n" +
		"*Batch Download:*\n\n" +
		"/batch \\- Download multiple URLs sekaligus\n" +
		"\n" +
		"*Admin Commands:*\n\n" +
		"/authorize ID \\- Izinkan user baru\n" +
		"/unauthorize ID \\- Cabut izin user\n" +
		"/users \\- Lihat daftar user\n\n" +
		"*Flag Opsional:*\n" +
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

	_, _ = s.Bot.Send(msg)
}

func (s *BotService) HandleDashboardCallback(callback *tgbotapi.CallbackQuery) {
	parts := strings.Split(callback.Data, ":")
	if len(parts) < 2 {
		return
	}
	action := parts[1]

	switch action {
	case "page":
		if len(parts) >= 3 {
			page, _ := strconv.Atoi(parts[2])
			s.TaskManager.Mu.Lock()
			s.TaskManager.StatusPages[callback.Message.Chat.ID] = page
			s.TaskManager.Mu.Unlock()
			s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Halaman %d", page+1)))
		}
		return
	case "status":
		s.HandleStatusFromCallback(callback)
		return
	case "settings":
		s.HandleSettingsFromCallback(callback)
		return
	case "ping":
		s.HandlePing(callback.Message)
		return
	case "speed":
		s.HandleSpeed(callback.Message)
		return
	}

	text := s.getDashboardModeText(action)
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) getDashboardModeText(action string) string {
	switch action {
	case "mirror":
		return "📥 *Mirror Mode*\n\n" +
			"Kirim file/media atau reply ke file dengan /mirror untuk memulai\\.\n\n" +
			"Atau kirim URL dengan format:\n" +
			"/mirror URL\n\n" +
			"*Flags:*\n" +
			"\\-z : Zip sebelum upload\n" +
			"\\-uz : Extract setelah download\n" +
			"\\-p PASSWORD : Password zip"

	case "leech":
		return "🔗 *Leech Mode*\n\n" +
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
		return "🎬 *YouTube\\-DLP Mode*\n\n" +
			"Download video dari YouTube dan 1000\\+ situs lainnya:\n" +
			"/ytdlp URL\n\n" +
			"Support:\n" +
			"\\- YouTube \\(video \\& playlist\\)\n" +
			"\\- Twitter/X\n" +
			"\\- Instagram\n" +
			"\\- TikTok\n" +
			"\\- Dan banyak lagi\\!"

	case "torrent":
		return "🧲 *Torrent Mode*\n\n" +
			"Download via magnet atau file torrent:\n" +
			"/torrent magnet:xt=urn:btih:xxxxx\n\n" +
			"Atau reply ke file \\.torrent dengan /torrent"

	case "clone":
		return "📂 *Clone Mode*\n\n" +
			"Cloning file atau folder Google Drive yang bersifat public\\.\n\n" +
			"Gunakan perintah:\n" +
			"/clone URL\\_GDRIVE\n\n" +
			"*Catatan:*\n" +
			"\\- Pastikan link bisa diakses publik\n" +
			"\\- Nama file/folder akan tetap sama"

	case "cancel":
		return "❌ *Cancel Task*\n\n" +
			"Gunakan perintah:\n" +
			"/cancel TaskID\n\n" +
			"Lihat daftar task aktif dengan /status"

	case "batch":
		return "📦 *Batch Download*\n\n" +
			"Download multiple URLs sekaligus:\n" +
			"```\n/batch\nURL1\nURL2\nURL3\n```\n\n" +
			"*Flags:*\n" +
			"\\-name NAME : Nama batch\n" +
			"\\-z : Zip semua hasil\n" +
			"\\-p PASSWORD : Password zip\n" +
			"\\-priority 1\\-10 : Prioritas\n\n" +
			"*Commands:*\n" +
			"/batchstatus \\- Status batch aktif\n" +
			"/cancelbatch ID \\- Cancel batch"

	case "search":
		return "🔍 *Search Torrent*\n\n" +
			"Cari torrent dari berbagai sumber:\n" +
			"/search keyword\n\n" +
			"*Sources:*\n" +
			"\\- SolidTorrents\n" +
			"\\- Nyaa\n" +
			"\\- PirateBay"

	default:
		return "❓ Aksi tidak dikenal"
	}
}

func (s *BotService) HandleStatusFromCallback(callback *tgbotapi.CallbackQuery) {
	tasks := s.TaskManager.GetActiveTasks()

	if len(tasks) == 0 {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "📭 *Tidak ada task aktif*")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
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
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) HandleSettingsFromCallback(callback *tgbotapi.CallbackQuery) {
	text := s.formatSettingsMessage()
	keyboard := s.getSettingsKeyboard()

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, _ = s.Bot.Send(msg)
}

func formatTaskLine(task TaskSnapshot) string {
	emoji := utils.StatusEmoji(string(task.Status))
	bar := utils.ProgressBar(task.Progress, 10)

	processedSize := task.DownloadedSize
	if task.Status == domain.StatusUploading {
		processedSize = task.UploadedSize
	}

	return fmt.Sprintf(
		"━━━━━━━━━━━━━━━━━━━━━━━━\n"+
			"%s *ID:* `%s` \\| *%s\\.\\.\\.*\n"+
			"%s\n"+
			"📄 *File:* %s\n"+
			"📦 *Processed:* %s / %s\n"+
			"⚡ *Speed:* %s \\| *CN:* %d \\| ⏱️ *ETA:* %s\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━",
		emoji,
		task.ID,
		utils.EscapeMarkdownV2(utils.FormatStatus(string(task.Status))),
		utils.EscapeMarkdownV2(bar),
		utils.EscapeMarkdownV2(utils.TruncateString(task.FileName, 40)),
		utils.EscapeMarkdownV2(utils.FormatBytes(processedSize)),
		utils.EscapeMarkdownV2(utils.FormatBytes(task.TotalSize)),
		utils.EscapeMarkdownV2(utils.FormatSpeed(task.Speed)),
		task.Connections,
		utils.EscapeMarkdownV2(utils.FormatDuration(task.ETA)),
	)
}
