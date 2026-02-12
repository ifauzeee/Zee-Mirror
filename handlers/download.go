package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"zee-mirror/internal/downloader"
	"zee-mirror/internal/organizer"
	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

const (
	BtnTextCloudLink = "☁️ Cloud Link"
	BtnTextIndexURL  = "🌐 Index URL"
)

func (s *BotService) HandleMirror(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name := utils.ParseFlags(args)
	var fileName string

	if name != "" {
		fileName = name
	}

	if message.ReplyToMessage != nil {
		fileID, replyName := s.extractFileFromReply(message.ReplyToMessage)
		if fileID != "" {
			go s.handleTelegramFileDownload(message, fileID, replyName, zip, unzip, password, quality)
			return
		}
	}

	if url != "" {
		if fileName == "" {
			fileName = utils.GetFileNameFromURL(url)
			if isGenericName(fileName) {
				resolvedName := utils.ResolveFileName(url)
				if resolvedName != "" {
					fileName = resolvedName
					slog.Debug("Resolved filename from header", "filename", fileName)
				}
			}
		}

		if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
			s.handleYTDLPGeneric(message, args, TypeYTDLP)
			return
		}
		replyID := 0
		if message.ReplyToMessage != nil {
			replyID = message.ReplyToMessage.MessageID
		}
		task, err := s.TaskManager.CreateTask(TypeMirror, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0)
		if err != nil {
			s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
			return
		}
		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.handleAutoDelete(task)
		slog.Info("Mirror task created", "taskID", task.ID, "url", url)
		return
	}

	lang := s.GetUserLanguage(message.From.ID)
	msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "reply_required"))
	msg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) extractFileFromReply(reply *tgbotapi.Message) (string, string) {
	var fileID, fileName string

	switch {
	case reply.Document != nil:
		fileID = reply.Document.FileID
		fileName = reply.Document.FileName
	case reply.Video != nil:
		fileID = reply.Video.FileID
		fileName = reply.Video.FileName
		if fileName == "" {
			fileName = fmt.Sprintf("video_%d.mp4", time.Now().Unix())
		}
	case reply.Audio != nil:
		fileID = reply.Audio.FileID
		fileName = reply.Audio.FileName
		if fileName == "" {
			fileName = fmt.Sprintf("audio_%d.mp3", time.Now().Unix())
		}
	case reply.Voice != nil:
		fileID = reply.Voice.FileID
		fileName = fmt.Sprintf("voice_%d.ogg", time.Now().Unix())
	case reply.VideoNote != nil:
		fileID = reply.VideoNote.FileID
		fileName = fmt.Sprintf("video_note_%d.mp4", time.Now().Unix())
	case reply.Animation != nil:
		fileID = reply.Animation.FileID
		fileName = reply.Animation.FileName
		if fileName == "" {
			fileName = fmt.Sprintf("animation_%d.mp4", time.Now().Unix())
		}
	case len(reply.Photo) > 0:
		photo := reply.Photo[len(reply.Photo)-1]
		fileID = photo.FileID
		fileName = fmt.Sprintf("photo_%d.jpg", time.Now().Unix())
	}

	return fileID, fileName
}

func (s *BotService) HandleLeech(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name := utils.ParseFlags(args)
	if url == "" {
		url = utils.ExtractMagnetFromText(args)
	}
	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_url"))
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		s.handleYTDLPGeneric(message, args, TypeYTDLPLeech)
		return
	}

	fileName := name
	if fileName == "" {
		fileName = utils.GetFileNameFromURL(url)
	}
	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}
	task, err := s.TaskManager.CreateTask(TypeLeech, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0)
	if err != nil {
		s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.handleAutoDelete(task)
	slog.Info("Leech task created", "taskID", task.ID, "url", url)
}

func (s *BotService) HandleYTDLP(message *tgbotapi.Message, args string) {
	s.handleYTDLPGeneric(message, args, TypeYTDLP)
}

func (s *BotService) HandleYTDLPLeech(message *tgbotapi.Message, args string) {
	s.handleYTDLPGeneric(message, args, TypeYTDLPLeech)
}

