package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
	"zee-mirror/internal/config"
	"zee-mirror/internal/database"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/downloader"
	"zee-mirror/internal/metrics"
	"zee-mirror/internal/queue"
	"zee-mirror/internal/recovery"
	"zee-mirror/internal/repository"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type TaskStatus = domain.TaskStatus

const (
	StatusQueued      = domain.StatusQueued
	StatusDownloading = domain.StatusDownloading
	StatusExtracting  = domain.StatusExtracting
	StatusZipping     = domain.StatusZipping
	StatusUploading   = domain.StatusUploading
	StatusCompleted   = domain.StatusCompleted
	StatusFailed      = domain.StatusFailed
	StatusCancelled   = domain.StatusCancelled
)

type TaskType = domain.TaskType

const (
	TypeMirror     = domain.TypeMirror
	TypeLeech      = domain.TypeLeech
	TypeYTDLP      = domain.TypeYTDLP
	TypeYTDLPLeech = domain.TypeYTDLPLeech
	TypeTorrent    = domain.TypeTorrent
	TypeClone      = domain.TypeClone
	TypeViking     = domain.TypeViking
	TypePlaylist   = domain.TypePlaylist
)

const (
	MarkdownV2       = "MarkdownV2"
	StatusHeaderText = "📊 *Status Task Aktif*\n\n"
	UnknownFile      = "unknown_file"
	UnlimitedStr     = "Unlimited"

	ModeLeech  = "leech"
	CmdRefresh = "refresh"
	CmdClose   = "close"
	CmdSystem  = "system"
	CmdHealth  = "health"
	CmdLogs    = "logs"

	IconFolder = "📁"
	IconFile   = "📄"
	IconOK     = "✅ OK"
	IconError  = "❌ ERROR"
)

type Task struct {
	DB repository.TaskRepository
	domain.Task
}

type TaskSnapshot = domain.TaskSnapshot

type YTDLPSession = domain.YTDLPSession

type TorrentSession = domain.TorrentSession

type TorrentFile = domain.TorrentFile

type TaskManager struct {
	Bot                  *tgbotapi.BotAPI
	DB                   repository.TaskRepository
	Aria2Engine          *downloader.Aria2Engine
	YTDLPEngine          *downloader.YTDLPEngine
	UserbotEngine        *downloader.UserbotEngine
	Config               *config.Config
	Tasks                map[string]*Task
	Queue                *queue.PriorityQueue
	QueueSignal          chan struct{}
	RateLimiter          *queue.UserRateLimiter
	LastStatusMsg        map[int64]int
	StatusPages          map[int64]int
	YTDLPSessions        map[string]*YTDLPSession
	TorrentSessions      map[string]*TorrentSession
	LastDashUpdateAt     map[int64]time.Time
	LastDashProgressSum  map[int64]float64
	LastTasksCount       map[int64]int
	ShutdownChan         chan struct{}
	ProcessTaskFunc      func(*Task)
	RefreshDashboardFunc func(int64, bool)
	CheckpointManager    *recovery.CheckpointManager
	DownloadDir          string
	RcloneDest           string
	ConfigDir            string
	Mu                   sync.RWMutex
	Wg                   sync.WaitGroup
	StatusMu             sync.Mutex
	ActiveCount          int
	MaxConcurrent        int
	StopDuplicate        bool
}

