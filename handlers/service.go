package handlers

import (
	"fmt"
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
	PathCache     sync.Map // For storing short-lived local paths for callbacks
}

func (s *BotService) StorePath(path string) string {
	id := uuid.New().String()[:8]
	s.PathCache.Store(id, path)
	// Optionally: set a timer to delete it after 1 hour
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
