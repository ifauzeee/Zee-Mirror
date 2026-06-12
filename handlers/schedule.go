package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zee-mirror/internal/domain"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

func (s *BotService) HandleSchedule(message *tgbotapi.Message, args string) {
	parts := strings.Fields(args)
	if len(parts) < 3 {
		s.Reply(message, "Usage: /schedule <type> <HH:MM> <url> [filename]\nType: mirror, zip, unzip, mp3, audio, video")
		return
	}

	taskType := parts[0]
	timeStr := parts[1]
	url := parts[2]
	var fileName string
	if len(parts) > 3 {
		fileName = strings.Join(parts[3:], " ")
	}

	validTypes := map[string]bool{"mirror": true, "zip": true, "unzip": true, "mp3": true, "audio": true, "video": true}
	if !validTypes[taskType] {
		s.Reply(message, fmt.Sprintf("Invalid type: %s. Valid: mirror, zip, unzip, mp3, audio, video", taskType))
		return
	}

	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		s.Reply(message, "Invalid time format. Use HH:MM (24-hour)")
		return
	}

	now := time.Now()
	scheduledAt := time.Date(now.Year(), now.Month(), now.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, now.Location())
	if scheduledAt.Before(now) {
		scheduledAt = scheduledAt.Add(24 * time.Hour)
	}

	st := domain.ScheduledTask{
		ID:          uuid.New().String(),
		TaskType:    taskType,
		URL:         url,
		FileName:    fileName,
		ChatID:      message.Chat.ID,
		UserID:      message.From.ID,
		Zip:         taskType == "zip",
		Unzip:       taskType == "unzip",
		Password:    "",
		Quality:     "",
		ScheduledAt: scheduledAt.Format(time.RFC3339),
		Status:      "pending",
	}

	if err := s.DB.SaveScheduled(context.Background(), st); err != nil {
		s.Reply(message, "Failed to schedule task")
		return
	}

	s.Reply(message, fmt.Sprintf("Task scheduled for %s", scheduledAt.Format("Mon Jan 2 15:04")))
}
