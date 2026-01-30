package handlers

import (
	"fmt"
	"log"
	"strings"

	"zee-mirror/internal/database"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserStats = database.UserStats
type DailyStats = database.DailyStats

func (s *BotService) HandleStats(message *tgbotapi.Message) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	stats, err := s.DB.GetBotStats()
	if err != nil {
		s.reply(message, "❌ *Gagal mengambil statistik*")
		return
	}

	userStats, _ := s.DB.GetUserStats(message.From.ID)
	dailyStats, _ := s.DB.GetTodayStats()
	userDailyStats, _ := s.DB.GetUserTodayStats(message.From.ID)

	log.Printf("[Stats] Generating stats for user %d", message.From.ID)
	text := s.formatStatsMessage(stats, userStats, dailyStats, userDailyStats)

	keyboard := s.getStatsKeyboard()

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, err = s.Bot.Send(msg)
	if err != nil {
		log.Printf("[Stats] Error sending stats message: %v", err)
		msg.ParseMode = ""
		msg.Text = "❌ *Gagal memformat statistik (Markdown Error)*\n\nFallback: Manual Stats\nTotal Tasks: " + fmt.Sprint(getIntValue(stats, "total_tasks"))
		_, _ = s.Bot.Send(msg)
	} else {
		log.Printf("[Stats] Stats message sent successfully to %d", message.From.ID)
	}
}

func (s *BotService) getStatsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 My Stats", "stats:my"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Today", "stats:today"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Weekly", "stats:weekly"),
			tgbotapi.NewInlineKeyboardButtonData("📉 Monthly", "stats:monthly"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "stats:refresh"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Close", "stats:close"),
		),
	)
}

