package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"gopkg.in/natefinch/lumberjack.v2"

	"zee-mirror/handlers"
	"zee-mirror/handlers/admin"
	"zee-mirror/handlers/basic"
	"zee-mirror/handlers/download"
	"zee-mirror/handlers/file"
	"zee-mirror/handlers/media"
	"zee-mirror/handlers/search"
	"zee-mirror/handlers/storage"
	"zee-mirror/internal/api"
	"zee-mirror/internal/cache"
	"zee-mirror/internal/config"
	"zee-mirror/internal/database"
	"zee-mirror/internal/metrics"
	"zee-mirror/internal/router"
	"zee-mirror/internal/service"
	"zee-mirror/internal/userbot"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"

	_ "zee-mirror/plugins/drive"
	_ "zee-mirror/plugins/mega"
	_ "zee-mirror/plugins/telegram"
	"zee-mirror/plugins/torrent"
	_ "zee-mirror/plugins/ytdlp"
)

func main() {
	_ = godotenv.Load()
	cfg := config.LoadConfig()

	_ = os.MkdirAll(cfg.ConfigDir, 0750)
	setupLogger(cfg)
	initSentry(cfg)

	slog.Info("Starting Zee-Mirror Bot...")

	if !cfg.Validate() {
		slog.Error("Invalid configuration")
		os.Exit(1)
	}

	db, err := database.NewDB(cfg.DBDriver, cfg.ConfigDir, cfg.DatabaseURL, "migrations")
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	bots, err := service.InitBots(cfg.BotTokens, cfg.TelegramAPI)
	if err != nil || len(bots) == 0 {
		slog.Error("Failed to initialize any bot instances", "error", err)
		os.Exit(1)
	}

	primaryBot := bots[0].Bot
	slog.Info("Authorized on account", "username", primaryBot.Self.UserName, "totalBots", len(bots))

	ub := userbot.GetInstance(cfg)
	if err := ub.Start(); err != nil {
		slog.Warn("Userbot failed to start", "error", err)
	}

	aria2Daemon := torrent.NewAria2Daemon(cfg.ConfigDir)
	if err := aria2Daemon.Start(); err != nil {
		slog.Error("Failed to start aria2 daemon", "error", err)
	}
	defer aria2Daemon.Stop()

	redisClient := cache.NewRedisClient(cfg.RedisURL)
	defer redisClient.Close()

	botSvc := handlers.NewBotService(primaryBot, cfg, db, db.DB, redisClient)

	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	go func() {
		for range sighup {
			slog.Info("Received SIGHUP, reloading config...")
			newCfg := config.Reload()
			botSvc.TaskManager.Mu.Lock()
			botSvc.TaskManager.Config = newCfg
			botSvc.TaskManager.MaxConcurrent = newCfg.MaxConcurrentDownloads
			botSvc.TaskManager.StopDuplicate = newCfg.StopDuplicate
			botSvc.TaskManager.Mu.Unlock()
			slog.Info("TaskManager config updated")
		}
	}()

	apiServer := api.NewServer(botSvc, cfg.DashboardPort)

	r := router.NewRouter(botSvc.BotService)
	setupRoutes(r)

	r.RegisterMagnetHandler(func(_ *service.BotService, m *tgbotapi.Message) {
		text := m.Text
		if text == "" {
			text = m.Caption
		}
		botSvc.HandleTorrent(m, text)
	})

	apiServer.SetRouter(r)
	apiServer.Start()

	go func() {
		time.Sleep(2 * time.Second)
		recovery := service.NewTaskRecovery(db, botSvc.TaskManager, botSvc.BotService)
		if err := recovery.RecoverIncompleteTasks(); err != nil {
			slog.Warn("Failed to auto-recover tasks", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := os.Stat(cfg.DownloadDir); err == nil {
					size, _ := utils.CalculateDirSize(cfg.DownloadDir)
					metrics.StorageUsage.WithLabelValues(cfg.DownloadDir).Set(float64(size))
					slog.Debug("Storage usage metric updated", "size", size)
				}
			}
		}
	}()

	if cfg.UseWebhook && cfg.WebhookURL != "" {
		slog.Info("🌐 Starting in WEBHOOK mode", "webhook_url", cfg.WebhookURL)

		if err := apiServer.SetupWebhook(); err != nil {
			slog.Error("Failed to setup webhook, falling back to polling", "error", err)
			for _, bi := range bots {
				startPolling(ctx, bi.Bot, botSvc, r)
			}
		} else {
			slog.Info("✅ Webhook mode active. Waiting for updates from Telegram...")
			<-ctx.Done()
			slog.Info("Shutting down gracefully (webhook mode)...")
			if err := apiServer.RemoveWebhook(); err != nil {
				slog.Warn("Failed to remove webhook on shutdown", "error", err)
			}
		}
	} else {
		slog.Info("📡 Starting in LONG POLLING mode", "botCount", len(bots))
		_ = apiServer.RemoveWebhook()
		for _, bi := range bots {
			startPolling(ctx, bi.Bot, botSvc, r)
		}
	}

	slog.Info("Initiating global shutdown sequence...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		slog.Warn("API Server shutdown error", "error", err)
	}

	botSvc.Shutdown()
	ub.Stop()
	aria2Daemon.Stop()

	sentry.Flush(5 * time.Second)
	slog.Info("Zee-Mirror Bot has been gracefully shut down.")
}

