package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

func HandleMirror(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	url, zip, unzip, password, quality := ParseFlags(args)
	var fileName string

	if message.ReplyToMessage != nil {
		reply := message.ReplyToMessage
		if reply.Document != nil {
			fileName = reply.Document.FileName
			go handleTelegramFileDownload(bot, message, reply.Document.FileID, fileName, zip, unzip, password, quality)
			return
		} else if reply.Video != nil {
			fileName = reply.Video.FileName
			if fileName == "" {
				fileName = fmt.Sprintf("video_%d.mp4", time.Now().Unix())
			}
			go handleTelegramFileDownload(bot, message, reply.Video.FileID, fileName, zip, unzip, password, quality)
			return
		}
	}

	if url != "" {
		fileName = GetFileNameFromURL(url)
		statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "📥 Memulai mirror…"))
		taskManager.CreateTask(TypeMirror, url, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password, quality)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nReply ke file atau berikan URL\\.")
	msg.ParseMode = MarkdownV2
	_, _ = bot.Send(msg)
}

func HandleLeech(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	url, zip, unzip, password, quality := ParseFlags(args)
	if url == "" {
		url = ExtractMagnetFromText(args)
	}
	if url == "" {
		url = ExtractURLFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL untuk di\\-leech\\.")
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	fileName := GetFileNameFromURL(url)
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🔗 Memulai leech…"))
	taskManager.CreateTask(TypeLeech, url, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password, quality)
}

func HandleYTDLP(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	url, zip, _, password, quality := ParseFlags(args)
	if url == "" {
		url = ExtractURLFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL video\\.")
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	if quality == "" && (strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")) {
		showYTDLPQualityMenu(bot, message, url, zip, password)
		return
	}

	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🎬 Mengambil info video…"))
	taskManager.CreateTask(TypeYTDLP, url, "video", message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, false, password, quality)
}

func showYTDLPQualityMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message, url string, zip bool, password string) {
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🎬 Menganalisa kualitas video…"))

	cmd := exec.Command("yt-dlp", "-j", "--no-playlist", url)
	output, err := cmd.Output()
	if err != nil {
		editStatusMessage(bot, statusMsg.Chat.ID, statusMsg.MessageID, fmt.Sprintf("❌ *Gagal mengambil info video:* %v", EscapeMarkdownV2(err.Error())))
		return
	}

	var data struct {
		Formats []struct {
			Height int     `json:"height"`
			FPS    float64 `json:"fps"`
			VCodec string  `json:"vcodec"`
		} `json:"formats"`
	}

	if err := json.Unmarshal(output, &data); err != nil {
		editStatusMessage(bot, statusMsg.Chat.ID, statusMsg.MessageID, "❌ *Gagal memproses data video*")
		return
	}

	resMap := make(map[int]float64)
	for _, f := range data.Formats {
		if f.VCodec != "none" && f.Height > 0 {
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
	taskManager.Mu.Lock()
	taskManager.YTDLPSessions[sessionID] = &YTDLPSession{
		URL:      url,
		Zip:      zip,
		Password: password,
	}
	taskManager.Mu.Unlock()

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
	_, _ = bot.Send(editMsg)
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

func HandleYTDLPQualityCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Sesi kadaluarsa"))
		return
	}

	quality := parts[1]
	sessionID := parts[2]

	if quality == "best" {
		quality = ""
	}

	taskManager.Mu.Lock()
	session, exists := taskManager.YTDLPSessions[sessionID]
	if exists {
		delete(taskManager.YTDLPSessions, sessionID)
	}
	taskManager.Mu.Unlock()

	if !exists {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Sesi tidak ditemukan"))
		return
	}

	_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Memulai download..."))

	text := fmt.Sprintf("🎬 *YT\\-DLP dimulai* kualiti: `%s`", quality)
	if quality == "" {
		text = "🎬 *YT\\-DLP dimulai* kualiti: `Terbaik`"
	}
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	_, _ = bot.Send(editMsg)

	taskManager.CreateTask(TypeYTDLP, session.URL, "video", callback.Message.Chat.ID, callback.Message.MessageID, callback.From.ID, session.Zip, false, session.Password, quality)
}

func HandleTorrent(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	url, zip, unzip, password, quality := ParseFlags(args)
	if url == "" {
		url = ExtractMagnetFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan magnet link\\.")
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	fileName := "torrent_download"
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🧲 Menambahkan torrent…"))
	taskManager.CreateTask(TypeTorrent, url, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password, quality)
}

func handleTelegramFileDownload(bot *tgbotapi.BotAPI, message *tgbotapi.Message, fileID, fileName string, zip, unzip bool, password, quality string) {
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "📥 *Memulai download…*"))

	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		editStatusMessage(bot, statusMsg.Chat.ID, statusMsg.MessageID, fmt.Sprintf("❌ *Error:* %s", EscapeMarkdownV2(err.Error())))
		return
	}

	fileURL := file.Link(bot.Token)
	taskManager.CreateTask(TypeMirror, fileURL, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password, quality)
}

