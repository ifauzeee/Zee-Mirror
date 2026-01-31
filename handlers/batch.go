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

	"zee-mirror/internal/domain"
	"zee-mirror/pkg/utils"

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

func NewBatchManager() *BatchManager {
	return &BatchManager{
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

		if utils.IsValidURL(line) {
			options.URLs = append(options.URLs, line)
		}
	}

	return options
}

func (s *BotService) HandleBatch(message *tgbotapi.Message, args string) {
	if args == "" {
		s.sendBatchHelp(message.Chat.ID)
		return
	}

	options := parseBatchArguments(args)

	if len(options.URLs) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nTidak ada URL valid yang ditemukan\\.\n\nGunakan `/batch` untuk melihat cara penggunaan\\.")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if !options.ZipAll {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("🔄 *Creating %d regular tasks\\.\\.\\.*", len(options.URLs)))
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)

		for _, url := range options.URLs {
			fileName := utils.GetFileNameFromURL(url)
			task := s.TaskManager.CreateTask(TypeMirror, url, fileName, message.Chat.ID, 0, message.From.ID, false, false, options.Password, "")
			log.Printf("[Batch-Mirror] Created task %s for %s", task.ID, url)
		}

		s.UpdateSharedDashboard(message.Chat.ID, true)
		return
	}

	if options.Name == "" {
		options.Name = fmt.Sprintf("batch_%s", time.Now().Format("20060102_150405"))
	}

	batch := s.createBatchTask(options.Name, options.URLs, message.Chat.ID, 0, message.From.ID, options.ZipAll, options.Password, options.Priority)

	s.UpdateSharedDashboard(message.Chat.ID, true)

	go s.processBatchTask(batch)

	log.Printf("[Batch] Created batch %s with %d URLs, priority: %d", batch.ID, len(options.URLs), options.Priority)
}

func (s *BotService) updateBatchStatus(batch *BatchTask) {
	s.UpdateSharedDashboard(batch.ChatID, false)
}

