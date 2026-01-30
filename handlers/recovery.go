package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"zee-mirror/internal/database"
	"zee-mirror/internal/domain"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TaskRecovery struct {
	DB          *database.DB
	TaskManager *TaskManager
	BotService  *BotService
}

func NewTaskRecovery(db *database.DB, tm *TaskManager, bs *BotService) *TaskRecovery {
	return &TaskRecovery{
		DB:          db,
		TaskManager: tm,
		BotService:  bs,
	}
}

func (tr *TaskRecovery) RecoverIncompleteTasks() error {
	log.Println("[Recovery] Checking for incomplete tasks...")

	tasks, err := tr.DB.GetRecoverableTasks()
	if err != nil {
		return fmt.Errorf("failed to get recoverable tasks: %v", err)
	}

	if len(tasks) == 0 {
		log.Println("[Recovery] No incomplete tasks found")
		return nil
	}

	log.Printf("[Recovery] Found %d incomplete tasks", len(tasks))

	recovered := 0
	skipped := 0

	for _, record := range tasks {
		if time.Since(record.CreatedAt) > 24*time.Hour {
			log.Printf("[Recovery] Skipping old task %s", record.ID)
			_ = tr.DB.UpdateTaskStatus(record.ID, "expired", "Task expired")
			skipped++
			continue
		}

		task := tr.createTaskFromRecord(record)
		if task == nil {
			skipped++
			continue
		}

		log.Printf("[Recovery] Recovering task %s", record.ID)

		tr.TaskManager.Mu.Lock()
		tr.TaskManager.Tasks[task.ID] = task
		tr.TaskManager.Mu.Unlock()

		go func(t *Task) {
			tr.TaskManager.Queue <- t
		}(task)

		recovered++
	}

	log.Printf("[Recovery] Recovered %d tasks, skipped %d", recovered, skipped)
	return nil
}

func (tr *TaskRecovery) createTaskFromRecord(record database.TaskRecord) *Task {
	ctx, cancel := context.WithCancel(context.Background())

	task := &Task{
		Task: domain.Task{
			ID:         record.ID,
			GID:        record.GID,
			Type:       domain.TaskType(record.Type),
			Status:     domain.StatusQueued,
			URL:        record.URL,
			FileName:   record.FileName,
			ChatID:     record.ChatID,
			UserID:     record.UserID,
			CreatedAt:  time.Now(),
			Ctx:        ctx,
			CancelFunc: cancel,
			Zip:        record.Zip,
			Unzip:      record.Unzip,
			Password:   record.Password,
		},
		DB: tr.DB,
	}

	return task
}

func (s *BotService) HandleRecoveryStatus(message *tgbotapi.Message) {
	if message.From.ID != s.Config.OwnerID {
		s.reply(message, "❌ *Akses Ditolak*")
		return
	}

	tasks, err := s.DB.GetRecoverableTasks()
	if err != nil {
		s.reply(message, "❌ *Gagal mengambil data recovery*")
		return
	}

	if len(tasks) == 0 {
		s.reply(message, "✅ *Tidak ada task yang perlu di\\-recover*")
		return
	}

	var text strings.Builder
	text.WriteString("🔄 *Recoverable Tasks*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	for i, t := range tasks {
		if i >= 10 {
			text.WriteString(fmt.Sprintf("\n_\\.\\.\\. dan %d task lainnya_", len(tasks)-10))
			break
		}
		text.WriteString(fmt.Sprintf("• `%s` \\| %s\n", t.ID, utils.EscapeMarkdownV2(t.Type)))
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━\n")
	text.WriteString("Use /recover to recover all tasks")

	s.reply(message, text.String())
}

func (s *BotService) HandleRecover(message *tgbotapi.Message) {
	if message.From.ID != s.Config.OwnerID {
		s.reply(message, "❌ *Akses Ditolak*")
		return
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, "🔄 *Recovering tasks\\.\\.\\.*")
	statusMsg.ParseMode = MarkdownV2
	sent, _ := s.Bot.Send(statusMsg)

	recovery := NewTaskRecovery(s.DB, s.TaskManager, s)
	if err := recovery.RecoverIncompleteTasks(); err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Recovery failed*\n\n%s", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	activeTasks := s.TaskManager.GetActiveTasks()
	s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("✅ *Recovery Complete*\n\nActive tasks: %d", len(activeTasks)))
}
