package handlers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleMirror(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	url, zip, unzip, password := ParseFlags(args)
	var fileName string

	if message.ReplyToMessage != nil {
		reply := message.ReplyToMessage
		if reply.Document != nil {
			fileName = reply.Document.FileName
			go handleTelegramFileDownload(bot, message, reply.Document.FileID, fileName, zip, unzip, password)
			return
		} else if reply.Video != nil {
			fileName = reply.Video.FileName
			if fileName == "" {
				fileName = fmt.Sprintf("video_%d.mp4", time.Now().Unix())
			}
			go handleTelegramFileDownload(bot, message, reply.Video.FileID, fileName, zip, unzip, password)
			return
		}
	}

	if url != "" {
		fileName = GetFileNameFromURL(url)
		statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "📥 Memulai mirror…"))
		taskManager.CreateTask(TypeMirror, url, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nReply ke file atau berikan URL\\.")
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)
}

func HandleLeech(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	url, zip, unzip, password := ParseFlags(args)
	if url == "" {
		url = ExtractMagnetFromText(args)
	}
	if url == "" {
		url = ExtractURLFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL untuk di\\-leech\\.")
		msg.ParseMode = "MarkdownV2"
		bot.Send(msg)
		return
	}

	fileName := GetFileNameFromURL(url)
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🔗 Memulai leech…"))
	taskManager.CreateTask(TypeLeech, url, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password)
}

func HandleYTDLP(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	url, zip, _, password := ParseFlags(args)
	if url == "" {
		url = ExtractURLFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL video\\.")
		msg.ParseMode = "MarkdownV2"
		bot.Send(msg)
		return
	}

	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🎬 Mengambil info video…"))
	taskManager.CreateTask(TypeYTDLP, url, "video", message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, false, password)
}

func HandleTorrent(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	url, zip, unzip, password := ParseFlags(args)
	if url == "" {
		url = ExtractMagnetFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan magnet link\\.")
		msg.ParseMode = "MarkdownV2"
		bot.Send(msg)
		return
	}

	fileName := "torrent_download"
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🧲 Menambahkan torrent…"))
	taskManager.CreateTask(TypeTorrent, url, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password)
}

func handleTelegramFileDownload(bot *tgbotapi.BotAPI, message *tgbotapi.Message, fileID, fileName string, zip, unzip bool, password string) {
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "📥 *Memulai download…*"))

	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		editStatusMessage(bot, statusMsg.Chat.ID, statusMsg.MessageID, fmt.Sprintf("❌ *Error:* %s", EscapeMarkdownV2(err.Error())))
		return
	}

	fileURL := file.Link(bot.Token)
	taskManager.CreateTask(TypeMirror, fileURL, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password)
}

func handleURLDownload(bot *tgbotapi.BotAPI, message *tgbotapi.Message, url, fileName string, taskType TaskType, zip, unzip bool, password string) {
	statusText := fmt.Sprintf("📥 *Memulai %s…*", string(taskType))
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, statusText))

	task := taskManager.CreateTask(taskType, url, fileName, message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, unzip, password)
	downloadWithAria2(bot, task)
}

func handleYTDLPDownload(bot *tgbotapi.BotAPI, message *tgbotapi.Message, url string, zip bool, password string) {
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🎬 *Mengambil info video…*"))
	task := taskManager.CreateTask(TypeYTDLP, url, "video", message.Chat.ID, statusMsg.MessageID, message.From.ID, zip, false, password)
	go downloadWithYTDLP(bot, task)
}

func downloadWithAria2(bot *tgbotapi.BotAPI, task *Task) {
	task.SetStatus(StatusDownloading)
	updateTaskStatus(bot, task)

	outputDir := filepath.Join(taskManager.DownloadDir, task.ID)
	os.MkdirAll(outputDir, 0755)

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

	cmd := exec.CommandContext(ctx, "aria2c", args...)
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		task.SetError(fmt.Sprintf("aria2c failed: %v", err))
		updateTaskStatus(bot, task)
		return
	}

	go parseAria2Progress(bot, task, stdout)
	cmd.Wait()

	if task.Status == StatusCancelled {
		cleanupTask(task)
		return
	}

	task.LocalPath = findDownloadedFile(outputDir)
	task.FileName = filepath.Base(task.LocalPath)

	if task.Unzip && IsArchiveFile(task.LocalPath) {
		extractArchive(bot, task)
	}

	if task.Zip {
		createZipArchive(bot, task)
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
	updateTaskStatus(bot, task)

	outputDir := filepath.Join(taskManager.DownloadDir, task.ID)
	os.MkdirAll(outputDir, 0755)

	args := []string{"-o", filepath.Join(outputDir, "%(title)s.%(ext)s"), "--newline", task.URL}
	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	stdout, _ := cmd.StdoutPipe()

	cmd.Start()
	go parseYTDLPProgress(bot, task, stdout)
	cmd.Wait()

	task.LocalPath = findDownloadedFile(outputDir)
	task.FileName = filepath.Base(task.LocalPath)

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
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Size() > maxSize {
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
		text = fmt.Sprintf("✅ *Completed\\!*\n📄 `%s`\n📁 `%s`",
			EscapeMarkdownV2(snapshot.FileName),
			EscapeMarkdownV2(snapshot.RemotePath))
	case StatusFailed:
		text = fmt.Sprintf("❌ *Failed\\!*\n📄 `%s`\nError: `%s`",
			EscapeMarkdownV2(snapshot.FileName),
			EscapeMarkdownV2(TruncateString(snapshot.Error, 100)))
	}

	editMsg := tgbotapi.NewEditMessageText(snapshot.ChatID, snapshot.MessageID, text)
	editMsg.ParseMode = "MarkdownV2"

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
	editMsg.ParseMode = "MarkdownV2"
	bot.Send(editMsg)
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
