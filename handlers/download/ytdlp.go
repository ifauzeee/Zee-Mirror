package download

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"zee-mirror/internal/downloader"
	"zee-mirror/internal/service"
	"zee-mirror/pkg/i18n"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

func HandleYTDLP(s *service.BotService, message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.Reply(message, service.GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name, subs, hardsub := utils.ParseFlags(args)
	_ = unzip

	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_url"))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if isYTDLPPlaylist(url) {
		go handleYTDLPPlaylist(s, message, url, name, zip, password, quality, subs, hardsub, service.TypeYTDLP)
		return
	}

	handleYTDLPGeneric(s, message, args, service.TypeYTDLP)
}

func HandleYTDLPLeech(s *service.BotService, message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.Reply(message, service.GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name, subs, hardsub := utils.ParseFlags(args)
	_ = unzip
	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" {
		lang := s.GetUserLanguage(message.From.ID)
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "invalid_url"))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if isYTDLPPlaylist(url) {
		go handleYTDLPPlaylist(s, message, url, name, zip, password, quality, subs, hardsub, service.TypeYTDLPLeech)
		return
	}

	handleYTDLPGeneric(s, message, args, service.TypeYTDLPLeech)
}

func handleYTDLPGeneric(s *service.BotService, message *tgbotapi.Message, args string, taskType service.TaskType) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.Reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, _, password, quality, name, subs, hardsub := utils.ParseFlags(args)
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
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		if sentMsg, err := s.Bot.Send(msg); err == nil {
			s.AutoDeleteMessage(message.Chat.ID, sentMsg.MessageID, 30*time.Second)
		}
		return
	}

	if quality == "" && (strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")) {
		showYTDLPQualityMenu(s, message, url, name, zip, password, taskType)
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

	task, err := s.TaskManager.CreateTask(taskType, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, false, password, quality, 0, subs, hardsub)
	if err != nil {
		s.HandleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.HandleAutoDelete(task)
	slog.Info("YTDLP task created", "taskID", task.ID, "type", taskType, "url", url, "fileName", fileName)
}

func isYTDLPPlaylist(url string) bool {
	return (strings.Contains(url, "/@") ||
		strings.Contains(url, "/channel/") ||
		strings.Contains(url, "/c/") ||
		strings.Contains(url, "/user/") ||
		strings.Contains(url, "/playlist?") ||
		strings.Contains(url, "&list=")) && !strings.Contains(url, "watch?v=")
}

func showYTDLPQualityMenu(s *service.BotService, message *tgbotapi.Message, url, name string, zip bool, password string, taskType service.TaskType) {
	lang := s.GetUserLanguage(message.From.ID)
	if isYTDLPPlaylist(url) {
		msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "ytdlp_playlist_error"))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
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
		s.EditMessage(statusMsg.Chat.ID, statusMsg.MessageID, i18n.T(lang, "ytdlp_analysis_failed", utils.EscapeMarkdownV2(err.Error())))
		s.AutoDeleteMessage(statusMsg.Chat.ID, statusMsg.MessageID, 20*time.Second)
		return
	}

	sortedHeights := getSortedHeights(resMap)

	sessionID := createYTDLPSession(s, url, name, zip, password, taskType)
	keyboard := buildYTDLPKeyboard(sortedHeights, resMap, sessionID)

	text := i18n.T(lang, "ytdlp_select_quality")
	if len(sortedHeights) == 0 {
		text = i18n.T(lang, "ytdlp_no_resolution")
	}

	editMsg := tgbotapi.NewEditMessageText(statusMsg.Chat.ID, statusMsg.MessageID, text)
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
}

func getSortedHeights(resMap map[int]float64) []int {
	var heights []int
	for h := range resMap {
		heights = append(heights, h)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(heights)))
	return heights
}

