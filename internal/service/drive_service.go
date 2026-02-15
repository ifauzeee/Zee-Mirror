package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) ListDriveFiles(path string) ([]DriveFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configPath := s.TaskManager.ConfigDir + "/rclone.conf"
	args := []string{
		"lsjson",
		path,
		"--config", configPath,
		"--no-modtime",
	}

	cmd := exec.CommandContext(ctx, "rclone", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rclone lsjson failed: %v", err)
	}

	var files []DriveFile
	if err := json.Unmarshal(output, &files); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return files, nil
}

func (s *BotService) FormatDriveFileList(path string, files []DriveFile) string {
	var text strings.Builder

	text.WriteString("📂 *FILE MANAGER \\- GDRIVE*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")
	text.WriteString(fmt.Sprintf("📍 *PATH:* `%s`\n\n", utils.EscapeMarkdownV2Code(path)))

	if len(files) == 0 {
		text.WriteString("📭 _Folder ini kosong_\n")
	} else {
		folderCount := 0
		fileCount := 0

		var folders []string
		for _, f := range files {
			if f.IsDir {
				folderCount++
				folders = append(folders, fmt.Sprintf("📁 `%s`", utils.EscapeMarkdownV2Code(utils.TruncateString(f.Name, 40))))
			}
		}

		if len(folders) > 0 {
			text.WriteString("📁 *FOLDERS*\n")
			for _, f := range folders {
				text.WriteString(f + "\n")
			}
			text.WriteString("\n")
		}

		var fileList []string
		for _, f := range files {
			if !f.IsDir {
				fileCount++
				size := utils.FormatBytes(f.Size)
				fileList = append(fileList, fmt.Sprintf("📄 `%s` \\(%s\\)",
					utils.EscapeMarkdownV2Code(utils.TruncateString(f.Name, 35)),
					utils.EscapeMarkdownV2(size)))
			}
		}

		if len(fileList) > 0 {
			text.WriteString("📄 *FILES*\n")
			for _, f := range fileList {
				text.WriteString(f + "\n")
			}
			text.WriteString("\n")
		}

		text.WriteString("📊 *SUMMARY*\n")
		text.WriteString(fmt.Sprintf("📂 Folders: `%d`\n", folderCount))
		text.WriteString(fmt.Sprintf("📄 Files: `%d`\n", fileCount))
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func (s *BotService) BuildDriveNavigationKeyboard(files []DriveFile, currentRelPath string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	folderCount := 0
	for _, f := range files {
		if f.IsDir && folderCount < 10 {
			folderCount++
			nextPath := f.Name
			if currentRelPath != "" {
				nextPath = strings.TrimSuffix(currentRelPath, "/") + "/" + f.Name
			}
			data := fmt.Sprintf("dr:c:%s", nextPath)
			if len(data) > 60 {
				id := s.StorePath(nextPath)
				data = fmt.Sprintf("dr:c:id:%s", id)
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("%d. 📁 %s", folderCount, utils.TruncateString(f.Name, 25)),
					data,
				),
			))
		}
	}

	fileCount := 0
	for _, f := range files {
		if !f.IsDir && fileCount < 10 {
			fileCount++
			filePath := f.Name
			if currentRelPath != "" {
				filePath = strings.TrimSuffix(currentRelPath, "/") + "/" + f.Name
			}
			data := fmt.Sprintf("dr:i:%s", filePath)
			if len(data) > 60 {
				id := s.StorePath(filePath)
				data = fmt.Sprintf("dr:i:id:%s", id)
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("%d. 📄 %s", fileCount, utils.TruncateString(f.Name, 25)),
					data,
				),
			))
		}
	}

	var navButtons []tgbotapi.InlineKeyboardButton
	refreshData := fmt.Sprintf("dr:c:%s", currentRelPath)
	if len(refreshData) > 60 {
		id := s.StorePath(currentRelPath)
		refreshData = fmt.Sprintf("dr:c:id:%s", id)
	}
	navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", refreshData))
	navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("🏠 Home", "dr:h"))
	navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "dr:x"))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(navButtons...))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}
