package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"zee-mirror/handlers"
	"zee-mirror/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	log.Println("🚀 Starting Zee-Mirror Bot...")

	cfg := config.LoadConfig()
	if !cfg.Validate() {
		log.Fatal("❌ Invalid configuration")
	}

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
	}

	bot.Debug = false
	log.Printf("✅ Authorized on account %s", bot.Self.UserName)

	service := handlers.NewBotService(bot, cfg)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Graceful shutdown
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
			handleMessage(service, update.Message, cfg)
		} else if update.CallbackQuery != nil {
			handleCallback(service, update.CallbackQuery)
		}
	}
}

func handleMessage(s *handlers.BotService, msg *tgbotapi.Message, cfg *config.Config) {
	if !cfg.IsAuthorized(msg.From.ID) {
		log.Printf("[Auth] Unauthorized access from %d (%s)", msg.From.ID, msg.From.UserName)
		return
	}

	if msg.IsCommand() {
		command := msg.Command()
		args := msg.CommandArguments()

		switch command {
		case "start":
			s.HandleStart(msg)
		case "help":
			s.HandleHelp(msg)
		case "mirror":
			s.HandleMirror(msg, args)
		case "leech":
			s.HandleLeech(msg, args)
		case "ytdlp":
			s.HandleYTDLP(msg, args)
		case "torrent":
			s.HandleTorrent(msg, args)
		case "status":
			s.HandleStatus(msg)
		case "cancel":
			s.HandleCancel(msg, args)
		case "search":
			s.HandleSearch(msg, args)
		case "settings":
			s.HandleSettings(msg)
		case "batch":
			s.HandleBatch(msg, args)
		case "batchstatus":
			s.HandleBatchStatus(msg)
		case "cancelbatch":
			s.HandleCancelBatch(msg, args)
		default:
			// Unknown command
		}
		return
	}

	// Handle non-command messages (e.g. magnet links, file replies)
	text := msg.Text
	if text == "" && msg.Caption != "" {
		text = msg.Caption
	}

	if text != "" {
		if strings.HasPrefix(text, "magnet:?") {
			s.HandleTorrent(msg, text)
			return
		}
		if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
			// Auto mirror/leech based on default setting? 
			// For now, let's not auto-start tasks on just URLs to avoid spam
		}
	}

	if msg.ReplyToMessage != nil && (msg.Document != nil || msg.Video != nil) {
		// This is handled in HandleMirror/HandleLeech when command is used
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