func createYTDLPSession(s *service.BotService, url, name string, zip bool, password string, taskType service.TaskType) string {
	sessionID := uuid.New().String()[:8]
	s.TaskManager.Mu.Lock()
	defer s.TaskManager.Mu.Unlock()
	s.TaskManager.YTDLPSessions[sessionID] = &service.YTDLPSession{
		URL:      url,
		FileName: name,
		Zip:      zip,
		Password: password,
		Type:     taskType,
	}
	return sessionID
}

func buildYTDLPKeyboard(sortedHeights []int, resMap map[int]float64, sessionID string) tgbotapi.InlineKeyboardMarkup {
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
		tgbotapi.NewInlineKeyboardButtonData("🎵 Audio Only", fmt.Sprintf("ytdlp_q:audio:%s", sessionID)),
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

func YTDLPQualityCallbackHandler(s *service.BotService, callback *tgbotapi.CallbackQuery, parts []string) {
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

	task, err := s.TaskManager.CreateTask(session.Type, session.URL, fileName, callback.Message.Chat.ID, callback.Message.MessageID, replyID, callback.From.ID, session.Zip, false, session.Password, quality, 0, "", false)
	if err != nil {
		s.HandleCreateTaskError(callback.Message.Chat.ID, callback.Message.MessageID, err)
		return
	}
	s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
	slog.Info("YTDLP task created from callback", "taskID", task.ID, "type", session.Type, "quality", quality)
}

func handleYTDLPPlaylist(s *service.BotService, message *tgbotapi.Message, url, name string, zip bool, password, quality, subs string, hardsub bool, taskType service.TaskType) {
	lang := s.GetUserLanguage(message.From.ID)
	statusMsg, _ := s.Bot.Send(tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "ytdlp_analysis")))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	metadata, err := s.TaskManager.YTDLPEngine.GetPlaylistMetadata(ctx, url)
	if err != nil {
		slog.Error("Playlist analysis failed", "error", err)
		s.EditMessage(statusMsg.Chat.ID, statusMsg.MessageID, i18n.T(lang, "ytdlp_analysis_failed", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	if len(metadata.Entries) == 0 {
		s.EditMessage(statusMsg.Chat.ID, statusMsg.MessageID, "❌ *Error*\n\nPlaylist kosong atau tidak ditemukan video\\.")
		return
	}

	totalItems := len(metadata.Entries)
	s.EditMessage(statusMsg.Chat.ID, statusMsg.MessageID, fmt.Sprintf("✅ *Playlist Diterima*\n\n🏷️ *Judul:* %s\n📦 *Jumlah Item:* %d\n\nSedang memproses item playlist\\.\\.\\.", utils.EscapeMarkdownV2(metadata.Title), totalItems))

	go handlePlaylistDownload(s, message, metadata, name, zip, password, quality, subs, hardsub, taskType)
}

func handlePlaylistDownload(s *service.BotService, message *tgbotapi.Message, metadata *downloader.PlaylistMetadata, name string, zip bool, password, quality, subs string, hardsub bool, taskType service.TaskType) {
	for i, entry := range metadata.Entries {
		if i >= 50 {
			slog.Warn("Playlist item limit reached", "limit", 50, "total", len(metadata.Entries))
			break
		}

		itemURL := entry.URL
		if itemURL == "" {
			itemURL = "https://www.youtube.com/watch?v=" + entry.ID
		}

		fileName := name
		if fileName != "" {
			fileName = fmt.Sprintf("%s - %03d", name, i+1)
		}

		task, err := s.TaskManager.CreateTask(taskType, itemURL, fileName, message.Chat.ID, message.MessageID, 0, message.From.ID, zip, false, password, quality, 0, subs, hardsub)
		if err != nil {
			slog.Error("Failed to create task for playlist item", "index", i, "error", err)
			continue
		}
		task.Mu.Lock()
		task.PlaylistCount = len(metadata.Entries)
		task.PlaylistIndex = i + 1
		task.Mu.Unlock()
		s.UpdateSharedDashboard(message.Chat.ID, false)
		time.Sleep(2 * time.Second)
	}
}
