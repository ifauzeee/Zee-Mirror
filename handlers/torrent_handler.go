package handlers

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"zee-mirror/internal/service"
	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

func (s *BotService) HandleTorrent(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, service.GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name, _, _ := utils.ParseFlags(args)
	if message.ReplyToMessage != nil && message.ReplyToMessage.Document != nil {
		fileID := message.ReplyToMessage.Document.FileID
		fileName := message.ReplyToMessage.Document.FileName
		fileSize := int64(message.ReplyToMessage.Document.FileSize)

		if strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
			go s.HandleTelegramFileDownload(message, fileID, fileName, fileSize, service.TypeMirror, zip, unzip, password, quality)
			return
		}
	}

	if url == "" {
		url = utils.ExtractMagnetFromText(args)
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_magnet"))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
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
	msg.ParseMode = tgbotapi.ModeMarkdownV2
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

	s.TaskManager.TorrentSessions[sessionID] = &service.TorrentSession{
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

	go s.FetchTorrentMetadataBackground(sessionID)

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
		confirmMsg.ParseMode = tgbotapi.ModeMarkdownV2
		sentMsg, _ := s.Bot.Send(confirmMsg)
		statusMsgID := sentMsg.MessageID

		fileName := session.FileName
		if fileName == "" {
			fileName = utils.GetFileNameFromURL(session.URL)
			if fileName == "unknown_file" {
				fileName = "torrent_download"
			}
		}

		task, err := s.TaskManager.CreateTask(service.TypeTorrent, session.URL, fileName, session.ChatID, statusMsgID, session.ReplyID, session.UserID, session.Zip, session.Unzip, session.Password, "", 0, "", false)
		if err != nil {
			s.handleCreateTaskError(session.ChatID, statusMsgID, err)
			return
		}
		s.UpdateSharedDashboard(session.ChatID, true)
		s.HandleAutoDelete(task)
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
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	sentMsg, _ := s.Bot.Send(msg)

	task, err := s.TaskManager.CreateTask(service.TypeTorrent, url, fileName, session.ChatID, sentMsg.MessageID, session.ReplyID, session.UserID, session.Zip, session.Unzip, session.Password, "", 0, "", false)
	if err != nil {
		s.handleCreateTaskError(session.ChatID, sentMsg.MessageID, err)
		return nil
	}
	s.UpdateSharedDashboard(session.ChatID, true)
	s.HandleAutoDelete(task)
	slog.Info("Torrent task created (selected files)", "taskID", task.ID, "selectedFiles", selectedFiles)

	return nil
}