func setupLogger(cfg *config.Config) {
	logPath := filepath.Join(cfg.ConfigDir, "zee-mirror.log")

	logFile := &lumberjack.Logger{
		Filename:   filepath.Clean(logPath),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     28,
		Compress:   true,
	}

	multi := io.MultiWriter(os.Stdout, logFile)

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

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.ToLower(cfg.LogFormat) == "json" {
		handler = slog.NewJSONHandler(multi, opts)
	} else {
		handler = slog.NewTextHandler(multi, opts)
	}

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

	slog.Info("Logging to file enabled", "format", cfg.LogFormat)
}

func initSentry(cfg *config.Config) {
	if cfg.SentryDSN == "" {
		return
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      getEnvOrDefault("APP_ENV", "production"),
		Release:          getEnvOrDefault("APP_RELEASE", "unknown"),
		TracesSampleRate: 0.2,
	}); err != nil {
		slog.Warn("Failed to initialize Sentry", "error", err)
		return
	}

	slog.Info("Sentry error tracking initialized")
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func startPolling(ctx context.Context, bot *tgbotapi.BotAPI, botSvc *handlers.BotService, r *router.Router) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	go func() {
		<-ctx.Done()
		slog.Info("Shutting down gracefully (polling mode)...")
		bot.StopReceivingUpdates()
	}()

	processUpdates(ctx, updates, botSvc, r)
}

func processUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, botSvc *handlers.BotService, r *router.Router) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("Update processor stopping...")
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			if update.Message != nil {
				isStart := update.Message.IsCommand() && update.Message.Command() == "start"

				if !botSvc.IsAuthorized(update.Message.From.ID) && !isStart {
					slog.Warn("Unauthorized access attempt", "userID", update.Message.From.ID, "username", update.Message.From.UserName, "text", update.Message.Text)

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, service.GetErrorMessage("ACCESS DENIED", "Anda belum terautentikasi untuk menggunakan bot ini.\nSilakan hubungi Owner untuk mendapatkan akses."))
					msg.ParseMode = tgbotapi.ModeMarkdownV2
					msg.ReplyToMessageID = update.Message.MessageID
					_, _ = botSvc.Bot.Send(msg)
					continue
				}
				go r.HandleMessage(update.Message)
			} else if update.CallbackQuery != nil {
				data := update.CallbackQuery.Data
				isHelp := strings.HasPrefix(data, "help:")

				if !botSvc.IsAuthorized(update.CallbackQuery.From.ID) && !isHelp {
					slog.Warn("Unauthorized callback attempt", "userID", update.CallbackQuery.From.ID, "username", update.CallbackQuery.From.UserName, "data", data)

					cb := tgbotapi.NewCallback(update.CallbackQuery.ID, "🚫 Access Denied")
					_, _ = botSvc.Bot.Request(cb)
					continue
				}
				go r.HandleCallback(update.CallbackQuery)
			}
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
	r.RegisterCommand("start", func(s *service.BotService, m *tgbotapi.Message) { basic.StartHandler(s, m) })
	r.RegisterCommand("help", func(s *service.BotService, m *tgbotapi.Message) { basic.HelpHandler(s, m) })
	r.RegisterCommand("settings", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleSettings(m)
	})
	r.RegisterCommand("ping", func(s *service.BotService, m *tgbotapi.Message) { basic.HandlePing(s, m) })
	r.RegisterCommand("speed", func(s *service.BotService, m *tgbotapi.Message) { basic.HandleSpeed(s, m) })
	r.RegisterCommand("stats", func(s *service.BotService, m *tgbotapi.Message) { basic.HandleStats(s, m) })
	r.RegisterCommand("lang", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleLanguage(m)
	})
	r.RegisterCommand("language", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleLanguage(m)
	})
	r.RegisterCommand("schedule", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleSchedule(m, m.CommandArguments())
	})
}

