package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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
	cmdHelp     = "help"
	cmdStats    = "stats"
)

func main() {
	cfg := config.LoadConfig()

	_ = os.MkdirAll(cfg.ConfigDir, 0750)
	logPath := filepath.Join(cfg.ConfigDir, "zee-mirror.log")
	logFile, err := os.OpenFile(filepath.Clean(logPath), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)

	var multi io.Writer = os.Stdout
	if err == nil {
		multi = io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multi)
	}

	banner := `
███████╗███████╗███████╗                       
╚══███╔╝██╔════╝██╔════╝                       
  ███╔╝ █████╗  █████╗                         
 ███╔╝  ██╔══╝  ██╔══╝                         
███████╗███████╗███████╗                       
╚══════╝╚══════╝╚══════╝                       
                                               
███╗   ███╗██╗██████╗ ██████╗  ██████╗ ██████╗ 
████╗ ████║██║██╔══██╗██╔══██╗██╔═══██╗██╔══██╗
██╔████╔██║██║██████╔╝██████╔╝██║   ██║██████╔╝
██║╚██╔╝██║██║██╔══██╗██╔══██╗██║   ██║██╔══██╗
██║ ╚═╝ ██║██║██║  ██║██║  ██║╚██████╔╝██║  ██║
╚═╝     ╚═╝╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝`

	_, _ = fmt.Fprintln(multi, banner)

	if err != nil {
		log.Printf("⚠️ Gagal membuka file log: %v", err)
	} else {
		log.Println("📝 Logging to file enabled: zee-mirror.log")
	}

	log.Println("🚀 Starting Zee-Mirror Bot...")

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

	log.Printf("[Command] Received: /%s from %d", command, msg.From.ID)

	switch command {
	case "start", cmdHelp, cmdSettings, "ping", "speed":
		handleBasicCommands(s, msg, command)
	case "mirror", "leech", "ytdlp", "torrent", "clone":
		handleDownloadCommands(s, msg, command, args)
	case "status", "cancel", "cancelall", "search":
		handleTaskCommands(s, msg, command, args)
	case cmdBatch, "batchstatus", "cancelbatch":
		handleBatchCommands(s, msg, command, args)
	case "authorize", "unauthorize", "users", "setlogchannel", "setalertchannel":
		handleAdminCommands(s, msg, command, args)
	case cmdStats:
		s.HandleStats(msg)
	case handlers.CmdSystem, handlers.CmdHealth, handlers.CmdLogs:
		handleSystemCommands(s, msg, command, args)
	case "ls", "dir", "mkdir", "rm", "mv", "share", "find":
		handleFileManagerCommands(s, msg, command, args)
	case "storages", "setstorage":
		handleStorageCommands(s, msg, command, args)
	case "extractaudio", "compress", "thumbnail", "screenshots", "subtitle", "convert", "mediainfo":
		handleMediaCommands(s, msg, command, args)
	case "recover", "recoverystatus":
		handleRecoveryCommands(s, msg, command)
	default:
		if strings.HasPrefix(command, "cancel_") {
			taskID := strings.TrimPrefix(command, "cancel_")
			s.HandleCancel(msg, taskID)
		}
	}
}

func handleBasicCommands(s *handlers.BotService, msg *tgbotapi.Message, command string) {
	switch command {
	case "start":
		s.HandleStart(msg)
	case cmdHelp:
		s.HandleHelp(msg)
	case cmdSettings:
		s.HandleSettings(msg)
	case "ping":
		s.HandlePing(msg)
	case "speed":
		s.HandleSpeed(msg)
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
	case "cancelall":
		s.HandleCancelAll(msg)
	case "search":
		s.HandleSearch(msg, args)
	}
}

func handleBatchCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	if command == cmdBatch {
		s.HandleBatch(msg, args)
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

	case "setalertchannel":
		s.HandleSetAlertChannel(msg, args)
	}
}

func handleSystemCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	switch command {
	case handlers.CmdSystem:
		s.HandleSystem(msg)
	case handlers.CmdHealth:
		s.HandleHealth(msg)
	case handlers.CmdLogs:
		s.HandleLogs(msg, args)
	}
}

func handleFileManagerCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	switch command {
	case "ls", "dir":
		s.HandleDriveList(msg, args, 0)
	case "mkdir":
		s.HandleDriveMkdir(msg, args)
	case "rm":
		s.HandleDriveDelete(msg, args)
	case "mv":
		s.HandleDriveMove(msg, args)
	case "share":
		s.HandleDriveShare(msg, args)
	case "find":
		s.HandleDriveSearch(msg, args)
	}
}

func handleStorageCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	switch command {
	case "storages":
		s.HandleStorages(msg)
	case "setstorage":
		s.HandleSetStorage(msg, args)
	}
}

func handleMediaCommands(s *handlers.BotService, msg *tgbotapi.Message, command, args string) {
	switch command {
	case "extractaudio":
		s.HandleExtractAudio(msg, args)
	case "compress":
		s.HandleCompressVideo(msg, args)
	case "thumbnail":
		s.HandleGenerateThumbnail(msg, args)
	case "screenshots":
		s.HandleScreenshots(msg, args)
	case "subtitle":
		s.HandleEmbedSubtitle(msg, args)
	case "convert":
		s.HandleConvertFormat(msg, args)
	case "mediainfo":
		s.HandleMediaInfo(msg, args)
	}
}

func handleRecoveryCommands(s *handlers.BotService, msg *tgbotapi.Message, command string) {
	switch command {
	case "recover":
		s.HandleRecover(msg)
	case "recoverystatus":
		s.HandleRecoveryStatus(msg)
	}
}

func handleCallback(s *handlers.BotService, cb *tgbotapi.CallbackQuery) {
	parts := strings.Split(cb.Data, ":")
	if len(parts) == 0 {
		return
	}

	prefix := parts[0]
	switch {
	case prefix == "dashboard", prefix == "help", prefix == "refresh_status":
		handleStatusCallbacks(s, cb, prefix)
	case prefix == "t_search", prefix == "t_page", prefix == "t_item", prefix == "t_back", prefix == "t_close":
		handleSearchCallbacks(s, cb, prefix, parts)
	case prefix == "ytdlp_q", prefix == "settings", prefix == "batch", prefix == "confirm":
		handleTaskActionCallbacks(s, cb, prefix, parts)
	case prefix == "stats", prefix == "system", prefix == "storage", prefix == "drive", prefix == "dr":
		handleSystemCallbacks(s, cb, prefix, parts)
	case prefix == "media_m":
		s.HandleMediaMirrorCallback(cb, parts)
	case strings.HasPrefix(prefix, "cancel_"):
		taskID := strings.TrimPrefix(prefix, "cancel_")
		s.HandleCancelCallback(cb, taskID)
	default:
		log.Printf("[Callback] Unknown callback: %s", cb.Data)
	}
}

func handleStatusCallbacks(s *handlers.BotService, cb *tgbotapi.CallbackQuery, prefix string) {
	if prefix == "refresh_status" {
		s.HandleRefreshStatusCallback(cb)
	} else {
		s.HandleDashboardCallback(cb)
	}
}

func handleSearchCallbacks(s *handlers.BotService, cb *tgbotapi.CallbackQuery, prefix string, parts []string) {
	if prefix == "t_search" {
		s.HandleSearchCallback(cb, parts)
	} else {
		s.HandleSearchNavCallback(cb, parts)
	}
}

func handleTaskActionCallbacks(s *handlers.BotService, cb *tgbotapi.CallbackQuery, prefix string, parts []string) {
	switch prefix {
	case "ytdlp_q":
		s.HandleYTDLPQualityCallback(cb, parts)
	case "settings":
		s.HandleSettingsCallback(cb, parts)
	case "batch":
		s.HandleBatchCallback(cb, parts)
	case "confirm":
		s.HandleConfirmCallback(cb, parts)
	}
}

func handleSystemCallbacks(s *handlers.BotService, cb *tgbotapi.CallbackQuery, prefix string, parts []string) {
	switch prefix {
	case cmdStats:
		s.HandleStatsCallback(cb, parts)
	case handlers.CmdSystem:
		s.HandleSystemCallback(cb, parts)
	case "storage":
		s.HandleStorageCallback(cb, parts)
	case "drive", "dr":
		s.HandleDriveCallback(cb, parts)
	}
}
