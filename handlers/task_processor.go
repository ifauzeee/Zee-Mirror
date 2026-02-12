package handlers

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zee-mirror/internal/downloader"
	"zee-mirror/internal/organizer"
)

func (s *BotService) processTask(task *Task) {
	task.Mu.RLock()
	status := task.Status
	url := task.URL
	task.Mu.RUnlock()

	if status == StatusCancelled {
		slog.Info("Skipping cancelled task", "taskID", task.ID)
		return
	}

	slog.Info("Starting task processing", "taskID", task.ID, "type", task.Type)

	if (task.Type == TypeMirror || task.Type == TypeLeech) && (strings.Contains(url, "/c/") || strings.Contains(url, "t.me/c/")) {
		if s.Config.UserSessionString != "" {
			slog.Info("Using Userbot engine for private link", "taskID", task.ID)
			s.downloadWithUserbot(task)
			return
		}
	}

	if (task.Type == TypeMirror || task.Type == TypeLeech) && (strings.Contains(url, "drive.google.com") || strings.Contains(url, "docs.google.com") || strings.Contains(url, "drive.usercontent.google.com")) {
		slog.Info("Detected Google Drive URL for Mirror/Leech, switching to local Rclone download", "taskID", task.ID)
		s.downloadGDriveWithRclone(task)
		return
	}

	switch task.Type {
	case TypeMirror, TypeLeech, TypeTorrent, TypeViking:
		s.downloadWithAria2(task)
	case TypeYTDLP, TypeYTDLPLeech:
		s.downloadWithYTDLP(task)
	case TypeClone:
		s.cloneWithRclone(task)
	}
}

func (s *BotService) downloadWithAria2(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	outputDir := filepath.Join(s.TaskManager.DownloadDir, task.ID)

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		s.updateTaskStatus(task)
		return
	}

	if strings.HasPrefix(task.URL, "file://") {
		if task.Type != TypeTorrent {
			s.handleLocalFileDownload(task, outputDir)
			return
		}
	}

	var firstUpdate = true
	lastUpdate := time.Now()
	err := s.TaskManager.Aria2Engine.Download(task.Ctx, &task.Task, outputDir, func(up downloader.ProgressUpdate) {
		task.UpdateFromProgressUpdate(up)

		if firstUpdate || time.Since(lastUpdate) >= 3*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
			_ = task.SaveToDB()
			firstUpdate = false
		}
	})

	if err != nil && task.Status != StatusCancelled {
		if s.TaskManager.IsShuttingDown() {
			return
		}
		if s.retryTask(task, err.Error()) {
			return
		}
		task.SetError(err.Error())
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}

	s.handlePostDownload(task, outputDir)
}

func (s *BotService) downloadWithYTDLP(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	outputDir := filepath.Join(s.TaskManager.DownloadDir, task.ID)

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		s.updateTaskStatus(task)
		return
	}

	lastUpdate := time.Now()
	err := s.TaskManager.YTDLPEngine.Download(task.Ctx, &task.Task, outputDir, func(up downloader.ProgressUpdate) {
		task.UpdateFromProgressUpdate(up)

		if time.Since(lastUpdate) >= 5*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
			_ = task.SaveToDB()
		}
	})

	if err != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}
		if s.TaskManager.IsShuttingDown() {
			return
		}
		if s.retryTask(task, err.Error()) {
			return
		}
		task.SetError(err.Error())
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}

	if task.Status == StatusCancelled {
		s.cleanupTask(task)
		return
	}

	task.LocalPath = findDownloadedFile(outputDir)
	if task.LocalPath == "" {
		task.SetError("Downloaded file not found or incomplete (.part files ignored)")
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}
	task.FileName = filepath.Base(task.LocalPath)

	if info, err := os.Stat(task.LocalPath); err == nil {
		task.DownloadedSize = info.Size()
		task.TotalSize = info.Size()
	}

	var uploadErr error
	if task.Type == TypeYTDLPLeech {
		uploadErr = s.UploadToTelegram(task)
	} else {
		uploadErr = s.UploadWithRclone(task)
	}

	if uploadErr != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}
		if s.TaskManager.IsShuttingDown() {
			return
		}
		if s.retryTask(task, uploadErr.Error()) {
			return
		}
		task.SetError(fmt.Sprintf("Upload failed: %v", uploadErr))
	} else {
		task.SetStatus(StatusCompleted)
	}
	s.updateTaskStatus(task)
	s.cleanupTask(task)
	s.handleAutoDelete(task)
}

