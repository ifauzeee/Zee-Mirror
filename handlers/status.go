package handlers

import (
	"fmt"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleStatus(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	UpdateSharedDashboard(bot, message.Chat.ID, true)
}

func UpdateSharedDashboard(bot *tgbotapi.BotAPI, chatID int64, forceNew bool) {
	taskManager.StatusMu.Lock()
	defer taskManager.StatusMu.Unlock()

	tasks := taskManager.GetActiveTasksByChat(chatID)

	if len(tasks) == 0 {
		taskManager.Mu.Lock()
		lastMsgID, exists := taskManager.LastStatusMsg[chatID]
		if exists {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, lastMsgID))
			delete(taskManager.LastStatusMsg, chatID)
			delete(lastStatusText, chatID)
		}
		taskManager.Mu.Unlock()

		if forceNew {
			msg := tgbotapi.NewMessage(chatID, "📭 *Tidak ada task aktif*\n\nGunakan /mirror, /leech, /ytdlp, atau /torrent untuk memulai\\.")
			msg.ParseMode = MarkdownV2
			_, _ = bot.Send(msg)
		}
		return
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	text := StatusHeaderText

	for i, task := range tasks {
		snapshot := task.GetSnapshot()
		emoji := StatusEmoji(string(snapshot.Status))
		bar := ProgressBar(snapshot.Progress, 10)

		text += fmt.Sprintf(
			"━━━━━━━━━━━━━━━━━━━━━━━━\n"+
				"%s *ID:* `%s` \\| *%s\\.\\.\\.*\n"+
				"%s\n"+
				"📄 *File:* %s\n"+
				"📦 *Size:* %s\n"+
				"⚡ *Speed:* %s \\| *CN:* %d \\| ⏱️ *ETA:* %s\n"+
				"🚫 *Action:* /cancel\\_%s\n",
			emoji,
			snapshot.ID,
			EscapeMarkdownV2(FormatStatus(string(snapshot.Status))),
			EscapeMarkdownV2(bar),
			EscapeMarkdownV2(TruncateString(snapshot.FileName, 40)),
			EscapeMarkdownV2(FormatBytes(snapshot.TotalSize)),
			EscapeMarkdownV2(FormatSpeed(snapshot.Speed)),
			snapshot.Connections,
			EscapeMarkdownV2(FormatDuration(snapshot.ETA)),
			snapshot.ID,
		)
		if i == len(tasks)-1 {
			text += "━━━━━━━━━━━━━━━━━━━━━━━━\n"
		}
	}

	text += fmt.Sprintf("_Total: %d task aktif_", len(tasks))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "refresh_status"),
		),
	)

	taskManager.Mu.RLock()
	lastMsgID, exists := taskManager.LastStatusMsg[chatID]
	taskManager.Mu.RUnlock()

	lastText, textExists := lastStatusText[chatID]

	if exists && forceNew {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, lastMsgID))
		exists = false
		delete(lastStatusText, chatID)
	}

	if exists && !forceNew {
		if textExists && lastText == text {
			return
		}

		editMsg := tgbotapi.NewEditMessageText(chatID, lastMsgID, text)
		editMsg.ParseMode = MarkdownV2
		editMsg.ReplyMarkup = &keyboard
		if _, err := bot.Send(editMsg); err == nil {
			lastStatusText[chatID] = text
			return
		} else if strings.Contains(err.Error(), "message is not modified") {
			lastStatusText[chatID] = text
			return
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	sentMsg, err := bot.Send(msg)
	if err == nil {
		taskManager.Mu.Lock()
		taskManager.LastStatusMsg[chatID] = sentMsg.MessageID
		taskManager.Mu.Unlock()
		lastStatusText[chatID] = text
	}
}

var lastStatusText = make(map[int64]string)

func HandleCancel(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	if args == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nGunakan: `/cancel <TaskID>`\n\nLihat daftar task dengan /status")
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	taskID := args

	if taskManager.CancelTask(taskID) {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ *Task `%s` dibatalkan*", taskID))
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		UpdateSharedDashboard(bot, message.Chat.ID, false)
		return
	}

	task := taskManager.GetTaskByGID(taskID)
	if task != nil {
		if taskManager.CancelTask(task.ID) {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ *Task `%s` dibatalkan*", task.ID))
			msg.ParseMode = MarkdownV2
			_, _ = bot.Send(msg)
			return
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ *Task `%s` tidak ditemukan*", EscapeMarkdownV2(taskID)))
	msg.ParseMode = MarkdownV2
	_, _ = bot.Send(msg)
}

func HandleCancelCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, taskID string) {
	if taskManager.CancelTask(taskID) {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Task dibatalkan"))

		text := "🚫 *Task Dibatalkan*\n\nID: `%s`"
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, fmt.Sprintf(text, taskID))
		editMsg.ParseMode = MarkdownV2
		_, _ = bot.Send(editMsg)
		UpdateSharedDashboard(bot, callback.Message.Chat.ID, false)
	} else {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Gagal membatalkan task"))
	}
}

func HandleRefreshStatusCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Refreshed"))
	UpdateSharedDashboard(bot, callback.Message.Chat.ID, false)
}

func HandleConfirmCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	if len(parts) < 2 {
		return
	}

	action := parts[1]
	switch action {
	case "yes":
		text := "✅ *Dikonfirmasi*"
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = bot.Send(editMsg)

	case "no":
		text := "❌ *Dibatalkan*"
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = bot.Send(editMsg)
	}
}
