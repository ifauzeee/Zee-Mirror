package handlers

import (
	"fmt"
	"zee-mirror/internal/domain"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	LineSeparator    = "━━━━━━━━━━━━━━━━━━━━━━"
	CompactSeparator = "──────────────────────"
	RepoURL          = "https://github.com/ifauzeee/Zee-Mirror"
)

func ProfessionalMessage(title string, content string) string {
	return fmt.Sprintf("✨ *%s* ✨\n%s\n\n%s\n\n%s",
		utils.EscapeMarkdownV2(title),
		LineSeparator,
		content,
		LineSeparator)
}

// HelpDetailMessage formats help detail with sections
func HelpDetailMessage(title, kegunaan, caraPakai, contoh, extra string) string {
	content := ""

	if kegunaan != "" {
		content += "📝 *KEGUNAAN*\n" + kegunaan + "\n\n"
	}

	if caraPakai != "" {
		content += "📌 *CARA PAKAI*\n" + caraPakai + "\n\n"
	}

	if contoh != "" {
		content += "💡 *CONTOH*\n" + contoh
	}

	if extra != "" {
		content += "\n\n" + extra
	}

	return ProfessionalMessage(title, content)
}

func GetWelcomeMessage(userName string) string {
	content := fmt.Sprintf("👋 Hai, *%s*\\!\n\n"+
		"Selamat datang di *ZEE\\-MIRROR*\\.\n"+
		"Bot serbaguna untuk Mirror, Leech, dan Media Tools\\.\n\n"+
		"🚀 *Didesain untuk kecepatan dan kemudahan\\.*",
		utils.EscapeMarkdownV2(userName))

	return ProfessionalMessage("ZEE-MIRROR BOT", content)
}

func GetStatusHeader() string {
	return fmt.Sprintf("📊 *STATUS TASK AKTIF*\n%s\n\n", LineSeparator)
}

func FormatTaskProfessional(taskSnapshot domain.TaskSnapshot) string {
	emoji := utils.StatusEmoji(string(taskSnapshot.Status))
	bar := utils.ProgressBar(taskSnapshot.Progress, 10)

	processedSize := taskSnapshot.DownloadedSize
	if taskSnapshot.Status == domain.StatusUploading {
		processedSize = taskSnapshot.UploadedSize
	}

	return fmt.Sprintf(
		"🔹 *ID:* `%s` \\| %s *%s*\n"+
			"%s\n"+
			"📄 *File:* `%s`\n"+
			"📦 *Size:* `%s / %s`\n"+
			"⚡ *Speed:* `%s` \\| ⏱️ *ETA:* `%s`\n"+
			"🚫 *Cancel:* /cancel\\_%s\n",
		taskSnapshot.ID,
		emoji,
		utils.EscapeMarkdownV2(utils.FormatStatus(string(taskSnapshot.Status))),
		utils.EscapeMarkdownV2(bar),
		utils.EscapeMarkdownV2(utils.TruncateString(taskSnapshot.FileName, 35)),
		utils.EscapeMarkdownV2(utils.FormatBytes(processedSize)),
		utils.EscapeMarkdownV2(utils.FormatBytes(taskSnapshot.TotalSize)),
		utils.EscapeMarkdownV2(utils.FormatSpeed(taskSnapshot.Speed)),
		utils.EscapeMarkdownV2(utils.FormatDuration(taskSnapshot.ETA)),
		utils.EscapeMarkdownV2(taskSnapshot.ID),
	)
}

func GetSuccessMessage(title, text string) string {
	return fmt.Sprintf("✅ *%s*\n%s\n\n%s",
		utils.EscapeMarkdownV2(title),
		LineSeparator,
		text)
}

func GetErrorMessage(title, text string) string {
	return fmt.Sprintf("❌ *%s*\n%s\n\n%s",
		utils.EscapeMarkdownV2(title),
		LineSeparator,
		text)
}

func GetStartKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("💻 Code Repo", RepoURL),
			tgbotapi.NewInlineKeyboardButtonData("❓ Help", "help:main"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Close", "dashboard:close"),
		),
	)
}
