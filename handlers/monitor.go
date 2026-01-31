package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type SystemStats struct {
	CPUUsage    float64
	MemoryTotal uint64
	MemoryUsed  uint64
	MemoryFree  uint64
	DiskTotal   uint64
	DiskUsed    uint64
	DiskFree    uint64
	DiskUsage   float64
	Uptime      time.Duration
	GoRoutines  int
	ActiveTasks int
	QueuedTasks int
	StartTime   time.Time
}

var botStartTime = time.Now()

func (s *BotService) HandleSystem(message *tgbotapi.Message) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	stats := s.getSystemStats()
	text := s.formatSystemStats(stats)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "system:refresh"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Detailed", "system:detailed"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Logs", "system:logs"),
			tgbotapi.NewInlineKeyboardButtonData("🧹 Cleanup", "system:cleanup"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "system:close"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) getSystemStats() SystemStats {
	stats := SystemStats{
		StartTime:   botStartTime,
		Uptime:      time.Since(botStartTime),
		GoRoutines:  runtime.NumGoroutine(),
		ActiveTasks: len(s.TaskManager.GetActiveTasks()),
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats.MemoryUsed = m.Alloc
	stats.MemoryTotal = m.Sys

	diskTotal, diskUsed, diskFree := s.getDiskStats(s.Config.DownloadDir)
	stats.DiskTotal = diskTotal
	stats.DiskUsed = diskUsed
	stats.DiskFree = diskFree
	if diskTotal > 0 {
		stats.DiskUsage = float64(diskUsed) / float64(diskTotal) * 100
	}

	stats.QueuedTasks = len(s.TaskManager.Queue)

	return stats
}

func (s *BotService) getDiskStats(path string) (total, used, free uint64) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, 0
	}
	_ = info
	usage := s.getDiskUsageOS()

	total = 100 * 1024 * 1024 * 1024
	used = uint64(usage * float64(total) / 100)
	free = total - used

	return
}

func (s *BotService) formatSystemStats(stats SystemStats) string {
	var content strings.Builder

	content.WriteString("⏱️ *RUNTIME*\n")
	content.WriteString(fmt.Sprintf("• Uptime: `%s`\n", utils.EscapeMarkdownV2(formatUptime(stats.Uptime))))
	content.WriteString(fmt.Sprintf("• Started: `%s`\n", utils.EscapeMarkdownV2(stats.StartTime.Format("02 Jan 15:04"))))
	content.WriteString(fmt.Sprintf("• Goroutines: `%d`\n\n", stats.GoRoutines))

	content.WriteString("📥 *TASKS*\n")
	content.WriteString(fmt.Sprintf("• Active: `%d`\n", stats.ActiveTasks))
	content.WriteString(fmt.Sprintf("• Queued: `%d`\n\n", stats.QueuedTasks))

	content.WriteString("💾 *MEMORY USAGE*\n")
	memBar := createUsageBar(float64(stats.MemoryUsed)/float64(stats.MemoryTotal)*100, 12)
	content.WriteString(fmt.Sprintf("• Used: `%s / %s`\n",
		utils.EscapeMarkdownV2(utils.FormatBytesUint64(stats.MemoryUsed)),
		utils.EscapeMarkdownV2(utils.FormatBytesUint64(stats.MemoryTotal))))
	content.WriteString(fmt.Sprintf("%s\n\n", utils.EscapeMarkdownV2(memBar)))

	content.WriteString("💿 *DISK STORAGE*\n")
	diskBar := createUsageBar(stats.DiskUsage, 12)
	content.WriteString(fmt.Sprintf("• Used: `%s / %s`\n",
		utils.EscapeMarkdownV2(utils.FormatBytesUint64(stats.DiskUsed)),
		utils.EscapeMarkdownV2(utils.FormatBytesUint64(stats.DiskTotal))))
	content.WriteString(utils.EscapeMarkdownV2(diskBar))

	return ProfessionalMessage("SYSTEM RESOURCES", content.String())
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm %ds", minutes, int(d.Seconds())%60)
}

