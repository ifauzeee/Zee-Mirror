package handlers

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type NotificationService struct {
	Bot            *tgbotapi.BotAPI
	AlertChannelID int64
	OwnerID        int64
	mu             sync.RWMutex
	DiskThreshold  float64
}

func NewNotificationService(bot *tgbotapi.BotAPI, alertChannelID, ownerID int64) *NotificationService {
	return &NotificationService{
		Bot:            bot,
		AlertChannelID: alertChannelID,
		OwnerID:        ownerID,
		DiskThreshold:  85.0,
	}
}

func (n *NotificationService) SetAlertChannel(channelID int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.AlertChannelID = channelID
}

func (n *NotificationService) SetDiskThreshold(threshold float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.DiskThreshold = threshold
}

func (n *NotificationService) SendAlert(alertType, title, message string) {
	targetID := n.AlertChannelID
	if targetID == 0 {
		targetID = n.OwnerID
	}

	if targetID == 0 {
		return
	}

	text := fmt.Sprintf(`🚨 *ALERT: %s*

%s %s

📝 *Details:*
%s

⏰ *Time:* %s`,
		utils.EscapeMarkdownV2(title),
		alertType,
		utils.EscapeMarkdownV2(title),
		utils.EscapeMarkdownV2(message),
		utils.EscapeMarkdownV2(time.Now().Format("02 Jan 2006 15:04:05")),
	)

	msg := tgbotapi.NewMessage(targetID, text)
	msg.ParseMode = "MarkdownV2"
	if _, err := n.Bot.Send(msg); err != nil {
		log.Printf("[Alert] Failed to send alert: %v", err)
	}
}

func (n *NotificationService) AlertDiskUsage(usage float64, path string) {
	if usage < n.DiskThreshold {
		return
	}

	message := fmt.Sprintf("Disk usage at %.1f%% on %s\n\nPlease free up some space to prevent download failures.", usage, path)
	n.SendAlert("💾", "High Disk Usage", message)
}

func (n *NotificationService) AlertSystemHealth(cpuUsage, memUsage float64) {
	var alerts []string

	if cpuUsage > 90 {
		alerts = append(alerts, fmt.Sprintf("CPU: %.1f%%", cpuUsage))
	}
	if memUsage > 90 {
		alerts = append(alerts, fmt.Sprintf("Memory: %.1f%%", memUsage))
	}

	if len(alerts) > 0 {
		message := fmt.Sprintf("High resource usage detected:\n%s", strings.Join(alerts, "\n"))
		n.SendAlert("🖥️", "System Health Warning", message)
	}
}

func (s *BotService) HandleSetAlertChannel(message *tgbotapi.Message, args string) {
	if message.From.ID != s.Config.OwnerID {
		s.reply(message, "❌ *Akses Ditolak*\nHanya Owner yang bisa menggunakan perintah ini\\.")
		return
	}

	channelID := message.Chat.ID

	if args != "" {
		var id int64
		if _, err := fmt.Sscanf(args, "%d", &id); err == nil {
			channelID = id
		}
	}

	if s.Notifications != nil {
		s.Notifications.SetAlertChannel(channelID)
	}

	_ = s.DB.SetSetting("alert_channel_id", fmt.Sprintf("%d", channelID))

	s.reply(message, fmt.Sprintf("✅ *Alert Channel Set*\n\nChannel ID: `%d`\n\nSemua alert akan dikirim ke channel ini\\.", channelID))
}
