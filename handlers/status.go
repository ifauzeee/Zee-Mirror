package handlers

import (
	"fmt"
	"sort"
	"strings"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var lastStatusText = make(map[int64]string)

func (s *BotService) HandleStatus(message *tgbotapi.Message) {
	s.UpdateSharedDashboard(message.Chat.ID, true)
}

func (s *BotService) UpdateSharedDashboard(chatID int64, forceNew bool) {
	s.TaskManager.StatusMu.Lock()
	defer s.TaskManager.StatusMu.Unlock()

	tasks := s.TaskManager.GetActiveTasksByChat(chatID)
	activeBatches := s.getActiveBatchesByChat(chatID)

	if len(tasks) == 0 && len(activeBatches) == 0 {
		s.handleEmptyTasks(chatID, forceNew)
		return
	}

	sortedTasks := sortTasksByCreationTime(tasks)
	page := s.getCurrentPage(chatID)
	processedTasks := s.processPagination(sortedTasks, &page, chatID)

	text := s.buildDashboardStatusText(processedTasks, activeBatches, page, len(sortedTasks))
	keyboard := buildNavigationKeyboard(page, calculateTotalPages(len(sortedTasks)))

	s.sendStatusMessage(chatID, text, keyboard, forceNew)
}

func (s *BotService) handleEmptyTasks(chatID int64, forceNew bool) {
	s.TaskManager.Mu.Lock()
	lastMsgID, exists := s.TaskManager.LastStatusMsg[chatID]
	if exists {
		_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(chatID, lastMsgID))
		delete(s.TaskManager.LastStatusMsg, chatID)
		delete(lastStatusText, chatID)
	}
	s.TaskManager.Mu.Unlock()

	if forceNew {
		msg := tgbotapi.NewMessage(chatID, "📭 *Tidak ada task aktif*\n\nGunakan /mirror, /leech, /ytdlp, /torrent, atau /clone untuk memulai\\.")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
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

func (s *BotService) getCurrentPage(chatID int64) int {
	s.TaskManager.Mu.RLock()
	page := s.TaskManager.StatusPages[chatID]
	s.TaskManager.Mu.RUnlock()
	return page
}

func (s *BotService) processPagination(tasks []*Task, page *int, chatID int64) []*Task {
	perPage := 5
	totalTasks := len(tasks)
	totalPages := (totalTasks + perPage - 1) / perPage

	if *page >= totalPages && totalPages > 0 {
		*page = totalPages - 1
		s.TaskManager.Mu.Lock()
		s.TaskManager.StatusPages[chatID] = *page
		s.TaskManager.Mu.Unlock()
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

func (s *BotService) buildDashboardStatusText(visibleTasks []*Task, batches []*BatchTask, page, totalTasks int) string {
	text := StatusHeaderText

	for _, batch := range batches {
		batch.Mu.RLock()
		text += fmt.Sprintf("📦 *Batch:* %s \\| 📊 %d/%d selesai\n\n",
			utils.EscapeMarkdownV2(batch.Name),
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
			emoji := utils.StatusEmoji(string(snapshot.Status))
			bar := utils.ProgressBar(snapshot.Progress, 10)

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
				utils.EscapeMarkdownV2(utils.FormatStatus(string(snapshot.Status))),
				utils.EscapeMarkdownV2(bar),
				utils.EscapeMarkdownV2(utils.TruncateString(snapshot.FileName, 40)),
				utils.EscapeMarkdownV2(utils.FormatBytes(snapshot.TotalSize)),
				utils.EscapeMarkdownV2(utils.FormatSpeed(snapshot.Speed)),
				snapshot.Connections,
				utils.EscapeMarkdownV2(utils.FormatDuration(snapshot.ETA)),
				utils.EscapeMarkdownV2(snapshot.ID),
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
		emoji := utils.StatusEmoji(string(snapshot.Status))
		bar := utils.ProgressBar(snapshot.Progress, 10)

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
			utils.EscapeMarkdownV2(utils.FormatStatus(string(snapshot.Status))),
			utils.EscapeMarkdownV2(bar),
			utils.EscapeMarkdownV2(utils.TruncateString(snapshot.FileName, 40)),
			utils.EscapeMarkdownV2(utils.FormatBytes(snapshot.TotalSize)),
			utils.EscapeMarkdownV2(utils.FormatSpeed(snapshot.Speed)),
			snapshot.Connections,
			utils.EscapeMarkdownV2(utils.FormatDuration(snapshot.ETA)),
			utils.EscapeMarkdownV2(snapshot.ID),
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

func (s *BotService) sendStatusMessage(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup, forceNew bool) {
	s.TaskManager.Mu.RLock()
	lastMsgID, exists := s.TaskManager.LastStatusMsg[chatID]
	s.TaskManager.Mu.RUnlock()

	lastText, textExists := lastStatusText[chatID]

	if exists && forceNew {
		_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(chatID, lastMsgID))
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
		if _, err := s.Bot.Send(editMsg); err == nil {
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
	sentMsg, err := s.Bot.Send(msg)
	if err == nil {
		s.TaskManager.Mu.Lock()
		s.TaskManager.LastStatusMsg[chatID] = sentMsg.MessageID
		s.TaskManager.Mu.Unlock()
		lastStatusText[chatID] = text
	}
}

func (s *BotService) HandleCancel(message *tgbotapi.Message, args string) {
	if args == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nGunakan: `/cancel <TaskID>`\n\nLihat daftar task dengan /status")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	taskID := args

	if s.TaskManager.CancelTask(taskID) {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ *Task `%s` dibatalkan*", taskID))
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		s.UpdateSharedDashboard(message.Chat.ID, false)
		return
	}

	task := s.TaskManager.GetTaskByGID(taskID)
	if task != nil {
		if s.TaskManager.CancelTask(task.ID) {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ *Task `%s` dibatalkan*", task.ID))
			msg.ParseMode = MarkdownV2
			_, _ = s.Bot.Send(msg)
			return
		}
	}

	bm := s.BatchManager
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
		_, _ = s.Bot.Send(msg)
		s.UpdateSharedDashboard(message.Chat.ID, false)
		return
	}

	if s.checkBatchSubTaskCancellation(taskID) {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ *Sub-Task `%s` dibatalkan*", taskID))
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		s.UpdateSharedDashboard(message.Chat.ID, false)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ *Task/Batch `%s` tidak ditemukan*", utils.EscapeMarkdownV2(taskID)))
	msg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) HandleCancelCallback(callback *tgbotapi.CallbackQuery, taskID string) {
	success := false

	if s.TaskManager.CancelTask(taskID) {
		success = true
	} else {
		bm := s.BatchManager
		bm.Mu.RLock()
		if batch, ok := bm.Batches[taskID]; ok {
			batch.CancelFunc()
			batch.SetStatus(StatusCancelled)
			success = true
		}
		bm.Mu.RUnlock()

		if !success {
			if s.checkBatchSubTaskCancellation(taskID) {
				success = true
			}
		}
	}

	if success {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Dibatalkan"))
		s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
	} else {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Gagal membatalkan / Id tidak ditemukan"))
	}
}

func (s *BotService) checkBatchSubTaskCancellation(taskID string) bool {
	bm := s.BatchManager
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

func (s *BotService) HandleRefreshStatusCallback(callback *tgbotapi.CallbackQuery) {
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Refreshed"))
	s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
}

func (s *BotService) HandleConfirmCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	if len(parts) < 2 {
		return
	}

	action := parts[1]
	switch action {
	case "yes":
		text := "✅ *Dikonfirmasi*"
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)

	case "no":
		text := "❌ *Dibatalkan*"
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)
	}
}

func (s *BotService) getActiveBatchesByChat(chatID int64) []*BatchTask {
	bm := s.BatchManager
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
