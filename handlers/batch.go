package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type BatchTask struct {
	ID          string
	Name        string
	URLs        []string
	SubTasks    []*Task
	Status      TaskStatus
	ChatID      int64
	MessageID   int
	UserID      int64
	CreatedAt   time.Time
	CompletedAt time.Time
	ZipAll      bool
	Password    string
	Priority    int
	TotalSize   int64
	Downloaded  int64
	Progress    float64
	Error       string
	LocalPath   string
	RemotePath  string
	RemoteURL   string
	Ctx         context.Context
	CancelFunc  context.CancelFunc
	Mu          sync.RWMutex
	Completed   int
	Failed      int
	DownloadDir string
}

type BatchManager struct {
	Batches       map[string]*BatchTask
	PriorityQueue []*BatchTask
	Mu            sync.RWMutex
}

var batchManager *BatchManager

func init() {
	batchManager = &BatchManager{
		Batches:       make(map[string]*BatchTask),
		PriorityQueue: make([]*BatchTask, 0),
	}
}

type BatchOptions struct {
	URLs     []string
	Name     string
	ZipAll   bool
	Password string
	Priority int
}

func parseBatchArguments(args string) *BatchOptions {
	lines := strings.Split(args, "\n")
	options := &BatchOptions{
		URLs:     make([]string, 0),
		Priority: 5,
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "-name ") {
			options.Name = strings.TrimPrefix(line, "-name ")
			continue
		}
		if strings.HasPrefix(line, "-n ") {
			options.Name = strings.TrimPrefix(line, "-n ")
			continue
		}
		if line == "-z" || line == "-zip" {
			options.ZipAll = true
			continue
		}
		if strings.HasPrefix(line, "-p ") {
			options.Password = strings.TrimPrefix(line, "-p ")
			continue
		}
		if strings.HasPrefix(line, "-priority ") {
			_, err := fmt.Sscanf(strings.TrimPrefix(line, "-priority "), "%d", &options.Priority)
			if err != nil {
				options.Priority = 5
			}
			if options.Priority < 1 {
				options.Priority = 1
			}
			if options.Priority > 10 {
				options.Priority = 10
			}
			continue
		}

		if IsValidURL(line) {
			options.URLs = append(options.URLs, line)
		}
	}

	return options
}

func HandleBatch(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args string) {
	if args == "" {
		sendBatchHelp(bot, message.Chat.ID)
		return
	}

	options := parseBatchArguments(args)

	if len(options.URLs) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nTidak ada URL valid yang ditemukan\\.\n\nGunakan `/batch` untuk melihat cara penggunaan\\.")
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	if !options.ZipAll {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("🔄 *Creating %d regular tasks...*", len(options.URLs)))
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)

		for _, url := range options.URLs {
			fileName := GetFileNameFromURL(url)
			task := taskManager.CreateTask(TypeMirror, url, fileName, message.Chat.ID, 0, message.From.ID, false, false, options.Password, "")
			log.Printf("[Batch-Mirror] Created task %s for %s", task.ID, url)
		}

		UpdateSharedDashboard(bot, message.Chat.ID, true)
		return
	}

	if options.Name == "" {
		options.Name = fmt.Sprintf("batch_%s", time.Now().Format("20060102_150405"))
	}

	batch := createBatchTask(options.Name, options.URLs, message.Chat.ID, 0, message.From.ID, options.ZipAll, options.Password, options.Priority)

	UpdateSharedDashboard(bot, message.Chat.ID, true)

	go processBatchTask(bot, batch)

	log.Printf("[Batch] Created batch %s with %d URLs, priority: %d", batch.ID, len(options.URLs), options.Priority)
}

func updateBatchStatus(bot *tgbotapi.BotAPI, batch *BatchTask) {
	UpdateSharedDashboard(bot, batch.ChatID, false)
}