func NewTaskManager(bot *tgbotapi.BotAPI, maxConcurrent int, downloadDir, rcloneDest, configDir string, processTaskFunc func(*Task), refreshDashboardFunc func(int64, bool), db repository.TaskRepository) *TaskManager {
	cfg := config.LoadConfig()

	tm := &TaskManager{
		Tasks:                make(map[string]*Task),
		Queue:                queue.NewPriorityQueue(),
		QueueSignal:          make(chan struct{}, 1000),
		RateLimiter:          queue.NewUserRateLimiter(5, 10),
		MaxConcurrent:        maxConcurrent,
		DownloadDir:          downloadDir,
		RcloneDest:           rcloneDest,
		ConfigDir:            configDir,
		YTDLPSessions:        make(map[string]*YTDLPSession),
		TorrentSessions:      make(map[string]*TorrentSession),
		ShutdownChan:         make(chan struct{}),
		Bot:                  bot,
		DB:                   db,
		LastStatusMsg:        make(map[int64]int),
		StatusPages:          make(map[int64]int),
		ProcessTaskFunc:      processTaskFunc,
		RefreshDashboardFunc: refreshDashboardFunc,
		Aria2Engine:          downloader.NewAria2Engine(configDir, cfg.Aria2RPCURL, cfg.Aria2RPCSecret),
		YTDLPEngine:          downloader.NewYTDLPEngine(configDir),
		UserbotEngine:        downloader.NewUserbotEngine(cfg),
		LastDashUpdateAt:     make(map[int64]time.Time),
		LastDashProgressSum:  make(map[int64]float64),
		LastTasksCount:       make(map[int64]int),
		Config:               cfg,
	}

	if dbInstance, ok := db.(*database.DB); ok {
		tm.CheckpointManager = recovery.NewCheckpointManager(dbInstance)
	}

	if db != nil {
		ctx := context.Background()
		activeTasks, err := db.GetActive(ctx)
		if err == nil {
			for _, rt := range activeTasks {
				taskCtx, cancel := context.WithCancel(context.Background())
				task := &Task{
					Task: domain.Task{
						ID:             rt.ID,
						GID:            rt.GID,
						Type:           domain.TaskType(rt.Type),
						Status:         domain.TaskStatus(rt.Status),
						URL:            rt.URL,
						FileName:       rt.FileName,
						LocalPath:      rt.LocalPath,
						RemotePath:     rt.RemotePath,
						RemoteURL:      rt.RemoteURL,
						TotalSize:      rt.TotalSize,
						DownloadedSize: rt.DownloadedSize,
						UploadedSize:   rt.UploadedSize,
						ChatID:         rt.ChatID,
						UserID:         rt.UserID,
						CreatedAt:      rt.CreatedAt,
						Zip:            rt.Zip,
						Unzip:          rt.Unzip,
						Password:       rt.Password,
						Error:          rt.Error,
						Ctx:            taskCtx,
						CancelFunc:     cancel,
						RetryCount:     rt.RetryCount,
						MaxRetries:     cfg.MaxRetries,
					},
					DB: db,
				}
				if rt.CompletedAt.Valid {
					task.CompletedAt = rt.CompletedAt.Time
				}

				if tm.CheckpointManager != nil {
					if cp, err := tm.CheckpointManager.GetCheckpoint(task.ID); err == nil && cp != nil {
						task.DownloadedSize = cp.DownloadedBytes
						task.TotalSize = cp.TotalBytes
						task.Progress = cp.Progress
						slog.Info("Restored task from checkpoint", "taskID", task.ID, "progress", task.Progress)
					}
				}

				tm.Tasks[task.ID] = task
				tm.Queue.Enqueue(task, 0)
				select {
				case tm.QueueSignal <- struct{}{}:
				default:
				}
			}
			slog.Info("Loaded active tasks from database", "count", len(activeTasks))
		}
	}

	for i := 0; i < maxConcurrent; i++ {
		go tm.worker(i)
	}

	go tm.startAutoRefresh()

	return tm
}

func (tm *TaskManager) startAutoRefresh() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ShutdownChan:
			return
		case <-ticker.C:
			tm.refreshActiveDashboards()
		}
	}
}

func (tm *TaskManager) refreshActiveDashboards() {
	tm.Mu.RLock()
	var chats []int64
	for chatID := range tm.LastStatusMsg {
		chats = append(chats, chatID)
	}
	tm.Mu.RUnlock()

	for _, chatID := range chats {
		if tm.RefreshDashboardFunc != nil {
			tm.RefreshDashboardFunc(chatID, false)
		}
	}
}

