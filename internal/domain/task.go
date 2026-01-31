package domain

import (
	"context"
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
	TypeMirror  TaskType = "mirror"
	TypeLeech   TaskType = "leech"
	TypeYTDLP   TaskType = "ytdlp"
	TypeTorrent TaskType = "torrent"
	TypeClone   TaskType = "clone"
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

	Ctx        context.Context
	CancelFunc context.CancelFunc
	Mu         sync.RWMutex
}

type TaskSnapshot struct {
	ID              string
	GID             string
	Type            TaskType
	Status          TaskStatus
	URL             string
	FileName        string
	LocalPath       string
	RemotePath      string
	RemoteURL       string
	TotalSize       int64
	DownloadedSize  int64
	UploadedSize    int64
	Speed           int64
	Connections     int
	ETA             time.Duration
	Progress        float64
	Error           string
	ChatID          int64
	MessageID       int
	UserID          int64
	CreatedAt       time.Time
	StartedAt       time.Time
	CompletedAt     time.Time
	Zip             bool
	Unzip           bool
	Password        string
	Quality         string
	OrigFileName    string
	ResultMessageID int
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
	}
}