func (s *BotService) handleYTDLPGeneric(message *tgbotapi.Message, args string, taskType TaskType) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, _, password, quality, name := utils.ParseFlags(args)
	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" && message.ReplyToMessage != nil {
		replyText := message.ReplyToMessage.Text
		if replyText == "" {
			replyText = message.ReplyToMessage.Caption
		}
		url = utils.ExtractURLFromText(replyText)
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_url"))
		msg.ParseMode = MarkdownV2
		if sentMsg, err := s.Bot.Send(msg); err == nil {
			s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 30*time.Second)
		}
		return
	}

	if quality == "" && (strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")) {
		s.showYTDLPQualityMenu(message, url, name, zip, password, taskType)
		return
	}

	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}

	fileName := name
	if fileName == "" {
		fileName = utils.GetFileNameFromURL(url)
	}

	task, err := s.TaskManager.CreateTask(taskType, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, false, password, quality, 0)
	if err != nil {
		s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.handleAutoDelete(task)
	slog.Info("YTDLP task created", "taskID", task.ID, "type", taskType, "url", url, "fileName", fileName)
}

func (s *BotService) isYTDLPPlaylist(url string) bool {
	return (strings.Contains(url, "/@") ||
		strings.Contains(url, "/channel/") ||
		strings.Contains(url, "/c/") ||
		strings.Contains(url, "/user/") ||
		strings.Contains(url, "/playlist?") ||
		strings.Contains(url, "&list=")) && !strings.Contains(url, "watch?v=")
}

func (s *BotService) showYTDLPQualityMenu(message *tgbotapi.Message, url, name string, zip bool, password string, taskType TaskType) {
	lang := s.GetUserLanguage(message.From.ID)
	if s.isYTDLPPlaylist(url) {
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "ytdlp_playlist_error"))
		msg.ParseMode = MarkdownV2
		sentMsg, _ := s.Bot.Send(msg)
		s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 15*time.Second)
		return
	}

	statusMsg, _ := s.Bot.Send(tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "ytdlp_analysis")))

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	resMap, err := s.TaskManager.YTDLPEngine.GetFormats(ctx, url)
	if err != nil {
		slog.Error("YTDLP analysis failed", "error", err)
		s.editStatusMessage(statusMsg.Chat.ID, statusMsg.MessageID, i18n.T(lang, "ytdlp_analysis_failed", utils.EscapeMarkdownV2(err.Error())))
		s.AutoDeleteMessage(statusMsg.Chat.ID, statusMsg.MessageID, 20*time.Second)
		return
	}

	sortedHeights := s.getSortedHeights(resMap)

	sessionID := s.createYTDLPSession(url, name, zip, password, taskType)
	keyboard := s.buildYTDLPKeyboard(sortedHeights, resMap, sessionID)

	text := i18n.T(lang, "ytdlp_select_quality")
	if len(sortedHeights) == 0 {
		text = i18n.T(lang, "ytdlp_no_resolution")
	}

	editMsg := tgbotapi.NewEditMessageText(statusMsg.Chat.ID, statusMsg.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) getSortedHeights(resMap map[int]float64) []int {
	var heights []int
	for h := range resMap {
		heights = append(heights, h)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(heights)))
	return heights
}

func (s *BotService) createYTDLPSession(url, name string, zip bool, password string, taskType TaskType) string {
	sessionID := uuid.New().String()[:8]
	s.TaskManager.Mu.Lock()
	defer s.TaskManager.Mu.Unlock()
	s.TaskManager.YTDLPSessions[sessionID] = &YTDLPSession{
		URL:      url,
		FileName: name,
		Zip:      zip,
		Password: password,
		Type:     taskType,
	}
	return sessionID
}

func (s *BotService) buildYTDLPKeyboard(sortedHeights []int, resMap map[int]float64, sessionID string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(sortedHeights); i += 2 {
		var row []tgbotapi.InlineKeyboardButton

		h1 := sortedHeights[i]
		label1 := formatQualityLabel(h1, resMap[h1])
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label1, fmt.Sprintf("ytdlp_q:%d:%s", h1, sessionID)))

		if i+1 < len(sortedHeights) {
			h2 := sortedHeights[i+1]
			label2 := formatQualityLabel(h2, resMap[h2])
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(label2, fmt.Sprintf("ytdlp_q:%d:%s", h2, sessionID)))
		}
		rows = append(rows, row)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🚀 Kualitas Terbaik", fmt.Sprintf("ytdlp_q:best:%s", sessionID)),
	))
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func formatQualityLabel(height int, fps float64) string {
	var label string
	switch height {
	case 4320:
		label = "8K (4320p)"
	case 2160:
		label = "4K (2160p)"
	case 1440:
		label = "2K (1440p)"
	default:
		label = fmt.Sprintf("%dp", height)
	}

	if fps > 30 {
		label += fmt.Sprintf(" %dfps", int(fps))
	}
	return label
}

func (s *BotService) HandleYTDLPQualityCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Sesi kadaluarsa"))
		return
	}

	quality := parts[1]
	sessionID := parts[2]

	if quality == "best" {
		quality = ""
	}

	s.TaskManager.Mu.Lock()
	session, exists := s.TaskManager.YTDLPSessions[sessionID]
	if exists {
		delete(s.TaskManager.YTDLPSessions, sessionID)
	}
	s.TaskManager.Mu.Unlock()

	if !exists {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Sesi tidak ditemukan"))
		return
	}

	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Memulai download..."))

	s.TaskManager.Mu.Lock()
	s.TaskManager.LastStatusMsg[callback.Message.Chat.ID] = callback.Message.MessageID
	s.TaskManager.Mu.Unlock()

	replyID := 0
	if callback.Message.ReplyToMessage != nil {
		replyID = callback.Message.ReplyToMessage.MessageID
	}
	fileName := session.FileName
	if fileName == "" {
		fileName = utils.GetFileNameFromURL(session.URL)
	}

	if err := s.CheckQuota(callback.From.ID); err != nil {
		_, _ = s.Bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, GetErrorMessage("QUOTA EXCEEDED", err.Error())))
		return
	}

	task, err := s.TaskManager.CreateTask(session.Type, session.URL, fileName, callback.Message.Chat.ID, callback.Message.MessageID, replyID, callback.From.ID, session.Zip, false, session.Password, quality, 0)
	if err != nil {
		s.handleCreateTaskError(callback.Message.Chat.ID, callback.Message.MessageID, err)
		return
	}
	s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
	slog.Info("YTDLP task created from callback", "taskID", task.ID, "type", session.Type, "quality", quality)
}

func (s *BotService) HandleTorrent(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name := utils.ParseFlags(args)
	if message.ReplyToMessage != nil && message.ReplyToMessage.Document != nil {
		fileID := message.ReplyToMessage.Document.FileID
		fileName := message.ReplyToMessage.Document.FileName

		if strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
			go s.handleTelegramFileDownload(message, fileID, fileName, zip, unzip, password, quality)
			return
		}
	}

	if url == "" {
		url = utils.ExtractMagnetFromText(args)
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_magnet"))
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}

	s.showTorrentSelectionMenu(message, url, name, zip, unzip, password, replyID)
}

func (s *BotService) showTorrentSelectionMenu(message *tgbotapi.Message, url, name string, zip, unzip bool, password string, replyID int) {
	sessionID := s.createTorrentSession(url, name, zip, unzip, password, message.Chat.ID, message.MessageID, replyID, message.From.ID)

	baseURL := s.Config.DashboardURL
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	dashboardURL := ""
	if strings.Contains(baseURL, "localhost") || strings.Contains(baseURL, "127.0.0.1") {
		dashboardURL = fmt.Sprintf("%s:%d/torrent-select/%s", baseURL, s.Config.DashboardPort, sessionID)
	} else {
		dashboardURL = fmt.Sprintf("%s/torrent-select/%s", baseURL, sessionID)
	}

	lang := s.GetUserLanguage(message.From.ID)
	text := i18n.T(lang, "torrent_menu_text")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Select All", fmt.Sprintf("torrent_sel:all:%s", sessionID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📋 Select Files (Web)", dashboardURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", fmt.Sprintf("torrent_sel:cancel:%s", sessionID)),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard

	sentMsg, err := s.Bot.Send(msg)
	if err != nil {
		slog.Error("Failed to send torrent selection menu", "error", err)
		return
	}

	s.TaskManager.Mu.Lock()
	if session, exists := s.TaskManager.TorrentSessions[sessionID]; exists {
		session.MessageID = sentMsg.MessageID
	}
	s.TaskManager.Mu.Unlock()

	slog.Info("Torrent selection menu shown", "sessionID", sessionID, "url", url)
}

func (s *BotService) createTorrentSession(url, name string, zip, unzip bool, password string, chatID int64, msgID, replyID int, userID int64) string {
	sessionID := uuid.New().String()[:8]

	s.TaskManager.Mu.Lock()
	defer s.TaskManager.Mu.Unlock()

	s.TaskManager.TorrentSessions[sessionID] = &TorrentSession{
		URL:           url,
		FileName:      name,
		Zip:           zip,
		Unzip:         unzip,
		Password:      password,
		SelectedFiles: nil,
		ChatID:        chatID,
		MessageID:     msgID,
		ReplyID:       replyID,
		UserID:        userID,
		IsFetching:    true,
	}

	go s.fetchTorrentMetadataBackground(sessionID)

	go func() {
		time.Sleep(1 * time.Hour)
		s.TaskManager.Mu.Lock()
		delete(s.TaskManager.TorrentSessions, sessionID)
		s.TaskManager.Mu.Unlock()
	}()

	return sessionID
}

func (s *BotService) HandleTorrentSelectionCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Sesi tidak valid"))
		return
	}

	action := parts[1]
	sessionID := parts[2]

	s.TaskManager.Mu.Lock()
	session, exists := s.TaskManager.TorrentSessions[sessionID]
	if !exists {
		s.TaskManager.Mu.Unlock()
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Sesi tidak ditemukan atau sudah kadaluarsa"))
		return
	}

	switch action {
	case "all":
		delete(s.TaskManager.TorrentSessions, sessionID)
		s.TaskManager.Mu.Unlock()

		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Memulai download semua file..."))

		_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))

		confirmMsg := tgbotapi.NewMessage(callback.Message.Chat.ID, "✅ *Download dimulai*\n\nMendownload semua file dalam torrent\\.\\.\\.")
		confirmMsg.ParseMode = MarkdownV2
		sentMsg, _ := s.Bot.Send(confirmMsg)
		statusMsgID := sentMsg.MessageID

		fileName := session.FileName
		if fileName == "" {
			fileName = utils.GetFileNameFromURL(session.URL)
			if fileName == "unknown_file" {
				fileName = "torrent_download"
			}
		}

		task, err := s.TaskManager.CreateTask(TypeTorrent, session.URL, fileName, session.ChatID, statusMsgID, session.ReplyID, session.UserID, session.Zip, session.Unzip, session.Password, "", 0)
		if err != nil {
			s.handleCreateTaskError(session.ChatID, statusMsgID, err)
			return
		}
		s.UpdateSharedDashboard(session.ChatID, true)
		s.handleAutoDelete(task)
		slog.Info("Torrent task created (all files)", "taskID", task.ID, "url", session.URL)

	case "cancel":
		delete(s.TaskManager.TorrentSessions, sessionID)
		s.TaskManager.Mu.Unlock()

		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Dibatalkan"))

		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, _ = s.Bot.Request(deleteMsg)

		slog.Info("Torrent session cancelled", "sessionID", sessionID)

	default:
		s.TaskManager.Mu.Unlock()
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Aksi tidak valid"))
	}
}

func (s *BotService) StartTorrentWithSelectedFiles(sessionID string, selectedFiles []int) error {
	s.TaskManager.Mu.Lock()
	session, exists := s.TaskManager.TorrentSessions[sessionID]
	if !exists {
		s.TaskManager.Mu.Unlock()
		return fmt.Errorf("session not found")
	}

	delete(s.TaskManager.TorrentSessions, sessionID)
	s.TaskManager.Mu.Unlock()

	url := session.URL
	if len(selectedFiles) > 0 {
		selectFileStr := ""
		for i, idx := range selectedFiles {
			if i > 0 {
				selectFileStr += ","
			}
			selectFileStr += fmt.Sprintf("%d", idx)
		}
		url = url + "#select=" + selectFileStr
	}

	fileName := session.FileName
	if fileName == "" {
		fileName = utils.GetFileNameFromURL(session.URL)
		if fileName == "unknown_file" {
			fileName = "torrent_download"
		}
	}

	_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(session.ChatID, session.MessageID))

	confirmText := fmt.Sprintf("✅ *Download dimulai*\n\nMendownload %d file yang dipilih\\.\\.\\.", len(selectedFiles))
	msg := tgbotapi.NewMessage(session.ChatID, confirmText)
	msg.ParseMode = MarkdownV2
	sentMsg, _ := s.Bot.Send(msg)

	task, err := s.TaskManager.CreateTask(TypeTorrent, url, fileName, session.ChatID, sentMsg.MessageID, session.ReplyID, session.UserID, session.Zip, session.Unzip, session.Password, "", 0)
	if err != nil {
		s.handleCreateTaskError(session.ChatID, sentMsg.MessageID, err)
		return nil
	}
	s.UpdateSharedDashboard(session.ChatID, true)
	s.handleAutoDelete(task)
	slog.Info("Torrent task created (selected files)", "taskID", task.ID, "selectedFiles", selectedFiles)

	return nil
}