func (tm *TaskManager) CreateTask(taskType TaskType, url, fileName string, chatID int64, msgID, replyID int, userID int64, zip, unzip bool, password, quality string, expectedTotalSize int64, subtitleLangs string, hardsub bool) (*Task, error) {
	if tm.StopDuplicate {
		tm.Mu.RLock()
		for _, t := range tm.Tasks {
			t.Mu.RLock()
			isFinished := t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusCancelled
			sameURL := t.URL == url
			sameQuality := t.Quality == quality
			t.Mu.RUnlock()

			if !isFinished && sameURL && sameQuality {
				tm.Mu.RUnlock()
				return nil, fmt.Errorf("%w: ID %s", domain.ErrDuplicateTask, t.ID)
			}
		}
		tm.Mu.RUnlock()

		if tm.DB != nil {
			oldTask, errDB := tm.DB.GetCompletedTaskByURL(context.Background(), url)
			if errDB == nil && oldTask != nil {
				return nil, fmt.Errorf("%w: file already exists in cloud/database", domain.ErrDuplicateTask)
			}
		}
	}

	if !tm.RateLimiter.Allow(userID) {
		return nil, fmt.Errorf("%w: limit 5 tasks/min exceeded", domain.ErrLimitExceeded)
	}

	ctx, cancel := context.WithCancel(context.Background())

	task := &Task{
		Task: domain.Task{
			ID:             uuid.New().String()[:8],
			Type:           taskType,
			Status:         StatusQueued,
			URL:            url,
			FileName:       fileName,
			OrigFileName:   fileName,
			ChatID:         chatID,
			MessageID:      msgID,
			ReplyMessageID: replyID,
			UserID:         userID,
			Zip:            zip,
			Unzip:          unzip,
			Password:       password,
			Quality:        quality,
			CreatedAt:      time.Now().UTC(),
			Ctx:            ctx,
			CancelFunc:     cancel,
			TotalSize:      expectedTotalSize,
			MaxRetries:     tm.Config.MaxRetries,
			SubtitleLangs:  subtitleLangs,
			Hardsub:        hardsub,
		},
		DB: tm.DB,
	}

	tm.Mu.Lock()
	tm.Tasks[task.ID] = task
	tm.Mu.Unlock()

	_ = task.SaveToDB()

	priority := 0
	if utils.IsAdmin(userID, tm.Config.OwnerID, tm.Config.AuthorizedUsers) {
		priority = 100
	}

	tm.Queue.Enqueue(task, priority)
	select {
	case tm.QueueSignal <- struct{}{}:
	default:
	}

	return task, nil
}

func (t *Task) SaveToDB() error {
	if t.DB == nil {
		return nil
	}

	t.Mu.RLock()
	defer t.Mu.RUnlock()

	record := domain.TaskRecord{
		ID:             t.ID,
		GID:            t.GID,
		Type:           string(t.Type),
		Status:         string(t.Status),
		URL:            t.URL,
		FileName:       t.FileName,
		LocalPath:      t.LocalPath,
		RemotePath:     t.RemotePath,
		RemoteURL:      t.RemoteURL,
		TotalSize:      t.TotalSize,
		DownloadedSize: t.DownloadedSize,
		UploadedSize:   t.UploadedSize,
		ChatID:         t.ChatID,
		UserID:         t.UserID,
		CreatedAt:      t.CreatedAt,
		Zip:            t.Zip,
		Unzip:          t.Unzip,
		Password:       t.Password,
		Error:          t.Error,
	}

	if !t.CompletedAt.IsZero() {
		record.CompletedAt = struct {
			Time  time.Time
			Valid bool
		}{Time: t.CompletedAt, Valid: true}
	}

	ctx := context.Background()
	err := t.DB.Save(ctx, record)
	if err != nil {
		slog.Error("Failed to save task to database", "taskID", t.ID, "status", t.Status, "error", err)
	}
	return err
}

func (tm *TaskManager) GetTask(taskID string) *Task {
	tm.Mu.RLock()
	defer tm.Mu.RUnlock()
	return tm.Tasks[taskID]
}

func (tm *TaskManager) GetTaskByGID(gid string) *Task {
	tm.Mu.RLock()
	defer tm.Mu.RUnlock()
	for _, task := range tm.Tasks {
		if task.GID == gid {
			return task
		}
	}
	return nil
}

func (tm *TaskManager) GetActiveTasks() []*Task {
	tm.Mu.RLock()
	defer tm.Mu.RUnlock()

	var active []*Task
	for _, task := range tm.Tasks {
		if task.Status != StatusCompleted && task.Status != StatusFailed && task.Status != StatusCancelled {
			active = append(active, task)
		}
	}

	sort.Slice(active, func(i, j int) bool {
		return active[i].CreatedAt.Before(active[j].CreatedAt)
	})

	return active
}