func setupDownloadRoutes(r *router.Router) {
	r.RegisterCommand("mirror", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleMirror(m, m.CommandArguments())
	})
	r.RegisterCommand("m", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleMirror(m, m.CommandArguments())
	})
	r.RegisterCommand("leech", func(s *service.BotService, m *tgbotapi.Message) { download.HandleLeech(s, m, m.CommandArguments()) })
	r.RegisterCommand("l", func(s *service.BotService, m *tgbotapi.Message) { download.HandleLeech(s, m, m.CommandArguments()) })
	r.RegisterCommand("viking", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleViking(m, m.CommandArguments())
	})
	r.RegisterCommand("v", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleViking(m, m.CommandArguments())
	})
	r.RegisterCommand("ytdlp", func(s *service.BotService, m *tgbotapi.Message) { download.HandleYTDLP(s, m, m.CommandArguments()) })
	r.RegisterCommand("y", func(s *service.BotService, m *tgbotapi.Message) { download.HandleYTDLP(s, m, m.CommandArguments()) })
	r.RegisterCommand("yt", func(s *service.BotService, m *tgbotapi.Message) { download.HandleYTDLP(s, m, m.CommandArguments()) })
	r.RegisterCommand("ytdlpleech", func(s *service.BotService, m *tgbotapi.Message) {
		download.HandleYTDLPLeech(s, m, m.CommandArguments())
	})
	r.RegisterCommand("yl", func(s *service.BotService, m *tgbotapi.Message) {
		download.HandleYTDLPLeech(s, m, m.CommandArguments())
	})
	r.RegisterCommand("torrent", func(s *service.BotService, m *tgbotapi.Message) { download.HandleTorrent(s, m, m.CommandArguments()) })
	r.RegisterCommand("t", func(s *service.BotService, m *tgbotapi.Message) { download.HandleTorrent(s, m, m.CommandArguments()) })
	r.RegisterCommand("clone", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleClone(m, m.CommandArguments())
	})
	r.RegisterCommand("cl", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleClone(m, m.CommandArguments())
	})
}

