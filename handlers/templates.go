package handlers

import (
	"fmt"
	"os"
	"time"
	"zee-mirror/internal/domain"
	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	LineSeparator    = "━━━━━━━━━━━━━━━━━━━━━━"
	CompactSeparator = "──────────────────────"
	RepoURL          = "https://github.com/ifauzeee/Zee-Mirror"
	UnknownSize      = "Unknown"
)

func buildTaskStatusText(snapshot domain.TaskSnapshot) string {
	var text string
	switch snapshot.Status {
	case domain.StatusCompleted:
		duration := calculateDuration(snapshot)
		sizeStr := determineSizeString(snapshot)

		text = fmt.Sprintf("%s\n\n"+
			"📄 *Name:* `%s`\n"+
			"📦 *Size:* `%s`\n"+
			"⏱ *Time:* `%s`\n"+
			"📁 *Path:* `%s`",
			i18n.MsgStatusCompleted,
			utils.EscapeMarkdownV2(snapshot.FileName),
			utils.EscapeMarkdownV2(sizeStr),
			utils.EscapeMarkdownV2(utils.FormatDuration(duration)),
			utils.EscapeMarkdownV2(snapshot.RemotePath))
	case domain.StatusFailed:
		text = fmt.Sprintf("%s\n📄 `%s`\nError: `%s`",
			i18n.MsgStatusFailed,
			utils.EscapeMarkdownV2(snapshot.FileName),
			utils.EscapeMarkdownV2(utils.TruncateString(snapshot.Error, 100)))
	default:
		return ""
	}
	return text
}

func calculateDuration(snapshot domain.TaskSnapshot) time.Duration {
	duration := snapshot.CompletedAt.Sub(snapshot.StartedAt)
	if snapshot.StartedAt.IsZero() {
		duration = snapshot.CompletedAt.Sub(snapshot.CreatedAt)
	}
	return duration
}

func determineSizeString(snapshot domain.TaskSnapshot) string {
	sizeStr := UnknownSize

	if snapshot.LocalPath != "" {
		if info, err := os.Stat(snapshot.LocalPath); err == nil && info.IsDir() {
			if dirSize, err := utils.CalculateDirSize(snapshot.LocalPath); err == nil && dirSize > 0 {
				sizeStr = utils.FormatBytes(dirSize)
			} else {
				if snapshot.TotalSize > 0 {
					sizeStr = utils.FormatBytes(snapshot.TotalSize)
				} else if snapshot.DownloadedSize > 0 {
					sizeStr = utils.FormatBytes(snapshot.DownloadedSize)
				}
			}
		} else {
			if snapshot.TotalSize > 0 {
				sizeStr = utils.FormatBytes(snapshot.TotalSize)
			} else if snapshot.DownloadedSize > 0 {
				sizeStr = utils.FormatBytes(snapshot.DownloadedSize)
			}
		}
	} else {
		if snapshot.TotalSize > 0 {
			sizeStr = utils.FormatBytes(snapshot.TotalSize)
		} else if snapshot.DownloadedSize > 0 {
			sizeStr = utils.FormatBytes(snapshot.DownloadedSize)
		}
	}

	return sizeStr
}

func ProfessionalMessage(title string, content string) string {
	return fmt.Sprintf("✨ *%s* ✨\n%s\n\n%s\n\n%s",
		utils.EscapeMarkdownV2(title),
		LineSeparator,
		content,
		LineSeparator)
}

func HelpDetailMessage(title, kegunaan, caraPakai, contoh, extra string) string {
	content := ""

	if kegunaan != "" {
		content += "📝 *KEGUNAAN*\n" + utils.EscapeMarkdownV2(kegunaan) + "\n\n"
	}

	if caraPakai != "" {
		content += "📌 *CARA PAKAI*\n" + utils.EscapeMarkdownV2(caraPakai) + "\n\n"
	}

	if contoh != "" {
		content += "💡 *CONTOH*\n" + utils.EscapeMarkdownV2(contoh)
	}

	if extra != "" {
		content += "\n\n" + utils.EscapeMarkdownV2(extra)
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

	totalSizeStr := utils.FormatBytes(taskSnapshot.TotalSize)
	if taskSnapshot.TotalSize == 0 {
		totalSizeStr = "Unknown"
	}

	return fmt.Sprintf(
		"🏷️ *ID:* `%s` • %s *%s*\n"+
			"%s\n"+
			"📄 *File:* `%s`\n"+
			"📦 *Size:* `%s / %s`\n"+
			"⚡ *Speed:* `%s` • ⏱️ *ETA:* `%s`\n"+
			"🚫 *Cancel:* /cancel\\_%s\n",
		utils.EscapeMarkdownV2Code(taskSnapshot.ID),
		emoji,
		utils.EscapeMarkdownV2(utils.FormatStatus(string(taskSnapshot.Status))),
		utils.EscapeMarkdownV2(bar),
		utils.EscapeMarkdownV2Code(utils.TruncateString(taskSnapshot.FileName, 35)),
		utils.EscapeMarkdownV2Code(utils.FormatBytes(processedSize)),
		utils.EscapeMarkdownV2Code(totalSizeStr),
		utils.EscapeMarkdownV2Code(utils.FormatSpeed(taskSnapshot.Speed)),
		utils.EscapeMarkdownV2Code(utils.FormatDuration(taskSnapshot.ETA)),
		utils.EscapeMarkdownV2(taskSnapshot.ID),
	)
}

func GetSuccessMessage(title, text string) string {
	return fmt.Sprintf("✅ *%s*\n%s\n\n%s",
		utils.EscapeMarkdownV2(title),
		LineSeparator,
		utils.EscapeMarkdownV2(text))
}

func GetErrorMessage(title, text string) string {
	return fmt.Sprintf("❌ *%s*\n%s\n\n%s",
		utils.EscapeMarkdownV2(title),
		LineSeparator,
		utils.EscapeMarkdownV2(text))
}

func GetStartKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("💻 Code Repo", RepoURL),
			tgbotapi.NewInlineKeyboardButtonData("❓ Help", "help:main"),
		),
	)
}

func GetHelpMainText() string {
	content := "Silakan pilih kategori bantuan di bawah untuk melihat detail fungsi dan cara penggunaan\\.\n\n" +
		"💡 *Klik tombol untuk membuka sub\\-menu\\.*"
	return ProfessionalMessage("PANDUAN BANTUAN", content)
}

func GetHelpKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📥 DOWNLOAD", "help:download"),
			tgbotapi.NewInlineKeyboardButtonData("📊 MONITOR", "help:monitor"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📁 FILES", "help:files"),
			tgbotapi.NewInlineKeyboardButtonData("🎵 MEDIA", "help:media"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 TASK", "help:task"),
			tgbotapi.NewInlineKeyboardButtonData("💾 STORAGE", "help:storage"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👑 ADMIN", "help:admin"),
			tgbotapi.NewInlineKeyboardButtonData("🔧 RECOVERY", "help:recovery"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ SETTINGS", "help:settings"),
			tgbotapi.NewInlineKeyboardButtonData("📋 ALL COMMANDS", "help:all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 HOME", "help:back"),
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "help:close"),
		),
	)
}