func (tm *TaskManager) CancelTask(taskID string) bool {
	tm.Mu.Lock()
	task, exists := tm.Tasks[taskID]
	if tm.CheckpointManager != nil && exists {
		if err := tm.CheckpointManager.DeleteCheckpoint(task.ID); err != nil {
			slog.Warn("Failed to delete checkpoint", "taskID", task.ID, "error", err)
		}
	}
	delete(tm.Tasks, task.ID)
	tm.Mu.Unlock()

	if !exists {
		return false
	}

	task.Mu.Lock()
	if task.Status == StatusCompleted || task.Status == StatusCancelled {
		task.Mu.Unlock()
		return false
	}

	task.Status = StatusCancelled
	task.CompletedAt = time.Now().UTC()
	if task.CancelFunc != nil {
		task.CancelFunc()
	}
	task.Mu.Unlock()

	err := task.SaveToDB()
	return err == nil
}

func (t *Task) Cancel(status TaskStatus) bool {
	t.Mu.Lock()
	if t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusCancelled {
		t.Mu.Unlock()
		return false
	}

	t.Status = status
	if t.CancelFunc != nil {
		t.CancelFunc()
	}
	t.Mu.Unlock()

	_ = t.SaveToDB()
	return true
}

func (tm *TaskManager) CancelAllTasks() int {
	tm.Mu.RLock()
	var taskIDs []string
	for id := range tm.Tasks {
		taskIDs = append(taskIDs, id)
	}
	tm.Mu.RUnlock()

	cancelledCount := 0
	for _, id := range taskIDs {
		if tm.CancelTask(id) {
			cancelledCount++
		}
	}
	return cancelledCount
}

func (tm *TaskManager) worker(_ int) {
	for {
		select {
		case <-tm.ShutdownChan:
			return
		case <-tm.QueueSignal:
			item := tm.Queue.DequeueNonBlocking()
			if item != nil {
				if task, ok := item.(*Task); ok {
					tm.Wg.Add(1)
					if tm.ProcessTaskFunc != nil {
						tm.ProcessTaskFunc(task)
					}
					tm.Wg.Done()
				}
			}
		}
	}
}

func (tm *TaskManager) IsShuttingDown() bool {
	select {
	case <-tm.ShutdownChan:
		return true
	default:
		return false
	}
}

func (t *Task) UpdateFromProgressUpdate(up downloader.ProgressUpdate) {
	t.Mu.Lock()
	defer t.Mu.Unlock()

	if up.FileName != "" {
		t.FileName = up.FileName
	}
	if up.Downloaded != 0 {
		t.DownloadedSize = up.Downloaded
	}
	if up.Total != 0 {
		t.TotalSize = up.Total
	}
	if up.Speed != 0 {
		t.Speed = up.Speed
	}
	if up.Progress != 0 {
		t.Progress = up.Progress
	}
	if up.Connections != 0 {
		t.Connections = up.Connections
	}
	if up.ETA != 0 {
		t.ETA = up.ETA
	}
	if up.Error != "" {
		t.Error = up.Error
	}
}

func (t *Task) SetStatus(status TaskStatus) {
	t.Mu.Lock()
	if t.Status == StatusCancelled && status != StatusCancelled {
		t.Mu.Unlock()
		return
	}
	oldStatus := t.Status
	t.Status = status
	if status == StatusDownloading {
		if t.Type == TypeYTDLP || t.Type == TypeMirror || t.Type == TypeLeech || t.Type == TypeTorrent {
			t.Connections = 16
		}
	}
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		t.CompletedAt = time.Now().UTC()
		if oldStatus != status {
			metrics.TasksTotal.WithLabelValues(string(t.Type), string(status)).Inc()
		}
	}
	t.Mu.Unlock()
	_ = t.SaveToDB()
}

func (t *Task) SetError(err string) {
	t.Mu.Lock()
	t.Error = err
	t.Status = StatusFailed
	t.CompletedAt = time.Now().UTC()
	metrics.TasksTotal.WithLabelValues(string(t.Type), "failed").Inc()
	t.Mu.Unlock()
	_ = t.SaveToDB()
}

func (t *Task) GetSnapshot() domain.TaskSnapshot {
	return t.Task.GetSnapshot()
}

func (s *BotService) HandleConfirmCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	if len(parts) < 2 {
		return
	}

	action := parts[1]
	switch action {
	case "yes":
		text := "✅ *Dikonfirmasi*"
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)
	case "no":
		text := "❌ *Dibatalkan*"
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(editMsg)
	}
}
