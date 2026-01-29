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
		task := taskManager.CreateTask(TypeMirror, url, fileName, message.Chat.ID, 0, message.From.ID, zip, unzip, password, quality)
		UpdateSharedDashboard(bot, message.Chat.ID, true)
		log.Printf("[Mirror] Task created: %s", task.ID)
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
	task := taskManager.CreateTask(TypeLeech, url, fileName, message.Chat.ID, 0, message.From.ID, zip, unzip, password, quality)
	UpdateSharedDashboard(bot, message.Chat.ID, true)
	log.Printf("[Leech] Task created: %s", task.ID)
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

	task := taskManager.CreateTask(TypeYTDLP, url, "video", message.Chat.ID, 0, message.From.ID, zip, false, password, quality)
	UpdateSharedDashboard(bot, message.Chat.ID, true)
	log.Printf("[YTDLP] Task created: %s", task.ID)
}

func showYTDLPQualityMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message, url string, zip bool, password string) {
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🎬 Menganalisa kualitas video…"))

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

	cookiesPath := filepath.Join(taskManager.ConfigDir, "cookies.txt")
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
		editStatusMessage(bot, statusMsg.Chat.ID, statusMsg.MessageID, fmt.Sprintf("❌ *Gagal menganalisa video:* %v\n\n_Pastikan URL valid atau coba lagi nanti\\._", EscapeMarkdownV2(err.Error())))
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
		editStatusMessage(bot, statusMsg.Chat.ID, statusMsg.MessageID, "❌ *Gagal memproses data video*")
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

	_, _ = bot.Request(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))

	task := taskManager.CreateTask(TypeYTDLP, session.URL, "video", callback.Message.Chat.ID, 0, callback.From.ID, session.Zip, false, session.Password, quality)
	UpdateSharedDashboard(bot, callback.Message.Chat.ID, true)
	log.Printf("[YTDLPCallback] Task created: %s", task.ID)
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
	task := taskManager.CreateTask(TypeTorrent, url, fileName, message.Chat.ID, 0, message.From.ID, zip, unzip, password, quality)
	UpdateSharedDashboard(bot, message.Chat.ID, true)
	log.Printf("[Torrent] Task created: %s", task.ID)
}

func handleTelegramFileDownload(bot *tgbotapi.BotAPI, message *tgbotapi.Message, fileID, fileName string, zip, unzip bool, password, quality string) {
	tgFile, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ *Error:* %s", EscapeMarkdownV2(err.Error())))
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	fileURL := tgFile.Link(bot.Token)
	task := taskManager.CreateTask(TypeMirror, fileURL, fileName, message.Chat.ID, 0, message.From.ID, zip, unzip, password, quality)
	UpdateSharedDashboard(bot, message.Chat.ID, true)
	log.Printf("[TGDownload] Task created: %s", task.ID)
}

