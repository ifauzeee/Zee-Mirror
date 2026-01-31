package handlers

import (
	"fmt"
	"log"
	"sync"
	"time"
	"zee-mirror/internal/config"
	"zee-mirror/internal/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type BotService struct {
	Bot           *tgbotapi.BotAPI
	TaskManager   *TaskManager
	BatchManager  *BatchManager
	Settings      *Settings
	Config        *config.Config
	DB            *database.DB
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

func NewBotService(bot *tgbotapi.BotAPI, cfg *config.Config, db *database.DB) *BotService {
	s := &BotService{
		Bot:          bot,
		Config:       cfg,
		DB:           db,
		Settings:     NewSettings(),
		BatchManager: NewBatchManager(),
	}

	if val, err := db.GetSetting("auto_delete_messages"); err == nil {
		s.Settings.AutoDeleteMessages = (val == "true")
	}
	if val, err := db.GetSetting("default_mode"); err == nil {
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
	if alertCh, err := db.GetSetting("alert_channel_id"); err == nil {
		_, _ = fmt.Sscanf(alertCh, "%d", &alertChannelID)
	}
	s.Notifications = NewNotificationService(bot, alertChannelID, cfg.OwnerID)

	_ = db.UpsertUser(cfg.OwnerID, "Owner", "owner")
	for _, id := range cfg.AuthorizedUsers {
		_ = db.UpsertUser(id, "Authorized User", "authorized")
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

	s.Settings.mu.RLock()
	active := s.Settings.AutoDeleteMessages
	s.Settings.mu.RUnlock()

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

	s.Settings.mu.RLock()
	active := s.Settings.AutoDeleteMessages
	s.Settings.mu.RUnlock()

	if !active {
		return
	}

	if task.MessageID != 0 {
		go func() {
			deleteCmd := tgbotapi.NewDeleteMessage(task.ChatID, task.MessageID)
			_, _ = s.Bot.Request(deleteCmd)
		}()
	}

	// If task is completed/failed/cancelled, delete the result message after 60s
	task.Mu.RLock()
	status := task.Status
	resultMsgID := task.ResultMessageID
	task.Mu.RUnlock()

	if (status == StatusCompleted || status == StatusFailed || status == StatusCancelled) && resultMsgID != 0 {
		go func() {
			log.Printf("[AutoDelete] Deleting result message %d for task %s in 60s", resultMsgID, task.ID)
			time.Sleep(60 * time.Second)
			deleteCmd := tgbotapi.NewDeleteMessage(task.ChatID, resultMsgID)
			_, _ = s.Bot.Request(deleteCmd)
		}()
	}
}