func (s *BotService) handleTelegramFileDownload(message *tgbotapi.Message, fileID, fileName string, zip, unzip bool, password, quality string) {
	tgFile, isOfficial, err := s.GetFileWithFallback(fileID)
	if err != nil {
		slog.Error("Failed to get file from Telegram", "error", err, "fileID", fileID)
		errText := err.Error()
		lang := s.GetUserLanguage(message.From.ID)
		msgText := fmt.Sprintf("❌ *Error:* %s", utils.EscapeMarkdownV2(errText))

		if strings.Contains(errText, "file is too big") {
			msgText += i18n.T(lang, "telegram_file_limit")
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	var fileURL string
	if filepath.IsAbs(tgFile.FilePath) {
		translatedPath := strings.Replace(tgFile.FilePath, "/var/lib/telegram-bot-api", s.Config.DownloadDir, 1)
		if _, errStat := os.Stat(translatedPath); errStat == nil {
			slog.Info("Local TG file detected", "path", translatedPath)
			fileURL = "file://" + translatedPath
		}
	}

	if fileURL == "" {
		if s.Config.TelegramAPI != "" && !isOfficial {
			fileEndpoint := strings.Replace(s.Config.TelegramAPI, "/bot%s/%s", "/file/bot%s/%s", 1)
			fileURL = fmt.Sprintf(fileEndpoint, s.Bot.Token, tgFile.FilePath)
		} else {
			fileURL = tgFile.Link(s.Bot.Token)
		}
	}

	slog.Debug("Telegram download initiated", "fileID", fileID, "filePath", tgFile.FilePath, "url", fileURL)

	taskType := TypeMirror
	if strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
		taskType = TypeTorrent
	}

	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}

	if taskType == TypeTorrent {
		s.showTorrentSelectionMenu(message, fileURL, fileName, zip, unzip, password, replyID)
		return
	}

	task, err := s.TaskManager.CreateTask(taskType, fileURL, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, int64(tgFile.FileSize))
	if err != nil {
		s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.handleAutoDelete(task)
	s.UpdateSharedDashboard(message.Chat.ID, true)
	slog.Info("Telegram download task created", "taskID", task.ID, "type", taskType)
}

func (s *BotService) handleLocalFileDownload(task *Task, outputDir string) {
	sourcePath := strings.TrimPrefix(task.URL, "file://")

	task.Mu.RLock()
	expectedSize := task.TotalSize
	task.Mu.RUnlock()

	s.updateTaskStatus(task)

	lastUpdate := time.Now()
	var sameSizeCount int
	var lastSize int64

	for {
		info, err := os.Stat(sourcePath)
		if err != nil {
			task.SetError(fmt.Sprintf("Local file not found: %v", err))
			s.updateTaskStatus(task)
			return
		}

		currentSize := info.Size()

		if expectedSize > 0 {
			if currentSize >= expectedSize {
				break
			}

			if time.Since(lastUpdate) >= 3*time.Second {
				task.Mu.Lock()
				task.DownloadedSize = currentSize
				task.Progress = float64(currentSize) / float64(expectedSize) * 100
				task.Mu.Unlock()
				s.updateTaskStatus(task)
				lastUpdate = time.Now()
			}
		} else {
			break
		}

		if currentSize == lastSize {
			sameSizeCount++
		} else {
			sameSizeCount = 0
		}
		lastSize = currentSize

		time.Sleep(1 * time.Second)
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		task.SetError(fmt.Sprintf("Local file not found: %v", err))
		s.updateTaskStatus(task)
		return
	}

	fileName := task.FileName
	if fileName == "" || fileName == UnknownFile {
		fileName = filepath.Base(sourcePath)
	}
	destPath := filepath.Join(outputDir, fileName)

	task.Mu.Lock()
	task.TotalSize = info.Size()
	task.DownloadedSize = 0
	task.Progress = 0
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	cleanedSource := filepath.Clean(sourcePath)
	source, err := os.Open(cleanedSource)
	if err != nil {
		task.SetError(fmt.Sprintf("Failed to open source file: %v", err))
		s.updateTaskStatus(task)
		return
	}
	defer func() { _ = source.Close() }()

	cleanedDest := filepath.Clean(destPath)
	dest, err := os.Create(cleanedDest)
	if err != nil {
		task.SetError(fmt.Sprintf("Failed to create destination file: %v", err))
		s.updateTaskStatus(task)
		return
	}
	defer func() { _ = dest.Close() }()

	buf := make([]byte, 32*1024)
	var copied int64
	startTime := time.Now()
	lastUpdate = time.Now()

	task.Mu.Lock()
	task.DownloadedSize = 0
	task.Progress = 0
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	for {
		nr, readErr := source.Read(buf)
		if nr > 0 {
			nw, writeErr := dest.Write(buf[0:nr])
			if nw > 0 {
				copied += int64(nw)
			}
			if writeErr != nil {
				task.SetError(fmt.Sprintf("Failed to copy file: %v", writeErr))
				s.updateTaskStatus(task)
				return
			}
			if nr != nw {
				task.SetError("Failed to copy file: short write")
				s.updateTaskStatus(task)
				return
			}
		}

		if time.Since(lastUpdate) >= 1*time.Second {
			task.Mu.Lock()
			task.DownloadedSize = copied
			if task.TotalSize > 0 {
				task.Progress = float64(copied) / float64(task.TotalSize) * 100
			}
			elapsed := time.Since(startTime).Seconds()
			if elapsed > 0 {
				task.Speed = int64(float64(copied) / elapsed)
				if task.Speed > 0 && task.TotalSize > 0 {
					remaining := task.TotalSize - copied
					task.ETA = time.Duration(remaining/task.Speed) * time.Second
				}
			}
			task.Mu.Unlock()
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			task.SetError(fmt.Sprintf("Failed to copy file: %v", readErr))
			s.updateTaskStatus(task)
			return
		}
	}

	task.Mu.Lock()
	task.DownloadedSize = task.TotalSize
	task.Progress = 100
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	s.handlePostDownload(task, outputDir)
}

func (s *BotService) downloadWithAria2(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	outputDir := filepath.Join(s.TaskManager.DownloadDir, task.ID)

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		s.updateTaskStatus(task)
		return
	}

	if strings.HasPrefix(task.URL, "file://") {
		if task.Type != TypeTorrent {
			s.handleLocalFileDownload(task, outputDir)
			return
		}
	}

	var firstUpdate = true
	lastUpdate := time.Now()
	err := s.TaskManager.Aria2Engine.Download(task.Ctx, &task.Task, outputDir, func(up downloader.ProgressUpdate) {
		task.UpdateFromProgressUpdate(up)

		if firstUpdate || time.Since(lastUpdate) >= 3*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
			_ = task.SaveToDB()
			firstUpdate = false
		}
	})

	if err != nil && task.Status != StatusCancelled {
		if s.TaskManager.IsShuttingDown() {
			return
		}
		if s.retryTask(task, err.Error()) {
			return
		}
		task.SetError(err.Error())
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}

	s.handlePostDownload(task, outputDir)
}

