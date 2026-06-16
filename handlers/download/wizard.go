package download

import (
	"fmt"
	"strings"
	"zee-mirror/internal/service"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	PromptMirrorURL = "🔗 Silakan kirim *URL* yang ingin Anda mirror:"
	PromptRename    = "✏️ Silakan kirim *Nama Baru* untuk file ini:"
)

func HandleMirrorWizard(s *service.BotService, message *tgbotapi.Message) {
	text := service.ProfessionalMessage("MIRROR WIZARD",
		"Selamat datang di Mirror Wizard\\.\n"+
			"Silakan pilih metode input yang Anda inginkan\\.")

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, _ = s.Bot.Send(msg)
}

func HandleMirrorWizardCallback(s *service.BotService, callback *tgbotapi.CallbackQuery, parts []string) {
	action := parts[1]

	switch action {
	case "input_url":
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, PromptMirrorURL)
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
		_, _ = s.Bot.Send(msg)

		_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))

	case "back_main":
		HandleMirrorWizard(s, callback.Message)
		text := service.ProfessionalMessage("MIRROR WIZARD",
			"Selamat datang di Mirror Wizard\\.\n"+
				"Silakan pilih metode input yang Anda inginkan\\.")

		edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		edit.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(edit)

	case "toggle_zip":
		toggleWizardOption(s, callback, "zip")

	case "opt_rename":
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, PromptRename)
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}

		_, _ = s.Bot.Send(msg)

	case "start":
		executeWizardTask(s, callback)
	}
}

func HandleWizardInput(s *service.BotService, message *tgbotapi.Message) {
	replyText := message.ReplyToMessage.Text

	if strings.Contains(replyText, "Silakan kirim URL") {
		url := message.Text
		if !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "magnet") {
			s.Reply(message, "❌ URL tidak valid. Harap kirim link http/https atau magnet.")
			return
		}

		showWizardOptionsMenu(s, message.Chat.ID, url, false)
	}
}

func showWizardOptionsMenu(s *service.BotService, chatID int64, url string, isZip bool) {
	cleanURL := utils.EscapeMarkdownV2(url)

	text := fmt.Sprintf("⚙️ *Mirror Options*\n\n"+
		"🔗 *URL:* `%s`\n"+
		"📦 *Zip:* %s\n\n"+
		"Silakan atur opsi sebelum memulai\\.",
		cleanURL,
		utils.BoolToEmoji(isZip),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.DisableWebPagePreview = true
	_, _ = s.Bot.Send(msg)
}

func toggleWizardOption(s *service.BotService, callback *tgbotapi.CallbackQuery, option string) {
	text := callback.Message.Text
	lines := strings.Split(text, "\n")
	var url string
	var isZip bool

	for _, line := range lines {
		if idx := strings.Index(line, "URL:"); idx != -1 {
			url = strings.TrimSpace(line[idx+len("URL:"):])
		}
		if strings.Contains(line, "Zip:") {
			if strings.Contains(line, "✅") {
				isZip = true
			}
		}
	}

	if option == "zip" {
		isZip = !isZip
	}

	cleanURL := utils.EscapeMarkdownV2(url)
	newText := fmt.Sprintf("⚙️ *Mirror Options*\n\n"+
		"🔗 *URL:* `%s`\n"+
		"📦 *Zip:* %s\n\n"+
		"Silakan atur opsi sebelum memulai\\.",
		cleanURL,
		utils.BoolToEmoji(isZip),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, newText)
	edit.ParseMode = tgbotapi.ModeMarkdownV2
	edit.DisableWebPagePreview = true
	_, _ = s.Bot.Send(edit)
}

func executeWizardTask(s *service.BotService, callback *tgbotapi.CallbackQuery) {
	text := callback.Message.Text
	lines := strings.Split(text, "\n")
	var url string
	var isZip bool

	for _, line := range lines {
		if idx := strings.Index(line, "URL:"); idx != -1 {
			url = strings.TrimSpace(line[idx+len("URL:"):])
		}
		if strings.Contains(line, "Zip:") {
			if strings.Contains(line, "✅") {
				isZip = true
			}
		}
	}

	if url == "" {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Failed to parse URL"))
		return
	}

	fileName := utils.GetFileNameFromURL(url)
	task, err := s.TaskManager.CreateTask(service.TypeMirror, url, fileName, callback.Message.Chat.ID, callback.Message.MessageID, 0, callback.From.ID, isZip, false, "", "", 0, "", false)

	if err != nil {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Error starting task"))
		return
	}

	s.UpdateSharedDashboard(callback.Message.Chat.ID, true)
	s.HandleAutoDelete(task)

	_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🚀 Task Started"))
}