func sendBatchHelp(bot *tgbotapi.BotAPI, chatID int64) {
	helpText := `📦 *Batch Download System*

*Penggunaan:*
` + "```" + `
/batch
URL1
URL2
URL3
` + "```" + `

*Flags Opsional:*
• ` + "`-name <nama>`" + ` \\- Nama batch
• ` + "`-z`" + ` atau ` + "`-zip`" + ` \\- Zip semua hasil
• ` + "`-p <password>`" + ` \\- Password untuk zip
• ` + "`-priority <1-10>`" + ` \\- Prioritas \\(default: 5\\)

*Contoh:*
` + "```" + `
/batch -name MyDownloads -z
[https://example.com/file1.zip](https://example.com/file1.zip)
[https://example.com/file2.mp4](https://example.com/file2.mp4)
[https://example.com/file3.rar](https://example.com/file3.rar)
` + "```" + `

*Fitur:*
✅ Download multiple URL sekaligus
✅ Zip semua hasil dalam satu archive
✅ Queue management dengan prioritas
✅ Progress tracking per\\-file dan total
✅ Cancel seluruh batch atau per\\-file`

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = MarkdownV2
	_, _ = bot.Send(msg)
}

func createBatchTask(name string, urls []string, chatID int64, msgID int, userID int64, zipAll bool, password string, priority int) *BatchTask {
	ctx, cancel := context.WithCancel(context.Background())

	batchID := uuid.New().String()[:8]
	downloadDir := filepath.Join(taskManager.DownloadDir, "batch_"+batchID)

	batch := &BatchTask{
		ID:          batchID,
		Name:        name,
		URLs:        urls,
		SubTasks:    make([]*Task, len(urls)),
		Status:      StatusQueued,
		ChatID:      chatID,
		MessageID:   msgID,
		UserID:      userID,
		CreatedAt:   time.Now(),
		ZipAll:      zipAll,
		Password:    password,
		Priority:    priority,
		Ctx:         ctx,
		CancelFunc:  cancel,
		DownloadDir: downloadDir,
	}

	for i, url := range urls {
		batch.SubTasks[i] = createBatchSubTask(batch, url, i)
	}

	batchManager.Mu.Lock()
	batchManager.Batches[batch.ID] = batch
	batchManager.PriorityQueue = append(batchManager.PriorityQueue, batch)

	sort.Slice(batchManager.PriorityQueue, func(i, j int) bool {
		return batchManager.PriorityQueue[i].Priority > batchManager.PriorityQueue[j].Priority
	})
	batchManager.Mu.Unlock()

	return batch
}

