package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"zee-mirror/internal/downloader"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

func (s *BotService) HandleMirror(message *tgbotapi.Message, args string) {
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
					log.Printf("[Mirror] Resolved filename from header: %s", fileName)
				}
			}
		}
		task := s.TaskManager.CreateTask(TypeMirror, url, fileName, message.Chat.ID, message.MessageID, message.From.ID, zip, unzip, password, quality)
		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.handleAutoDelete(task)
		log.Printf("[Mirror] Task created: %s", task.ID)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nReply ke file atau berikan URL\\.")
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
	url, zip, unzip, password, quality, name := utils.ParseFlags(args)
	if url == "" {
		url = utils.ExtractMagnetFromText(args)
	}
	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL untuk di\\-leech\\.")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	fileName := name
	if fileName == "" {
		fileName = utils.GetFileNameFromURL(url)
	}
	task := s.TaskManager.CreateTask(TypeLeech, url, fileName, message.Chat.ID, message.MessageID, message.From.ID, zip, unzip, password, quality)
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.handleAutoDelete(task)
	log.Printf("[Leech] Task created: %s", task.ID)
}

func (s *BotService) HandleYTDLP(message *tgbotapi.Message, args string) {
	url, zip, _, password, quality, _ := utils.ParseFlags(args)
	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL video\\.")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if quality == "" && (strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")) {
		s.showYTDLPQualityMenu(message, url, zip, password)
		return
	}

	task := s.TaskManager.CreateTask(TypeYTDLP, url, "video", message.Chat.ID, message.MessageID, message.From.ID, zip, false, password, quality)
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.handleAutoDelete(task)
	log.Printf("[YTDLP] Task created: %s", task.ID)
}

func (s *BotService) showYTDLPQualityMenu(message *tgbotapi.Message, url string, zip bool, password string) {
	statusMsg, _ := s.Bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🎬 Menganalisa kualitas video…"))

	args := []string{
		"-j",
		"--no-playlist",
		"--no-check-certificate",
		"--extractor-args", "youtube:player-client=web,web_embedded,ios,mweb,tv",
		"--socket-timeout", "60",
		"--add-header", "Accept-Language: en-US,en;q=0.9",
		"--add-header", "Referer: https://www.youtube.com/",
		"--remote-components", "ejs:github",
		"--js-runtime", "node",
		"--cache-dir", "/home/botuser/.cache/yt-dlp-final",
	}

	cookiesPath := filepath.Join(s.Config.ConfigDir, "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		log.Printf("[YTDLP-Info] Using cookies from: %s", cookiesPath)
		args = append(args, "--cookies", cookiesPath)
		args = append(args, "--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	}

	args = append(args, url)
	cmd := exec.Command("yt-dlp", args...)

	output, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		log.Printf("[YTDLP-Info] Error: %v, Stderr: %s", err, stderr)
		s.editStatusMessage(statusMsg.Chat.ID, statusMsg.MessageID, fmt.Sprintf("❌ *Gagal menganalisa video:* %v\n\n_Pastikan URL valid atau coba lagi nanti\\._", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	var data struct {
		Formats []struct {
			FormatID string  `json:"format_id"`
			Height   int     `json:"height"`
			FPS      float64 `json:"fps"`
			VCodec   string  `json:"vcodec"`
		} `json:"formats"`
	}

	if err := json.Unmarshal(output, &data); err != nil {
		s.editStatusMessage(statusMsg.Chat.ID, statusMsg.MessageID, "❌ *Gagal memproses data video*")
		return
	}

	resMap := make(map[int]float64)
	for _, f := range data.Formats {
		log.Printf("[YTDLP-Format] ID: %s, Res: %dp, FPS: %.1f, VCodec: %s", f.FormatID, f.Height, f.FPS, f.VCodec)

		if strings.HasPrefix(f.FormatID, "sb") {
			continue
		}

		if f.Height > 0 && f.VCodec != "" && f.VCodec != "none" {
			if f.FPS > resMap[f.Height] {
				resMap[f.Height] = f.FPS
			}
		}
	}

	var sortedHeights []int
	for h := range resMap {
		sortedHeights = append(sortedHeights, h)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedHeights)))

	sessionID := uuid.New().String()[:8]
	s.TaskManager.Mu.Lock()
	s.TaskManager.YTDLPSessions[sessionID] = &YTDLPSession{
		URL:      url,
		Zip:      zip,
		Password: password,
	}
	s.TaskManager.Mu.Unlock()

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(sortedHeights); i += 2 {
		var row []tgbotapi.InlineKeyboardButton

		h1 := sortedHeights[i]
		fps1 := resMap[h1]
		label1 := formatQualityLabel(h1, fps1)
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label1, fmt.Sprintf("ytdlp_q:%d:%s", h1, sessionID)))

		if i+1 < len(sortedHeights) {
			h2 := sortedHeights[i+1]
			fps2 := resMap[h2]
			label2 := formatQualityLabel(h2, fps2)
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(label2, fmt.Sprintf("ytdlp_q:%d:%s", h2, sessionID)))
		}

		rows = append(rows, row)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🚀 Kualitas Terbaik", fmt.Sprintf("ytdlp_q:best:%s", sessionID)),
	))
	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	text := "📽️ *Pilih Kualitas Video*\n\nVideo ini mendukung resolusi berikut:"
	if len(sortedHeights) == 0 {
		text = "📽️ *Pilih Kualitas Video*\n\nResolusi tidak terdeteksi, gunakan kualitas terbaik:"
	}

	editMsg := tgbotapi.NewEditMessageText(statusMsg.Chat.ID, statusMsg.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
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

	task := s.TaskManager.CreateTask(TypeYTDLP, session.URL, "video", callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, session.Zip, false, session.Password, quality)
	s.UpdateSharedDashboard(callback.Message.Chat.ID, false)
	log.Printf("[YTDLPCallback] Task created: %s", task.ID)
}

