package basic

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"zee-mirror/internal/domain"
	"zee-mirror/internal/service"
	"zee-mirror/pkg/chart"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	CmdRefresh = "refresh"
)

type UserStats = domain.UserStats
type DailyStats = domain.DailyStats

func HandleStats(s *service.BotService, message *tgbotapi.Message) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	ctx := context.Background()
	stats, err := s.DB.GetBotStats(ctx)
	if err != nil {
		s.Reply(message, "❌ *Gagal mengambil statistik*")
		return
	}

	userStats, _ := s.DB.GetUserStats(ctx, message.From.ID)
	dailyStats, _ := s.DB.GetTodayStats(ctx)
	userDailyStats, _ := s.DB.GetUserTodayStats(ctx, message.From.ID)
	weeklyStats, _ := s.DB.GetWeeklyStats(ctx)

	slog.Info("Generating stats", "userID", message.From.ID)

	text := FormatStatsMessage(stats, userStats, dailyStats, userDailyStats)

	keyboard := GetStatsKeyboard()

	chartBytes, err := chart.GenerateWeeklyStatsChart(weeklyStats)
	if err == nil && len(chartBytes) > 0 {
		msg := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileBytes{Name: "stats.png", Bytes: chartBytes})
		msg.Caption = text
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		msg.ReplyMarkup = keyboard

		sentMsg, err := s.Bot.Send(msg)
		if err != nil {
			slog.Error("Error sending stats photo", "error", err, "userID", message.From.ID)

			fallbackMsg := tgbotapi.NewMessage(message.Chat.ID, text)
			fallbackMsg.ParseMode = tgbotapi.ModeMarkdownV2
			fallbackMsg.ReplyMarkup = keyboard
			if sentTextMsg, err := s.Bot.Send(fallbackMsg); err == nil {
				s.AutoDeleteMessage(message.Chat.ID, sentTextMsg.MessageID, 60*time.Second)
			}
		} else {
			s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 60*time.Second)
		}
	} else {
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		msg.ReplyMarkup = keyboard
		sentMsg, err := s.Bot.Send(msg)
		if err != nil {
			slog.Error("Error sending stats message", "error", err, "userID", message.From.ID)
			msg.ParseMode = ""
			msg.Text = "❌ *Gagal memformat statistik (Markdown Error)*\n\nFallback: Manual Stats\nTotal Tasks: " + fmt.Sprint(getIntValue(stats, "total_tasks"))
			_, _ = s.Bot.Send(msg)
		} else {
			s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 60*time.Second)
		}
	}
}

func GetStatsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.InlineKeyboardMarkup{}
}

