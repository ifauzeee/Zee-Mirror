package handlers

import (
	"database/sql"
	"zee-mirror/internal/config"
	"zee-mirror/internal/database"
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	CmdSystem = "system"
	CmdHealth = "health"
	CmdLogs   = "logs"
)

type BotService struct {
	*service.BotService
}

func NewBotService(bot *tgbotapi.BotAPI, cfg *config.Config, db *database.DB, sqlDB *sql.DB) *BotService {
	return &BotService{
		BotService: service.NewBotService(bot, cfg, db, sqlDB),
	}
}
