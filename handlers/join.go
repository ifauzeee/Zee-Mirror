package handlers

import (
	"fmt"
	"strings"
	"time"
	"zee-mirror/internal/userbot"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) HandleJoin(m *tgbotapi.Message, args string) {
	if args == "" {
		reply := tgbotapi.NewMessage(m.Chat.ID, "⚠️ *Format Salah*\n\nGunakan: `/join <link invite>`\nContoh: `/join https://t.me/+FnboT0dqFiozNzA1`")
		reply.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(reply)
		return
	}

	var hash string
	switch {
	case strings.Contains(args, "+"):
		parts := strings.Split(args, "+")
		if len(parts) > 1 {
			hash = parts[1]
		}
	case strings.Contains(args, "joinchat/"):
		parts := strings.Split(args, "joinchat/")
		if len(parts) > 1 {
			hash = parts[1]
		}
	default:
		hash = args
	}

	hash = strings.TrimSpace(hash)
	hash = strings.Split(hash, "?")[0]
	hash = strings.Split(hash, "/")[0]

	if hash == "" {
		msg := tgbotapi.NewMessage(m.Chat.ID, "❌ *Error:* Could not extract invite hash from link")
		msg.ParseMode = MarkdownV2
		msg.ReplyToMessageID = m.MessageID
		s.AutoDeleteMessage(m.Chat.ID, m.MessageID, 30*time.Second)
		_, _ = s.Bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(m.Chat.ID, fmt.Sprintf("🔄 *Mencoba bergabung ke channel\\.\\.\\.*\nHash: `%s`", hash))
	msg.ParseMode = MarkdownV2
	sentMsg, err := s.Bot.Send(msg)
	if err != nil {
		msg = tgbotapi.NewMessage(m.Chat.ID, fmt.Sprintf("🔄 Mencoba bergabung ke channel...\nHash: %s", hash))
		sentMsg, _ = s.Bot.Send(msg)
	}

	ub := userbot.GetInstance(s.Config)
	result, err := ub.JoinChat(hash)

	text := ""
	if err != nil {
		if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			text = "✅ *Sudah Bergabung\\!* Bot sudah ada di dalam channel tersebut\\."
		} else {
			text = fmt.Sprintf("❌ *Gagal Bergabung*\nError: `%s`", utils.EscapeMarkdownV2(err.Error()))
		}
	} else {
		if result == "Already joined" {
			text = "✅ *Sudah Bergabung\\!* Bot sudah ada di dalam channel tersebut\\."
		} else {
			text = fmt.Sprintf("✅ *Berhasil\\!* %s", utils.EscapeMarkdownV2(result))
		}
	}

	if sentMsg.MessageID != 0 {
		edit := tgbotapi.NewEditMessageText(m.Chat.ID, sentMsg.MessageID, text)
		edit.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(edit)
	} else {
		finalMsg := tgbotapi.NewMessage(m.Chat.ID, text)
		finalMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(finalMsg)
	}
}
