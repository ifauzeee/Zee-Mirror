package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"zee-mirror/handlers"
	"zee-mirror/internal/config"
	"zee-mirror/internal/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	cmdBatch    = "batch"
	cmdSettings = "settings"
)

func main() {
	log.Println("🚀 Starting Zee-Mirror Bot...")

	cfg := config.LoadConfig()
	if !cfg.Validate() {
		log.Fatal("❌ Invalid configuration")
	}

	db, err := database.NewDB(cfg.ConfigDir)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}

	var bot *tgbotapi.BotAPI

	if cfg.TelegramAPI != "" {
		bot, err = tgbotapi.NewBotAPIWithAPIEndpoint(cfg.BotToken, cfg.TelegramAPI)
		if err == nil {
			log.Printf("🌐 Using custom API endpoint: %s", cfg.TelegramAPI)
		}
	} else {
		bot, err = tgbotapi.NewBotAPI(cfg.BotToken)
	}

	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
	}

	bot.Debug = false
	log.Printf("✅ Authorized on account %s", bot.Self.UserName)

	service := handlers.NewBotService(bot, cfg, db)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		service.Shutdown()
		os.Exit(0)
	}()

	for update := range updates {
		if update.Message != nil {
			handleMessage(service, update.Message)
		} else if update.CallbackQuery != nil {
			handleCallback(service, update.CallbackQuery)
		}
	}
}

func handleMessage(s *handlers.BotService, msg *tgbotapi.Message) {
	if !s.IsAuthorized(msg.From.ID) {
		log.Printf("[Auth] Unauthorized access from %d (%s)", msg.From.ID, msg.From.UserName)
		return
	}

	_ = s.DB.UpsertUser(msg.From.ID, msg.From.UserName, "user")

	if msg.IsCommand() {
		handleCommand(s, msg)
		return
	}

	text := msg.Text
	if text == "" && msg.Caption != "" {
		text = msg.Caption
	}

	if text != "" {
		if strings.HasPrefix(text, "magnet:?") {
			s.HandleTorrent(msg, text)
			return
		}
	}
}

func handleCommand(s *handlers.BotService, msg *tgbotapi.Message) {
	command := msg.Command()
	args := msg.CommandArguments()

	switch command {
	case "start", "help", cmdSettings:
		handleBasicCommands(s, msg, command)
	case "mirror", "leech", "ytdlp", "torrent", "clone":
		handleDownloadCommands(s, msg, command, args)
	case "status", "cancel", "search":
		handleTaskCommands(s, msg, command, args)
	case cmdBatch, "batchstatus", "cancelbatch":
		handleBatchCommands(s, msg, command, args)
	case "authorize", "unauthorize", "users":
		handleAdminCommands(s, msg, command, args)
	}
}

func handleBasicCommands(s *handlers.BotService, msg *tgbotapi.Message, command string) {
	switch command {
	case "start":
		s.HandleStart(msg)
	case "help":
		s.HandleHelp(msg)
	case cmdSettings:
		s.HandleSettings(msg)
	}
}

func handleDownloadCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	switch command {
	case "mirror":
		s.HandleMirror(msg, args)
	case "leech":
		s.HandleLeech(msg, args)
	case "ytdlp":
		s.HandleYTDLP(msg, args)
	case "torrent":
		s.HandleTorrent(msg, args)
	case "clone":
		s.HandleClone(msg, args)
	}
}

func handleTaskCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	switch command {
	case "status":
		s.HandleStatus(msg)
	case "cancel":
		s.HandleCancel(msg, args)
	case "search":
		s.HandleSearch(msg, args)
	}
}

func handleBatchCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	switch command {
	case cmdBatch:
		s.HandleBatch(msg, args)
	case "batchstatus":
		s.HandleBatchStatus(msg)
	case "cancelbatch":
		s.HandleCancelBatch(msg, args)
	}
}

func handleAdminCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	switch command {
	case "authorize":
		s.HandleAuthorize(msg, args)
	case "unauthorize":
		s.HandleUnauthorize(msg, args)
	case "users":
		s.HandleUsers(msg)
	}
}

func handleCallback(s *handlers.BotService, cb *tgbotapi.CallbackQuery) {
	data := cb.Data
	parts := strings.Split(data, ":")

	if len(parts) == 0 {
		return
	}

	prefix := parts[0]
	switch {
	case prefix == "dashboard":
		s.HandleDashboardCallback(cb)
	case prefix == "ytdlp_q":
		s.HandleYTDLPQualityCallback(cb, parts)
	case prefix == "t_search":
		s.HandleSearchCallback(cb, parts)
	case prefix == "t_page", prefix == "t_item", prefix == "t_back", prefix == "t_close":
		s.HandleSearchNavCallback(cb, parts)
	case prefix == "settings":
		s.HandleSettingsCallback(cb, parts)
	case prefix == "batch":
		s.HandleBatchCallback(cb, parts)
	case prefix == "refresh_status":
		s.HandleRefreshStatusCallback(cb)
	case prefix == "confirm":
		s.HandleConfirmCallback(cb, parts)
	case strings.HasPrefix(prefix, "cancel_"):
		taskID := strings.TrimPrefix(prefix, "cancel_")
		s.HandleCancelCallback(cb, taskID)
	default:
		log.Printf("[Callback] Unknown callback: %s", data)
	}
}
