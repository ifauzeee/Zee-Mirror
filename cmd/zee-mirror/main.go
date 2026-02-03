package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"zee-mirror/handlers"
	"zee-mirror/internal/api"
	"zee-mirror/internal/config"
	"zee-mirror/internal/database"
	"zee-mirror/internal/router"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.LoadConfig()

	_ = os.MkdirAll(cfg.ConfigDir, 0750)
	setupLogger(cfg)

	slog.Info("Starting Zee-Mirror Bot...")

	if !cfg.Validate() {
		slog.Error("Invalid configuration")
		os.Exit(1)
	}

	db, err := database.NewDB(cfg.ConfigDir)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	bot, err := initBot(cfg)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	slog.Info("Authorized on account", "username", bot.Self.UserName)

	service := handlers.NewBotService(bot, cfg, db)

	apiServer := api.NewAPIServer(service, cfg.DashboardPort)
	apiServer.Start()

	r := router.NewRouter(service)
	setupRoutes(r)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Info("Shutting down...")
		service.Shutdown()
		os.Exit(0)
	}()

	processUpdates(updates, service, r)
}

func setupLogger(cfg *config.Config) {
	logPath := filepath.Join(cfg.ConfigDir, "zee-mirror.log")
	logFile, err := os.OpenFile(filepath.Clean(logPath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)

	var multi io.Writer = os.Stdout
	if err == nil {
		multi = io.MultiWriter(os.Stdout, logFile)
	}

	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(multi, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	banner := `
  ______ ______ ______     __  __ _____ _____  _____   ____  _____
 |___  /|  ____|  ____|   |  \/  |_   _|  __ \|  __ \ / __ \|  __ \
    / / | |__  | |__      | \  / | | | | |__) | |__) | |  | | |__) |
   / /  |  __| |  __|     | |\/| | | | |  _  /|  _  /| |  | |  _  /
  / /__ | |____| |____    | |  | |_| |_| | \ \| | \ \| |__| | | \ \
 /_____||______|______|   |_|  |_|_____|_|  \_\_|  \_\\____/|_|  \_\`

	_, _ = fmt.Fprintln(multi, banner)

	if err != nil {
		slog.Warn("Gagal membuka file log", "error", err)
	} else {
		slog.Info("Logging to file enabled: zee-mirror.log")
	}
}

func initBot(cfg *config.Config) (*tgbotapi.BotAPI, error) {
	var bot *tgbotapi.BotAPI
	var err error

	if cfg.TelegramAPI != "" {
		bot, err = tgbotapi.NewBotAPIWithAPIEndpoint(cfg.BotToken, cfg.TelegramAPI)
		if err == nil {
			slog.Info("Using custom API endpoint", "endpoint", cfg.TelegramAPI)
		}
	} else {
		bot, err = tgbotapi.NewBotAPI(cfg.BotToken)
	}

	if err != nil {
		return nil, err
	}

	bot.Debug = (cfg.LogLevel == "debug")
	return bot, nil
}

func processUpdates(updates tgbotapi.UpdatesChannel, service *handlers.BotService, r *router.Router) {
	for update := range updates {
		if update.Message != nil {
			if !service.IsAuthorized(update.Message.From.ID) {
				slog.Warn("Unauthorized access", "userID", update.Message.From.ID, "username", update.Message.From.UserName)
				continue
			}
			go r.HandleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			if !service.IsAuthorized(update.CallbackQuery.From.ID) {
				slog.Warn("Unauthorized callback", "userID", update.CallbackQuery.From.ID, "username", update.CallbackQuery.From.UserName)
				continue
			}
			go r.HandleCallback(update.CallbackQuery)
		}
	}
}

func setupRoutes(r *router.Router) {
	setupBasicRoutes(r)
	setupDownloadRoutes(r)
	setupAdminRoutes(r)
	setupFileManagerRoutes(r)
	setupMediaRoutes(r)
	setupCallbackRoutes(r)
}

func setupBasicRoutes(r *router.Router) {
	r.RegisterCommand("start", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleStart(m) })
	r.RegisterCommand("help", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleHelp(m) })
	r.RegisterCommand("settings", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSettings(m) })
	r.RegisterCommand("ping", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandlePing(m) })
	r.RegisterCommand("speed", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSpeed(m) })
	r.RegisterCommand("stats", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleStats(m) })
}

func setupDownloadRoutes(r *router.Router) {
	r.RegisterCommand("mirror", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleMirror(m, m.CommandArguments()) })
	r.RegisterCommand("m", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleMirror(m, m.CommandArguments()) })
	r.RegisterCommand("leech", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleLeech(m, m.CommandArguments()) })
	r.RegisterCommand("l", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleLeech(m, m.CommandArguments()) })
	r.RegisterCommand("ytdlp", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleYTDLP(m, m.CommandArguments()) })
	r.RegisterCommand("y", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleYTDLP(m, m.CommandArguments()) })
	r.RegisterCommand("ytdlpleech", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleYTDLPLeech(m, m.CommandArguments()) })
	r.RegisterCommand("yl", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleYTDLPLeech(m, m.CommandArguments()) })
	r.RegisterCommand("torrent", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleTorrent(m, m.CommandArguments()) })
	r.RegisterCommand("t", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleTorrent(m, m.CommandArguments()) })
	r.RegisterCommand("clone", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleClone(m, m.CommandArguments()) })
	r.RegisterCommand("cl", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleClone(m, m.CommandArguments()) })
}

func setupAdminRoutes(r *router.Router) {
	r.RegisterCommand("status", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleStatus(m) })
	r.RegisterCommand("st", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleStatus(m) })
	r.RegisterCommand("cancel", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleCancel(m, m.CommandArguments()) })
	r.RegisterCommand("c", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleCancel(m, m.CommandArguments()) })
	r.RegisterCommand("cancelall", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleCancelAll(m) })
	r.RegisterCommand("search", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSearch(m, m.CommandArguments()) })
	r.RegisterCommand("batch", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleBatch(m, m.CommandArguments()) })
	r.RegisterCommand("authorize", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleAuthorize(m, m.CommandArguments()) })
	r.RegisterCommand("unauthorize", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleUnauthorize(m, m.CommandArguments()) })
	r.RegisterCommand("removeuser", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleRemoveUser(m, m.CommandArguments()) })
	r.RegisterCommand("setrole", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSetRole(m, m.CommandArguments()) })
	r.RegisterCommand("setlimit", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSetLimit(m, m.CommandArguments()) })
	r.RegisterCommand("setexpire", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSetExpire(m, m.CommandArguments()) })
	r.RegisterCommand("users", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleUsers(m) })
	r.RegisterCommand("setalertchannel", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSetAlertChannel(m, m.CommandArguments()) })
	r.RegisterCommand(handlers.CmdSystem, func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSystem(m) })
	r.RegisterCommand(handlers.CmdHealth, func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleHealth(m) })
	r.RegisterCommand(handlers.CmdLogs, func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleLogs(m, m.CommandArguments()) })
	r.RegisterCommand("recover", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleRecover(m) })
	r.RegisterCommand("recoverystatus", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleRecoveryStatus(m) })
}

func setupFileManagerRoutes(r *router.Router) {
	fmHandler := func(s *handlers.BotService, m *tgbotapi.Message) {
		cmd := m.Command()
		args := m.CommandArguments()
		switch cmd {
		case "ls", "dir":
			s.HandleDriveList(m, args, 0)
		case "mkdir":
			s.HandleDriveMkdir(m, args)
		case "rm":
			s.HandleDriveDelete(m, args)
		case "mv":
			s.HandleDriveMove(m, args)
		case "share":
			s.HandleDriveShare(m, args)
		case "find":
			s.HandleDriveSearch(m, args)
		}
	}
	r.RegisterCommand("ls", fmHandler)
	r.RegisterCommand("dir", fmHandler)
	r.RegisterCommand("mkdir", fmHandler)
	r.RegisterCommand("rm", fmHandler)
	r.RegisterCommand("mv", fmHandler)
	r.RegisterCommand("share", fmHandler)
	r.RegisterCommand("find", fmHandler)

	r.RegisterCommand("storages", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleStorages(m) })
	r.RegisterCommand("setstorage", func(s *handlers.BotService, m *tgbotapi.Message) { s.HandleSetStorage(m, m.CommandArguments()) })
}

func setupMediaRoutes(r *router.Router) {
	mediaHandler := func(s *handlers.BotService, m *tgbotapi.Message) {
		cmd := m.Command()
		args := m.CommandArguments()
		switch cmd {
		case "extractaudio":
			s.HandleExtractAudio(m, args)
		case "compress":
			s.HandleCompressVideo(m, args)
		case "thumbnail":
			s.HandleGenerateThumbnail(m, args)
		case "screenshots":
			s.HandleScreenshots(m, args)
		case "subtitle":
			s.HandleEmbedSubtitle(m, args)
		case "convert":
			s.HandleConvertFormat(m, args)
		case "mediainfo":
			s.HandleMediaInfo(m, args)
		}
	}
	r.RegisterCommand("extractaudio", mediaHandler)
	r.RegisterCommand("compress", mediaHandler)
	r.RegisterCommand("thumbnail", mediaHandler)
	r.RegisterCommand("screenshots", mediaHandler)
	r.RegisterCommand("subtitle", mediaHandler)
	r.RegisterCommand("convert", mediaHandler)
	r.RegisterCommand("mediainfo", mediaHandler)
}

func setupCallbackRoutes(r *router.Router) {
	r.RegisterCallback("dashboard", func(s *handlers.BotService, cb *tgbotapi.CallbackQuery) { s.HandleDashboardCallback(cb) })
	r.RegisterCallback("help", func(s *handlers.BotService, cb *tgbotapi.CallbackQuery) { s.HandleDashboardCallback(cb) })
	r.RegisterCallback("refresh_status", func(s *handlers.BotService, cb *tgbotapi.CallbackQuery) { s.HandleRefreshStatusCallback(cb) })

	searchHandler := func(s *handlers.BotService, cb *tgbotapi.CallbackQuery) {
		parts := strings.Split(cb.Data, ":")
		if parts[0] == "t_search" {
			s.HandleSearchCallback(cb, parts)
		} else {
			s.HandleSearchNavCallback(cb, parts)
		}
	}
	r.RegisterCallback("t_search", searchHandler)
	r.RegisterCallback("t_page", searchHandler)
	r.RegisterCallback("t_item", searchHandler)
	r.RegisterCallback("t_back", searchHandler)
	r.RegisterCallback("t_close", searchHandler)

	taskActionHandler := func(s *handlers.BotService, cb *tgbotapi.CallbackQuery) {
		parts := strings.Split(cb.Data, ":")
		switch parts[0] {
		case "ytdlp_q":
			s.HandleYTDLPQualityCallback(cb, parts)
		case "settings":
			s.HandleSettingsCallback(cb, parts)
		case "batch":
			s.HandleBatchCallback(cb, parts)
		case "confirm":
			s.HandleConfirmCallback(cb, parts)
		case "torrent_sel":
			s.HandleTorrentSelectionCallback(cb, parts)
		}
	}
	r.RegisterCallback("ytdlp_q", taskActionHandler)
	r.RegisterCallback("settings", taskActionHandler)
	r.RegisterCallback("batch", taskActionHandler)
	r.RegisterCallback("confirm", taskActionHandler)
	r.RegisterCallback("torrent_sel", taskActionHandler)

	systemHandler := func(s *handlers.BotService, cb *tgbotapi.CallbackQuery) {
		parts := strings.Split(cb.Data, ":")
		switch parts[0] {
		case "stats":
			s.HandleStatsCallback(cb, parts)
		case handlers.CmdSystem:
			s.HandleSystemCallback(cb, parts)
		case "storage":
			s.HandleStorageCallback(cb, parts)
		case "drive", "dr":
			s.HandleDriveCallback(cb, parts)
		}
	}
	r.RegisterCallback("stats", systemHandler)
	r.RegisterCallback(handlers.CmdSystem, systemHandler)
	r.RegisterCallback("storage", systemHandler)
	r.RegisterCallback("drive", systemHandler)
	r.RegisterCallback("dr", systemHandler)

	r.RegisterCallback("media_m", func(s *handlers.BotService, cb *tgbotapi.CallbackQuery) {
		parts := strings.Split(cb.Data, ":")
		s.HandleMediaMirrorCallback(cb, parts)
	})
}
