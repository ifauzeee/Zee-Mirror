package downloader

import (
	"context"
	"time"
	"zee-mirror/internal/domain"
)

type ProgressUpdate struct {
	FileName    string
	Error       string
	Message     string
	Downloaded  int64
	Total       int64
	Speed       int64
	Progress    float64
	ETA         time.Duration
	Connections int
}

func (p ProgressUpdate) Found() bool {
	return p.Downloaded > 0 || p.Total > 0 || p.Speed > 0 || p.Progress > 0 || p.FileName != ""
}

type PlaylistMetadata struct {
	Title   string          `json:"title"`
	Entries []PlaylistEntry `json:"entries"`
}

type PlaylistEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Index int    `json:"playlist_index"`
}

type DownloadEngine interface {
	Download(ctx context.Context, task *domain.Task, outputDir string, onProgress func(ProgressUpdate)) error
}

type FormatLister interface {
	GetFormats(ctx context.Context, url string) (map[int]float64, error)
	GetPlaylistMetadata(ctx context.Context, url string) (*PlaylistMetadata, error)
	IsPlaylist(url string) bool
}

type MediaDownloader interface {
	DownloadEngine
	FormatLister
}
