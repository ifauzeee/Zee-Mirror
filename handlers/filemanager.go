package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type DriveFile struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	Size     int64  `json:"Size"`
	MimeType string `json:"MimeType"`
	ModTime  string `json:"ModTime"`
	IsDir    bool   `json:"IsDir"`
	Path     string `json:"Path"`
}

func (s *BotService) HandleDriveList(message *tgbotapi.Message, args string, editMessageID int) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	slog.Info("Drive list request", "userID", message.From.ID, "args", args, "editMsgID", editMessageID)

	fullPath, relPath := s.resolveDrivePath(args)
	slog.Debug("Listing drive path", "fullPath", fullPath)

	sent, err := s.sendLoadingStatus(message.Chat.ID, editMessageID)
	if err != nil {
		slog.Error("Error preparing status message", "error", err)
		return
	}

	files, err := s.listDriveFiles(fullPath)
	if err != nil {
		s.handleDriveError(message.Chat.ID, sent.MessageID, "Gagal memuat daftar file", err)
		return
	}

	slog.Debug("Drive list found items", "count", len(files))

	if s.tryJumpToFileInfo(message, sent.MessageID, relPath, files) {
		return
	}

	s.renderDriveList(message.Chat.ID, sent.MessageID, relPath, files)
}

func (s *BotService) resolveDrivePath(args string) (string, string) {
	basePath := strings.TrimSuffix(s.TaskManager.RcloneDest, "/")
	fullPath := basePath

	if args != "" {
		if strings.HasPrefix(args, "/") {
			remoteName := strings.Split(s.TaskManager.RcloneDest, ":")[0]
			fullPath = remoteName + ":" + args
		} else {
			fullPath = basePath + "/" + strings.TrimPrefix(args, "/")
		}
	}
	fullPath = strings.TrimSuffix(fullPath, "/")

	baseRel := ""
	if strings.Contains(s.TaskManager.RcloneDest, ":") {
		parts := strings.SplitN(s.TaskManager.RcloneDest, ":", 2)
		if len(parts) > 1 {
			baseRel = strings.Trim(parts[1], "/")
		}
	}

	relPath := ""
	if strings.Contains(fullPath, ":") {
		parts := strings.SplitN(fullPath, ":", 2)
		if len(parts) > 1 {
			fullPathAfterRemote := strings.Trim(parts[1], "/")
			if baseRel != "" && (fullPathAfterRemote == baseRel || strings.HasPrefix(fullPathAfterRemote, baseRel+"/")) {
				relPath = strings.TrimPrefix(fullPathAfterRemote, baseRel)
				relPath = strings.TrimPrefix(relPath, "/")
			} else {
				relPath = "/" + fullPathAfterRemote
			}
		}
	}
	return fullPath, relPath
}

func (s *BotService) sendLoadingStatus(chatID int64, editMessageID int) (*tgbotapi.Message, error) {
	loadingText := "🔍 *Memuat daftar file\\.\\.\\.*"
	if editMessageID != 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, editMessageID, loadingText)
		editMsg.ParseMode = MarkdownV2
		m, err := s.Bot.Send(editMsg)
		if err == nil {
			return &m, nil
		}
	}

	statusMsg := tgbotapi.NewMessage(chatID, loadingText)
	statusMsg.ParseMode = MarkdownV2
	m, err := s.Bot.Send(statusMsg)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *BotService) handleDriveError(chatID int64, messageID int, title string, err error) {
	slog.Error("Drive operation error", "title", title, "error", err)
	errorText := fmt.Sprintf("❌ *%s*\n\nError: %s", title, utils.EscapeMarkdownV2(err.Error()))
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, errorText)
	editMsg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) tryJumpToFileInfo(message *tgbotapi.Message, messageID int, relPath string, files []DriveFile) bool {
	if len(files) == 1 && !files[0].IsDir {
		fileName := files[0].Name
		if relPath == fileName || strings.HasSuffix(relPath, "/"+fileName) {
			slog.Info("Target is a file, jumping to info view", "relPath", relPath, "messageID", messageID)
			s.handleDriveFileInfoDetailed(message.Chat.ID, messageID, relPath)
			return true
		}
	}
	return false
}