func setupAdminRoutes(r *router.Router) {
	r.RegisterCommand("status", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleStatus(m)
	})
	r.RegisterCommand("st", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleStatus(m)
	})
	r.RegisterCommand("cancel", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleCancel(m, m.CommandArguments())
	})
	r.RegisterCommand("c", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleCancel(m, m.CommandArguments())
	})
	r.RegisterCommand("cancelall", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleCancelAll(m)
	})
	r.RegisterCommand("search", func(s *service.BotService, m *tgbotapi.Message) { search.HandleSearch(s, m, m.CommandArguments()) })
	r.RegisterCommand("searchall", func(s *service.BotService, m *tgbotapi.Message) { search.HandleSearch(s, m, m.CommandArguments()) })
	r.RegisterCommand("batch", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleBatch(m, m.CommandArguments())
	})
	r.RegisterCommand("authorize", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleAuthorize(m, m.CommandArguments())
	})
	r.RegisterCommand("unauthorize", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleUnauthorize(m, m.CommandArguments())
	})
	r.RegisterCommand("removeuser", func(s *service.BotService, m *tgbotapi.Message) { admin.RemoveUserHandler(s, m, m.CommandArguments()) })
	r.RegisterCommand("setrole", func(s *service.BotService, m *tgbotapi.Message) { admin.SetRoleHandler(s, m, m.CommandArguments()) })
	r.RegisterCommand("setlimit", func(s *service.BotService, m *tgbotapi.Message) { admin.SetLimitHandler(s, m, m.CommandArguments()) })
	r.RegisterCommand("setexpire", func(s *service.BotService, m *tgbotapi.Message) { admin.SetExpireHandler(s, m, m.CommandArguments()) })
	r.RegisterCommand("users", func(s *service.BotService, m *tgbotapi.Message) { admin.HandleUserList(s, m) })
	r.RegisterCommand("setalertchannel", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleSetAlertChannel(m, m.CommandArguments())
	})
	r.RegisterCommand(handlers.CmdSystem, func(s *service.BotService, m *tgbotapi.Message) { basic.HandleSystem(s, m) })
	r.RegisterCommand(handlers.CmdHealth, func(s *service.BotService, m *tgbotapi.Message) { basic.HandleHealth(s, m) })
	r.RegisterCommand(handlers.CmdLogs, func(s *service.BotService, m *tgbotapi.Message) { basic.HandleLogs(s, m, m.CommandArguments()) })
	r.RegisterCommand("recover", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleRecover(m)
	})
	r.RegisterCommand("recoverystatus", func(s *service.BotService, m *tgbotapi.Message) {
		(&handlers.BotService{BotService: s}).HandleRecoveryStatus(m)
	})
	r.RegisterCommand("join", func(s *service.BotService, m *tgbotapi.Message) { basic.HandleJoin(s, m, m.CommandArguments()) })
}

func setupFileManagerRoutes(r *router.Router) {
	fmHandler := func(s *service.BotService, m *tgbotapi.Message) {
		cmd := m.Command()
		args := m.CommandArguments()
		switch cmd {
		case "ls", "dir":
			file.HandleDriveList(s, m, args, 0)
		case "mkdir":
			file.HandleDriveMkdir(s, m, args)
		case "rm":
			file.HandleDriveDelete(s, m, args)
		case "mv":
			file.HandleDriveMove(s, m, args)
		case "share":
			file.HandleDriveShare(s, m, args)
		case "find":
			file.HandleDriveSearch(s, m, args)
		}
	}
	r.RegisterCommand("ls", fmHandler)
	r.RegisterCommand("dir", fmHandler)
	r.RegisterCommand("mkdir", fmHandler)
	r.RegisterCommand("rm", fmHandler)
	r.RegisterCommand("mv", fmHandler)
	r.RegisterCommand("share", fmHandler)
	r.RegisterCommand("find", fmHandler)

	r.RegisterCommand("storages", func(s *service.BotService, m *tgbotapi.Message) { storage.HandleStorages(s, m) })
	r.RegisterCommand("setstorage", func(s *service.BotService, m *tgbotapi.Message) { storage.HandleSetStorage(s, m, m.CommandArguments()) })
}

func setupMediaRoutes(r *router.Router) {
	mediaHandler := func(s *service.BotService, m *tgbotapi.Message) {
		cmd := m.Command()
		args := m.CommandArguments()
		switch cmd {
		case "extractaudio":
			media.HandleExtractAudio(s, m, args)
		case "compress":
			media.HandleCompressVideo(s, m, args)
		case "thumbnail":
			media.HandleGenerateThumbnail(s, m, args)
		case "screenshots":
			media.HandleScreenshots(s, m, args)
		case "subtitle":
			media.HandleEmbedSubtitle(s, m, args)
		case "hardsub":
			media.HandleHardsub(s, m, args)
		case "rescale":
			media.HandleRescale(s, m, args)
		case "convert":
			media.HandleConvertFormat(s, m, args)
		case "mediainfo":
			media.HandleMediaInfo(s, m, args)
		}
	}
	r.RegisterCommand("extractaudio", mediaHandler)
	r.RegisterCommand("compress", mediaHandler)
	r.RegisterCommand("thumbnail", mediaHandler)
	r.RegisterCommand("screenshots", mediaHandler)
	r.RegisterCommand("subtitle", mediaHandler)
	r.RegisterCommand("hardsub", mediaHandler)
	r.RegisterCommand("rescale", mediaHandler)
	r.RegisterCommand("convert", mediaHandler)
	r.RegisterCommand("mediainfo", mediaHandler)
}

