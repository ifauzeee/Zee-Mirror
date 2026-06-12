package service

import (
	"fmt"
	"os"
	"strings"
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

func buildTaskStatusText(lang string, snapshot domain.TaskSnapshot) string {
	var text string
	switch snapshot.Status {
	case domain.StatusCompleted:
		duration := calculateDuration(snapshot)
		sizeStr := determineSizeString(snapshot)

		playlistLine := ""
		if snapshot.PlaylistCount > 0 {
			playlistLine = fmt.Sprintf("\n📋 *Playlist:* \\[%d/%d\\]", snapshot.PlaylistIndex, snapshot.PlaylistCount)
		}

		text = fmt.Sprintf("%s\n\n"+
			"📄 *Name:* `%s`\n"+
			"📦 *Size:* `%s`\n"+
			"⏱ *Time:* `%s`\n"+
			"📁 *Path:* `%s`%s",
			i18n.T(lang, "status_completed"),
			utils.EscapeMarkdownV2Code(snapshot.FileName),
			utils.EscapeMarkdownV2Code(sizeStr),
			utils.EscapeMarkdownV2Code(utils.FormatDuration(duration)),
			utils.EscapeMarkdownV2Code(snapshot.RemotePath),
			playlistLine)
	case domain.StatusFailed:
		text = fmt.Sprintf("%s\n📄 `%s`\nError: `%s`",
			i18n.T(lang, "status_failed"),
			utils.EscapeMarkdownV2Code(snapshot.FileName),
			utils.EscapeMarkdownV2Code(utils.TruncateString(snapshot.Error, 100)))
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

func GetWelcomeMessage(lang, userName string) string {
	content := i18n.T(lang, "welcome_content", utils.EscapeMarkdownV2(userName))
	return ProfessionalMessage(i18n.T(lang, "welcome_title"), content)
}

func GetStatusHeader() string {
	return fmt.Sprintf("📊 *STATUS TASK AKTIF*\n%s\n\n", LineSeparator)
}

func FormatTaskProfessional(lang string, taskSnapshot domain.TaskSnapshot) string {
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

	msgLine := ""
	if taskSnapshot.ProcessingMessage != "" {
		msgLine = fmt.Sprintf("💬 *Status:* _%s_\n", utils.EscapeMarkdownV2(taskSnapshot.ProcessingMessage))
	} else if taskSnapshot.Status == domain.StatusDownloading && taskSnapshot.Progress >= 100 {
		msgLine = "💬 *Status:* _Memproses file..._\n"
	}

	playlistInfo := ""
	if taskSnapshot.PlaylistCount > 0 {
		playlistInfo = fmt.Sprintf("📋 *Playlist:* \\[%d/%d\\]\n", taskSnapshot.PlaylistIndex, taskSnapshot.PlaylistCount)
	}

	return fmt.Sprintf(
		"🏷️ *ID:* `%s` • %s *%s*\n"+
			"%s\n"+
			"%s"+
			"%s"+
			"📄 *File:* `%s`\n"+
			"📦 *Size:* `%s / %s`\n"+
			"⚡ *Speed:* `%s` • ⏱️ *ETA:* `%s`\n"+
			"🚫 *Cancel:* /cancel\\_%s\n",
		utils.EscapeMarkdownV2Code(taskSnapshot.ID),
		emoji,
		utils.EscapeMarkdownV2(i18n.T(lang, strings.ToLower(string(taskSnapshot.Status)))),
		utils.EscapeMarkdownV2(bar),
		msgLine,
		playlistInfo,
		utils.EscapeMarkdownV2Code(utils.TruncateString(taskSnapshot.FileName, 35)),
		utils.EscapeMarkdownV2Code(utils.FormatBytes(processedSize)),
		utils.EscapeMarkdownV2Code(totalSizeStr),
		utils.EscapeMarkdownV2Code(utils.FormatSpeed(taskSnapshot.Speed)),
		utils.EscapeMarkdownV2Code(utils.FormatDuration(taskSnapshot.ETA)),
		utils.EscapeMarkdownV2(taskSnapshot.ID),
	)
}

func GetStartKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_menu"), "help:main"),
		),
	)
}

func GetHelpMainText(lang string) string {
	content := i18n.T(lang, "help_content")
	return ProfessionalMessage(i18n.T(lang, "help_title"), content)
}

func GetHelpKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_download"), "help:download"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_monitor"), "help:monitor"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_files"), "help:files"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_media"), "help:media"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_task"), "help:task"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_storage"), "help:storage"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_admin"), "help:admin"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_recovery"), "help:recovery"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_settings"), "help:settings"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_all"), "help:all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_home"), "help:back"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_close"), "help:close"),
		),
	)
}

func GetDownloadHelpKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📥 General", "help:sub_dl_general"),
			tgbotapi.NewInlineKeyboardButtonData("🎬 Video", "help:sub_dl_video"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧲 Torrent", "help:sub_dl_torrent"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Advanced", "help:sub_dl_adv"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_back"), "help:main"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_close"), "help:close"),
		),
	)
}

func GetSettingsKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💾 Storage", "settings:cat_storage"),
			tgbotapi.NewInlineKeyboardButtonData("👤 User", "settings:cat_user"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Global", "settings:cat_global"),
			tgbotapi.NewInlineKeyboardButtonData("🛡️ Admin", "settings:cat_admin"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_back"), "help:main"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "help_close"), "help:close"),
		),
	)
}