func processBatchTask(bot *tgbotapi.BotAPI, batch *BatchTask) {
	log.Printf("[Batch %s] Starting processing %d URLs", batch.ID, len(batch.URLs))

	if err := os.MkdirAll(batch.DownloadDir, 0750); err != nil {
		batch.SetError(fmt.Sprintf("Failed to create batch directory: %v", err))
		updateBatchStatus(bot, batch)
		return
	}

	batch.SetStatus(StatusDownloading)
	updateBatchStatus(bot, batch)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)

	for i, url := range batch.URLs {
		select {
		case <-batch.Ctx.Done():
			log.Printf("[Batch %s] Cancelled", batch.ID)
			batch.SetStatus(StatusCancelled)
			updateBatchStatus(bot, batch)
			return
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(idx int, downloadURL string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			subTask := batch.SubTasks[idx]

			err := downloadBatchItem(batch, subTask)

			if err != nil {
				batch.Mu.Lock()
				subTask.SetError(err.Error())
				batch.Failed++
				log.Printf("[Batch %s] Sub-task %s failed: %v", batch.ID, subTask.ID, err)
				batch.Progress = float64(batch.Completed+batch.Failed) / float64(len(batch.URLs)) * 100
				batch.Mu.Unlock()
			} else {
				if !batch.ZipAll {
					subTask.SetStatus(StatusUploading)
					updateBatchStatus(bot, batch)

					upErr := UploadWithRclone(bot, subTask)

					batch.Mu.Lock()
					if upErr != nil {
						subTask.SetError(upErr.Error())
						batch.Failed++
						log.Printf("[Batch %s] Sub-task %s upload failed: %v", batch.ID, subTask.ID, upErr)
					} else {
						subTask.SetStatus(StatusCompleted)
						batch.Completed++
						batch.Downloaded += subTask.DownloadedSize
						log.Printf("[Batch %s] Sub-task %s completed", batch.ID, subTask.ID)
					}
					batch.Progress = float64(batch.Completed+batch.Failed) / float64(len(batch.URLs)) * 100
					batch.Mu.Unlock()
				} else {
					batch.Mu.Lock()
					subTask.SetStatus(StatusCompleted)
					batch.Completed++
					batch.Downloaded += subTask.DownloadedSize
					log.Printf("[Batch %s] Sub-task %s downloaded (waiting for zip)", batch.ID, subTask.ID)
					batch.Progress = float64(batch.Completed+batch.Failed) / float64(len(batch.URLs)) * 100
					batch.Mu.Unlock()
				}
			}

			updateBatchStatus(bot, batch)
		}(i, url)
	}

	wg.Wait()

	select {
	case <-batch.Ctx.Done():
		batch.SetStatus(StatusCancelled)
		updateBatchStatus(bot, batch)
		cleanupBatch(batch)
		return
	default:
	}

	if batch.ZipAll && batch.Completed > 0 {
		batch.SetStatus(StatusZipping)
		updateBatchStatus(bot, batch)

		err := zipBatchResults(batch)
		if err != nil {
			batch.SetError(fmt.Sprintf("Zip failed: %v", err))
			updateBatchStatus(bot, batch)
			cleanupBatch(batch)
			return
		}

		batch.SetStatus(StatusUploading)
		updateBatchStatus(bot, batch)

		err = uploadBatchResults(batch)
		if err != nil {
			batch.SetError(fmt.Sprintf("Upload failed: %v", err))
			updateBatchStatus(bot, batch)
			cleanupBatch(batch)
			return
		}
	}

	batch.SetStatus(StatusCompleted)
	batch.CompletedAt = time.Now()
	updateBatchStatus(bot, batch)

	sendBatchCompletionMessage(bot, batch)
	cleanupBatch(batch)
}

func createBatchSubTask(batch *BatchTask, url string, index int) *Task {
	ctx, cancel := context.WithCancel(batch.Ctx)

	fileName := GetFileNameFromURL(url)
	if fileName == "unknown_file" {
		fileName = fmt.Sprintf("file_%d", index+1)
	}

	task := &Task{
		ID:         fmt.Sprintf("%s_%d", batch.ID, index+1),
		Type:       TypeLeech,
		Status:     StatusQueued,
		URL:        url,
		FileName:   fileName,
		ChatID:     batch.ChatID,
		MessageID:  batch.MessageID,
		UserID:     batch.UserID,
		CreatedAt:  time.Now(),
		Ctx:        ctx,
		CancelFunc: cancel,
	}

	return task
}

func downloadBatchItem(batch *BatchTask, task *Task) error {
	task.SetStatus(StatusDownloading)

	taskDir := filepath.Join(batch.DownloadDir, task.ID)
	if err := os.MkdirAll(taskDir, 0750); err != nil {
		return fmt.Errorf("failed to create task directory: %v", err)
	}

	configPath := filepath.Join(taskManager.ConfigDir, "cookies.txt")
	args := []string{
		task.URL,
		"-d", taskDir,
		"--max-connection-per-server=16",
		"--split=16",
		"--min-split-size=1M",
		"--max-concurrent-downloads=1",
		"--file-allocation=none",
		"--continue=true",
		"--auto-file-renaming=false",
		"--summary-interval=1",
		"--download-result=full",
		"--console-log-level=notice",
		"--allow-overwrite=true",
		"--load-cookies=" + configPath,
	}

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "aria2c", args...)
	output, err := cmd.CombinedOutput()

	if task.Status == StatusCancelled {
		return fmt.Errorf("cancelled")
	}

	if err != nil {
		return fmt.Errorf("aria2c failed: %v, output: %s", err, string(output))
	}

	downloadedFile := findDownloadedFile(taskDir)
	if downloadedFile == "" {
		return fmt.Errorf("no file downloaded")
	}

	task.LocalPath = downloadedFile
	task.FileName = filepath.Base(downloadedFile)

	if info, err := os.Stat(downloadedFile); err == nil {
		task.Mu.Lock()
		task.TotalSize = info.Size()
		task.DownloadedSize = info.Size()
		task.Progress = 100
		task.Mu.Unlock()

		batch.Mu.Lock()
		batch.TotalSize += info.Size()
		batch.Mu.Unlock()
	}

	return nil
}

