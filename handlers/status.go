package handlers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"zee-mirror/pkg/i18n"
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

	tm := s.TaskManager
	tm.Mu.RLock()
	page := tm.StatusPages[chatID]
	tasks := tm.GetActiveTasks()
	tm.Mu.RUnlock()

	bm := s.BatchManager
	bm.Mu.RLock()
	var batches []*BatchTask
	for _, b := range bm.Batches {
		if b.Status != StatusCompleted && b.Status != StatusFailed && b.Status != StatusCancelled {
			batches = append(batches, b)
		}
	}
	bm.Mu.RUnlock()

	totalTasks := len(tasks) + len(batches)
	if totalTasks == 0 {
		tm.Mu.Lock()
		lastMsgID, exists := tm.LastStatusMsg[chatID]

		if !exists {
			key := fmt.Sprintf("dashboard_msg_%d", chatID)
			if val, err := s.SettingsRepo.Get(context.Background(), key); err == nil && val != "" {
				var storedID int
				if n, _ := fmt.Sscanf(val, "%d", &storedID); n > 0 && storedID != 0 {
					lastMsgID = storedID
					exists = true
				}
			}
		}

		if exists {
			_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(chatID, lastMsgID))
			delete(tm.LastStatusMsg, chatID)
			delete(lastStatusText, chatID)
			delete(tm.LastDashUpdateAt, chatID)
			delete(tm.LastDashProgressSum, chatID)
			_ = s.SettingsRepo.Set(context.Background(), fmt.Sprintf("dashboard_msg_%d", chatID), "")
		}
		tm.Mu.Unlock()
		if forceNew {
			msg := tgbotapi.NewMessage(chatID, i18n.MsgNoActiveTasks)
			msg.ParseMode = MarkdownV2
			sentMsg, err := s.Bot.Send(msg)
			if err == nil {
				s.AutoDeleteMessage(chatID, sentMsg.MessageID, 30*time.Second)
			}
		}
		return
	}

	var currentProgressSum float64
	for _, t := range tasks {
		snapshot := t.GetSnapshot()
		currentProgressSum += snapshot.Progress
	}
	for _, b := range batches {
		b.Mu.RLock()
		currentProgressSum += b.Progress
		b.Mu.RUnlock()
	}

	tm.Mu.Lock()
	lastUpdateAt := tm.LastDashUpdateAt[chatID]
	lastProgressSum := tm.LastDashProgressSum[chatID]
	lastCount := tm.LastTasksCount[chatID]

	shouldUpdate := forceNew ||
		totalTasks != lastCount ||
		time.Since(lastUpdateAt) >= 3*time.Second ||
		(currentProgressSum-lastProgressSum) >= 5.0 ||
		(lastProgressSum-currentProgressSum) >= 5.0

	if !shouldUpdate {
		tm.Mu.Unlock()
		return
	}

	tm.LastDashUpdateAt[chatID] = time.Now()
	tm.LastDashProgressSum[chatID] = currentProgressSum
	tm.LastTasksCount[chatID] = totalTasks
	tm.Mu.Unlock()

	text := s.buildStatusDashboardText(tasks, batches, page)
	totalPages := (totalTasks + 4) / 5
	keyboard := buildNavigationKeyboard(page, totalPages)

	s.sendStatusMessage(chatID, text, keyboard, forceNew)
}

func (s *BotService) HandleRefreshStatusCallback(callback *tgbotapi.CallbackQuery) {
	s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
	callbackConfig := tgbotapi.NewCallback(callback.ID, i18n.MsgDashboardRefreshed)
	_, _ = s.Bot.Request(callbackConfig)
}

func calculateTotalPages(totalTasks int) int {
	return (totalTasks + 4) / 5
}