func setupCallbackRoutes(r *router.Router) {
	r.RegisterCallback("dashboard", func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "dashboard") {
			return
		}
		(&handlers.BotService{BotService: s}).HandleDashboardCallback(cb)
	})
	r.RegisterCallback("help", func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "help") {
			return
		}
		basic.HandleHelpCallback(s, cb, "")
	})
	r.RegisterCallback("refresh_status", func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "refresh_status") {
			return
		}
		(&handlers.BotService{BotService: s}).HandleRefreshStatusCallback(cb)
	})

	searchHandler := func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "search") {
			return
		}
		parts := strings.Split(cb.Data, ":")
		if parts[0] == "t_search" {
			search.HandleSearchCallback(s, cb, parts)
		} else {
			search.HandleSearchNavCallback(s, cb, parts)
		}
	}
	r.RegisterCallback("t_search", searchHandler)
	r.RegisterCallback("t_page", searchHandler)
	r.RegisterCallback("t_item", searchHandler)
	r.RegisterCallback("t_back", searchHandler)
	r.RegisterCallback("t_close", searchHandler)

	taskActionHandler := func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "task_action") {
			return
		}
		parts := strings.Split(cb.Data, ":")
		switch parts[0] {
		case "ytdlp_q":
			(&handlers.BotService{BotService: s}).HandleYTDLPQualityCallback(cb, parts)
		case "settings":
			(&handlers.BotService{BotService: s}).HandleSettingsCallback(cb, parts)
		case "batch":
			(&handlers.BotService{BotService: s}).HandleBatchCallback(cb, parts)
		case "confirm":
			(&handlers.BotService{BotService: s}).HandleConfirmCallback(cb, parts)
		case "torrent_sel":
			download.HandleTorrentSelectionCallback(s, cb, parts)
		}
	}
	r.RegisterCallback("ytdlp_q", taskActionHandler)
	r.RegisterCallback("settings", taskActionHandler)
	r.RegisterCallback("batch", taskActionHandler)
	r.RegisterCallback("confirm", taskActionHandler)
	r.RegisterCallback("torrent_sel", taskActionHandler)

	systemHandler := func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "system") {
			return
		}
		parts := strings.Split(cb.Data, ":")
		switch parts[0] {
		case "stats":
			basic.HandleStatsCallback(s, cb, parts)
		case handlers.CmdSystem:
			basic.HandleSystemCallback(s, cb, parts)
		case "storage":
			storage.HandleStorageCallback(s, cb, parts)
		case "drive", "dr":
			file.HandleDriveCallback(s, cb, parts)
		}
	}
	r.RegisterCallback("stats", systemHandler)
	r.RegisterCallback(handlers.CmdSystem, systemHandler)
	r.RegisterCallback("storage", systemHandler)
	r.RegisterCallback("drive", systemHandler)
	r.RegisterCallback("dr", systemHandler)

	r.RegisterCallback("media_m", func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "media_m") {
			return
		}
		parts := strings.Split(cb.Data, ":")
		media.HandleMediaMirrorCallback(s, cb, parts)
	})
	r.RegisterCallback("media", func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "media") {
			return
		}
		parts := strings.Split(cb.Data, ":")
		media.HandleMediaMenuCallback(s, cb, parts)
	})

	r.RegisterCallback("wizard", func(s *service.BotService, cb *tgbotapi.CallbackQuery) {
		if !ensureCallbackMessage(s, cb, "wizard") {
			return
		}
		parts := strings.Split(cb.Data, ":")
		(&handlers.BotService{BotService: s}).HandleMirrorWizardCallback(cb, parts)
	})
}

func ensureCallbackMessage(s *service.BotService, cb *tgbotapi.CallbackQuery, route string) bool {
	if cb == nil {
		slog.Warn("Ignoring nil callback", "route", route)
		return false
	}
	if cb.Message == nil {
		slog.Warn("Ignoring callback without message", "route", route, "data", cb.Data)
		if cb.ID != "" {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(cb.ID, "Unsupported callback context"))
		}
		return false
	}
	return true
}
