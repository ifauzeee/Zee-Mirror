package service

import (
	"fmt"
	"log/slog"
	"time"

	"zee-mirror/internal/domain"
	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

func (s *BotService) ShowTorrentSelectionMenu(message *tgbotapi.Message, url, name string, zip, unzip bool, password string, replyID int) {
	sessionID := s.CreateTorrentSession(url, name, zip, unzip, password, message.Chat.ID, message.MessageID, replyID, message.From.ID)

	lang := s.GetUserLanguage(message.From.ID)
	text := i18n.T(lang, "torrent_menu_text")

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2

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

func (s *BotService) CreateTorrentSession(url, name string, zip, unzip bool, password string, chatID int64, msgID, replyID int, userID int64) string {
	sessionID := uuid.New().String()[:12]

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

		task, err := s.TaskManager.CreateTask(TypeTorrent, session.URL, fileName, session.ChatID, statusMsgID, session.ReplyID, session.UserID, session.Zip, session.Unzip, session.Password, "", 0, "", false)
		if err != nil {
			s.HandleCreateTaskError(session.ChatID, statusMsgID, err)
			return
		}
		s.UpdateSharedDashboard(session.ChatID, true)
		s.HandleAutoDelete(task)
		slog.Info("Torrent task created (all files)", "taskID", task.ID, "url", session.URL)

	case "browse", "page":
		offset := 0
		if len(parts) >= 4 {
			_, _ = fmt.Sscanf(parts[3], "%d", &offset)
		}

		if session.IsFetching {
			s.TaskManager.Mu.Unlock()
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "⏳ Masih mengambil metadata..."))
			return
		}

		if len(session.Files) == 0 {
			s.TaskManager.Mu.Unlock()
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Gagal mendapatkan daftar file"))
			return
		}

		s.TaskManager.Mu.Unlock()
		s.ShowTorrentBrowseMenu(callback, sessionID, offset)

	case "toggle":
		if len(parts) < 5 {
			s.TaskManager.Mu.Unlock()
			return
		}

		fileIdx := 0
		offset := 0
		_, _ = fmt.Sscanf(parts[3], "%d", &fileIdx)
		_, _ = fmt.Sscanf(parts[4], "%d", &offset)

		found := false
		for i, idx := range session.SelectedFiles {
			if idx == fileIdx {
				session.SelectedFiles = append(session.SelectedFiles[:i], session.SelectedFiles[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			session.SelectedFiles = append(session.SelectedFiles, fileIdx)
		}

		s.TaskManager.Mu.Unlock()
		s.ShowTorrentBrowseMenu(callback, sessionID, offset)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Pilihan diperbarui"))

	case "start_sel":
		selected := session.SelectedFiles
		if len(selected) == 0 {
			s.TaskManager.Mu.Unlock()
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "⚠️ Pilih minimal satu file!"))
			return
		}
		s.TaskManager.Mu.Unlock()
		_ = s.StartTorrentWithSelectedFiles(sessionID, selected)

	case "back":
		s.TaskManager.Mu.Unlock()

		lang := s.GetUserLanguage(callback.From.ID)
		text := i18n.T(lang, "torrent_menu_text")

		edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		edit.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(edit)

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

func (s *BotService) ShowTorrentBrowseMenu(callback *tgbotapi.CallbackQuery, sessionID string, offset int) {
	s.TaskManager.Mu.RLock()
	session, exists := s.TaskManager.TorrentSessions[sessionID]
	if !exists {
		s.TaskManager.Mu.RUnlock()
		return
	}

	files := session.Files
	s.TaskManager.Mu.RUnlock()

	lang := s.GetUserLanguage(callback.From.ID)
	text := i18n.T(lang, "torrent_browse_title", len(files))

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = MarkdownV2

	_, _ = s.Bot.Request(edit)
}

func (s *BotService) StartTorrentWithSelectedFiles(sessionID string, selectedFiles []int) error {
	s.TaskManager.Mu.Lock()
	session, exists := s.TaskManager.TorrentSessions[sessionID]
	if !exists {
		s.TaskManager.Mu.Unlock()
		return fmt.Errorf("%w: session not found", domain.ErrNotFound)
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

	task, err := s.TaskManager.CreateTask(TypeTorrent, url, fileName, session.ChatID, sentMsg.MessageID, session.ReplyID, session.UserID, session.Zip, session.Unzip, session.Password, "", 0, "", false)
	if err != nil {
		s.HandleCreateTaskError(session.ChatID, sentMsg.MessageID, err)
		return nil
	}
	s.UpdateSharedDashboard(session.ChatID, true)
	s.HandleAutoDelete(task)
	slog.Info("Torrent task created (selected files)", "taskID", task.ID, "selectedFiles", selectedFiles)

	return nil
}
