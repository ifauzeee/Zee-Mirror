package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"zee-mirror/internal/config"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/repository"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type BotService struct {
	Bot           *tgbotapi.BotAPI
	TaskManager   *TaskManager
	BatchManager  *BatchManager
	Settings      *Settings
	Config        *config.Config
	Notifications *NotificationService
	DB            repository.FullRepository
	UserRepo      repository.UserRepository
	SettingsRepo  repository.SettingsRepository
	PathCache     sync.Map
}

func (s *BotService) StorePath(path string) string {
	id := uuid.New().String()[:8]
	s.PathCache.Store(id, path)
	go func() {
		time.Sleep(1 * time.Hour)
		s.PathCache.Delete(id)
	}()
	return id
}

func (s *BotService) GetPath(id string) (string, bool) {
	val, ok := s.PathCache.Load(id)
	if !ok {
		return "", false
	}
	return val.(string), true
}

func NewBotService(bot *tgbotapi.BotAPI, cfg *config.Config, db repository.FullRepository) *BotService {
	s := &BotService{
		Bot:          bot,
		Config:       cfg,
		DB:           db,
		UserRepo:     db,
		SettingsRepo: db,
		Settings:     NewSettings(),
		BatchManager: NewBatchManager(),
	}

	ctx := context.Background()
	if val, err := db.Get(ctx, "auto_delete_messages"); err == nil {
		s.Settings.AutoDeleteMessages = (val == "true")
	}
	if val, err := db.Get(ctx, "default_mode"); err == nil {
		s.Settings.DefaultMode = val
	}

	processFunc := func(task *Task) {
		s.processTask(task)
	}

	refreshFunc := func(chatID int64, forceNew bool) {
		s.UpdateSharedDashboard(chatID, forceNew)
	}

	tm := NewTaskManager(bot, cfg.MaxConcurrentDownloads, cfg.DownloadDir, cfg.RcloneDest, cfg.ConfigDir, processFunc, refreshFunc, db)
	tm.StopDuplicate = cfg.StopDuplicate
	s.TaskManager = tm

	var alertChannelID int64
	if alertCh, err := db.Get(ctx, "alert_channel_id"); err == nil {
		_, _ = fmt.Sscanf(alertCh, "%d", &alertChannelID)
	}
	s.Notifications = NewNotificationService(bot, alertChannelID, cfg.OwnerID)

	_ = db.Upsert(ctx, domain.User{
		ID:                cfg.OwnerID,
		Username:          "Owner",
		Role:              "owner",
		CreatedAt:         time.Now(),
		MaxDailyTasks:     -1,
		MaxDailyBandwidth: -1,
	})

	for _, id := range cfg.AuthorizedUsers {
		_ = db.Upsert(ctx, domain.User{
			ID:                id,
			Username:          "Authorized User",
			Role:              "authorized",
			CreatedAt:         time.Now(),
			MaxDailyTasks:     cfg.DefaultMaxDailyTasks,
			MaxDailyBandwidth: cfg.DefaultMaxDailyBandwidth,
		})
	}

	go s.startDiskCleanupWorker()
	s.StartResourceMonitor()

	return s
}

func (s *BotService) Shutdown() {
	if s.TaskManager != nil {
		close(s.TaskManager.ShutdownChan)
		s.TaskManager.Wg.Wait()
	}
}

func (s *BotService) AutoDeleteMessage(chatID int64, messageID int, delay time.Duration) {
	if s.Settings == nil {
		return
	}

	s.Settings.Mu.RLock()
	active := s.Settings.AutoDeleteMessages
	s.Settings.Mu.RUnlock()

	if !active || messageID == 0 {
		return
	}

	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		deleteCmd := tgbotapi.NewDeleteMessage(chatID, messageID)
		_, _ = s.Bot.Request(deleteCmd)
	}()
}

func (s *BotService) handleAutoDelete(task *Task) {
	if s.Settings == nil {
		return
	}

	s.Settings.Mu.RLock()
	active := s.Settings.AutoDeleteMessages
	s.Settings.Mu.RUnlock()

	if !active {
		return
	}

	if task.MessageID != 0 {
		go func() {
			deleteCmd := tgbotapi.NewDeleteMessage(task.ChatID, task.MessageID)
			_, _ = s.Bot.Request(deleteCmd)
		}()
	}

	if task.ReplyMessageID != 0 {
		go func() {
			deleteCmd := tgbotapi.NewDeleteMessage(task.ChatID, task.ReplyMessageID)
			_, _ = s.Bot.Request(deleteCmd)
		}()
	}
}

func (s *BotService) AutoDeleteCommandAndReply(message *tgbotapi.Message) {
	s.AutoDeleteMessage(message.Chat.ID, message.MessageID, 0)
	if message.ReplyToMessage != nil && !IsMediaMessage(message.ReplyToMessage) {
		s.AutoDeleteMessage(message.Chat.ID, message.ReplyToMessage.MessageID, 0)
	}
}
func (s *BotService) handleCreateTaskError(chatID int64, messageID int, err error) {
	slog.Info("handleCreateTaskError called", "chatID", chatID, "error", err)

	text := ""
	var keyboard *tgbotapi.InlineKeyboardMarkup

	if dupErr, ok := err.(*DuplicateTaskError); ok {
		slog.Info("Duplicate detected", "msg", dupErr.Message, "remoteURL", dupErr.RemoteURL)
		text = fmt.Sprintf("⚠️ *Download Dibatalkan*\n\n%s", utils.EscapeMarkdownV2(dupErr.Message))

		if dupErr.RemoteURL != "" && strings.HasPrefix(dupErr.RemoteURL, "http") {
			kb := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(BtnTextCloudLink, dupErr.RemoteURL),
				),
			)
			keyboard = &kb
		}
	} else {
		text = fmt.Sprintf("❌ *Gagal membuat task:*\n%s", utils.EscapeMarkdownV2(err.Error()))
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyToMessageID = messageID
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	_, sendErr := s.Bot.Send(msg)
	if sendErr != nil {
		slog.Warn("Failed to send error with reply, trying without reply", "error", sendErr)
		msg.ReplyToMessageID = 0
		_, sendErr = s.Bot.Send(msg)
		if sendErr != nil {
			slog.Error("Failed to send error message completely", "error", sendErr)
			return
		}
	}
}