func (s *BotService) downloadWithUserbot(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	outputDir := filepath.Join(s.TaskManager.DownloadDir, task.ID)

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("failed to create output dir: %v", err))
		s.updateTaskStatus(task)
		return
	}

	var firstUpdate = true
	lastUpdate := time.Now()
	err := s.TaskManager.UserbotEngine.Download(task.Ctx, &task.Task, outputDir, func(up downloader.ProgressUpdate) {
		task.UpdateFromProgressUpdate(up)

		if firstUpdate || time.Since(lastUpdate) >= 3*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
			_ = task.SaveToDB()
			firstUpdate = false
		}
	})

	if err != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}
		task.SetError(err.Error())
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}

	if task.Status == StatusCancelled {
		s.cleanupTask(task)
		return
	}

	s.handlePostDownload(task, outputDir)
}

func (s *BotService) handlePostDownload(task *Task, outputDir string) {
	if task.Status == StatusCancelled {
		s.cleanupTask(task)
		return
	}

	task.LocalPath = findDownloadedFile(outputDir)
	if task.LocalPath == "" {
		if task.Error == "" {
			task.SetError("Downloaded file not found")
		}
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}
	task.FileName = filepath.Base(task.LocalPath)

	if task.Unzip && organizer.IsArchiveFile(task.LocalPath) {
		if err := s.extractArchive(task); err != nil {
			task.SetError(fmt.Sprintf("Extraction failed: %v", err))
			s.updateTaskStatus(task)
			s.cleanupTask(task)
			return
		}
	}

	if task.Zip {
		if err := s.createZipArchive(task); err != nil {
			task.SetError(fmt.Sprintf("Compression failed: %v", err))
			s.updateTaskStatus(task)
			s.cleanupTask(task)
			return
		}
	}

	var err error
	switch task.Type {
	case TypeLeech:
		err = s.UploadToTelegram(task)
	case TypeViking:
		err = s.UploadToViking(task)
	default:
		err = s.UploadWithRclone(task)
	}

	if err != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}
		task.SetError(fmt.Sprintf("Upload failed: %v", err))
	} else {
		task.SetStatus(StatusCompleted)
	}
	s.updateTaskStatus(task)
	s.cleanupTask(task)
	s.handleAutoDelete(task)
}

func (s *BotService) retryTask(task *Task, originalErr string) bool {
	task.Mu.Lock()
	if task.Status == StatusCancelled {
		task.Mu.Unlock()
		return false
	}

	if task.RetryCount >= task.MaxRetries {
		task.Mu.Unlock()
		return false
	}

	task.RetryCount++
	retries := task.RetryCount
	task.Mu.Unlock()

	backoff := 5 * time.Second
	for i := 1; i < retries; i++ {
		backoff *= 2
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
			break
		}
	}

	slog.Info("Retrying task due to error",
		"taskID", task.ID,
		"retry", retries,
		"max", task.MaxRetries,
		"backoff", backoff,
		"error", originalErr)

	go func() {
		time.Sleep(backoff)

		task.Mu.Lock()
		task.Status = StatusQueued
		task.Error = ""
		task.Mu.Unlock()

		s.updateTaskStatus(task)
		s.TaskManager.Queue <- task
	}()

	return true
}

func findDownloadedFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var candidates []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".aria2") ||
			strings.HasSuffix(name, ".part") ||
			strings.HasSuffix(name, ".ytdl") ||
			strings.HasSuffix(name, ".temp") {
			continue
		}
		candidates = append(candidates, filepath.Join(dir, name))
	}

	if len(candidates) == 1 {
		return candidates[0]
	}

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