func downloadWithAria2(bot *tgbotapi.BotAPI, task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	updateTaskStatus(bot, task)

	outputDir := filepath.Join(taskManager.DownloadDir, task.ID)
	//nolint:gosec
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		updateTaskStatus(bot, task)
		return
	}

	args := []string{
		"--dir=" + outputDir,
		"--allow-overwrite=true",
		"--max-connection-per-server=16",
		"--split=32",
		"--min-split-size=1M",
		"--max-overall-download-limit=0",
		"--max-resume-failure-tries=0",
		"--retry-wait=1",
		"--connect-timeout=30",
		"--timeout=30",
		"--console-log-level=notice",
		"--summary-interval=1",
		"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"--header=Accept-Encoding: gzip, deflate",
		"--async-dns=true",
		"--file-allocation=none",
		"--disk-cache=128M",
		"--enable-mmap=true",
		"--check-certificate=false",
		"--optimize-concurrent-downloads=true",
		"--max-file-not-found=2",
		"--disable-ipv6=true",
		"--enable-http-pipelining=true",
		"--peer-id-prefix=-AZ2060-",
		"--peer-agent=Transmission/2.94",
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
	err := cmd.Wait()
	if err != nil && task.Status != StatusCancelled {
		task.SetError(fmt.Sprintf("aria2c execution failed: %v", err))
		updateTaskStatus(bot, task)
		cleanupTask(task)
		return
	}

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
	//nolint:gosec
	if err := os.MkdirAll(outputDir, 0777); err != nil {
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
		"--format-sort", "res,fps,codec:vp9,vcodec,br",
		"--extractor-args", "youtube:player-client=web,web_embedded,ios,mweb,tv",
		"--socket-timeout", "60",
		"--concurrent-fragments", "16",
		"--buffer-size", "1M",
		"--add-header", "Accept-Language: en-US,en;q=0.9",
		"--add-header", "Referer: https://www.youtube.com/",
		"--remote-components", "ejs:github",
		"--js-runtime", "node",
		"--cache-dir", "/home/botuser/.cache/yt-dlp-final",
	}

	cookiesPath := filepath.Join(taskManager.ConfigDir, "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		args = append(args, "--cookies", cookiesPath)
		args = append(args, "--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	}

	if task.Quality != "" {
		format := fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", task.Quality, task.Quality)
		args = append(args, "-f", format)
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
		if task.Status == StatusCancelled {
			cleanupTask(task)
			return
		}

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

	if task.Status == StatusCancelled {
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
	statusRegex := regexp.MustCompile(`\[#\w+\s+(\S+)/(\S+)\((\d+)%\)(?:\s+CN:(\d+))?.*DL:(\S+)(?:\s+ETA:(\S+))?\]`)
	lastUpdate := time.Now()

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) < 200 {
			log.Printf("[Aria2 Output] %s", line)
		}

		matches := statusRegex.FindStringSubmatch(line)
		if len(matches) >= 5 {
			downloadedStr := matches[1]
			totalStr := matches[2]
			pctStr := matches[3]
			cnStr := matches[4]
			speedStr := matches[5]
			etaStr := ""
			if len(matches) >= 7 {
				etaStr = matches[6]
			}

			downloaded := ParseBytesString(downloadedStr)
			total := ParseBytesString(totalStr)
			speed := ParseBytesString(speedStr)

			task.Mu.Lock()
			task.DownloadedSize = downloaded
			task.TotalSize = total
			task.Speed = speed
			if cn, err := strconv.Atoi(cnStr); err == nil {
				task.Connections = cn
			}
			if pct, err := strconv.ParseFloat(pctStr, 64); err == nil {
				task.Progress = pct
			}
			if total > 0 && downloaded > 0 && task.Progress == 0 {
				task.Progress = float64(downloaded) / float64(total) * 100
			}
			if etaStr != "" {
				etaStr = strings.TrimRight(etaStr, "]")
				if d, err := time.ParseDuration(etaStr); err == nil {
					task.ETA = d
				}
			}
			task.Mu.Unlock()
		}

		if time.Since(lastUpdate) >= 3*time.Second {
			updateTaskStatus(bot, task)
			lastUpdate = time.Now()
		}
	}
}

func parseYTDLPProgress(bot *tgbotapi.BotAPI, task *Task, reader io.ReadCloser) {
	scanner := bufio.NewScanner(reader)
	progressRegex := regexp.MustCompile(`\[download\]\s+([\d\.]+)%\s+of\s+(?:~)?(\S+)\s+at\s+(\S+)\s+ETA\s+(\S+)`)
	lastUpdate := time.Now()

	for scanner.Scan() {
		line := scanner.Text()

		log.Printf("[YT-DLP %s] %s", task.ID, line)

		if strings.Contains(line, "ERROR:") {
			task.Mu.Lock()
			task.Error = line
			task.Mu.Unlock()
		}

		if strings.HasPrefix(line, "[download] Destination:") {
			title := strings.TrimPrefix(line, "[download] Destination:")
			title = filepath.Base(strings.TrimSpace(title))
			task.Mu.Lock()
			task.FileName = title
			task.Mu.Unlock()
		} else if strings.HasPrefix(line, "[download]") && strings.Contains(line, "has already been downloaded") {
			title := strings.TrimPrefix(line, "[download]")
			title = strings.TrimSuffix(title, " has already been downloaded")
			title = filepath.Base(strings.TrimSpace(title))
			task.Mu.Lock()
			task.FileName = title
			task.Mu.Unlock()
		}

		matches := progressRegex.FindStringSubmatch(line)
		if len(matches) >= 5 {
			pctStr := matches[1]
			totalStr := matches[2]
			speedStr := matches[3]
			etaStr := matches[4]

			task.Mu.Lock()
			if pct, err := strconv.ParseFloat(pctStr, 64); err == nil {
				task.Progress = pct
			}
			task.TotalSize = ParseBytesString(totalStr)
			task.Speed = ParseBytesString(speedStr)
			task.Connections = 16
			if d, err := parseYTDLPDuration(etaStr); err == nil {
				task.ETA = d
			}
			task.Mu.Unlock()
		}

		if time.Since(lastUpdate) >= 5*time.Second {
			updateTaskStatus(bot, task)
			lastUpdate = time.Now()
		}
	}
}

func parseYTDLPDuration(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	var h, m, s_ int
	var err error

	switch len(parts) {
	case 3:
		h, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		m, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		s_, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, err
		}
	case 2:
		m, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		s_, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
	default:
		s_, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
	}

	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s_)*time.Second, nil
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

	if snapshot.Status != StatusCompleted && snapshot.Status != StatusFailed && snapshot.Status != StatusCancelled {
		UpdateSharedDashboard(bot, snapshot.ChatID, false)
		return
	}

	UpdateSharedDashboard(bot, snapshot.ChatID, false)

	var text string
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
	default:
		return
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

	_, _ = bot.Send(msg)
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