func (s *BotService) formatStatsMessage(stats map[string]interface{}, userStats *UserStats, dailyStats *DailyStats, userDailyStats *DailyStats) string {
	var text strings.Builder

	text.WriteString("📊 *STATISTIK DASHBOARD*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	totalTasks := getIntValue(stats, "total_tasks")
	completedTasks := getIntValue(stats, "completed_tasks")
	failedTasks := getIntValue(stats, "failed_tasks")
	totalBandwidth := getInt64Value(stats, "total_bandwidth")

	text.WriteString("🌐 *GLOBAL STATS*\n")
	text.WriteString(fmt.Sprintf("📥 Total Tasks: `%d`\n", totalTasks))
	text.WriteString(fmt.Sprintf("✅ Completed: `%d`\n", completedTasks))
	text.WriteString(fmt.Sprintf("❌ Failed: `%d`\n", failedTasks))
	text.WriteString(fmt.Sprintf("📊 Success Rate: `%.1f%%`\n", calculateSuccessRate(completedTasks, totalTasks)))
	text.WriteString(fmt.Sprintf("📈 Bandwidth: `%s`\n\n", utils.EscapeMarkdownV2(utils.FormatBytes(totalBandwidth))))

	if dailyStats != nil {
		text.WriteString("📅 *HARI INI \\(GLOBAL\\)*\n")
		text.WriteString(fmt.Sprintf("📥 Tasks: `%d`\n", dailyStats.TotalTasks))
		text.WriteString(fmt.Sprintf("✅ Completed: `%d`\n", dailyStats.CompletedTasks))
		text.WriteString(fmt.Sprintf("📈 Bandwidth: `%s`\n\n", utils.EscapeMarkdownV2(utils.FormatBytes(dailyStats.TotalBandwidth))))
	}

	if userDailyStats != nil && userDailyStats.TotalTasks > 0 {
		text.WriteString("👤 *MY TODAY*\n")
		text.WriteString(fmt.Sprintf("📥 Tasks: `%d`\n", userDailyStats.TotalTasks))
		text.WriteString(fmt.Sprintf("📊 Bandwidth: `%s`\n\n", utils.EscapeMarkdownV2(utils.FormatBytes(userDailyStats.TotalBandwidth))))
	}

	if userStats != nil {
		text.WriteString("👤 *MY ALL\\-TIME STATS*\n")
		text.WriteString(fmt.Sprintf("📥 Total Tasks: `%d`\n", userStats.TotalDownloads))
		text.WriteString(fmt.Sprintf("📊 Bandwidth: `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(userStats.TotalBandwidth))))
		if !userStats.LastActive.IsZero() {
			text.WriteString(fmt.Sprintf("🕐 Active: `%s`\n", utils.EscapeMarkdownV2(userStats.LastActive.Format("15:04"))))
		}
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func (s *BotService) formatUserStatsDetailed(stats *UserStats) string {
	if stats == nil {
		return "📭 *Belum ada statistik untuk Anda*"
	}

	var text strings.Builder
	text.WriteString("👤 *USER STATISTICS*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	text.WriteString("🆔 *USER INFO*\n")
	text.WriteString(fmt.Sprintf("👤 Name: @%s\n", utils.EscapeMarkdownV2(stats.Username)))
	text.WriteString(fmt.Sprintf("🆔 ID: `%d`\n", stats.UserID))
	if !stats.LastActive.IsZero() {
		text.WriteString(fmt.Sprintf("🕐 Last Active: `%s`\n", utils.EscapeMarkdownV2(stats.LastActive.Format("02 Jan 15:04"))))
	}
	text.WriteString("\n")

	text.WriteString("📊 *ACTIVITY*\n")
	text.WriteString(fmt.Sprintf("📥 Total Tasks: `%d`\n", stats.TotalDownloads))
	text.WriteString(fmt.Sprintf("✅ Success: `%d`\n", stats.SuccessfulTasks))
	text.WriteString(fmt.Sprintf("❌ Failed: `%d`\n", stats.FailedTasks))
	text.WriteString(fmt.Sprintf("📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(stats.TotalBandwidth))))

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func (s *BotService) formatDailyStatsDetailed(stats *DailyStats, title string, userStats *DailyStats) string {
	if stats == nil {
		return "📭 *Belum ada statistik*"
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("📅 *STATISTIK %s*\n", strings.ToUpper(utils.EscapeMarkdownV2(title))))
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	text.WriteString(fmt.Sprintf("📆 *%s*\n\n", stats.Date.Format("02 Jan 2006")))

	text.WriteString("🌐 *GLOBAL ACTIVITY*\n")
	text.WriteString(fmt.Sprintf("📥 Total Tasks: `%d`\n", stats.TotalTasks))
	text.WriteString(fmt.Sprintf("✅ Completed: `%d`\n", stats.CompletedTasks))
	text.WriteString(fmt.Sprintf("❌ Failed: `%d`\n", stats.FailedTasks))
	text.WriteString(fmt.Sprintf("📊 Success Rate: `%.1f%%`\n", calculateSuccessRate(stats.CompletedTasks, stats.TotalTasks)))
	text.WriteString(fmt.Sprintf("📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(stats.TotalBandwidth))))
	if stats.AverageSpeed > 0 {
		text.WriteString(fmt.Sprintf("⚡ Avg Speed: `%s`\n", utils.EscapeMarkdownV2(utils.FormatSpeed(stats.AverageSpeed))))
	}
	if stats.PeakConcurrent > 0 {
		text.WriteString(fmt.Sprintf("🔥 Peak Concurrent: `%d`\n", stats.PeakConcurrent))
	}

	if userStats != nil {
		text.WriteString("\n👤 *YOUR ACTIVITY TODAY*\n")
		text.WriteString(fmt.Sprintf("📥 Total Tasks: `%d`\n", userStats.TotalTasks))
		text.WriteString(fmt.Sprintf("✅ Success: `%d`\n", userStats.CompletedTasks))
		text.WriteString(fmt.Sprintf("❌ Failed: `%d`\n", userStats.FailedTasks))
		text.WriteString(fmt.Sprintf("📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(userStats.TotalBandwidth))))
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func (s *BotService) formatWeeklyStats(stats []DailyStats) string {
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

		text.WriteString(fmt.Sprintf("%s `%s` : %d tasks\n",
			icon,
			day.Date.Format("02/01"),
			day.TotalTasks))
	}

	text.WriteString("\n📊 *SUMMARY*\n")
	text.WriteString(fmt.Sprintf("📥 Total: `%d` tasks\n", totalTasks))
	text.WriteString(fmt.Sprintf("✅ Completed: `%d`\n", totalCompleted))
	text.WriteString(fmt.Sprintf("📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(totalBandwidth))))

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func (s *BotService) formatMonthlyStats(stats []DailyStats) string {
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
		text.WriteString(fmt.Sprintf("Week %d: `%d` tasks\n└ `%s`\n",
			week,
			ws.tasks,
			utils.EscapeMarkdownV2(utils.FormatBytes(ws.bandwidth))))
	}

	text.WriteString("\n📊 *SUMMARY*\n")
	text.WriteString(fmt.Sprintf("📥 Total: `%d` tasks\n", totalTasks))
	text.WriteString(fmt.Sprintf("✅ Completed: `%d`\n", totalCompleted))
	text.WriteString(fmt.Sprintf("📈 Bandwidth: `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(totalBandwidth))))

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func (s *BotService) HandleStatsCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[1]
	var text string

	switch action {
	case "my":
		userStats, err := s.DB.GetUserStats(callback.From.ID)
		if err != nil {
			text = "❌ *Gagal mengambil statistik Anda*"
		} else {
			text = s.formatUserStatsDetailed(userStats)
		}

	case "today":
		dailyStats, err := s.DB.GetTodayStats()
		userDailyStats, _ := s.DB.GetUserTodayStats(callback.From.ID)
		if err != nil {
			text = "❌ *Gagal mengambil statistik hari ini*"
		} else {
			text = s.formatDailyStatsDetailed(dailyStats, "Hari Ini", userDailyStats)
		}

	case "weekly":
		weeklyStats, err := s.DB.GetWeeklyStats()
		if err != nil {
			text = "❌ *Gagal mengambil statistik mingguan*"
		} else {
			text = s.formatWeeklyStats(weeklyStats)
		}

	case "monthly":
		monthlyStats, err := s.DB.GetMonthlyStats()
		if err != nil {
			text = "❌ *Gagal mengambil statistik bulanan*"
		} else {
			text = s.formatMonthlyStats(monthlyStats)
		}

	case CmdRefresh:
		stats, _ := s.DB.GetBotStats()
		userStats, _ := s.DB.GetUserStats(callback.From.ID)
		dailyStats, _ := s.DB.GetTodayStats()
		userDailyStats, _ := s.DB.GetUserTodayStats(callback.From.ID)
		text = s.formatStatsMessage(stats, userStats, dailyStats, userDailyStats)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Statistics refreshed!"))

	case CmdClose:
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, _ = s.Bot.Request(deleteMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Closed"))
		return
	}

	if text != "" {
		var keyboard tgbotapi.InlineKeyboardMarkup
		if action == "refresh" {
			keyboard = s.getStatsKeyboard()
		} else {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "stats:refresh"),
					tgbotapi.NewInlineKeyboardButtonData("❌ Close", "stats:close"),
				),
			)
		}

		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		editMsg.ReplyMarkup = &keyboard
		_, _ = s.Bot.Send(editMsg)
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