func (s *BotService) renderDriveList(chatID int64, messageID int, relPath string, files []DriveFile) {
	currentPathForUI := relPath
	if !strings.HasPrefix(currentPathForUI, "/") {
		currentPathForUI = "/" + currentPathForUI
	}

	text := s.formatDriveFileList(currentPathForUI, files)
	keyboard := s.buildDriveNavigationKeyboard(files, relPath)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard

	if _, err := s.Bot.Send(editMsg); err != nil {
		slog.Error("Error sending final list message", "error", err)
		s.handleDriveError(chatID, messageID, "Gagal menampilkan daftar file", err)
	}
}

func (s *BotService) listDriveFiles(path string) ([]DriveFile, error) {
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

func (s *BotService) formatDriveFileList(path string, files []DriveFile) string {
	var text strings.Builder

	text.WriteString("📂 *FILE MANAGER \\- GDRIVE*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")
	text.WriteString(fmt.Sprintf("📍 *PATH:* `%s`\n\n", utils.EscapeMarkdownV2(path)))

	if len(files) == 0 {
		text.WriteString("📭 _Folder ini kosong_\n")
	} else {
		folderCount := 0
		fileCount := 0

		var folders []string
		for _, f := range files {
			if f.IsDir {
				folderCount++
				folders = append(folders, fmt.Sprintf("%s `%s`", IconFolder, utils.EscapeMarkdownV2(utils.TruncateString(f.Name, 40))))
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
				icon := getFileIcon(f.Name)
				size := utils.FormatBytes(f.Size)
				fileList = append(fileList, fmt.Sprintf("%s `%s` \\(%s\\)",
					icon,
					utils.EscapeMarkdownV2(utils.TruncateString(f.Name, 35)),
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

func (s *BotService) buildDriveNavigationKeyboard(files []DriveFile, currentRelPath string) tgbotapi.InlineKeyboardMarkup {
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
					fmt.Sprintf("%d. %s %s", folderCount, IconFolder, utils.TruncateString(f.Name, 25)),
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
					fmt.Sprintf("%d. %s %s", fileCount, getFileIcon(f.Name), utils.TruncateString(f.Name, 25)),
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

func (s *BotService) buildSearchNavigationKeyboard(files []DriveFile) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	maxShow := 20
	count := 0
	for i, f := range files {
		if count >= maxShow {
			break
		}
		count++

		path := f.Path
		if path == "" {
			path = f.Name
		}

		action := "i"
		if f.IsDir {
			action = "c"
		}

		data := fmt.Sprintf("dr:%s:%s", action, path)
		if len(data) > 60 {
			id := s.StorePath(path)
			data = fmt.Sprintf("dr:%s:id:%s", action, id)
		}

		var icon string
		if f.IsDir {
			icon = IconFolder
		} else {
			icon = getFileIcon(f.Name)
		}

		name := filepath.Base(f.Name)

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%d. %s %s", i+1, icon, utils.TruncateString(name, 25)),
				data,
			),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "dr:x"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (s *BotService) HandleDriveMkdir(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}
	if args == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/mkdir nama_folder`")
		return
	}

	folderPath := s.TaskManager.RcloneDest + "/" + args

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configPath := s.TaskManager.ConfigDir + "/rclone.conf"
	cmd := exec.CommandContext(ctx, "rclone", "mkdir", folderPath, "--config", configPath)

	if err := cmd.Run(); err != nil {
		s.reply(message, fmt.Sprintf("❌ *Gagal membuat folder*\n\nError: %s", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	s.reply(message, fmt.Sprintf("✅ *Folder Berhasil Dibuat*\n\n📁 `%s`", utils.EscapeMarkdownV2(args)))
}

func (s *BotService) HandleDriveDelete(message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.reply(message, "❌ *Akses Ditolak*\nHanya Admin yang bisa menghapus file\\.")
		return
	}

	if args == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/rm nama_file_atau_folder`")
		return
	}

	targetPath := s.TaskManager.RcloneDest + "/" + args

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Ya, Hapus", fmt.Sprintf("dr:df:%s", args)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "dr:xf"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID,
		fmt.Sprintf("⚠️ *Konfirmasi Hapus*\n\nAnda yakin ingin menghapus:\n📁 `%s`\\?",
			utils.EscapeMarkdownV2(targetPath)))
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) HandleDriveMove(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/mv sumber tujuan`")
		return
	}

	source := s.TaskManager.RcloneDest + "/" + parts[0]
	dest := s.TaskManager.RcloneDest + "/" + parts[1]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	configPath := s.TaskManager.ConfigDir + "/rclone.conf"
	cmd := exec.CommandContext(ctx, "rclone", "moveto", source, dest, "--config", configPath)

	if err := cmd.Run(); err != nil {
		s.reply(message, fmt.Sprintf("❌ *Gagal memindahkan file*\n\nError: %s", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	s.reply(message, fmt.Sprintf("✅ *File Berhasil Dipindahkan*\n\n📄 `%s`\n➡️ `%s`",
		utils.EscapeMarkdownV2(parts[0]),
		utils.EscapeMarkdownV2(parts[1])))
}

func (s *BotService) HandleDriveShare(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	if args == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/share nama_file`")
		return
	}

	targetPath := s.TaskManager.RcloneDest + "/" + args

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, "🔗 *Generating share link\\.\\.\\.*")
	statusMsg.ParseMode = MarkdownV2
	sent, _ := s.Bot.Send(statusMsg)

	var link string
	if s.Config.IndexURL != "" {
		// Normalize paths
		targetPathSlash := strings.ReplaceAll(targetPath, "\\", "/")
		rcloneDestSlash := strings.ReplaceAll(s.TaskManager.RcloneDest, "\\", "/")
		rcloneDestSlash = strings.TrimRight(rcloneDestSlash, "/")

		var relPath string
		if strings.HasPrefix(targetPathSlash, rcloneDestSlash) {
			relPath = strings.TrimPrefix(targetPathSlash, rcloneDestSlash)
		} else {
			parts := strings.SplitN(targetPathSlash, ":", 2)
			if len(parts) > 1 {
				relPath = parts[1]
			} else {
				relPath = targetPathSlash
			}
		}

		relPath = strings.TrimLeft(relPath, "/")
		pathParts := strings.Split(relPath, "/")
		for i, part := range pathParts {
			pathParts[i] = url.PathEscape(part)
		}
		encodedPath := strings.Join(pathParts, "/")
		baseURL := strings.TrimRight(s.Config.IndexURL, "/")
		link = fmt.Sprintf("%s/%s", baseURL, encodedPath)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		configPath := s.TaskManager.ConfigDir + "/rclone.conf"
		cmd := exec.CommandContext(ctx, "rclone", "link", targetPath, "--config", configPath)
		output, err := cmd.Output()

		if err != nil {
			editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sent.MessageID,
				fmt.Sprintf("❌ *Gagal generate link*\n\nError: %s", utils.EscapeMarkdownV2(err.Error())))
			editMsg.ParseMode = MarkdownV2
			_, _ = s.Bot.Send(editMsg)
			return
		}
		link = strings.TrimSpace(string(output))
	}

	text := fmt.Sprintf("✅ *Share Link Generated*\n\n📄 File: `%s`\n🔗 Link: %s",
		utils.EscapeMarkdownV2(args),
		utils.EscapeMarkdownV2(link))

	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sent.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) HandleDriveSearch(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}
	if args == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/find keyword`")
		return
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, "🔍 *Mencari file\\.\\.\\.*")
	statusMsg.ParseMode = MarkdownV2
	sent, _ := s.Bot.Send(statusMsg)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	configPath := s.TaskManager.ConfigDir + "/rclone.conf"
	searchPath := s.TaskManager.RcloneDest

	slog.Info("Searching drive", "query", args, "path", searchPath)

	cmd := exec.CommandContext(ctx, "rclone", "lsjson", searchPath, "--config", configPath,
		"--include", "*"+args+"*", "-R", "--max-depth", "5", "--ignore-case")
	output, err := cmd.Output()

	if err != nil {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sent.MessageID,
			fmt.Sprintf("❌ *Gagal mencari file*\n\nError: %s", utils.EscapeMarkdownV2(err.Error())))
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)
		return
	}

	var rawFiles []DriveFile
	if err := json.Unmarshal(output, &rawFiles); err != nil {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sent.MessageID,
			"❌ *Gagal parsing hasil pencarian*")
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)
		return
	}

	var files []DriveFile
	queryLower := strings.ToLower(args)
	for _, f := range rawFiles {
		if strings.Contains(strings.ToLower(f.Name), queryLower) {
			files = append(files, f)
		}
	}

	text := s.formatSearchResults(args, files)
	keyboard := s.buildSearchNavigationKeyboard(files)
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sent.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) formatSearchResults(query string, files []DriveFile) string {
	var text strings.Builder

	text.WriteString("🔍 *HASIL PENCARIAN*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")
	text.WriteString(fmt.Sprintf("🔎 Query: `%s`\n\n", utils.EscapeMarkdownV2(query)))

	if len(files) == 0 {
		text.WriteString("📭 _Tidak ada file ditemukan_\n")
	} else {
		maxShow := 20
		if len(files) > maxShow {
			text.WriteString(fmt.Sprintf("📊 Menampilkan %d dari %d hasil\n\n", maxShow, len(files)))
		}

		text.WriteString("📂 *FILES FOUND*\n")

		for i, f := range files {
			if i >= maxShow {
				break
			}
			icon := IconFolder
			if !f.IsDir {
				icon = getFileIcon(f.Name)
			}
			size := ""
			if !f.IsDir {
				size = fmt.Sprintf(" \\(%s\\)", utils.EscapeMarkdownV2(utils.FormatBytes(f.Size)))
			}
			path := f.Path
			if path == "" {
				path = f.Name
			}
			text.WriteString(fmt.Sprintf("%d\\. %s `%s`%s\n", i+1, icon, utils.EscapeMarkdownV2(utils.TruncateString(path, 45)), size))
		}
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func (s *BotService) HandleDriveCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[1]
	switch action {
	case "cd", "c":
		s.handleCDCallback(callback, parts)
	case "home", "h":
		s.handleHomeCallback(callback)
	case "info", "i":
		s.handleInfoCallback(callback, parts)
	case CmdClose, "x":
		s.handleCloseCallback(callback)
	case "confirm_delete", "df":
		s.handleConfirmDeleteCallback(callback, parts)
	case "cancel_delete", "xf":
		s.handleCancelDeleteCallback(callback)
	}
}

func (s *BotService) resolveCallbackPath(parts []string) string {
	if len(parts) < 3 {
		return ""
	}
	fullPath := strings.Join(parts[2:], ":")
	if strings.HasPrefix(fullPath, "id:") {
		id := strings.TrimPrefix(fullPath, "id:")
		if cached, ok := s.GetPath(id); ok {
			return cached
		}
	}
	return fullPath
}

func (s *BotService) handleCDCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	fullPath := s.resolveCallbackPath(parts)
	msg := &tgbotapi.Message{
		Chat: callback.Message.Chat,
		From: callback.From,
	}
	s.HandleDriveList(msg, fullPath, callback.Message.MessageID)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "📂 Membuka folder..."))
}

func (s *BotService) handleHomeCallback(callback *tgbotapi.CallbackQuery) {
	msg := &tgbotapi.Message{
		Chat: callback.Message.Chat,
		From: callback.From,
	}
	s.HandleDriveList(msg, "", callback.Message.MessageID)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🏠 Kembali ke awal"))
}

func (s *BotService) handleInfoCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	fullPath := s.resolveCallbackPath(parts)
	s.handleDriveFileInfoDetailed(callback.Message.Chat.ID, callback.Message.MessageID, fullPath)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

func (s *BotService) handleCloseCallback(callback *tgbotapi.CallbackQuery) {
	deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
	_, _ = s.Bot.Request(deleteMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Closed"))
}

func (s *BotService) handleConfirmDeleteCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	fullPath := s.resolveCallbackPath(parts)
	s.executeDelete(callback, fullPath)
}

func (s *BotService) handleCancelDeleteCallback(callback *tgbotapi.CallbackQuery) {
	deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
	_, _ = s.Bot.Request(deleteMsg)
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Cancelled"))
}

func (s *BotService) executeDelete(callback *tgbotapi.CallbackQuery, fileName string) {
	if fileName == "" || fileName == "/" || fileName == "." {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Invalid path"))
		return
	}

	targetPath := s.TaskManager.RcloneDest + "/" + fileName
	slog.Info("Attempting to delete drive file", "path", targetPath)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	configPath := s.TaskManager.ConfigDir + "/rclone.conf"

	cmd := exec.CommandContext(ctx, "rclone", "deletefile", targetPath, "--config", configPath)
	_, err := cmd.CombinedOutput()

	if err != nil {
		slog.Warn("deletefile failed, trying purge", "error", err, "path", targetPath)

		cmdPurge := exec.CommandContext(ctx, "rclone", "purge", targetPath, "--config", configPath)
		outputPurge, errPurge := cmdPurge.CombinedOutput()

		if errPurge != nil {
			slog.Error("purge also failed", "error", errPurge, "path", targetPath)
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Failed to delete"))

			errMsg := fmt.Sprintf("❌ *Gagal menghapus file/folder*\n\nTarget: `%s`\nError: `%s`\nOutput: `%s`",
				utils.EscapeMarkdownV2(fileName),
				utils.EscapeMarkdownV2(errPurge.Error()),
				utils.EscapeMarkdownV2(utils.TruncateString(string(outputPurge), 100)))

			s.reply(callback.Message, errMsg)
			return
		}
		slog.Info("Successfully purged path", "path", targetPath)
	} else {
		slog.Info("Successfully deleted file", "path", targetPath)
	}

	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Deleted successfully"))

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID,
		fmt.Sprintf("✅ *File Dihapus*\n\n📁 `%s`", utils.EscapeMarkdownV2(fileName)))
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}

	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) handleDriveFileInfoDetailed(chatID int64, messageID int, relPath string) {
	targetPath, configPath := s.resolvePaths(relPath)

	slog.Debug("Detailed drive info request", "relPath", relPath, "targetPath", targetPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	file := s.getDriveMetadata(ctx, targetPath, configPath)
	cloudLink := s.getCloudLink(ctx, targetPath, configPath)

	text := s.buildFileInfoMessage(relPath, file)
	keyboard := s.buildFileInfoKeyboard(relPath, cloudLink)

	s.sendOrEditMessage(chatID, messageID, text, keyboard)
}

func (s *BotService) resolvePaths(relPath string) (string, string) {
	basePath := strings.TrimSuffix(s.TaskManager.RcloneDest, "/")
	cleanedRel := strings.Trim(relPath, "/")
	targetPath := basePath
	if cleanedRel != "" {
		targetPath = basePath + "/" + cleanedRel
	}
	configPath := s.TaskManager.ConfigDir + "/rclone.conf"
	return targetPath, configPath
}

func (s *BotService) getDriveMetadata(ctx context.Context, targetPath, configPath string) DriveFile {
	cmd := exec.CommandContext(ctx, "rclone", "lsjson", targetPath, "--config", configPath)
	output, err := cmd.Output()
	if err != nil {
		slog.Error("rclone lsjson failed for info", "error", err, "path", targetPath)
		return DriveFile{}
	}

	var files []DriveFile
	if errJson := json.Unmarshal(output, &files); errJson == nil && len(files) > 0 {
		file := files[0]
		slog.Debug("Drive metadata found", "name", file.Name, "size", file.Size)
		return file
	}
	return DriveFile{}
}

func (s *BotService) getCloudLink(ctx context.Context, targetPath, configPath string) string {
	if s.Config.IndexURL != "" {
		return s.generateIndexLink(targetPath)
	}

	linkCmd := exec.CommandContext(ctx, "rclone", "link", targetPath, "--config", configPath)
	linkOutput, _ := linkCmd.Output()
	return strings.TrimSpace(string(linkOutput))
}

func (s *BotService) generateIndexLink(targetPath string) string {
	targetPathSlash := strings.ReplaceAll(targetPath, "\\", "/")
	rcloneDestSlash := strings.ReplaceAll(s.TaskManager.RcloneDest, "\\", "/")
	rcloneDestSlash = strings.TrimRight(rcloneDestSlash, "/")

	var relPath string
	if strings.HasPrefix(targetPathSlash, rcloneDestSlash) {
		relPath = strings.TrimPrefix(targetPathSlash, rcloneDestSlash)
	} else {
		parts := strings.SplitN(targetPathSlash, ":", 2)
		if len(parts) > 1 {
			relPath = parts[1]
		} else {
			relPath = targetPathSlash
		}
	}

	relPath = strings.TrimLeft(relPath, "/")
	pathParts := strings.Split(relPath, "/")
	for i, part := range pathParts {
		pathParts[i] = url.PathEscape(part)
	}
	encodedPath := strings.Join(pathParts, "/")
	baseURL := strings.TrimRight(s.Config.IndexURL, "/")
	return fmt.Sprintf("%s/%s", baseURL, encodedPath)
}

func (s *BotService) buildFileInfoMessage(relPath string, file DriveFile) string {
	var text strings.Builder
	text.WriteString(fmt.Sprintf("%s *FILE INFORMATION*\n", getFileIcon(relPath)))
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")
	text.WriteString(fmt.Sprintf("📄 *Name:* `%s`\n", utils.EscapeMarkdownV2(filepath.Base(relPath))))

	if file.Name != "" {
		text.WriteString(fmt.Sprintf("📦 *Size:* `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(file.Size))))
		text.WriteString(fmt.Sprintf("🕒 *Modified:* `%s`\n", utils.EscapeMarkdownV2(file.ModTime)))
		text.WriteString(fmt.Sprintf("🏷️ *MimeType:* `%s`\n", utils.EscapeMarkdownV2(file.MimeType)))
	} else {
		text.WriteString("_Metadata limited atau rclone sedang sibuk_\n")
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return text.String()
}

