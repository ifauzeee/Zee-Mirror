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
	activeBatches := getActiveBatchesByChat(chatID)

	if len(tasks) == 0 && len(activeBatches) == 0 {
		handleEmptyTasks(bot, chatID, forceNew)
		return
	}

	sortedTasks := sortTasksByCreationTime(tasks)
	page := getCurrentPage(chatID)
	processedTasks := processPagination(sortedTasks, &page, chatID)

	text := buildDashboardStatusText(processedTasks, activeBatches, page, len(sortedTasks))
	keyboard := buildNavigationKeyboard(page, calculateTotalPages(len(sortedTasks)))

	sendStatusMessage(bot, chatID, text, keyboard, forceNew)
}

func handleEmptyTasks(bot *tgbotapi.BotAPI, chatID int64, forceNew bool) {
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
}

func sortTasksByCreationTime(tasks []*Task) []*Task {
	sortedTasks := make([]*Task, len(tasks))
	copy(sortedTasks, tasks)
	sort.Slice(sortedTasks, func(i, j int) bool {
		return sortedTasks[i].CreatedAt.Before(sortedTasks[j].CreatedAt)
	})
	return sortedTasks
}

func getCurrentPage(chatID int64) int {
	taskManager.Mu.RLock()
	page := taskManager.StatusPages[chatID]
	taskManager.Mu.RUnlock()
	return page
}

func processPagination(tasks []*Task, page *int, chatID int64) []*Task {
	perPage := 5
	totalTasks := len(tasks)
	totalPages := (totalTasks + perPage - 1) / perPage

	if *page >= totalPages && totalPages > 0 {
		*page = totalPages - 1
		taskManager.Mu.Lock()
		taskManager.StatusPages[chatID] = *page
		taskManager.Mu.Unlock()
	}

	start := *page * perPage
	end := start + perPage
	if end > totalTasks {
		end = totalTasks
	}

	return tasks[start:end]
}

func calculateTotalPages(totalTasks int) int {
	perPage := 5
	return (totalTasks + perPage - 1) / perPage
}

func buildDashboardStatusText(visibleTasks []*Task, batches []*BatchTask, page, totalTasks int) string {
	text := StatusHeaderText

	for _, batch := range batches {
		batch.Mu.RLock()
		text += fmt.Sprintf("📦 *Batch:* %s \\| 📊 %d/%d selesai\n\n",
			EscapeMarkdownV2(batch.Name),
			batch.Completed,
			len(batch.URLs),
		)

		visibleBatchTasks := 5
		if len(batch.SubTasks) < visibleBatchTasks {
			visibleBatchTasks = len(batch.SubTasks)
		}

		for i := 0; i < visibleBatchTasks; i++ {
			task := batch.SubTasks[i]
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
				EscapeMarkdownV2(snapshot.ID),
			)
		}

		if len(batch.SubTasks) > visibleBatchTasks {
			text += fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━━━━\n_\\.\\.\\. dan %d task lainnya_\n", len(batch.SubTasks)-visibleBatchTasks)
		}

		if len(batch.SubTasks) > 0 {
			text += "━━━━━━━━━━━━━━━━━━━━━━━━\n"
		}
		text += fmt.Sprintf("_Total: %d task dalam batch `%s`_\n\n", len(batch.SubTasks), batch.ID)
		batch.Mu.RUnlock()
	}

	totalPages := calculateTotalPages(totalTasks)
	if totalPages > 1 {
		text += fmt.Sprintf("📄 *Halaman:* %d/%d\n\n", page+1, totalPages)
	}

	for i, task := range visibleTasks {
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
			EscapeMarkdownV2(snapshot.ID),
		)
		if i == len(visibleTasks)-1 {
			text += "━━━━━━━━━━━━━━━━━━━━━━━━\n"
		}
	}

	if len(batches) > 0 {
		text += fmt.Sprintf("_Active: %d batches, %d regular tasks_", len(batches), len(visibleTasks))
	} else {
		text += fmt.Sprintf("_Total: %d task aktif_", len(visibleTasks))
	}

	return text
}

func buildNavigationKeyboard(page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	navRow := tgbotapi.NewInlineKeyboardRow()

	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Prev", fmt.Sprintf("dashboard:page:%d", page-1)))
	}
	navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "refresh_status"))
	if page < totalPages-1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("dashboard:page:%d", page+1)))
	}
	rows = append(rows, navRow)

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func sendStatusMessage(bot *tgbotapi.BotAPI, chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup, forceNew bool) {
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

	bm := GetBatchManager()
	bm.Mu.RLock()
	foundBatch := false
	var targetBatch *BatchTask
	for _, batch := range bm.Batches {
		if batch.ID == taskID {
			targetBatch = batch
			foundBatch = true
			break
		}
	}
	bm.Mu.RUnlock()

	if foundBatch {
		targetBatch.CancelFunc()
		targetBatch.SetStatus(StatusCancelled)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ *Batch `%s` dibatalkan*", taskID))
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		UpdateSharedDashboard(bot, message.Chat.ID, false)
		return
	}

	if checkBatchSubTaskCancellation(taskID) {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ *Sub-Task `%s` dibatalkan*", taskID))
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		UpdateSharedDashboard(bot, message.Chat.ID, false)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ *Task/Batch `%s` tidak ditemukan*", EscapeMarkdownV2(taskID)))
	msg.ParseMode = MarkdownV2
	_, _ = bot.Send(msg)
}

func HandleCancelCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, taskID string) {
	success := false

	if taskManager.CancelTask(taskID) {
		success = true
	} else {
		bm := GetBatchManager()
		bm.Mu.RLock()
		if batch, ok := bm.Batches[taskID]; ok {
			batch.CancelFunc()
			batch.SetStatus(StatusCancelled)
			success = true
		}
		bm.Mu.RUnlock()

		if !success {
			if checkBatchSubTaskCancellation(taskID) {
				success = true
			}
		}
	}

	if success {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Dibatalkan"))
		UpdateSharedDashboard(bot, callback.Message.Chat.ID, false)
	} else {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Gagal membatalkan / Id tidak ditemukan"))
	}
}

func checkBatchSubTaskCancellation(taskID string) bool {
	bm := GetBatchManager()
	bm.Mu.RLock()
	defer bm.Mu.RUnlock()

	for _, batch := range bm.Batches {
		for _, sub := range batch.SubTasks {
			if sub.ID == taskID {
				sub.CancelFunc()
				sub.SetStatus(StatusCancelled)
				return true
			}
		}
	}
	return false
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

func getActiveBatchesByChat(chatID int64) []*BatchTask {
	bm := GetBatchManager()
	bm.Mu.RLock()
	defer bm.Mu.RUnlock()

	var active []*BatchTask
	for _, batch := range bm.Batches {
		if batch.ChatID == chatID &&
			batch.Status != StatusCompleted &&
			batch.Status != StatusFailed &&
			batch.Status != StatusCancelled {
			active = append(active, batch)
		}
	}
	return active
}
