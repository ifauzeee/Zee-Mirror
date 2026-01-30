package handlers

import (
	"zee-mirror/internal/config"
	"zee-mirror/internal/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotService struct {
	Bot          *tgbotapi.BotAPI
	TaskManager  *TaskManager
	BatchManager *BatchManager
	Settings     *Settings
	Config       *config.Config
	DB           *database.DB
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

	_ = db.UpsertUser(cfg.OwnerID, "Owner", "owner")
	for _, id := range cfg.AuthorizedUsers {
		_ = db.UpsertUser(id, "Authorized User", "authorized")
	}

	go s.startDiskCleanupWorker()

	return s
}

func (s *BotService) Shutdown() {
	if s.TaskManager != nil {
		close(s.TaskManager.ShutdownChan)
		s.TaskManager.Wg.Wait()
	}
}