func (s *BotService) HandleTorrent(message *tgbotapi.Message, args string) {
	url, zip, unzip, password, quality, name := utils.ParseFlags(args)
	if url == "" {
		url = utils.ExtractMagnetFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan magnet link\\.")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	fileName := name
	if fileName == "" {
		fileName = "torrent_download"
	}
	task := s.TaskManager.CreateTask(TypeTorrent, url, fileName, message.Chat.ID, message.MessageID, message.From.ID, zip, unzip, password, quality)
	s.UpdateSharedDashboard(message.Chat.ID, true)
	s.handleAutoDelete(task)
	log.Printf("[Torrent] Task created: %s", task.ID)
}

func (s *BotService) handleTelegramFileDownload(message *tgbotapi.Message, fileID, fileName string, zip, unzip bool, password, quality string) {
	tgFile, err := s.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		errText := err.Error()
		msgText := fmt.Sprintf("❌ *Error:* %s", utils.EscapeMarkdownV2(errText))

		if strings.Contains(errText, "file is too big") {
			msgText += "\n\n⚠️ *Limitasi Telegram:* Bot hanya dapat mengunduh file hingga 20MB melalui server resmi\\. Gunakan *Local Bot API Server* untuk mengunduh file hingga 2GB\\."
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	var fileURL string
	if filepath.IsAbs(tgFile.FilePath) {
		translatedPath := strings.Replace(tgFile.FilePath, "/var/lib/telegram-bot-api", s.Config.DownloadDir, 1)
		if _, err := os.Stat(translatedPath); err == nil {
			log.Printf("[TGDownload] Local file detected: %s", translatedPath)
			fileURL = "file://" + translatedPath
		}
	}

	if fileURL == "" {
		if s.Config.TelegramAPI != "" {
			fileEndpoint := strings.Replace(s.Config.TelegramAPI, "/bot%s/%s", "/file/bot%s/%s", 1)
			fileURL = fmt.Sprintf(fileEndpoint, s.Bot.Token, tgFile.FilePath)
		} else {
			fileURL = tgFile.Link(s.Bot.Token)
		}
	}

	log.Printf("[TGDownload] FileID: %s, FilePath: %s, FinalURL: %s", fileID, tgFile.FilePath, fileURL)
	task := s.TaskManager.CreateTask(TypeMirror, fileURL, fileName, message.Chat.ID, 0, message.From.ID, zip, unzip, password, quality)
	s.UpdateSharedDashboard(message.Chat.ID, true)
	log.Printf("[TGDownload] Task created: %s", task.ID)
}

func (s *BotService) handleLocalFileDownload(task *Task, outputDir string) {
	sourcePath := strings.TrimPrefix(task.URL, "file://")

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

	//nolint:gosec
	source, err := os.Open(sourcePath)
	if err != nil {
		task.SetError(fmt.Sprintf("Failed to open source file: %v", err))
		s.updateTaskStatus(task)
		return
	}
	defer func() { _ = source.Close() }()

	//nolint:gosec
	dest, err := os.Create(destPath)
	if err != nil {
		task.SetError(fmt.Sprintf("Failed to create destination file: %v", err))
		s.updateTaskStatus(task)
		return
	}
	defer func() { _ = dest.Close() }()

	_, err = io.Copy(dest, source)
	if err != nil {
		task.SetError(fmt.Sprintf("Failed to copy file: %v", err))
		s.updateTaskStatus(task)
		return
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
	//nolint:gosec
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		s.updateTaskStatus(task)
		return
	}

	if strings.HasPrefix(task.URL, "file://") {
		s.handleLocalFileDownload(task, outputDir)
		return
	}

	lastUpdate := time.Now()
	err := s.TaskManager.Aria2Engine.Download(task.Ctx, &task.Task, outputDir, func(up downloader.ProgressUpdate) {
		task.Mu.Lock()
		if up.Downloaded != 0 {
			task.DownloadedSize = up.Downloaded
		}
		if up.Total != 0 {
			task.TotalSize = up.Total
		}
		if up.Speed != 0 {
			task.Speed = up.Speed
		}
		if up.Progress != 0 {
			task.Progress = up.Progress
		}
		if up.Connections != 0 {
			task.Connections = up.Connections
		}
		if up.ETA != 0 {
			task.ETA = up.ETA
		}
		task.Mu.Unlock()

		if time.Since(lastUpdate) >= 3*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
		}
	})

	if err != nil && task.Status != StatusCancelled {
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

	if task.Unzip && utils.IsArchiveFile(task.LocalPath) {
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
	if task.Type == TypeLeech {
		err = s.UploadToTelegram(task)
	} else {
		err = s.UploadWithRclone(task)
	}

	if err != nil {
		task.SetError(fmt.Sprintf("Upload failed: %v", err))
	} else {
		task.SetStatus(StatusCompleted)
	}
	s.updateTaskStatus(task)
	s.cleanupTask(task)
	s.handleAutoDelete(task)
}

func (s *BotService) updateTaskProgress(task *Task, up downloader.ProgressUpdate) {
	task.Mu.Lock()
	if up.FileName != "" {
		task.FileName = up.FileName
	}
	if up.Progress != 0 {
		task.Progress = up.Progress
	}
	if up.Total != 0 {
		task.TotalSize = up.Total
	}
	if up.Downloaded != 0 {
		task.DownloadedSize = up.Downloaded
	}
	if up.Speed != 0 {
		task.Speed = up.Speed
	}
	if up.ETA != 0 {
		task.ETA = up.ETA
	}
	if up.Error != "" {
		task.Error = up.Error
	}
	task.Mu.Unlock()
}

func (s *BotService) downloadWithYTDLP(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	outputDir := filepath.Join(s.TaskManager.DownloadDir, task.ID)
	//nolint:gosec
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		s.updateTaskStatus(task)
		return
	}

	lastUpdate := time.Now()
	err := s.TaskManager.YTDLPEngine.Download(task.Ctx, &task.Task, outputDir, func(up downloader.ProgressUpdate) {
		s.updateTaskProgress(task, up)

		if time.Since(lastUpdate) >= 5*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
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

	if err := s.UploadWithRclone(task); err != nil {
		task.SetError(fmt.Sprintf("Upload failed: %v", err))
	} else {
		task.SetStatus(StatusCompleted)
	}
	s.updateTaskStatus(task)
	s.cleanupTask(task)
	s.handleAutoDelete(task)
}

func findDownloadedFile(dir string) string {
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

	text := buildTaskStatusText(snapshot)

	if snapshot.Status == StatusCompleted && utils.IsVideoFile(snapshot.FileName) && snapshot.LocalPath != "" {
		task.Mu.RLock()
		existingID := task.ResultMessageID
		task.Mu.RUnlock()

		if existingID == 0 {
			if s.sendVideoWithThumbnail(task, text) {
				return
			}
		}
	}

	s.sendFinalMessage(task, text)
}

func buildTaskStatusText(snapshot TaskSnapshot) string {
	var text string
	switch snapshot.Status {
	case StatusCompleted:
		duration := calculateDuration(snapshot)
		sizeStr := determineSizeString(snapshot)

		text = fmt.Sprintf("✅ *Completed\\!*\n\n"+
			"📄 *Name:* `%s`\n"+
			"📦 *Size:* `%s`\n"+
			"⏱ *Time:* `%s`\n"+
			"📁 *Path:* `%s`",
			utils.EscapeMarkdownV2(snapshot.FileName),
			utils.EscapeMarkdownV2(sizeStr),
			utils.EscapeMarkdownV2(utils.FormatDuration(duration)),
			utils.EscapeMarkdownV2(snapshot.RemotePath))
	case StatusFailed:
		text = fmt.Sprintf("❌ *Failed\\!*\n📄 `%s`\nError: `%s`",
			utils.EscapeMarkdownV2(snapshot.FileName),
			utils.EscapeMarkdownV2(utils.TruncateString(snapshot.Error, 100)))
	default:
		return ""
	}
	return text
}

func calculateDuration(snapshot TaskSnapshot) time.Duration {
	duration := snapshot.CompletedAt.Sub(snapshot.StartedAt)
	if snapshot.StartedAt.IsZero() {
		duration = snapshot.CompletedAt.Sub(snapshot.CreatedAt)
	}
	return duration
}

func determineSizeString(snapshot TaskSnapshot) string {
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

func (s *BotService) sendVideoWithThumbnail(task *Task, text string) bool {
	snapshot := task.GetSnapshot()
	if thumb, err := GenerateThumbnail(snapshot.LocalPath, s.TaskManager.DownloadDir); err == nil {
		photo := tgbotapi.NewPhoto(snapshot.ChatID, tgbotapi.FilePath(thumb))
		photo.Caption = text
		photo.ParseMode = MarkdownV2
		if snapshot.RemoteURL != "" {
			photo.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("☁️ Cloud Link", snapshot.RemoteURL),
				),
			)
		}
		if sentMsg, err := s.Bot.Send(photo); err == nil {
			task.Mu.Lock()
			task.ResultMessageID = sentMsg.MessageID
			task.Mu.Unlock()
			log.Printf("[AutoDelete] Captured result video msg ID %d for task %s", sentMsg.MessageID, task.ID)
			_ = os.Remove(thumb)
			return true
		}
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
		edit := tgbotapi.NewEditMessageCaption(snapshot.ChatID, msgID, text)
		edit.ParseMode = MarkdownV2
		if snapshot.RemoteURL != "" {
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("☁️ Cloud Link", snapshot.RemoteURL),
				),
			)
			edit.ReplyMarkup = &keyboard
		}
		if _, err := s.Bot.Send(edit); err != nil {
			log.Printf("[FinalMessage] Failed to edit caption: %v", err)

			log.Printf("[FinalMessage] Fallback to sending new message")
		} else {
			return
		}
	}

	msg := tgbotapi.NewMessage(snapshot.ChatID, text)
	msg.ParseMode = MarkdownV2

	if snapshot.Status == StatusCompleted && snapshot.RemoteURL != "" {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("☁️ Cloud Link", snapshot.RemoteURL),
			),
		)
		msg.ReplyMarkup = keyboard
	}

	if sentMsg, err := s.Bot.Send(msg); err == nil {
		task.Mu.Lock()
		task.ResultMessageID = sentMsg.MessageID
		task.Mu.Unlock()
		log.Printf("[AutoDelete] Captured result final msg ID %d for task %s", sentMsg.MessageID, task.ID)
	}
}

func (s *BotService) editStatusMessage(chatID int64, msgID int, text string) {
	editMsg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	editMsg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) processTask(task *Task) {
	task.Mu.RLock()
	status := task.Status
	task.Mu.RUnlock()

	if status == StatusCancelled {
		log.Printf("[Task %s] Skipping cancelled task", task.ID)
		return
	}

	log.Printf("[Task %s] Starting processing type: %s", task.ID, task.Type)

	switch task.Type {
	case TypeMirror, TypeLeech, TypeTorrent:
		s.downloadWithAria2(task)
	case TypeYTDLP:
		s.downloadWithYTDLP(task)
	case TypeClone:
		s.cloneWithRclone(task)
	}
}

func isGenericName(name string) bool {
	uuidRegex := regexp.MustCompile(`^[a-fA-F0-9]{8}(-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}$`)
	if uuidRegex.MatchString(name) {
		return true
	}

	hexRegex := regexp.MustCompile(`^[a-fA-F0-9]{16,}$`)
	return hexRegex.MatchString(name)
}