func (s *BotService) handlePostDownload(task *Task, outputDir string) {
	if task.Status == StatusCancelled {
		s.cleanupTask(task)
		return
	}

	task.LocalPath = findDownloadedFile(outputDir)
	if task.LocalPath == "" {
		if task.Error == "" {
			task.SetError("Downloaded file not found")
		}
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}
	task.FileName = filepath.Base(task.LocalPath)

	if task.Unzip && organizer.IsArchiveFile(task.LocalPath) {
		if err := s.extractArchive(task); err != nil {
			task.SetError(fmt.Sprintf("Extraction failed: %v", err))
			s.updateTaskStatus(task)
			s.cleanupTask(task)
			return
		}
	}

	if task.Zip {
		if err := s.createZipArchive(task); err != nil {
			task.SetError(fmt.Sprintf("Compression failed: %v", err))
			s.updateTaskStatus(task)
			s.cleanupTask(task)
			return
		}
	}

	var err error
	switch task.Type {
	case TypeLeech:
		err = s.UploadToTelegram(task)
	case TypeViking:
		err = s.UploadToViking(task)
	default:
		err = s.UploadWithRclone(task)
	}

	if err != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}
		task.SetError(fmt.Sprintf("Upload failed: %v", err))
	} else {
		task.SetStatus(StatusCompleted)
	}
	s.updateTaskStatus(task)
	s.cleanupTask(task)
	s.handleAutoDelete(task)
}

func (s *BotService) downloadWithYTDLP(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	outputDir := filepath.Join(s.TaskManager.DownloadDir, task.ID)

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		s.updateTaskStatus(task)
		return
	}

	lastUpdate := time.Now()
	err := s.TaskManager.YTDLPEngine.Download(task.Ctx, &task.Task, outputDir, func(up downloader.ProgressUpdate) {
		task.UpdateFromProgressUpdate(up)

		if time.Since(lastUpdate) >= 5*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
			_ = task.SaveToDB()
		}
	})

	if err != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}

		if s.TaskManager.IsShuttingDown() {
			return
		}

		if s.retryTask(task, err.Error()) {
			return
		}

		task.SetError(err.Error())
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}

	if task.Status == StatusCancelled {
		s.cleanupTask(task)
		return
	}

	task.LocalPath = findDownloadedFile(outputDir)
	if task.LocalPath == "" {
		task.SetError("Downloaded file not found or incomplete (.part files ignored)")
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}
	task.FileName = filepath.Base(task.LocalPath)

	if info, err := os.Stat(task.LocalPath); err == nil {
		task.DownloadedSize = info.Size()
		task.TotalSize = info.Size()
	}

	var uploadErr error
	if task.Type == TypeYTDLPLeech {
		uploadErr = s.UploadToTelegram(task)
	} else {
		uploadErr = s.UploadWithRclone(task)
	}

	if uploadErr != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}
		if s.TaskManager.IsShuttingDown() {
			return
		}
		if s.retryTask(task, uploadErr.Error()) {
			return
		}
		task.SetError(fmt.Sprintf("Upload failed: %v", uploadErr))
	} else {
		task.SetStatus(StatusCompleted)
	}
	s.updateTaskStatus(task)
	s.cleanupTask(task)
	s.handleAutoDelete(task)
}

func findDownloadedFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var candidates []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".aria2") ||
			strings.HasSuffix(name, ".part") ||
			strings.HasSuffix(name, ".ytdl") ||
			strings.HasSuffix(name, ".temp") {
			continue
		}
		candidates = append(candidates, filepath.Join(dir, name))
	}

	if len(candidates) == 1 {
		return candidates[0]
	}

	var result string
	var maxSize int64
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		name := strings.ToLower(info.Name())
		if strings.HasSuffix(name, ".part") || strings.HasSuffix(name, ".ytdl") || strings.HasSuffix(name, ".aria2") {
			return nil
		}

		if info.Size() > maxSize {
			maxSize = info.Size()
			result = path
		}
		return nil
	})
	return result
}

func (s *BotService) updateTaskStatus(task *Task) {
	snapshot := task.GetSnapshot()

	if snapshot.Status != StatusCompleted && snapshot.Status != StatusFailed && snapshot.Status != StatusCancelled {
		s.UpdateSharedDashboard(snapshot.ChatID, false)
		return
	}

	s.UpdateSharedDashboard(snapshot.ChatID, false)

	lang := s.GetUserLanguage(snapshot.UserID)
	text := buildTaskStatusText(lang, snapshot)

	if snapshot.Status == StatusCompleted && organizer.IsVideoFile(snapshot.FileName) && snapshot.LocalPath != "" {
		task.Mu.RLock()
		existingID := task.ResultMessageID
		task.Mu.RUnlock()

		if existingID == 0 {
			if s.sendVideoWithThumbnail(task, text) {
				return
			}
		}
	}

	slog.Debug("Sending final task message", "task_id", task.ID, "status", snapshot.Status)
	s.sendFinalMessage(task, text)
}

func (s *BotService) sendVideoWithThumbnail(task *Task, text string) bool {
	snapshot := task.GetSnapshot()
	if thumb, err := GenerateThumbnail(snapshot.LocalPath, s.TaskManager.DownloadDir); err == nil {
		photo := tgbotapi.NewPhoto(snapshot.ChatID, tgbotapi.FilePath(thumb))
		photo.Caption = text
		photo.ParseMode = MarkdownV2
		if snapshot.RemoteURL != "" {
			btnText := BtnTextCloudLink
			if s.Config.IndexURL != "" {
				btnText = BtnTextIndexURL
			}
			photo.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(btnText, snapshot.RemoteURL),
				),
			)
		}
		sentMsg, sendErr := s.Bot.Send(photo)
		if sendErr == nil {
			task.Mu.Lock()
			task.ResultMessageID = sentMsg.MessageID
			task.Mu.Unlock()
			slog.Info("Captured result video message ID", "message_id", sentMsg.MessageID, "task_id", task.ID)
			_ = os.Remove(thumb)
			return true
		}
		slog.Error("Failed to send video with thumbnail", "error", sendErr, "task_id", task.ID)
		_ = os.Remove(thumb)
	}
	return false
}

func (s *BotService) sendFinalMessage(task *Task, text string) {
	snapshot := task.GetSnapshot()

	task.Mu.RLock()
	msgID := task.ResultMessageID
	task.Mu.RUnlock()

	if msgID != 0 {
		editCaption := tgbotapi.NewEditMessageCaption(snapshot.ChatID, msgID, text)
		editCaption.ParseMode = MarkdownV2
		if snapshot.RemoteURL != "" {
			btnText := BtnTextCloudLink
			if s.Config.IndexURL != "" {
				btnText = BtnTextIndexURL
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(btnText, snapshot.RemoteURL),
				),
			)
			editCaption.ReplyMarkup = &keyboard
		}

		if _, err := s.Bot.Send(editCaption); err == nil {
			return
		}

		editText := tgbotapi.NewEditMessageText(snapshot.ChatID, msgID, text)
		editText.ParseMode = MarkdownV2
		if snapshot.RemoteURL != "" {
			btnText := BtnTextCloudLink
			if s.Config.IndexURL != "" {
				btnText = BtnTextIndexURL
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL(btnText, snapshot.RemoteURL),
				),
			)
			editText.ReplyMarkup = &keyboard
		}

		if _, err := s.Bot.Send(editText); err == nil {
			return
		}

		slog.Warn("Failed to edit existing message, sending new one", "taskID", task.ID, "msgID", msgID)
	}

	msg := tgbotapi.NewMessage(snapshot.ChatID, text)
	msg.ParseMode = MarkdownV2

	if snapshot.Status == StatusCompleted && snapshot.RemoteURL != "" {
		btnText := BtnTextCloudLink
		if s.Config.IndexURL != "" {
			btnText = BtnTextIndexURL
		}
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(btnText, snapshot.RemoteURL),
			),
		)
		msg.ReplyMarkup = keyboard
	}

	if sentMsg, err := s.Bot.Send(msg); err == nil {
		task.Mu.Lock()
		task.ResultMessageID = sentMsg.MessageID
		task.Mu.Unlock()
		slog.Info("Captured result final message ID", "message_id", sentMsg.MessageID, "task_id", task.ID)
	} else {
		slog.Error("Failed to send final task message", "error", err, "task_id", task.ID, "chatID", snapshot.ChatID)
	}
}

