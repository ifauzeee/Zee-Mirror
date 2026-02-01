package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"
	"zee-mirror/internal/config"
	"zee-mirror/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type BotService struct {
	Bot           *tgbotapi.BotAPI
	TaskManager   *TaskManager
	BatchManager  *BatchManager
	Settings      *Settings
	Config        *config.Config
	DB            repository.FullRepository
	UserRepo      repository.UserRepository
	SettingsRepo  repository.SettingsRepository
	Notifications *NotificationService
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
	s.TaskManager = tm

	var alertChannelID int64
	if alertCh, err := db.Get(ctx, "alert_channel_id"); err == nil {
		_, _ = fmt.Sscanf(alertCh, "%d", &alertChannelID)
	}
	s.Notifications = NewNotificationService(bot, alertChannelID, cfg.OwnerID)

	_ = db.Upsert(ctx, cfg.OwnerID, "Owner", "owner")
	for _, id := range cfg.AuthorizedUsers {
		_ = db.Upsert(ctx, id, "Authorized User", "authorized")
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

	// User's command message is deleted immediately
	if task.MessageID != 0 {
		go func() {
			deleteCmd := tgbotapi.NewDeleteMessage(task.ChatID, task.MessageID)
			_, _ = s.Bot.Request(deleteCmd)
		}()
	}

	// User's reply message (e.g. file being mirrored) is also deleted
	if task.ReplyMessageID != 0 {
		go func() {
			deleteCmd := tgbotapi.NewDeleteMessage(task.ChatID, task.ReplyMessageID)
			_, _ = s.Bot.Request(deleteCmd)
		}()
	}
}

func (s *BotService) AutoDeleteCommandAndReply(message *tgbotapi.Message) {
	s.AutoDeleteMessage(message.Chat.ID, message.MessageID, 0)
	if message.ReplyToMessage != nil {
		s.AutoDeleteMessage(message.Chat.ID, message.ReplyToMessage.MessageID, 0)
	}
}
