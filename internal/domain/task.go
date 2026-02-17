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
	StatusFetching    TaskStatus = "fetching"
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
	TypeViking     TaskType = "viking"
	TypePlaylist   TaskType = "playlist"
)

type Task struct {
	Ctx             context.Context
	CancelFunc      context.CancelFunc
	CreatedAt       time.Time
	StartedAt       time.Time
	CompletedAt     time.Time
	ID              string
	GID             string
	URL             string
	FileName        string
	LocalPath       string
	RemotePath      string
	RemoteURL       string
	Error           string
	Password        string
	Quality         string
	OrigFileName    string
	Type            TaskType
	Status          TaskStatus
	SubtitleLangs   string
	TotalSize       int64
	DownloadedSize  int64
	UploadedSize    int64
	Speed           int64
	UserID          int64
	ChatID          int64
	ETA             time.Duration
	Progress        float64
	Mu              sync.RWMutex
	Connections     int
	MessageID       int
	ResultMessageID int
	ReplyMessageID  int
	RetryCount      int
	MaxRetries      int
	PlaylistCount   int
	PlaylistIndex   int
	Zip             bool
	Unzip           bool
	Hardsub         bool
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
	Password       string
	Error          string
	CreatedAt      time.Time
	CompletedAt    sql.NullTime
	TotalSize      int64
	DownloadedSize int64
	UploadedSize   int64
	ChatID         int64
	UserID         int64
	Zip            bool
	Unzip          bool
	RetryCount     int
}

type UserStats struct {
	LastActive      time.Time
	Username        string
	UserID          int64
	TotalBandwidth  int64
	TotalDownloads  int
	TotalUploads    int
	SuccessfulTasks int
	FailedTasks     int
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
	CreatedAt       time.Time     `json:"createdAt"`
	StartedAt       time.Time     `json:"startedAt"`
	CompletedAt     time.Time     `json:"completedAt"`
	ID              string        `json:"id"`
	GID             string        `json:"gid"`
	URL             string        `json:"url"`
	FileName        string        `json:"fileName"`
	LocalPath       string        `json:"localPath"`
	RemotePath      string        `json:"remotePath"`
	RemoteURL       string        `json:"remoteURL"`
	Error           string        `json:"error"`
	Password        string        `json:"password"`
	Quality         string        `json:"quality"`
	OrigFileName    string        `json:"origFileName"`
	Type            TaskType      `json:"type"`
	Status          TaskStatus    `json:"status"`
	SubtitleLangs   string        `json:"subtitleLangs,omitempty"`
	TotalSize       int64         `json:"totalSize"`
	DownloadedSize  int64         `json:"downloadedSize"`
	UploadedSize    int64         `json:"uploadedSize"`
	Speed           int64         `json:"speed"`
	ChatID          int64         `json:"chatID"`
	UserID          int64         `json:"userID"`
	ETA             time.Duration `json:"eta"`
	Progress        float64       `json:"progress"`
	Connections     int           `json:"connections"`
	MessageID       int           `json:"messageID"`
	ResultMessageID int           `json:"resultMessageID"`
	ReplyMessageID  int           `json:"replyMessageID"`
	RetryCount      int           `json:"retryCount"`
	MaxRetries      int           `json:"maxRetries"`
	PlaylistCount   int           `json:"playlistCount,omitempty"`
	PlaylistIndex   int           `json:"playlistIndex,omitempty"`
	Zip             bool          `json:"zip"`
	Unzip           bool          `json:"unzip"`
	Hardsub         bool          `json:"hardsub,omitempty"`
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
		RetryCount:      t.RetryCount,
		MaxRetries:      t.MaxRetries,
		PlaylistCount:   t.PlaylistCount,
		PlaylistIndex:   t.PlaylistIndex,
		SubtitleLangs:   t.SubtitleLangs,
		Hardsub:         t.Hardsub,
	}
}

type TaskCheckpoint struct {
	LastUpdate      time.Time
	TaskID          string
	DownloadedBytes int64
	TotalBytes      int64
	Progress        float64
}