func (s *BotService) buildFileInfoKeyboard(relPath, cloudLink string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	if cloudLink != "" && utils.IsValidURL(cloudLink) {
		btnText := BtnTextCloudLink
		if s.Config.IndexURL != "" {
			btnText = BtnTextIndexURL
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(btnText, cloudLink),
		))
	}

	dirPath := filepath.Dir(relPath)
	if dirPath == "." || dirPath == "/" {
		dirPath = ""
	}

	backData := fmt.Sprintf("dr:c:%s", dirPath)
	if len(backData) > 60 {
		id := s.StorePath(dirPath)
		backData = fmt.Sprintf("dr:c:id:%s", id)
	}

	deleteData := fmt.Sprintf("dr:df:%s", relPath)
	if len(deleteData) > 60 {
		id := s.StorePath(relPath)
		deleteData = fmt.Sprintf("dr:df:id:%s", id)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Folder", backData),
		tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete", deleteData),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "dr:x"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (s *BotService) sendOrEditMessage(chatID int64, messageID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard

	if _, err := s.Bot.Send(editMsg); err != nil {
		slog.Error("Failed to send/edit info message", "error", err)
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = MarkdownV2
		msg.ReplyMarkup = keyboard
		_, _ = s.Bot.Send(msg)
	}
}