func (s *BotService) sendBatchHelp(chatID int64) {
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
✅ Progress tracking per\\-file dan total`
	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) createBatchTask(name string, urls []string, chatID int64, msgID int, userID int64, zipAll bool, password string, priority int) *BatchTask {
	ctx, cancel := context.WithCancel(context.Background())

	batchID := uuid.New().String()[:8]
	downloadDir := filepath.Join(s.TaskManager.DownloadDir, "batch_"+batchID)

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
		batch.SubTasks[i] = s.createBatchSubTask(batch, url, i)
	}

	s.BatchManager.Mu.Lock()
	s.BatchManager.Batches[batch.ID] = batch
	s.BatchManager.PriorityQueue = append(s.BatchManager.PriorityQueue, batch)

	sort.Slice(s.BatchManager.PriorityQueue, func(i, j int) bool {
		return s.BatchManager.PriorityQueue[i].Priority > s.BatchManager.PriorityQueue[j].Priority
	})
	s.BatchManager.Mu.Unlock()

	return batch
}

func (s *BotService) processBatchTask(batch *BatchTask) {
	log.Printf("[Batch %s] Starting processing %d URLs", batch.ID, len(batch.URLs))

	if err := os.MkdirAll(batch.DownloadDir, 0750); err != nil {
		batch.SetError(fmt.Sprintf("Failed to create batch directory: %v", err))
		s.updateBatchStatus(batch)
		return
	}

	batch.SetStatus(StatusDownloading)
	s.updateBatchStatus(batch)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)

	for i, url := range batch.URLs {
		select {
		case <-batch.Ctx.Done():
			log.Printf("[Batch %s] Cancelled", batch.ID)
			batch.SetStatus(StatusCancelled)
			s.updateBatchStatus(batch)
			return
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(idx int, downloadURL string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			subTask := batch.SubTasks[idx]

			err := s.downloadBatchItem(batch, subTask)

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
					s.updateBatchStatus(batch)

					upErr := s.UploadWithRclone(subTask)

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

			s.updateBatchStatus(batch)
		}(i, url)
	}

	wg.Wait()

	select {
	case <-batch.Ctx.Done():
		batch.SetStatus(StatusCancelled)
		s.updateBatchStatus(batch)
		s.cleanupBatch(batch)
		return
	default:
	}

	if batch.ZipAll && batch.Completed > 0 {
		batch.SetStatus(StatusZipping)
		s.updateBatchStatus(batch)

		err := s.zipBatchResults(batch)
		if err != nil {
			batch.SetError(fmt.Sprintf("Zip failed: %v", err))
			s.updateBatchStatus(batch)
			s.cleanupBatch(batch)
			return
		}

		batch.SetStatus(StatusUploading)
		s.updateBatchStatus(batch)

		err = s.uploadBatchResults(batch)
		if err != nil {
			batch.SetError(fmt.Sprintf("Upload failed: %v", err))
			s.updateBatchStatus(batch)
			s.cleanupBatch(batch)
			return
		}
	}

	batch.SetStatus(StatusCompleted)
	batch.CompletedAt = time.Now()
	s.updateBatchStatus(batch)

	s.sendBatchCompletionMessage(batch)
	s.cleanupBatch(batch)
}

func (s *BotService) createBatchSubTask(batch *BatchTask, url string, index int) *Task {
	ctx, cancel := context.WithCancel(batch.Ctx)

	fileName := utils.GetFileNameFromURL(url)
	if fileName == UnknownFile {
		fileName = fmt.Sprintf("file_%d", index+1)
	}

	task := &Task{
		Task: domain.Task{
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
		},
		DB: s.TaskManager.DB,
	}

	return task
}

func (s *BotService) downloadBatchItem(batch *BatchTask, task *Task) error {
	task.SetStatus(StatusDownloading)

	taskDir := filepath.Join(batch.DownloadDir, task.ID)
	if err := os.MkdirAll(taskDir, 0750); err != nil {
		return fmt.Errorf("failed to create task directory: %v", err)
	}

	configPath := filepath.Join(s.Config.ConfigDir, "cookies.txt")
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

func (s *BotService) zipBatchResults(batch *BatchTask) error {
	log.Printf("[Batch %s] Zipping results...", batch.ID)

	zipFileName := utils.SanitizeFileName(batch.Name) + ".zip"
	zipPath := filepath.Join(s.TaskManager.DownloadDir, "batch_"+batch.ID+"_output", zipFileName)

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

func (s *BotService) uploadBatchResults(batch *BatchTask) error {
	log.Printf("[Batch %s] Uploading results...", batch.ID)

	uploadPath := batch.LocalPath
	if uploadPath == "" {
		uploadPath = batch.DownloadDir
	}

	remotePath := filepath.Join(s.TaskManager.RcloneDest, filepath.Base(uploadPath))
	batch.RemotePath = remotePath

	configPath := filepath.Join(s.Config.ConfigDir, "rclone.conf")
	args := []string{
		"copy",
		uploadPath,
		s.TaskManager.RcloneDest,
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
		filepath.Join(s.TaskManager.RcloneDest, uploadName),
	}

	linkCmd := exec.CommandContext(ctx, "rclone", linkArgs...)
	linkOutput, linkErr := linkCmd.Output()
	if linkErr == nil {
		batch.RemoteURL = strings.TrimSpace(string(linkOutput))
	}

	log.Printf("[Batch %s] Upload completed. URL: %s", batch.ID, batch.RemoteURL)
	return nil
}

func (s *BotService) sendBatchCompletionMessage(batch *BatchTask) {
	duration := batch.CompletedAt.Sub(batch.CreatedAt)

	text := fmt.Sprintf(`✅ *Batch Download Selesai\\!*

🏷️ *Name:* %s
📊 *Total Files:* %d
✅ *Berhasil:* %d
❌ *Gagal:* %d
📦 *Total Size:* %s
⏱️ *Duration:* %s`,
		utils.EscapeMarkdownV2(batch.Name),
		len(batch.URLs),
		batch.Completed,
		batch.Failed,
		utils.EscapeMarkdownV2(utils.FormatBytes(batch.TotalSize)),
		utils.EscapeMarkdownV2(utils.FormatDuration(duration)),
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
				tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "batch:close:none"),
			),
		)
		msg.ReplyMarkup = keyboard
	}

	_, _ = s.Bot.Send(msg)
}

func (s *BotService) cleanupBatch(batch *BatchTask) {
	if batch.DownloadDir != "" {
		_ = os.RemoveAll(batch.DownloadDir)
	}

	outputDir := filepath.Join(s.TaskManager.DownloadDir, "batch_"+batch.ID+"_output")
	_ = os.RemoveAll(outputDir)

	go func() {
		time.Sleep(1 * time.Hour)
		s.BatchManager.Mu.Lock()
		delete(s.BatchManager.Batches, batch.ID)
		for i, b := range s.BatchManager.PriorityQueue {
			if b.ID == batch.ID {
				s.BatchManager.PriorityQueue = append(s.BatchManager.PriorityQueue[:i], s.BatchManager.PriorityQueue[i+1:]...)
				break
			}
		}
		s.BatchManager.Mu.Unlock()
	}()
}

func (s *BotService) HandleBatchCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[1]
	batchID := parts[2]

	s.BatchManager.Mu.RLock()
	batch, exists := s.BatchManager.Batches[batchID]
	s.BatchManager.Mu.RUnlock()

	if !exists {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Batch not found"))
		return
	}

	switch action {
	case CmdRefresh:
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Refreshed"))
		s.updateBatchStatus(batch)

	case "cancel":
		batch.CancelFunc()
		batch.SetStatus(StatusCancelled)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🚫 Batch cancelled"))
		s.updateBatchStatus(batch)

	case CmdClose:
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, _ = s.Bot.Request(deleteMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Closed"))
		return
	}
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

func (s *BotService) checkBatchSubTaskCancellation(taskID string) bool {
	s.BatchManager.Mu.RLock()
	defer s.BatchManager.Mu.RUnlock()

	for _, batch := range s.BatchManager.Batches {
		batch.Mu.RLock()
		for _, sub := range batch.SubTasks {
			if sub.ID == taskID {
				sub.Cancel(StatusCancelled)
				batch.Mu.RUnlock()
				return true
			}
		}
		batch.Mu.RUnlock()
	}
	return false
}