func FormatStatsMessage(stats map[string]interface{}, userStats *UserStats, dailyStats *DailyStats, userDailyStats *DailyStats) string {
	var text strings.Builder

	text.WriteString("📊 *STATISTIK DASHBOARD*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	totalTasks := getIntValue(stats, "total_tasks")
	completedTasks := getIntValue(stats, "completed_tasks")
	failedTasks := getIntValue(stats, "failed_tasks")
	totalBandwidth := getInt64Value(stats, "total_bandwidth")

	text.WriteString("🌐 *GLOBAL STATS*\n")
	fmt.Fprintf(&text, "📥 Total Tasks: `%d`\n", totalTasks)
	fmt.Fprintf(&text, "✅ Completed: `%d`\n", completedTasks)
	fmt.Fprintf(&text, "❌ Failed: `%d`\n", failedTasks)
	fmt.Fprintf(&text, "📊 Success Rate: `%.1f%%`\n", calculateSuccessRate(completedTasks, totalTasks))
	fmt.Fprintf(&text, "📈 Bandwidth: `%s`\n\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(totalBandwidth)))

	if dailyStats != nil {
		text.WriteString("📅 *HARI INI \\(GLOBAL\\)*\n")
		fmt.Fprintf(&text, "📥 Tasks: `%d`\n", dailyStats.TotalTasks)
		fmt.Fprintf(&text, "✅ Completed: `%d`\n", dailyStats.CompletedTasks)
		fmt.Fprintf(&text, "📈 Bandwidth: `%s`\n\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(dailyStats.TotalBandwidth)))
	}

	if userDailyStats != nil && userDailyStats.TotalTasks > 0 {
		text.WriteString("👤 *MY TODAY*\n")
		fmt.Fprintf(&text, "📥 Tasks: `%d`\n", userDailyStats.TotalTasks)
		fmt.Fprintf(&text, "📊 Bandwidth: `%s`\n\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(userDailyStats.TotalBandwidth)))
	}

	if userStats != nil {
		text.WriteString("👤 *MY ALL\\-TIME STATS*\n")
		fmt.Fprintf(&text, "📥 Total Tasks: `%d`\n", userStats.TotalDownloads)
		fmt.Fprintf(&text, "📊 Bandwidth: `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(userStats.TotalBandwidth)))
		if !userStats.LastActive.IsZero() {
			fmt.Fprintf(&text, "🕐 Active: `%s`\n", utils.EscapeMarkdownV2Code(userStats.LastActive.Format("15:04")))
		}
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func formatUserStatsDetailed(stats *UserStats) string {
	if stats == nil {
		return "📭 *Belum ada statistik untuk Anda*"
	}

	var text strings.Builder
	text.WriteString("👤 *USER STATISTICS*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	text.WriteString("🆔 *USER INFO*\n")
	fmt.Fprintf(&text, "👤 Name: @%s\n", utils.EscapeMarkdownV2(stats.Username))
	fmt.Fprintf(&text, "🆔 ID: `%d`\n", stats.UserID)
	if !stats.LastActive.IsZero() {
		fmt.Fprintf(&text, "🕐 Last Active: `%s`\n", utils.EscapeMarkdownV2Code(stats.LastActive.Format("02 Jan 15:04")))
	}
	text.WriteString("\n")

	text.WriteString("📊 *ACTIVITY*\n")
	fmt.Fprintf(&text, "📥 Total Tasks: `%d`\n", stats.TotalDownloads)
	fmt.Fprintf(&text, "✅ Success: `%d`\n", stats.SuccessfulTasks)
	fmt.Fprintf(&text, "❌ Failed: `%d`\n", stats.FailedTasks)
	fmt.Fprintf(&text, "📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(stats.TotalBandwidth)))

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func formatDailyStatsDetailed(stats *DailyStats, title string, userStats *DailyStats) string {
	if stats == nil {
		return "📭 *Belum ada statistik*"
	}

	var text strings.Builder
	fmt.Fprintf(&text, "📅 *STATISTIK %s*\n", strings.ToUpper(utils.EscapeMarkdownV2(title)))
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Fprintf(&text, "📆 *%s*\n\n", stats.Date.Format("02 Jan 2006"))

	text.WriteString("🌐 *GLOBAL ACTIVITY*\n")
	fmt.Fprintf(&text, "📥 Total Tasks: `%d`\n", stats.TotalTasks)
	fmt.Fprintf(&text, "✅ Completed: `%d`\n", stats.CompletedTasks)
	fmt.Fprintf(&text, "❌ Failed: `%d`\n", stats.FailedTasks)
	fmt.Fprintf(&text, "📊 Success Rate: `%.1f%%`\n", calculateSuccessRate(stats.CompletedTasks, stats.TotalTasks))
	fmt.Fprintf(&text, "📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(stats.TotalBandwidth)))
	if stats.AverageSpeed > 0 {
		fmt.Fprintf(&text, "⚡ Avg Speed: `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatSpeed(stats.AverageSpeed)))
	}
	if stats.PeakConcurrent > 0 {
		fmt.Fprintf(&text, "🔥 Peak Concurrent: `%d`\n", stats.PeakConcurrent)
	}

	if userStats != nil {
		text.WriteString("\n👤 *YOUR ACTIVITY TODAY*\n")
		fmt.Fprintf(&text, "📥 Total Tasks: `%d`\n", userStats.TotalTasks)
		fmt.Fprintf(&text, "✅ Success: `%d`\n", userStats.CompletedTasks)
		fmt.Fprintf(&text, "❌ Failed: `%d`\n", userStats.FailedTasks)
		fmt.Fprintf(&text, "📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(userStats.TotalBandwidth)))
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func formatWeeklyStats(stats []DailyStats) string {
	var text strings.Builder
	text.WriteString("📈 *STATISTIK MINGGUAN*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	totalTasks := 0
	totalCompleted := 0
	totalFailed := 0

	var totalBandwidth int64

	text.WriteString("🗓️ *LAST 7 DAYS*\n")

	for _, day := range stats {
		totalTasks += day.TotalTasks
		totalCompleted += day.CompletedTasks
		totalFailed += day.FailedTasks
		totalBandwidth += day.TotalBandwidth

		icon := "▫️"
		if day.TotalTasks > 0 {
			icon = "▪️"
		}

		fmt.Fprintf(&text, "%s `%s` : %d tasks\n",
			icon,
			day.Date.Format("02/01"),
			day.TotalTasks)
	}

	text.WriteString("\n📊 *SUMMARY*\n")
	fmt.Fprintf(&text, "📥 Total: `%d` tasks\n", totalTasks)
	fmt.Fprintf(&text, "✅ Completed: `%d`\n", totalCompleted)
	fmt.Fprintf(&text, "📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(totalBandwidth)))

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func formatMonthlyStats(stats []DailyStats) string {
	var text strings.Builder
	text.WriteString("📉 *STATISTIK BULANAN*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	totalTasks := 0
	totalCompleted := 0
	totalFailed := 0
	var totalBandwidth int64

	weekStats := make(map[int]struct {
		tasks     int
		completed int
		bandwidth int64
	})

	var weeks []int

	for _, day := range stats {
		totalTasks += day.TotalTasks
		totalCompleted += day.CompletedTasks
		totalFailed += day.FailedTasks
		totalBandwidth += day.TotalBandwidth

		_, week := day.Date.ISOWeek()
		if _, exists := weekStats[week]; !exists {
			weeks = append(weeks, week)
		}

		ws := weekStats[week]
		ws.tasks += day.TotalTasks
		ws.completed += day.CompletedTasks
		ws.bandwidth += day.TotalBandwidth
		weekStats[week] = ws
	}

	text.WriteString("📅 *WEEKLY BREAK*\n")

	for _, week := range weeks {
		ws := weekStats[week]
		fmt.Fprintf(&text, "Week %d: `%d` tasks\n└ `%s`\n",
			week,
			ws.tasks,
			utils.EscapeMarkdownV2Code(utils.FormatBytes(ws.bandwidth)))
	}

	text.WriteString("\n📊 *SUMMARY*\n")
	fmt.Fprintf(&text, "📥 Total: `%d` tasks\n", totalTasks)
	fmt.Fprintf(&text, "✅ Completed: `%d`\n", totalCompleted)
	fmt.Fprintf(&text, "📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(totalBandwidth)))

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func HandleStatsCallback(s *service.BotService, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	ctx := context.Background()
	action := parts[1]
	var text string

	switch action {
	case "my":
		userStats, err := s.DB.GetUserStats(ctx, callback.From.ID)
		if err != nil {
			text = "❌ *Gagal mengambil statistik Anda*"
		} else {
			text = formatUserStatsDetailed(userStats)
		}

	case "today":
		dailyStats, err := s.DB.GetTodayStats(ctx)
		userDailyStats, _ := s.DB.GetUserTodayStats(ctx, callback.From.ID)
		if err != nil {
			text = "❌ *Gagal mengambil statistik hari ini*"
		} else {
			text = formatDailyStatsDetailed(dailyStats, "Hari Ini", userDailyStats)
		}

	case "weekly":
		weeklyStats, err := s.DB.GetWeeklyStats(ctx)
		if err != nil {
			text = "❌ *Gagal mengambil statistik mingguan*"
		} else {
			text = formatWeeklyStats(weeklyStats)
		}

	case "monthly":
		monthlyStats, err := s.DB.GetMonthlyStats(ctx)
		if err != nil {
			text = "❌ *Gagal mengambil statistik bulanan*"
		} else {
			text = formatMonthlyStats(monthlyStats)
		}

	case CmdRefresh:
		stats, _ := s.DB.GetBotStats(ctx)
		userStats, _ := s.DB.GetUserStats(ctx, callback.From.ID)
		dailyStats, _ := s.DB.GetTodayStats(ctx)
		userDailyStats, _ := s.DB.GetUserTodayStats(ctx, callback.From.ID)
		text = FormatStatsMessage(stats, userStats, dailyStats, userDailyStats)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Statistics refreshed!"))

	case CmdClose:
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, _ = s.Bot.Request(deleteMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Closed"))
		return
	}

	if text != "" {
		if callback.Message.Photo != nil {
			editMsg := tgbotapi.NewEditMessageCaption(callback.Message.Chat.ID, callback.Message.MessageID, text)
			editMsg.ParseMode = tgbotapi.ModeMarkdownV2
			_, _ = s.Bot.Send(editMsg)
		} else {
			editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
			editMsg.ParseMode = tgbotapi.ModeMarkdownV2
			_, _ = s.Bot.Send(editMsg)
		}
	}
}

func calculateSuccessRate(completed, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(completed) / float64(total) * 100
}

func getIntValue(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}

func getInt64Value(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case float64:
			return int64(val)
		}
	}
	return 0
}
