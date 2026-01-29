package handlers

import (
	"zee-mirror/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotService struct {
	Bot          *tgbotapi.BotAPI
	TaskManager  *TaskManager
	BatchManager *BatchManager
	Settings     *Settings
	Config       *config.Config
}

func NewBotService(bot *tgbotapi.BotAPI, cfg *config.Config) *BotService {
	s := &BotService{
		Bot:          bot,
		Config:       cfg,
		Settings:     NewSettings(),
		BatchManager: NewBatchManager(),
	}

	processFunc := func(task *Task) {
		s.processTask(task)
	}

	refreshFunc := func(chatID int64, forceNew bool) {
		s.UpdateSharedDashboard(chatID, forceNew)
	}

	tm := NewTaskManager(bot, cfg.MaxConcurrentDownloads, cfg.DownloadDir, cfg.RcloneDest, cfg.ConfigDir, processFunc, refreshFunc)
	s.TaskManager = tm

	return s
}

func (s *BotService) Shutdown() {
	if s.TaskManager != nil {
		close(s.TaskManager.ShutdownChan)
		s.TaskManager.Wg.Wait()
	}
}
