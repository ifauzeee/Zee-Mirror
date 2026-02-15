package media

import (
	"zee-mirror/handlers/basic"
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleMediaMenuCallback(s *service.BotService, cb *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(cb.ID, "Unknown media action"))
		return
	}

	actionMap := map[string]string{
		"extract":  "cmd_extractaudio",
		"compress": "cmd_compress",
		"thumb":    "cmd_thumbnail",
		"screens":  "cmd_screenshots",
		"subtitle": "cmd_subtitle",
		"hardsub":  "cmd_hardsub",
		"rescale":  "cmd_rescale",
		"convert":  "cmd_convert",
		"info":     "cmd_mediainfo",
	}

	helpAction, ok := actionMap[parts[1]]
	if !ok {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(cb.ID, "Unknown media action"))
		return
	}

	basic.HandleHelpCallback(s, cb, helpAction)
}
