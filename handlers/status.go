package handlers

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleStatus(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	tasks := taskManager.GetActiveTasks()

	if len(tasks) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📭 *Tidak ada task aktif*\n\nGunakan /mirror, /leech, /ytdlp, atau /torrent untuk memulai.")
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	text := StatusHeaderText

	for _, task := range tasks {
		snapshot := task.GetSnapshot()
		emoji := StatusEmoji(string(snapshot.Status))
		bar := ProgressBar(snapshot.Progress, 10)

		text += fmt.Sprintf(
			"%s *ID:* `%s`\n"+
				"📄 %s\n"+
				"%s\n"+
				"⚡ %s | ⏱ %s\n\n",
			emoji,
			snapshot.ID,
			EscapeMarkdownV2(TruncateString(snapshot.FileName, 40)),
			bar,
			EscapeMarkdownV2(FormatSpeed(snapshot.Speed)),
			EscapeMarkdownV2(FormatDuration(snapshot.ETA)),
		)
	}

	text += fmt.Sprintf("_Total: %d task aktif_", len(tasks))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "refresh_status"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, _ = bot.Send(msg)
}

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
	} else {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Gagal membatalkan task"))
	}
}

func HandleRefreshStatusCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	tasks := taskManager.GetActiveTasks()

	if len(tasks) == 0 {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "Tidak ada task aktif"))

		text := "📭 *Tidak ada task aktif*"
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = bot.Send(editMsg)
		return
	}

	_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Refreshed"))

	text := StatusHeaderText

	for _, task := range tasks {
		snapshot := task.GetSnapshot()
		emoji := StatusEmoji(string(snapshot.Status))
		bar := ProgressBar(snapshot.Progress, 10)

		text += fmt.Sprintf(
			"%s *ID:* `%s`\n"+
				"📄 %s\n"+
				"%s\n"+
				"⚡ %s \\| ⏱ %s\n\n",
			emoji,
			snapshot.ID,
			EscapeMarkdownV2(TruncateString(snapshot.FileName, 40)),
			bar,
			EscapeMarkdownV2(FormatSpeed(snapshot.Speed)),
			EscapeMarkdownV2(FormatDuration(snapshot.ETA)),
		)
	}

	text += fmt.Sprintf("_Total: %d task aktif_", len(tasks))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "refresh_status"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = bot.Send(editMsg)
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