func downloadWithAria2(bot *tgbotapi.BotAPI, task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	updateTaskStatus(bot, task)

	outputDir := filepath.Join(taskManager.DownloadDir, task.ID)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		updateTaskStatus(bot, task)
		return
	}

	args := []string{
		"--dir=" + outputDir,
		"--allow-overwrite=true",
		"--max-connection-per-server=16",
		"--split=16",
		"--summary-interval=1",
		"--console-log-level=notice",
		task.URL,
	}

	if IsMagnetLink(task.URL) {
		args = append(args, "--seed-time=0")
	}

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	//nolint:gosec
	cmd := exec.CommandContext(ctx, "aria2c", args...)
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		task.SetError(fmt.Sprintf("aria2c failed: %v", err))
		updateTaskStatus(bot, task)
		return
	}

	go parseAria2Progress(bot, task, stdout)
	_ = cmd.Wait()

	if task.Status == StatusCancelled {
		cleanupTask(task)
		return
	}

	task.LocalPath = findDownloadedFile(outputDir)
	task.FileName = filepath.Base(task.LocalPath)

	if task.Unzip && IsArchiveFile(task.LocalPath) {
		if err := extractArchive(bot, task); err != nil {
			task.SetError(fmt.Sprintf("Extraction failed: %v", err))
			updateTaskStatus(bot, task)
			cleanupTask(task)
			return
		}
	}

	if task.Zip {
		if err := createZipArchive(bot, task); err != nil {
			task.SetError(fmt.Sprintf("Compression failed: %v", err))
			updateTaskStatus(bot, task)
			cleanupTask(task)
			return
		}
	}

	if err := uploadWithRclone(bot, task); err != nil {
		task.SetError(fmt.Sprintf("Upload failed: %v", err))
	} else {
		task.SetStatus(StatusCompleted)
	}
	updateTaskStatus(bot, task)
	cleanupTask(task)
	handleAutoDelete(bot, task)
}

func downloadWithYTDLP(bot *tgbotapi.BotAPI, task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	updateTaskStatus(bot, task)

	outputDir := filepath.Join(taskManager.DownloadDir, task.ID)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		updateTaskStatus(bot, task)
		return
	}

	args := []string{
		"-o", filepath.Join(outputDir, "%(title)s.%(ext)s"),
		"--newline",
		"--no-playlist",
		"--continue",
		"--merge-output-format", "mp4",
		"--no-check-certificate",
		"--format-sort", "res,vcodec:h264",
		"--extractor-args", "youtube:player-client=tv,android,web",
		"--remote-components", "ejs:github",
	}

	args = append(args, "--js-runtime", "node")

	if task.Quality != "" {
		format := fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", task.Quality, task.Quality)
		args = append(args, "-f", format)
	}

	cookiesPath := filepath.Join(taskManager.ConfigDir, "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		args = append(args, "--cookies", cookiesPath)
	}

	args = append(args, task.URL)

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	//nolint:gosec
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		task.SetError(fmt.Sprintf("yt-dlp failed to start: %v", err))
		updateTaskStatus(bot, task)
		return
	}
	go parseYTDLPProgress(bot, task, stdout)

	if err := cmd.Wait(); err != nil {
		task.Mu.RLock()
		capturedErr := task.Error
		task.Mu.RUnlock()

		errMsg := fmt.Sprintf("yt-dlp error: %v", err)
		if capturedErr != "" {
			errMsg = capturedErr
		}

		task.SetError(errMsg)
		updateTaskStatus(bot, task)
		cleanupTask(task)
		return
	}

	task.LocalPath = findDownloadedFile(outputDir)
	if task.LocalPath == "" {
		task.SetError("Downloaded file not found or incomplete (.part files ignored)")
		updateTaskStatus(bot, task)
		cleanupTask(task)
		return
	}
	task.FileName = filepath.Base(task.LocalPath)

	if info, err := os.Stat(task.LocalPath); err == nil {
		task.DownloadedSize = info.Size()
		task.TotalSize = info.Size()
	}

	if err := uploadWithRclone(bot, task); err != nil {
		task.SetError(fmt.Sprintf("Upload failed: %v", err))
	} else {
		task.SetStatus(StatusCompleted)
	}
	updateTaskStatus(bot, task)
	cleanupTask(task)
}