func getFileIcon(filename string) string {
	lower := strings.ToLower(filename)

	switch {
	case strings.HasSuffix(lower, ".mp4"), strings.HasSuffix(lower, ".mkv"),
		strings.HasSuffix(lower, ".avi"), strings.HasSuffix(lower, ".mov"):
		return "🎬"
	case strings.HasSuffix(lower, ".mp3"), strings.HasSuffix(lower, ".flac"),
		strings.HasSuffix(lower, ".wav"), strings.HasSuffix(lower, ".aac"):
		return "🎵"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"),
		strings.HasSuffix(lower, ".png"), strings.HasSuffix(lower, ".gif"):
		return "🖼️"
	case strings.HasSuffix(lower, ".pdf"):
		return "📕"
	case strings.HasSuffix(lower, ".doc"), strings.HasSuffix(lower, ".docx"):
		return IconFile
	case strings.HasSuffix(lower, ".xls"), strings.HasSuffix(lower, ".xlsx"):
		return "📊"
	case strings.HasSuffix(lower, ".zip"), strings.HasSuffix(lower, ".rar"),
		strings.HasSuffix(lower, ".7z"), strings.HasSuffix(lower, ".tar"):
		return "🗜️"
	case strings.HasSuffix(lower, ".exe"), strings.HasSuffix(lower, ".msi"):
		return "⚙️"
	case strings.HasSuffix(lower, ".iso"):
		return "💿"
	default:
		return IconFile
	}
}
