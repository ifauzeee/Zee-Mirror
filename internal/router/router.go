package router

import (
	"log/slog"
	"strings"

	"zee-mirror/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandHandler func(s *handlers.BotService, msg *tgbotapi.Message)
type CallbackHandler func(s *handlers.BotService, cb *tgbotapi.CallbackQuery)

type Router struct {
	service          *handlers.BotService
	commands         map[string]CommandHandler
	callbacks        map[string]CallbackHandler
	defaultCommand   CommandHandler
	callbackPrefixes map[string]CallbackHandler
}

func NewRouter(service *handlers.BotService) *Router {
	return &Router{
		service:          service,
		commands:         make(map[string]CommandHandler),
		callbacks:        make(map[string]CallbackHandler),
		callbackPrefixes: make(map[string]CallbackHandler),
	}
}

func (r *Router) RegisterCommand(name string, handler CommandHandler) {
	r.commands[name] = handler
}

func (r *Router) RegisterCallback(prefix string, handler CallbackHandler) {
	r.callbackPrefixes[prefix] = handler
}

func (r *Router) HandleMessage(msg *tgbotapi.Message) {
	if msg.IsCommand() {
		command := msg.Command()
		if handler, ok := r.commands[command]; ok {
			handler(r.service, msg)
			return
		}

		// Handle dynamic commands like cancel_
		if strings.HasPrefix(command, "cancel_") {
			taskID := strings.TrimPrefix(command, "cancel_")
			r.service.HandleCancel(msg, taskID)
			return
		}

		if r.defaultCommand != nil {
			r.defaultCommand(r.service, msg)
		}
		return
	}

	// Handle non-command messages (like magnet links)
	text := msg.Text
	if text == "" && msg.Caption != "" {
		text = msg.Caption
	}

	if text != "" {
		if strings.HasPrefix(text, "magnet:?") {
			r.service.HandleTorrent(msg, text)
			return
		}
	}
}

func (r *Router) HandleCallback(cb *tgbotapi.CallbackQuery) {
	parts := strings.Split(cb.Data, ":")
	if len(parts) == 0 {
		return
	}

	prefix := parts[0]

	// Handle dynamic prefixes like cancel_
	if strings.HasPrefix(prefix, "cancel_") {
		taskID := strings.TrimPrefix(prefix, "cancel_")
		r.service.HandleCancelCallback(cb, taskID)
		return
	}

	if handler, ok := r.callbackPrefixes[prefix]; ok {
		handler(r.service, cb)
		return
	}

	slog.Warn("Unknown callback received", "data", cb.Data)
}