func (s *BotService) buildStatusDashboardText(tasks []*Task, batches []*BatchTask, page int) string {
	text := GetStatusHeader()

	allTasks := make([]interface{}, 0, len(tasks)+len(batches))
	for _, b := range batches {
		allTasks = append(allTasks, b)
	}
	for _, t := range tasks {
		allTasks = append(allTasks, t)
	}

	sort.Slice(allTasks, func(i, j int) bool {
		var timeI, timeJ time.Time

		if t, ok := allTasks[i].(*Task); ok {
			timeI = t.CreatedAt
		} else if b, ok := allTasks[i].(*BatchTask); ok {
			timeI = b.CreatedAt
		}

		if t, ok := allTasks[j].(*Task); ok {
			timeJ = t.CreatedAt
		} else if b, ok := allTasks[j].(*BatchTask); ok {
			timeJ = b.CreatedAt
		}

		return timeI.After(timeJ)
	})

	totalTasks := len(allTasks)
	start := page * 5
	end := start + 5
	if end > totalTasks {
		end = totalTasks
	}

	if start >= totalTasks {
		return GetErrorMessage("PAGING ERROR", i18n.MsgPagingError)
	}

	visibleItems := allTasks[start:end]
	var visibleTasksCount, visibleBatchesCount int

	for _, item := range visibleItems {
		if batch, ok := item.(*BatchTask); ok {
			visibleBatchesCount++
			batch.Mu.RLock()
			emoji := utils.StatusEmoji(string(batch.Status))
			batchID := utils.EscapeMarkdownV2(batch.ID)
			text += fmt.Sprintf("📦 *Batch:* `%s` • %s *%s*\n"+
				"📈 *Progres:* `%.1f%%` \\(%d/%d\\)\n"+
				"🚫 *Cancel:* /cancel\\_%s\n\n",
				batchID,
				emoji,
				utils.EscapeMarkdownV2(utils.FormatStatus(string(batch.Status))),
				batch.Progress,
				batch.Completed,
				len(batch.SubTasks),
				batchID,
			)
			batch.Mu.RUnlock()
		} else if task, ok := item.(*Task); ok {
			visibleTasksCount++
			snapshot := task.GetSnapshot()
			text += FormatTaskProfessional(snapshot) + "\n"
		}
	}

	totalPages := calculateTotalPages(totalTasks)
	text += CompactSeparator + "\n"
	if totalPages > 1 {
		text += fmt.Sprintf("📄 *Halaman:* %d/%d • ", page+1, totalPages)
	}

	if len(batches) > 0 {
		text += fmt.Sprintf("_Active: %d batches, %d regular tasks_", len(batches), visibleTasksCount)
	} else {
		text += fmt.Sprintf("_Total: %d task aktif_", visibleTasksCount)
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

	if !exists {
		key := fmt.Sprintf("dashboard_msg_%d", chatID)
		if val, err := s.SettingsRepo.Get(context.Background(), key); err == nil && val != "" {
			var storedID int
			if n, _ := fmt.Sscanf(val, "%d", &storedID); n > 0 && storedID != 0 {
				lastMsgID = storedID
				exists = true
				s.TaskManager.Mu.Lock()
				s.TaskManager.LastStatusMsg[chatID] = storedID
				s.TaskManager.Mu.Unlock()
			}
		}
	}

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

		key := fmt.Sprintf("dashboard_msg_%d", chatID)
		_ = s.SettingsRepo.Set(context.Background(), key, fmt.Sprintf("%d", sentMsg.MessageID))
	}
}

func (s *BotService) HandleCancel(message *tgbotapi.Message, args string) {
	if args == "" {
		s.reply(message, GetErrorMessage("CANCEL ERROR", "Gunakan: /cancel <TaskID>\n\nLihat daftar task dengan /status"))
		return
	}

	taskID := args

	if s.TaskManager.CancelTask(taskID) {
		s.reply(message, GetSuccessMessage("TASK CANCELLED", fmt.Sprintf(i18n.MsgTaskCancelled, taskID)))
		s.UpdateSharedDashboard(message.Chat.ID, false)
		return
	}

	task := s.TaskManager.GetTaskByGID(taskID)
	if task != nil {
		if s.TaskManager.CancelTask(task.ID) {
			s.reply(message, fmt.Sprintf("✅ *Task `%s` dibatalkan*", task.ID))
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
		s.reply(message, GetSuccessMessage("BATCH CANCELLED", fmt.Sprintf(i18n.MsgBatchCancelled, taskID)))
		s.UpdateSharedDashboard(message.Chat.ID, false)
		return
	}

	if s.checkBatchSubTaskCancellation(taskID) {
		s.reply(message, GetSuccessMessage("SUB-TASK CANCELLED", fmt.Sprintf(i18n.MsgSubTaskCancelled, taskID)))
		s.UpdateSharedDashboard(message.Chat.ID, false)
		return
	}

	s.reply(message, GetErrorMessage("NOT FOUND", fmt.Sprintf(i18n.MsgTaskNotFound, utils.EscapeMarkdownV2(taskID))))
}

func (s *BotService) HandleCancelAll(message *tgbotapi.Message) {
	if !s.IsAdmin(message.From.ID) {
		msg := tgbotapi.NewMessage(message.Chat.ID, GetErrorMessage("PERMISSION DENIED", i18n.MsgAdminOnly))
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	count := s.TaskManager.CancelAllTasks()

	s.BatchManager.Mu.RLock()
	var batchIDs []string
	for id := range s.BatchManager.Batches {
		batchIDs = append(batchIDs, id)
	}
	s.BatchManager.Mu.RUnlock()

	for _, id := range batchIDs {
		s.BatchManager.Mu.Lock()
		if b, exists := s.BatchManager.Batches[id]; exists {
			if b.Status != StatusCompleted && b.Status != StatusFailed && b.Status != StatusCancelled {
				if b.CancelFunc != nil {
					b.CancelFunc()
				}
				b.Status = StatusCancelled
				b.CompletedAt = time.Now()
				count++
			}
		}
		s.BatchManager.Mu.Unlock()
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, GetSuccessMessage("CANCEL ALL", fmt.Sprintf(i18n.MsgAllTasksCancelled, count)))
	msg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(msg)
	s.UpdateSharedDashboard(message.Chat.ID, false)
}

func (s *BotService) HandleCancelCallback(callback *tgbotapi.CallbackQuery, taskID string) {
	success := false

	if s.TaskManager.CancelTask(taskID) {
		success = true
	} else {
		task := s.TaskManager.GetTaskByGID(taskID)
		if task != nil {
			if s.TaskManager.CancelTask(task.ID) {
				success = true
			}
		}
	}

	if success {
		callbackConfig := tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Task %s dibatalkan", taskID))
		_, _ = s.Bot.Request(callbackConfig)
		s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
	} else {
		callbackConfig := tgbotapi.NewCallback(callback.ID, "❌ Gagal membatalkan task")
		_, _ = s.Bot.Request(callbackConfig)
	}
}