func createUsageBar(percentage float64, width int) string {
	filled := int(percentage * float64(width) / 100)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	icon := "🟢"
	if percentage > 70 {
		icon = "🟡"
	}
	if percentage > 90 {
		icon = "🔴"
	}

	return fmt.Sprintf("%s %s %.1f%%", icon, bar, percentage)
}

func (s *BotService) HandleHealth(message *tgbotapi.Message) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	checks := s.performHealthChecks()
	text := s.formatHealthCheck(checks)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(msg)
}

type HealthCheck struct {
	Name    string
	Status  string
	Message string
	Healthy bool
}

func (s *BotService) performHealthChecks() []HealthCheck {
	var checks []HealthCheck

	aria2Check := HealthCheck{Name: "Aria2"}
	if _, err := os.Stat("/usr/bin/aria2c"); err == nil {
		aria2Check.Status = IconOK
		aria2Check.Message = "Aria2 binary found"
		aria2Check.Healthy = true
	} else {
		aria2Check.Status = "⚠️ WARNING"
		aria2Check.Message = "Aria2 binary not found"
		aria2Check.Healthy = true
	}
	checks = append(checks, aria2Check)

	rcloneCheck := HealthCheck{Name: "Rclone"}
	if _, err := os.Stat(s.TaskManager.ConfigDir + "/rclone.conf"); err == nil {
		rcloneCheck.Status = IconOK
		rcloneCheck.Message = "Config file found"
		rcloneCheck.Healthy = true
	} else {
		rcloneCheck.Status = IconError
		rcloneCheck.Message = "Config file missing"
		rcloneCheck.Healthy = false
	}
	checks = append(checks, rcloneCheck)

	diskCheck := HealthCheck{Name: "Disk Space"}
	diskUsage := s.getDiskUsageOS()
	switch {
	case diskUsage < 85:
		diskCheck.Status = IconOK
		diskCheck.Message = fmt.Sprintf("%.1f%% used", diskUsage)
		diskCheck.Healthy = true
	case diskUsage < 95:
		diskCheck.Status = "⚠️ WARNING"
		diskCheck.Message = fmt.Sprintf("%.1f%% used - Low space", diskUsage)
		diskCheck.Healthy = true
	default:
		diskCheck.Status = "❌ CRITICAL"
		diskCheck.Message = fmt.Sprintf("%.1f%% used - Very low!", diskUsage)
		diskCheck.Healthy = false
	}
	checks = append(checks, diskCheck)

	dbCheck := HealthCheck{Name: "Database"}
	ctx := context.Background()
	if err := s.DB.Ping(ctx); err == nil {
		dbCheck.Status = IconOK
		dbCheck.Message = "SQLite connection OK"
		dbCheck.Healthy = true
	} else {
		dbCheck.Status = IconError
		dbCheck.Message = "Database error"
		dbCheck.Healthy = false
	}
	checks = append(checks, dbCheck)

	dlCheck := HealthCheck{Name: "Download Dir"}
	if info, err := os.Stat(s.Config.DownloadDir); err == nil && info.IsDir() {
		dlCheck.Status = IconOK
		dlCheck.Message = "Directory accessible"
		dlCheck.Healthy = true
	} else {
		dlCheck.Status = IconError
		dlCheck.Message = "Directory not accessible"
		dlCheck.Healthy = false
	}
	checks = append(checks, dlCheck)

	return checks
}

func (s *BotService) formatHealthCheck(checks []HealthCheck) string {
	var content strings.Builder

	allHealthy := true
	for _, c := range checks {
		if !c.Healthy {
			allHealthy = false
			break
		}
	}

	status := "🟢 *SYSTEMS NORMAL*"
	if !allHealthy {
		status = "🔴 *ISSUES DETECTED*"
	}

	content.WriteString(fmt.Sprintf("STATUS: %s\n%s\n\n", status, CompactSeparator))

	for _, c := range checks {
		icon := "✅"
		if !c.Healthy {
			if strings.Contains(c.Status, "WARNING") {
				icon = "⚠️"
			} else {
				icon = "❌"
			}
		}

		content.WriteString(fmt.Sprintf("%s *%s*\n", icon, utils.EscapeMarkdownV2(c.Name)))
		content.WriteString(fmt.Sprintf("└ _%s_\n", utils.EscapeMarkdownV2(c.Message)))
	}

	content.WriteString(fmt.Sprintf("\n%s\n🕐 Checked: `%s`", CompactSeparator, utils.EscapeMarkdownV2(time.Now().Format("15:04:05"))))

	return ProfessionalMessage("HEALTH CHECK", content.String())
}

