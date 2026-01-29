package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"zee-mirror/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Config struct {
	BotToken               string
	OwnerID                int64
	AuthorizedUsers        []int64
	RcloneDest             string
	MaxConcurrentDownloads int
	DownloadDir            string
	ConfigDir              string
}

var AppConfig Config

func loadConfig() Config {
	cfg := Config{
		BotToken:               os.Getenv("BOT_TOKEN"),
		RcloneDest:             os.Getenv("RCLONE_DEST"),
		DownloadDir:            "/app/downloads",
		ConfigDir:              "/app/config",
		MaxConcurrentDownloads: 3,
	}

	if ownerIDStr := os.Getenv("OWNER_ID"); ownerIDStr != "" {
		if id, err := strconv.ParseInt(ownerIDStr, 10, 64); err == nil {
			cfg.OwnerID = id
		}
	}

	if authUsers := os.Getenv("AUTHORIZED_USERS"); authUsers != "" {
		for _, idStr := range strings.Split(authUsers, ",") {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				cfg.AuthorizedUsers = append(cfg.AuthorizedUsers, id)
			}
		}
	}

	if maxDL := os.Getenv("MAX_CONCURRENT_DOWNLOADS"); maxDL != "" {
		if n, err := strconv.Atoi(maxDL); err == nil && n > 0 {
			cfg.MaxConcurrentDownloads = n
		}
	}

	if dlDir := os.Getenv("DOWNLOAD_DIR"); dlDir != "" {
		cfg.DownloadDir = dlDir
	}
	if cfgDir := os.Getenv("CONFIG_DIR"); cfgDir != "" {
		cfg.ConfigDir = cfgDir
	}

	return cfg
}

func validateConfig(cfg Config) bool {
	if cfg.BotToken == "" {
		log.Fatal("❌ BOT_TOKEN tidak ditemukan! Set environment variable BOT_TOKEN")
		return false
	}
	if cfg.OwnerID == 0 {
		log.Fatal("❌ OWNER_ID tidak ditemukan! Set environment variable OWNER_ID")
		return false
	}
	if cfg.RcloneDest == "" {
		log.Println("⚠️ RCLONE_DEST tidak di-set, menggunakan default: gdrive:/MirrorBot")
		cfg.RcloneDest = "gdrive:/MirrorBot"
	}
	return true
}

func isAuthorized(userID int64) bool {
	if userID == AppConfig.OwnerID {
		return true
	}
	for _, id := range AppConfig.AuthorizedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

func main() {
	log.Println("🚀 Starting Zee-Mirror Bot...")

	AppConfig = loadConfig()
	if !validateConfig(AppConfig) {
		os.Exit(1)
	}

	os.MkdirAll(AppConfig.DownloadDir, 0755)
	os.MkdirAll(AppConfig.ConfigDir, 0755)

	bot, err := tgbotapi.NewBotAPI(AppConfig.BotToken)
	if err != nil {
		log.Fatalf("❌ Gagal inisialisasi bot: %v", err)
	}

	log.Printf("✅ Bot authorized sebagai @%s", bot.Self.UserName)

	handlers.InitTaskManager(bot, AppConfig.MaxConcurrentDownloads, AppConfig.DownloadDir, AppConfig.RcloneDest, AppConfig.ConfigDir)
	handlers.InitSettings()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("🛑 Shutting down gracefully...")
		handlers.ShutdownTaskManager()
		os.Exit(0)
	}()

	log.Println("📡 Menunggu pesan...")

	for update := range updates {
		if update.CallbackQuery != nil {
			go handleCallback(bot, update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			continue
		}

		log.Printf("📩 Pesan dari %s (ID: %d): %s", update.Message.From.UserName, update.Message.From.ID, update.Message.Text)

		if !isAuthorized(update.Message.From.ID) {
			log.Printf("⛔ Akses ditolak untuk user ID: %d", update.Message.From.ID)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⛔ *Anda tidak memiliki akses ke bot ini\\.*")
			msg.ParseMode = "MarkdownV2"
			bot.Send(msg)
			continue
		}

		if update.Message.IsCommand() {
			go handleCommand(bot, update.Message)
		}
	}
}

func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	cmd := message.Command()
	args := message.CommandArguments()

	switch cmd {
	case "start":
		handlers.HandleStart(bot, message)
	case "help":
		handlers.HandleHelp(bot, message)
	case "mirror":
		handlers.HandleMirror(bot, message, args)
	case "leech":
		handlers.HandleLeech(bot, message, args)
	case "ytdlp":
		handlers.HandleYTDLP(bot, message, args)
	case "torrent":
		handlers.HandleTorrent(bot, message, args)
	case "status":
		handlers.HandleStatus(bot, message)
	case "cancel":
		handlers.HandleCancel(bot, message, args)
	case "settings":
		handlers.HandleSettings(bot, message)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "❓ Command tidak dikenal\\. Gunakan /help untuk melihat daftar command\\.")
		msg.ParseMode = "MarkdownV2"
		bot.Send(msg)
	}
}

func handleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	if !isAuthorized(callback.From.ID) {
		bot.Request(tgbotapi.NewCallback(callback.ID, "⛔ Akses ditolak"))
		return
	}

	data := callback.Data
	parts := strings.Split(data, ":")

	switch parts[0] {
	case "dashboard":
		handlers.HandleDashboardCallback(bot, callback)
	case "cancel_task":
		if len(parts) > 1 {
			handlers.HandleCancelCallback(bot, callback, parts[1])
		}
	case "settings":
		handlers.HandleSettingsCallback(bot, callback, parts)
	case "confirm":
		handlers.HandleConfirmCallback(bot, callback, parts)
	case "refresh_status":
		handlers.HandleRefreshStatusCallback(bot, callback)
	case "ytdlp_q":
		handlers.HandleYTDLPQualityCallback(bot, callback, parts)
	default:
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}
