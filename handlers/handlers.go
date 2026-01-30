package handlers

import (
	"context"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type TaskStatus string

const (
	StatusQueued      TaskStatus = "queued"
	StatusDownloading TaskStatus = "downloading"
	StatusExtracting  TaskStatus = "extracting"
	StatusZipping     TaskStatus = "zipping"
	StatusUploading   TaskStatus = "uploading"
	StatusCompleted   TaskStatus = "completed"
	StatusFailed      TaskStatus = "failed"
	StatusCancelled   TaskStatus = "cancelled"
)

type TaskType string

const (
	TypeMirror  TaskType = "mirror"
	TypeLeech   TaskType = "leech"
	TypeYTDLP   TaskType = "ytdlp"
	TypeTorrent TaskType = "torrent"
)

const (
	MarkdownV2       = "MarkdownV2"
	StatusHeaderText = "📊 *Status Task Aktif*\n\n"
	UnknownFile      = "unknown_file"
)

type Task struct {
	ID             string
	GID            string
	Type           TaskType
	Status         TaskStatus
	URL            string
	FileName       string
	LocalPath      string
	RemotePath     string
	RemoteURL      string
	TotalSize      int64
	DownloadedSize int64
	UploadedSize   int64
	Speed          int64
	Connections    int
	ETA            time.Duration
	Progress       float64
	Error          string
	ChatID         int64
	MessageID      int
	UserID         int64
	CreatedAt      time.Time
	StartedAt      time.Time
	CompletedAt    time.Time

	Zip          bool
	Unzip        bool
	Password     string
	Quality      string
	OrigFileName string

	Ctx        context.Context
	CancelFunc context.CancelFunc
	Mu         sync.RWMutex
}

type TaskSnapshot struct {
	ID             string
	GID            string
	Type           TaskType
	Status         TaskStatus
	URL            string
	FileName       string
	LocalPath      string
	RemotePath     string
	RemoteURL      string
	TotalSize      int64
	DownloadedSize int64
	UploadedSize   int64
	Speed          int64
	Connections    int
	ETA            time.Duration
	Progress       float64
	Error          string
	ChatID         int64
	MessageID      int
	UserID         int64
	CreatedAt      time.Time
	StartedAt      time.Time
	CompletedAt    time.Time
	Zip            bool
	Unzip          bool
	Password       string
	Quality        string
	OrigFileName   string
}

type YTDLPSession struct {
	URL      string
	Zip      bool
	Password string
}

type TaskManager struct {
	Tasks                map[string]*Task
	Queue                chan *Task
	ActiveCount          int
	MaxConcurrent        int
	DownloadDir          string
	RcloneDest           string
	ConfigDir            string
	YTDLPSessions        map[string]*YTDLPSession
	Mu                   sync.RWMutex
	StatusMu             sync.Mutex
	Wg                   sync.WaitGroup
	ShutdownChan         chan struct{}
	Bot                  *tgbotapi.BotAPI
	LastStatusMsg        map[int64]int
	StatusPages          map[int64]int
	ProcessTaskFunc      func(*Task)
	RefreshDashboardFunc func(int64, bool)
}

func NewTaskManager(bot *tgbotapi.BotAPI, maxConcurrent int, downloadDir, rcloneDest, configDir string, processTaskFunc func(*Task), refreshDashboardFunc func(int64, bool)) *TaskManager {
	tm := &TaskManager{
		Tasks:                make(map[string]*Task),
		Queue:                make(chan *Task, 100),
		MaxConcurrent:        maxConcurrent,
		DownloadDir:          downloadDir,
		RcloneDest:           rcloneDest,
		ConfigDir:            configDir,
		YTDLPSessions:        make(map[string]*YTDLPSession),
		ShutdownChan:         make(chan struct{}),
		Bot:                  bot,
		LastStatusMsg:        make(map[int64]int),
		StatusPages:          make(map[int64]int),
		ProcessTaskFunc:      processTaskFunc,
		RefreshDashboardFunc: refreshDashboardFunc,
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

func (tm *TaskManager) CreateTask(taskType TaskType, url, fileName string, chatID int64, msgID int, userID int64, zip, unzip bool, password, quality string) *Task {
	ctx, cancel := context.WithCancel(context.Background())

	task := &Task{
		ID:           uuid.New().String()[:8],
		Type:         taskType,
		Status:       StatusQueued,
		URL:          url,
		FileName:     fileName,
		OrigFileName: fileName,
		ChatID:       chatID,
		MessageID:    msgID,
		UserID:       userID,
		Zip:          zip,
		Unzip:        unzip,
		Password:     password,
		Quality:      quality,
		CreatedAt:    time.Now(),
		Ctx:          ctx,
		CancelFunc:   cancel,
	}

	tm.Mu.Lock()
	tm.Tasks[task.ID] = task
	tm.Mu.Unlock()

	tm.Queue <- task

	return task
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

func (tm *TaskManager) GetActiveTasksByChat(chatID int64) []*Task {
	tm.Mu.RLock()
	defer tm.Mu.RUnlock()

	var active []*Task
	for _, task := range tm.Tasks {
		if task.ChatID == chatID && task.Status != StatusCompleted && task.Status != StatusFailed && task.Status != StatusCancelled {
			active = append(active, task)
		}
	}
	return active
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
	return active
}

func (tm *TaskManager) CancelTask(taskID string) bool {
	tm.Mu.Lock()
	task, exists := tm.Tasks[taskID]
	tm.Mu.Unlock()

	if !exists {
		return false
	}

	task.Mu.Lock()
	defer task.Mu.Unlock()

	if task.Status == StatusCompleted || task.Status == StatusCancelled {
		return false
	}

	task.Status = StatusCancelled
	if task.CancelFunc != nil {
		task.CancelFunc()
	}

	return true
}

func (tm *TaskManager) worker(_ int) {
	for {
		select {
		case <-tm.ShutdownChan:
			return
		case task := <-tm.Queue:
			tm.Wg.Add(1)
			if tm.ProcessTaskFunc != nil {
				tm.ProcessTaskFunc(task)
			}
			tm.Wg.Done()
		}
	}
}

func (t *Task) UpdateProgress(downloaded, total int64, speed int64) {
	t.Mu.Lock()
	defer t.Mu.Unlock()

	t.DownloadedSize = downloaded
	t.TotalSize = total
	t.Speed = speed

	if total > 0 {
		t.Progress = float64(downloaded) / float64(total) * 100
	}

	if speed > 0 {
		remaining := total - downloaded
		t.ETA = time.Duration(remaining/speed) * time.Second
	}
}

func (t *Task) SetStatus(status TaskStatus) {
	t.Mu.Lock()
	t.Status = status
	if status == StatusDownloading {
		if t.Type == TypeYTDLP || t.Type == TypeMirror || t.Type == TypeLeech || t.Type == TypeTorrent {
			t.Connections = 16
		}
	}
	if status == StatusCompleted || status == StatusFailed {
		t.CompletedAt = time.Now()
	}
	t.Mu.Unlock()
}

func (t *Task) SetError(err string) {
	t.Mu.Lock()
	defer t.Mu.Unlock()
	t.Error = err
	t.Status = StatusFailed
	t.CompletedAt = time.Now()
}

func (t *Task) GetSnapshot() TaskSnapshot {
	t.Mu.RLock()
	defer t.Mu.RUnlock()
	return TaskSnapshot{
		ID:             t.ID,
		GID:            t.GID,
		Type:           t.Type,
		Status:         t.Status,
		URL:            t.URL,
		FileName:       t.FileName,
		LocalPath:      t.LocalPath,
		RemotePath:     t.RemotePath,
		RemoteURL:      t.RemoteURL,
		TotalSize:      t.TotalSize,
		DownloadedSize: t.DownloadedSize,
		UploadedSize:   t.UploadedSize,
		Speed:          t.Speed,
		Connections:    t.Connections,
		ETA:            t.ETA,
		Progress:       t.Progress,
		Error:          t.Error,
		ChatID:         t.ChatID,
		MessageID:      t.MessageID,
		UserID:         t.UserID,
		CreatedAt:      t.CreatedAt,
		StartedAt:      t.StartedAt,
		CompletedAt:    t.CompletedAt,
		Zip:            t.Zip,
		Unzip:          t.Unzip,
		Password:       t.Password,
		Quality:        t.Quality,
	}
}
