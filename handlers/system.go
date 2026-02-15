package handlers

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) HandlePing(message *tgbotapi.Message) {
	start := time.Now()

	msg := tgbotapi.NewMessage(message.Chat.ID, "🏓 *Pinging\\.\\.\\.*")
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	sentMsg, err := s.Bot.Send(msg)
	if err != nil {
		return
	}

	elapsed := time.Since(start)

	text := fmt.Sprintf("🏓 *Pong\\!* `%v`", elapsed.Round(time.Millisecond))
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentMsg.MessageID, text)
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) HandleSpeed(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🚀 *Running Speedtest\\.\\.\\.*")
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	sentMsg, err := s.Bot.Send(msg)
	if err != nil {
		return
	}

	go func() {
		cmd := exec.Command("speedtest-cli", "--simple")
		output, err := cmd.CombinedOutput()
		if err != nil {
			text := fmt.Sprintf("❌ *Speedtest Error*\n\n`%s`", utils.EscapeMarkdownV2Code(err.Error()))
			editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentMsg.MessageID, text)
			editMsg.ParseMode = tgbotapi.ModeMarkdownV2
			_, _ = s.Bot.Send(editMsg)
			return
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		var result strings.Builder
		result.WriteString("🚀 *Speedtest Result*\n\n")
		for _, line := range lines {
			result.WriteString(fmt.Sprintf("• `%s`\n", utils.EscapeMarkdownV2Code(line)))
		}

		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentMsg.MessageID, result.String())
		editMsg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(editMsg)
	}()
}