func (s *BotService) HandleLogs(message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.reply(message, GetErrorMessage("ACCESS DENIED", "Hanya Admin yang bisa melihat logs\\."))
		return
	}

	logPath := filepath.Join(s.Config.ConfigDir, "zee-mirror.log")

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		s.reply(message, GetErrorMessage("FILE NOT FOUND", "File log tidak ditemukan atau belum dibuat\\."))
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	tempFileName := fmt.Sprintf("zee-mirror_logs_%s.log", timestamp)
	tempPath := filepath.Join(os.TempDir(), tempFileName)

	input, err := os.ReadFile(filepath.Clean(logPath))
	if err != nil {
		s.reply(message, GetErrorMessage("READ ERROR", fmt.Sprintf("Gagal membaca log: %v", err)))
		return
	}

	err = os.WriteFile(tempPath, input, 0600)
	if err != nil {
		s.reply(message, GetErrorMessage("WRITE ERROR", fmt.Sprintf("Gagal membuat file temp: %v", err)))
		return
	}
	defer func() { _ = os.Remove(tempPath) }()

	file := tgbotapi.NewDocument(message.Chat.ID, tgbotapi.FilePath(tempPath))
	file.Caption = fmt.Sprintf("📝 *Application Logs:* `%s`", utils.EscapeMarkdownV2(tempFileName))
	file.ParseMode = MarkdownV2

	_, err = s.Bot.Send(file)
	if err != nil {
		s.reply(message, GetErrorMessage("SEND ERROR", fmt.Sprintf("Gagal mengirim file: %v", err)))
	}
}

func (s *BotService) HandleSystemCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[1]

	switch action {
	case CmdRefresh:
		stats := s.getSystemStats()
		text := s.formatSystemStats(stats)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "system:refresh"),
				tgbotapi.NewInlineKeyboardButtonData("📊 Detailed", "system:detailed"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📈 Logs", "system:logs"),
				tgbotapi.NewInlineKeyboardButtonData("🧹 Cleanup", "system:cleanup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "system:close"),
			),
		)

		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		editMsg.ReplyMarkup = &keyboard
		_, _ = s.Bot.Send(editMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Refreshed"))

	case "detailed":
		checks := s.performHealthChecks()
		text := s.formatHealthCheck(checks)

		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📊 Detailed view"))

	case "logs":
		msg := &tgbotapi.Message{
			Chat: callback.Message.Chat,
			From: callback.From,
		}
		s.HandleLogs(msg, "")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	case "logs_file":
		msg := &tgbotapi.Message{
			Chat: callback.Message.Chat,
			From: callback.From,
		}
		s.HandleLogs(msg, "file")
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Sending file..."))

	case "cleanup":
		if callback.From.ID != s.Config.OwnerID {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Owner only"))
			return
		}
		go s.performDiskCleanup()
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🧹 Cleanup started"))

	case CmdClose:
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, _ = s.Bot.Request(deleteMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Closed"))
		return
	}
}

func (s *BotService) StartResourceMonitor() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-s.TaskManager.ShutdownChan:
				return
			case <-ticker.C:
				s.checkResourceAlerts()
			}
		}
	}()

	slog.Info("Resource monitoring started")
}

func (s *BotService) checkResourceAlerts() {
	diskUsage := s.getDiskUsageOS()

	if diskUsage > 90 && s.Notifications != nil {
		s.Notifications.AlertDiskUsage(diskUsage, s.Config.DownloadDir)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memUsage := float64(m.Alloc) / float64(m.Sys) * 100

	if memUsage > 90 && s.Notifications != nil {
		s.Notifications.AlertSystemHealth(0, memUsage)
	}
}
