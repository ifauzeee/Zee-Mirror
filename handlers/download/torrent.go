package download

import (
	"strings"

	"zee-mirror/internal/service"
	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleTorrent(s *service.BotService, message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.Reply(message, service.GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name, _, _ := utils.ParseFlags(args)
	if message.ReplyToMessage != nil && message.ReplyToMessage.Document != nil {
		fileID := message.ReplyToMessage.Document.FileID
		fileName := message.ReplyToMessage.Document.FileName

		if strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
			go s.HandleTelegramFileDownload(message, fileID, fileName, zip, unzip, password, quality)
			return
		}
	}

	if url == "" {
		url = utils.ExtractMagnetFromText(args)
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_magnet"))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}

	s.ShowTorrentSelectionMenu(message, url, name, zip, unzip, password, replyID)
}

func HandleTorrentSelectionCallback(s *service.BotService, callback *tgbotapi.CallbackQuery, parts []string) {
	s.HandleTorrentSelectionCallback(callback, parts)
}