func zipBatchResults(batch *BatchTask) error {
	log.Printf("[Batch %s] Zipping results...", batch.ID)

	zipFileName := SanitizeFileName(batch.Name) + ".zip"
	zipPath := filepath.Join(taskManager.DownloadDir, "batch_"+batch.ID+"_output", zipFileName)

	if err := os.MkdirAll(filepath.Dir(zipPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	args := []string{
		"a",
		"-tzip",
		zipPath,
		batch.DownloadDir + "/*",
		"-y",
		"-r",
	}

	if batch.Password != "" {
		args = append(args, "-p"+batch.Password)
		args = append(args, "-mhe=on")
	}

	ctx, cancel := context.WithCancel(batch.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "7z", args...)
	output, err := cmd.CombinedOutput()

	if batch.Status == StatusCancelled {
		return fmt.Errorf("cancelled")
	}

	if err != nil {
		return fmt.Errorf("7z failed: %v, output: %s", err, string(output))
	}

	batch.LocalPath = zipPath
	if info, err := os.Stat(zipPath); err == nil {
		batch.TotalSize = info.Size()
	}

	log.Printf("[Batch %s] Zip created: %s", batch.ID, zipPath)
	return nil
}

func uploadBatchResults(batch *BatchTask) error {
	log.Printf("[Batch %s] Uploading results...", batch.ID)

	uploadPath := batch.LocalPath
	if uploadPath == "" {
		uploadPath = batch.DownloadDir
	}

	remotePath := filepath.Join(taskManager.RcloneDest, filepath.Base(uploadPath))
	batch.RemotePath = remotePath

	configPath := filepath.Join(taskManager.ConfigDir, "rclone.conf")
	args := []string{
		"copy",
		uploadPath,
		taskManager.RcloneDest,
		"--config", configPath,
		"--progress",
		"--stats", "2s",
		"--stats-one-line",
		"--transfers", "10",
		"--checkers", "20",
		"--drive-chunk-size", "256M",
		"--drive-upload-cutoff", "256M",
		"--buffer-size", "128M",
		"-v",
	}

	ctx, cancel := context.WithCancel(batch.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", args...)
	output, err := cmd.CombinedOutput()

	if batch.Status == StatusCancelled {
		return fmt.Errorf("cancelled")
	}

	if err != nil {
		return fmt.Errorf("rclone failed: %v, output: %s", err, string(output))
	}

	uploadName := filepath.Base(uploadPath)
	linkArgs := []string{
		"link",
		"--config", configPath,
		filepath.Join(taskManager.RcloneDest, uploadName),
	}

	linkCmd := exec.CommandContext(ctx, "rclone", linkArgs...)
	linkOutput, linkErr := linkCmd.Output()
	if linkErr == nil {
		batch.RemoteURL = strings.TrimSpace(string(linkOutput))
	}

	log.Printf("[Batch %s] Upload completed. URL: %s", batch.ID, batch.RemoteURL)
	return nil
}

func sendBatchCompletionMessage(bot *tgbotapi.BotAPI, batch *BatchTask) {
	duration := batch.CompletedAt.Sub(batch.CreatedAt)

	text := fmt.Sprintf(`✅ *Batch Download Selesai\\!*

🏷️ *Name:* %s
📊 *Total Files:* %d
✅ *Berhasil:* %d
❌ *Gagal:* %d
📦 *Total Size:* %s
⏱️ *Duration:* %s`,
		EscapeMarkdownV2(batch.Name),
		len(batch.URLs),
		batch.Completed,
		batch.Failed,
		EscapeMarkdownV2(FormatBytes(batch.TotalSize)),
		EscapeMarkdownV2(FormatDuration(duration)),
	)

	if batch.RemoteURL != "" {
		text += fmt.Sprintf("\n\n🔗 *Link:* [Download](%s)", batch.RemoteURL)
	}

	msg := tgbotapi.NewMessage(batch.ChatID, text)
	msg.ParseMode = MarkdownV2

	if batch.RemoteURL != "" {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("📥 Download", batch.RemoteURL),
			),
		)
		msg.ReplyMarkup = keyboard
	}

	_, _ = bot.Send(msg)
}

func cleanupBatch(batch *BatchTask) {
	if batch.DownloadDir != "" {
		_ = os.RemoveAll(batch.DownloadDir)
	}

	outputDir := filepath.Join(taskManager.DownloadDir, "batch_"+batch.ID+"_output")
	_ = os.RemoveAll(outputDir)

	go func() {
		time.Sleep(1 * time.Hour)
		batchManager.Mu.Lock()
		delete(batchManager.Batches, batch.ID)
		for i, b := range batchManager.PriorityQueue {
			if b.ID == batch.ID {
				batchManager.PriorityQueue = append(batchManager.PriorityQueue[:i], batchManager.PriorityQueue[i+1:]...)
				break
			}
		}
		batchManager.Mu.Unlock()
	}()
}

func HandleBatchCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[1]
	batchID := parts[2]

	batchManager.Mu.RLock()
	batch, exists := batchManager.Batches[batchID]
	batchManager.Mu.RUnlock()

	if !exists {
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Batch not found"))
		return
	}

	switch action {
	case "refresh":
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Refreshed"))
		updateBatchStatus(bot, batch)

	case "cancel":
		batch.CancelFunc()
		batch.SetStatus(StatusCancelled)
		_, _ = bot.Request(tgbotapi.NewCallback(callback.ID, "🚫 Batch cancelled"))
		updateBatchStatus(bot, batch)
	}
}