func (s *BotService) editStatusMessage(chatID int64, msgID int, text string) {
	editMsg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	editMsg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) downloadWithUserbot(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	outputDir := filepath.Join(s.TaskManager.DownloadDir, task.ID)

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		s.updateTaskStatus(task)
		return
	}

	var firstUpdate = true
	lastUpdate := time.Now()
	err := s.TaskManager.UserbotEngine.Download(task.Ctx, &task.Task, outputDir, func(up downloader.ProgressUpdate) {
		task.UpdateFromProgressUpdate(up)

		if firstUpdate || time.Since(lastUpdate) >= 3*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
			_ = task.SaveToDB()
			firstUpdate = false
		}
	})

	if err != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}

		task.SetError(err.Error())
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}

	if task.Status == StatusCancelled {
		s.cleanupTask(task)
		return
	}

	s.handlePostDownload(task, outputDir)
}

func (s *BotService) processTask(task *Task) {
	task.Mu.RLock()
	status := task.Status
	url := task.URL
	task.Mu.RUnlock()

	if status == StatusCancelled {
		slog.Info("Skipping cancelled task", "taskID", task.ID)
		return
	}

	slog.Info("Starting task processing", "taskID", task.ID, "type", task.Type)

	if (task.Type == TypeMirror || task.Type == TypeLeech) && (strings.Contains(url, "/c/") || strings.Contains(url, "t.me/c/")) {
		if s.Config.UserSessionString != "" {
			slog.Info("Using Userbot engine for private link", "taskID", task.ID)
			s.downloadWithUserbot(task)
			return
		}
	}

	if (task.Type == TypeMirror || task.Type == TypeLeech) && (strings.Contains(url, "drive.google.com") || strings.Contains(url, "docs.google.com") || strings.Contains(url, "drive.usercontent.google.com")) {
		slog.Info("Detected Google Drive URL for Mirror/Leech, switching to local Rclone download", "taskID", task.ID)
		s.downloadGDriveWithRclone(task)
		return
	}

	switch task.Type {
	case TypeMirror, TypeLeech, TypeTorrent, TypeViking:
		s.downloadWithAria2(task)
	case TypeYTDLP, TypeYTDLPLeech:
		s.downloadWithYTDLP(task)
	case TypeClone:
		s.cloneWithRclone(task)
	}
}

func (s *BotService) retryTask(task *Task, originalErr string) bool {
	task.Mu.Lock()
	if task.Status == StatusCancelled {
		task.Mu.Unlock()
		return false
	}

	if task.RetryCount >= task.MaxRetries {
		task.Mu.Unlock()
		return false
	}

	task.RetryCount++
	retries := task.RetryCount
	task.Mu.Unlock()

	backoff := 5 * time.Second
	for i := 1; i < retries; i++ {
		backoff *= 2
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
			break
		}
	}

	slog.Info("Retrying task due to error",
		"taskID", task.ID,
		"retry", retries,
		"max", task.MaxRetries,
		"backoff", backoff,
		"error", originalErr)

	go func() {
		time.Sleep(backoff)

		task.Mu.Lock()
		task.Status = StatusQueued
		task.Error = ""
		task.Mu.Unlock()

		s.updateTaskStatus(task)
		s.TaskManager.Queue <- task
	}()

	return true
}

func isGenericName(name string) bool {
	uuidRegex := regexp.MustCompile(`^[a-fA-F0-9]{8}(-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}$`)
	if uuidRegex.MatchString(name) {
		return true
	}

	hexRegex := regexp.MustCompile(`^[a-fA-F0-9]{16,}$`)
	return hexRegex.MatchString(name)
}
