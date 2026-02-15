package handlers

import (
	"strings"

	"log/slog"
	"zee-mirror/handlers/admin"
	"zee-mirror/handlers/basic"
	"zee-mirror/handlers/download"
	"zee-mirror/handlers/storage"
	"zee-mirror/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Dispatcher struct {
	Service *service.BotService
}

func NewDispatcher(s *service.BotService) *Dispatcher {
	return &Dispatcher{Service: s}
}

func (d *Dispatcher) HandleMessage(message *tgbotapi.Message) {
	if message.From != nil {
		d.Service.SyncUser(message.From)
	}

	if message.Text == "" && message.Caption == "" && message.Document == nil && message.Video == nil && message.Audio == nil && message.Photo == nil {
		return
	}

	text := message.Text
	if text == "" {
		text = message.Caption
	}

	if strings.HasPrefix(text, "magnet:?") {
		d.Service.AutoDeleteCommandAndReply(message)
		download.HandleTorrent(d.Service, message, text)
		return
	}

	if strings.HasPrefix(text, "/") {
		cmd, args := parseCommand(text)

		if strings.HasPrefix(cmd, "cancel_") {
			taskID := strings.TrimPrefix(cmd, "cancel_")
			d.Service.HandleCancel(message, taskID)
			return
		}

		switch cmd {
		case "start":
			basic.StartHandler(d.Service, message)
		case "help":
			basic.HelpHandler(d.Service, message)
		case "ping":
			basic.HandlePing(d.Service, message)
		case "speed", "speedtest":
			basic.HandleSpeed(d.Service, message)
		case "stats":
			basic.HandleStats(d.Service, message)
		case "system":
			basic.HandleSystem(d.Service, message)
		case "health":
			basic.HandleHealth(d.Service, message)
		case "logs":
			basic.HandleLogs(d.Service, message, args)
		case "join":
			basic.HandleJoin(d.Service, message, args)

		case "mirror", "m":
			download.HandleMirror(d.Service, message, args)
		case "leech", "l":
			download.HandleLeech(d.Service, message, args)
		case "ytdlp", "y", "watch":
			download.HandleYTDLP(d.Service, message, args)
		case "yleech", "yl":
			download.HandleYTDLPLeech(d.Service, message, args)
		case "torrent", "magnet":
			download.HandleTorrent(d.Service, message, args)

		case "auth", "a":
			admin.HandleAuth(d.Service, message, args)
		case "unauth", "u":
			admin.HandleUnauth(d.Service, message, args)
		case "ban":
			slog.Warn("HandleBan not implemented")
		case "unban":
			slog.Warn("HandleUnban not implemented")
		case "users":
			admin.HandleUserList(d.Service, message)
		case "broadcast", "bc":
			slog.Warn("HandleBroadcast not implemented")
		case "shell", "sh":
			slog.Warn("HandleShell not implemented")
		case "exec":
			slog.Warn("HandleExec not implemented")
		case "restart":
			slog.Warn("HandleRestart not implemented")

		case "storage", "drives":
			storage.HandleStorages(d.Service, message)
		case "setstorage":
			storage.HandleSetStorage(d.Service, message, args)

		}
	} else {
		if message.Document != nil || message.Video != nil || message.Audio != nil || message.Photo != nil {
		}
	}
}

func (d *Dispatcher) HandleCallback(callback *tgbotapi.CallbackQuery) {
	if callback.From != nil {
		d.Service.SyncUser(callback.From)
	}

	data := callback.Data

	if strings.HasPrefix(data, "cancel_") {
		taskID := strings.TrimPrefix(data, "cancel_")
		d.Service.HandleCancelCallback(callback, taskID)
		return
	}

	parts := strings.Split(data, ":")
	if len(parts) == 0 {
		return
	}

	prefix := parts[0]

	switch prefix {
	case "help":
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}
		basic.HandleHelpCallback(d.Service, callback, action)
	case "stats":
		basic.HandleStatsCallback(d.Service, callback, parts)
	case "storage":
		storage.HandleStorageCallback(d.Service, callback, parts)
	case "dashboard":
		if len(parts) > 1 && parts[1] == "ping" {
			basic.HandlePingFromCallback(d.Service, callback)
		} else if len(parts) > 1 && parts[1] == "speed" {
			basic.HandleSpeedFromCallback(d.Service, callback)
		}
	case "torrent_sel":
		download.HandleTorrentSelectionCallback(d.Service, callback, parts)
	case "cancel":
	}
}

func parseCommand(text string) (string, string) {
	parts := strings.SplitN(text, " ", 2)
	cmd := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
	if idx := strings.Index(cmd, "@"); idx != -1 {
		cmd = cmd[:idx]
	}
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}
	return cmd, args
}
