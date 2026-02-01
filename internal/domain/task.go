package domain

import (
	"context"
	"database/sql"
	"sync"
	"time"
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
	TypeMirror     TaskType = "mirror"
	TypeLeech      TaskType = "leech"
	TypeYTDLP      TaskType = "ytdlp"
	TypeYTDLPLeech TaskType = "ytdlp_leech"
	TypeTorrent    TaskType = "torrent"
	TypeClone      TaskType = "clone"
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

	Zip             bool
	Unzip           bool
	Password        string
	Quality         string
	OrigFileName    string
	ResultMessageID int
	ReplyMessageID  int

	Ctx        context.Context
	CancelFunc context.CancelFunc
	Mu         sync.RWMutex
}

type TaskRecord struct {
	ID             string
	GID            string
	Type           string
	Status         string
	URL            string
	FileName       string
	LocalPath      string
	RemotePath     string
	RemoteURL      string
	TotalSize      int64
	DownloadedSize int64
	UploadedSize   int64
	ChatID         int64
	UserID         int64
	CreatedAt      time.Time
	CompletedAt    sql.NullTime
	Zip            bool
	Unzip          bool
	Password       string
	Error          string
}

type UserStats struct {
	UserID          int64
	Username        string
	TotalDownloads  int
	TotalUploads    int
	TotalBandwidth  int64
	SuccessfulTasks int
	FailedTasks     int
	LastActive      time.Time
}

type DailyStats struct {
	Date           time.Time
	TotalTasks     int
	CompletedTasks int
	FailedTasks    int
	TotalBandwidth int64
	AverageSpeed   int64
	PeakConcurrent int
}

type TaskSnapshot struct {
	ID              string        `json:"id"`
	GID             string        `json:"gid"`
	Type            TaskType      `json:"type"`
	Status          TaskStatus    `json:"status"`
	URL             string        `json:"url"`
	FileName        string        `json:"fileName"`
	LocalPath       string        `json:"localPath"`
	RemotePath      string        `json:"remotePath"`
	RemoteURL       string        `json:"remoteURL"`
	TotalSize       int64         `json:"totalSize"`
	DownloadedSize  int64         `json:"downloadedSize"`
	UploadedSize    int64         `json:"uploadedSize"`
	Speed           int64         `json:"speed"`
	Connections     int           `json:"connections"`
	ETA             time.Duration `json:"eta"`
	Progress        float64       `json:"progress"`
	Error           string        `json:"error"`
	ChatID          int64         `json:"chatID"`
	MessageID       int           `json:"messageID"`
	UserID          int64         `json:"userID"`
	CreatedAt       time.Time     `json:"createdAt"`
	StartedAt       time.Time     `json:"startedAt"`
	CompletedAt     time.Time     `json:"completedAt"`
	Zip             bool          `json:"zip"`
	Unzip           bool          `json:"unzip"`
	Password        string        `json:"password"`
	Quality         string        `json:"quality"`
	OrigFileName    string        `json:"origFileName"`
	ResultMessageID int           `json:"resultMessageID"`
	ReplyMessageID  int           `json:"replyMessageID"`
}

func (t *Task) GetSnapshot() TaskSnapshot {
	t.Mu.RLock()
	defer t.Mu.RUnlock()
	return TaskSnapshot{
		ID:              t.ID,
		GID:             t.GID,
		Type:            t.Type,
		Status:          t.Status,
		URL:             t.URL,
		FileName:        t.FileName,
		LocalPath:       t.LocalPath,
		RemotePath:      t.RemotePath,
		RemoteURL:       t.RemoteURL,
		TotalSize:       t.TotalSize,
		DownloadedSize:  t.DownloadedSize,
		UploadedSize:    t.UploadedSize,
		Speed:           t.Speed,
		Connections:     t.Connections,
		ETA:             t.ETA,
		Progress:        t.Progress,
		Error:           t.Error,
		ChatID:          t.ChatID,
		MessageID:       t.MessageID,
		UserID:          t.UserID,
		CreatedAt:       t.CreatedAt,
		StartedAt:       t.StartedAt,
		CompletedAt:     t.CompletedAt,
		Zip:             t.Zip,
		Unzip:           t.Unzip,
		Password:        t.Password,
		Quality:         t.Quality,
		OrigFileName:    t.OrigFileName,
		ResultMessageID: t.ResultMessageID,
		ReplyMessageID:  t.ReplyMessageID,
	}
}