func parseAria2Progress(bot *tgbotapi.BotAPI, task *Task, reader io.ReadCloser) {
	scanner := bufio.NewScanner(reader)
	progressRegex := regexp.MustCompile(`[\(\[\s](\d+)%`)
	speedRegex := regexp.MustCompile(`DL:(\S+)`)
	etaRegex := regexp.MustCompile(`ETA:(\S+)`)
	lastUpdate := time.Now()

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) < 200 {
			log.Printf("[Aria2 Output] %s", line)
		}

		matches := progressRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
				task.Progress = pct
			}
		}

		speedMatches := speedRegex.FindStringSubmatch(line)
		if len(speedMatches) >= 2 {
			speedStr := speedMatches[1]
			speedStr = strings.TrimRight(speedStr, "]")
			task.Speed = ParseBytesString(speedStr)
		}

		etaMatches := etaRegex.FindStringSubmatch(line)
		if len(etaMatches) >= 2 {
			etaStr := etaMatches[1]
			etaStr = strings.TrimRight(etaStr, "]")
			if d, err := time.ParseDuration(etaStr); err == nil {
				task.ETA = d
			}
		}

		if time.Since(lastUpdate) >= 3*time.Second {
			updateTaskStatus(bot, task)
			lastUpdate = time.Now()
		}
	}
}

func parseYTDLPProgress(bot *tgbotapi.BotAPI, task *Task, reader io.ReadCloser) {
	scanner := bufio.NewScanner(reader)
	progressRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)%`)
	lastUpdate := time.Now()

	for scanner.Scan() {
		line := scanner.Text()

		log.Printf("[YT-DLP %s] %s", task.ID, line)

		if strings.Contains(line, "ERROR:") {
			task.Mu.Lock()
			task.Error = line
			task.Mu.Unlock()
		}

		matches := progressRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
				task.Progress = pct
			}
			if time.Since(lastUpdate) >= 5*time.Second {
				updateTaskStatus(bot, task)
				lastUpdate = time.Now()
			}
		}
	}
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

func updateTaskStatus(bot *tgbotapi.BotAPI, task *Task) {
	snapshot := task.GetSnapshot()
	emoji := StatusEmoji(string(snapshot.Status))
	bar := ProgressBar(snapshot.Progress, 10)

	text := fmt.Sprintf("%s *%s*\n📄 `%s`\n`%s`\n⚡ %s \\| ⏱ %s",
		emoji, EscapeMarkdownV2(FormatStatus(string(snapshot.Status))),
		EscapeMarkdownV2(TruncateString(snapshot.FileName, 50)),
		EscapeMarkdownV2(bar),
		EscapeMarkdownV2(FormatSpeed(snapshot.Speed)),
		EscapeMarkdownV2(FormatDuration(snapshot.ETA)))

	switch snapshot.Status {
	case StatusCompleted:
		duration := snapshot.CompletedAt.Sub(snapshot.StartedAt)
		if snapshot.StartedAt.IsZero() {
			duration = snapshot.CompletedAt.Sub(snapshot.CreatedAt)
		}

		sizeStr := "Unknown"
		if snapshot.TotalSize > 0 {
			sizeStr = FormatBytes(snapshot.TotalSize)
		} else if snapshot.DownloadedSize > 0 {
			sizeStr = FormatBytes(snapshot.DownloadedSize)
		}

		text = fmt.Sprintf("✅ *Completed\\!*\n\n"+
			"📄 *Name:* `%s`\n"+
			"📦 *Size:* `%s`\n"+
			"⏱ *Time:* `%s`\n"+
			"📁 *Path:* `%s`",
			EscapeMarkdownV2(snapshot.FileName),
			EscapeMarkdownV2(sizeStr),
			EscapeMarkdownV2(FormatDuration(duration)),
			EscapeMarkdownV2(snapshot.RemotePath))
	case StatusFailed:
		text = fmt.Sprintf("❌ *Failed\\!*\n📄 `%s`\nError: `%s`",
			EscapeMarkdownV2(snapshot.FileName),
			EscapeMarkdownV2(TruncateString(snapshot.Error, 100)))
	}

	editMsg := tgbotapi.NewEditMessageText(snapshot.ChatID, snapshot.MessageID, text)
	editMsg.ParseMode = MarkdownV2

	if snapshot.Status == StatusDownloading || snapshot.Status == StatusUploading {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", fmt.Sprintf("cancel_task:%s", snapshot.ID)),
			),
		)
		editMsg.ReplyMarkup = &keyboard
	} else if snapshot.Status == StatusCompleted && snapshot.RemoteURL != "" {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("☁️ Cloud Link", snapshot.RemoteURL),
			),
		)
		editMsg.ReplyMarkup = &keyboard
	}

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("[StatusUpdate] Gagal update status task %s: %v", task.ID, err)
	}
}

func editStatusMessage(bot *tgbotapi.BotAPI, chatID int64, msgID int, text string) {
	editMsg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	editMsg.ParseMode = MarkdownV2
	_, _ = bot.Send(editMsg)
}

func processTask(bot *tgbotapi.BotAPI, task *Task) {
	log.Printf("[Task %s] Starting processing type: %s", task.ID, task.Type)

	switch task.Type {
	case TypeMirror, TypeLeech, TypeTorrent:
		downloadWithAria2(bot, task)
	case TypeYTDLP:
		downloadWithYTDLP(bot, task)
	}
}