func HandleBatchStatus(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	batchManager.Mu.RLock()
	defer batchManager.Mu.RUnlock()

	if len(batchManager.Batches) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📭 *Tidak ada batch aktif*")
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	text := "📦 *Active Batches:*\n\n"
	for _, batch := range batchManager.Batches {
		if batch.Status == StatusCompleted || batch.Status == StatusCancelled || batch.Status == StatusFailed {
			continue
		}
		emoji := StatusEmoji(string(batch.Status))
		text += fmt.Sprintf("%s `%s` \\- %s \\(%d/%d\\)\n",
			emoji,
			batch.ID,
			EscapeMarkdownV2(batch.Name),
			batch.Completed,
			len(batch.URLs),
		)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = MarkdownV2
	_, _ = bot.Send(msg)
}

func HandleCancelBatch(bot *tgbotapi.BotAPI, message *tgbotapi.Message, batchID string) {
	batchManager.Mu.RLock()
	batch, exists := batchManager.Batches[batchID]
	batchManager.Mu.RUnlock()

	if !exists {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ *Batch `%s` tidak ditemukan*", batchID))
		msg.ParseMode = MarkdownV2
		_, _ = bot.Send(msg)
		return
	}

	batch.CancelFunc()
	batch.SetStatus(StatusCancelled)
	updateBatchStatus(bot, batch)

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ *Batch `%s` dibatalkan*", batchID))
	msg.ParseMode = MarkdownV2
	_, _ = bot.Send(msg)
}

func (b *BatchTask) SetStatus(status TaskStatus) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	b.Status = status
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		b.CompletedAt = time.Now()
	}
}

func (b *BatchTask) SetError(err string) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	b.Error = err
	b.Status = StatusFailed
	b.CompletedAt = time.Now()
}

func GetBatchManager() *BatchManager {
	return batchManager
}
