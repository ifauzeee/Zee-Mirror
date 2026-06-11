package telegram

import (
	"zee-mirror/internal/config"
)

func init() {
}
type TelegramEngine struct {
	Config *config.Config
}

func NewTelegramEngine(cfg *config.Config) *TelegramEngine {
	return &TelegramEngine{Config: cfg}
}
