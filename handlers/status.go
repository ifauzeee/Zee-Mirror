package handlers

import (
	"fmt"
	"strings"
	"time"

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
		if exists {
			_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(chatID, lastMsgID))
			delete(tm.LastStatusMsg, chatID)
			delete(lastStatusText, chatID)
		}
		tm.Mu.Unlock()
		if forceNew {
			msg := tgbotapi.NewMessage(chatID, "❌ *Tidak ada task aktif\\.*")
			msg.ParseMode = MarkdownV2
			sentMsg, err := s.Bot.Send(msg)
			if err == nil {
				s.AutoDeleteMessage(chatID, sentMsg.MessageID, 30*time.Second)
			}
		}
		return
	}

	text := s.buildStatusDashboardText(tasks, batches, page)
	totalPages := (totalTasks + 4) / 5
	keyboard := buildNavigationKeyboard(page, totalPages)

	s.sendStatusMessage(chatID, text, keyboard, forceNew)
}

func (s *BotService) HandleRefreshStatusCallback(callback *tgbotapi.CallbackQuery) {
	s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
	callbackConfig := tgbotapi.NewCallback(callback.ID, "🔄 Dashboard direfresh")
	_, _ = s.Bot.Request(callbackConfig)
}

func calculateTotalPages(totalTasks int) int {
	return (totalTasks + 4) / 5
}

func (s *BotService) buildStatusDashboardText(tasks []*Task, batches []*BatchTask, page int) string {
	text := GetStatusHeader()

	allTasks := make([]interface{}, 0)
	for _, b := range batches {
		allTasks = append(allTasks, b)
	}
	for _, t := range tasks {
		allTasks = append(allTasks, t)
	}

	totalTasks := len(allTasks)
	start := page * 5
	end := start + 5
	if end > totalTasks {
		end = totalTasks
	}

	if start >= totalTasks {
		return GetErrorMessage("PAGING ERROR", "Halaman tidak ditemukan.")
	}

	visibleItems := allTasks[start:end]
	var visibleTasks []*Task
	var visibleBatches []*BatchTask

	for _, item := range visibleItems {
		if t, ok := item.(*Task); ok {
			visibleTasks = append(visibleTasks, t)
		} else if b, ok := item.(*BatchTask); ok {
			visibleBatches = append(visibleBatches, b)
		}
	}

	for _, batch := range visibleBatches {
		batch.Mu.RLock()
		emoji := utils.StatusEmoji(string(batch.Status))
		text += fmt.Sprintf("📦 *Batch:* `%s` \\| %s *%s*\n"+
			"📈 *Progres:* `%.1f%%` \\(%d/%d\\)\n"+
			"🚫 *Cancel:* /cancel\\_%s\n\n",
			batch.ID,
			emoji,
			utils.EscapeMarkdownV2(utils.FormatStatus(string(batch.Status))),
			batch.Progress,
			batch.Completed,
			len(batch.SubTasks),
			batch.ID,
		)
		batch.Mu.RUnlock()
	}

	for _, task := range visibleTasks {
		snapshot := task.GetSnapshot()
		text += FormatTaskProfessional(snapshot) + "\n"
	}

	totalPages := calculateTotalPages(totalTasks)
	text += CompactSeparator + "\n"
	if totalPages > 1 {
		text += fmt.Sprintf("📄 *Halaman:* %d/%d \\| ", page+1, totalPages)
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
		s.reply(message, GetErrorMessage("CANCEL ERROR", "Gunakan: /cancel <TaskID\\>\n\nLihat daftar task dengan /status"))
		return
	}

	taskID := args

	if s.TaskManager.CancelTask(taskID) {
		s.reply(message, GetSuccessMessage("TASK CANCELLED", fmt.Sprintf("Task `%s` telah dibatalkan.", taskID)))
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
		s.reply(message, fmt.Sprintf("✅ *Batch `%s` dibatalkan*", taskID))
		s.UpdateSharedDashboard(message.Chat.ID, false)
		return
	}

	if s.checkBatchSubTaskCancellation(taskID) {
		s.reply(message, fmt.Sprintf("✅ *Sub-Task `%s` dibatalkan*", taskID))
		s.UpdateSharedDashboard(message.Chat.ID, false)
		return
	}

	s.reply(message, fmt.Sprintf("❌ *Task/Batch `%s` tidak ditemukan*", utils.EscapeMarkdownV2(taskID)))
}

func (s *BotService) HandleCancelAll(message *tgbotapi.Message) {
	if !s.IsAdmin(message.From.ID) {
		msg := tgbotapi.NewMessage(message.Chat.ID, GetErrorMessage("PERMISSION DENIED", "Fitur ini hanya untuk Admin/Owner."))
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	count := s.TaskManager.CancelAllTasks()
	msg := tgbotapi.NewMessage(message.Chat.ID, GetSuccessMessage("CANCEL ALL", fmt.Sprintf("%d tugas aktif telah dibatalkan.", count)))
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
