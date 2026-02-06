package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"zee-mirror/internal/organizer"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) UploadWithRclone(task *Task) error {
	task.SetStatus(StatusUploading)
	task.Mu.Lock()
	task.Progress = 0
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	uploadPath := task.LocalPath
	if uploadPath == "" {
		return fmt.Errorf("no file to upload")
	}

	if info, err := os.Stat(uploadPath); err == nil {
		task.Mu.Lock()
		if info.IsDir() {
			dirSize, err := utils.CalculateDirSize(uploadPath)
			if err != nil {
				slog.Warn("Could not calculate directory size", "error", err, "path", uploadPath)
				task.TotalSize = info.Size()
			} else {
				task.TotalSize = dirSize
			}
		} else {
			task.TotalSize = info.Size()
		}
		task.Mu.Unlock()
	}

	remoteDest := s.TaskManager.RcloneDest
	if s.Config.SmartAutoOrganization {
		if subFolder := organizer.GetTargetFolder(task.FileName); subFolder != "" {
			remoteDest = filepath.Join(remoteDest, subFolder)
			slog.Info("Smart Auto Organization: moving to subfolder", "taskID", task.ID, "subFolder", subFolder)
		}
	}

	remotePath := filepath.Join(remoteDest, task.FileName)
	task.RemotePath = remotePath

	rcloneDest := remoteDest
	if info, err := os.Stat(uploadPath); err == nil && info.IsDir() {
		rcloneDest = remotePath
	}

	configPath := filepath.Join(s.TaskManager.ConfigDir, "rclone.conf")
	args := []string{
		"copy",
		uploadPath,
		rcloneDest,
		"--config", configPath,
		"--progress",
		"--stats", "1s",
		"--stats-one-line",
		"--transfers", "10",
		"--checkers", "20",
		"--drive-chunk-size", "256M",
		"--drive-upload-cutoff", "256M",
		"--buffer-size", "128M",
		"--low-level-retries", "10",
		"--use-mmap",
		"--size-only",
		"--no-traverse",
		"--drive-pacer-min-sleep", "10ms",
		"--drive-pacer-burst", "200",
		"--log-level", "NOTICE",
	}

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	slog.Info("Starting rclone upload", "taskID", task.ID, "args", strings.Join(args, " "))

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start rclone: %v", err)
	}

	done := make(chan struct{})

	go s.estimateUploadProgress(task, done)

	go s.parseRcloneProgress(task, stderrPipe)
	go s.parseRcloneProgress(task, stdoutPipe)

	err = cmd.Wait()
	close(done)

	if err != nil {
		if task.Status == StatusCancelled {
			return fmt.Errorf("task cancelled")
		}
		return fmt.Errorf("rclone failed: %v", err)
	}

	if task.Status == StatusCancelled {
		return fmt.Errorf("task cancelled")
	}

	isDir := false
	if info, err := os.Stat(uploadPath); err == nil {
		isDir = info.IsDir()
	}
	s.generateRcloneLink(ctx, task, configPath, isDir)

	task.Mu.Lock()
	task.Progress = 100
	task.Mu.Unlock()
	return nil
}

func (s *BotService) estimateUploadProgress(task *Task, done chan struct{}) {
	const estimatedSpeedBytesPerSec = 3.5 * 1024 * 1024

	task.Mu.RLock()
	totalSize := task.TotalSize
	task.Mu.RUnlock()

	if totalSize <= 0 {
		return
	}

	estimatedTotalSeconds := float64(totalSize) / estimatedSpeedBytesPerSec
	if estimatedTotalSeconds < 3 {
		return
	}

	startTime := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			elapsed := time.Since(startTime).Seconds()

			estimatedProgress := (elapsed / estimatedTotalSeconds) * 100

			if estimatedProgress > 95 {
				estimatedProgress = 95
			}

			estimatedUploaded := int64(elapsed * estimatedSpeedBytesPerSec)
			if estimatedUploaded > totalSize {
				estimatedUploaded = totalSize
			}

			remainingSeconds := estimatedTotalSeconds - elapsed
			if remainingSeconds < 0 {
				remainingSeconds = 0
			}

			task.Mu.Lock()
			if estimatedProgress > task.Progress {
				task.Progress = estimatedProgress
				task.UploadedSize = estimatedUploaded
				task.Speed = int64(estimatedSpeedBytesPerSec)
				task.ETA = time.Duration(remainingSeconds) * time.Second
			}
			task.Mu.Unlock()

			s.updateTaskStatus(task)
		}
	}
}

func (s *BotService) generateRcloneLink(ctx context.Context, task *Task, configPath string, isDirUpload bool) {
	currentRemotePath := task.RemotePath
	if currentRemotePath == "" {
		currentRemotePath = filepath.Join(s.TaskManager.RcloneDest, task.FileName)
	}

	currentRemotePath = strings.ReplaceAll(currentRemotePath, "\\", "/")

	if s.Config.IndexURL != "" {
		if s.generateIndexURL(ctx, task, configPath, currentRemotePath) {
			return
		}
	}

	s.generateDirectLink(ctx, task, configPath, currentRemotePath, isDirUpload)
}

func (s *BotService) generateIndexURL(ctx context.Context, task *Task, configPath, currentRemotePath string) bool {
	if s.generateIDBasedIndexURL(ctx, task, configPath, currentRemotePath) {
		return true
	}

	return s.generatePathBasedIndexURL(task, currentRemotePath)
}

func (s *BotService) generateIDBasedIndexURL(ctx context.Context, task *Task, configPath, currentRemotePath string) bool {
	var fileID, parentID string

	lsArgs := []string{
		"lsjson",
		currentRemotePath,
		"--config", configPath,
		"--no-modtime",
		"--no-mimetype",
	}
	lsCmd := exec.CommandContext(ctx, "rclone", lsArgs...)
	if lsOutput, err := lsCmd.Output(); err == nil {
		var files []map[string]interface{}
		if json.Unmarshal(lsOutput, &files) == nil && len(files) > 0 {
			if id, ok := files[0]["ID"].(string); ok {
				fileID = id
			} else if id, ok := files[0]["Id"].(string); ok {
				fileID = id
			}
		}
	} else {
		slog.Warn("Failed to get File ID", "path", currentRemotePath, "error", err)
	}

	parentPath := path.Dir(currentRemotePath)
	grandParentPath := path.Dir(parentPath)
	parentName := path.Base(parentPath)

	linkArgsParent := []string{
		"lsjson",
		grandParentPath,
		"--config", configPath,
		"--dirs-only",
		"--no-modtime",
		"--no-mimetype",
	}

	linkCmdParent := exec.CommandContext(ctx, "rclone", linkArgsParent...)
	if linkOutputParent, err := linkCmdParent.Output(); err == nil {
		var files []map[string]interface{}
		if json.Unmarshal(linkOutputParent, &files) == nil {
			for _, folder := range files {
				if name, ok := folder["Name"].(string); ok && name == parentName {
					if id, ok := folder["ID"].(string); ok {
						parentID = id
					} else if id, ok := folder["Id"].(string); ok {
						parentID = id
					}
					break
				}
			}
		}
	} else {
		slog.Warn("Failed to list grandparent directory", "grandParentPath", grandParentPath, "error", err)
	}

	if fileID != "" && parentID != "" {
		baseURL := strings.TrimRight(s.Config.IndexURL, "/")
		encodedFileName := url.PathEscape(task.FileName)

		task.RemoteURL = fmt.Sprintf("%s/en/folder/%s/file/%s/%s", baseURL, parentID, fileID, encodedFileName)
		slog.Info("Generated ID-based Index URL", "url", task.RemoteURL)
		return true
	}

	slog.Warn("Could not generate ID-based URL (missing IDs)", "fileID", fileID, "parentID", parentID, "parentPath", parentPath, "parentName", parentName)
	return false
}

func (s *BotService) generatePathBasedIndexURL(task *Task, currentRemotePath string) bool {
	remotePathSlash := strings.ReplaceAll(currentRemotePath, "\\", "/")
	rcloneDestSlash := strings.ReplaceAll(s.TaskManager.RcloneDest, "\\", "/")
	rcloneDestSlash = strings.TrimRight(rcloneDestSlash, "/")

	var relPath string
	if strings.HasPrefix(remotePathSlash, rcloneDestSlash) {
		relPath = strings.TrimPrefix(remotePathSlash, rcloneDestSlash)
	} else {
		parts := strings.SplitN(remotePathSlash, ":", 2)
		if len(parts) > 1 {
			relPath = parts[1]
		} else {
			relPath = remotePathSlash
		}
	}

	relPath = strings.TrimLeft(relPath, "/")
	pathParts := strings.Split(relPath, "/")
	for i, part := range pathParts {
		pathParts[i] = url.PathEscape(part)
	}
	encodedPath := strings.Join(pathParts, "/")

	baseURL := strings.TrimRight(s.Config.IndexURL, "/")
	task.RemoteURL = fmt.Sprintf("%s/%s", baseURL, encodedPath)
	slog.Info("Generated Path-based Index URL", "url", task.RemoteURL)
	return true
}

func (s *BotService) generateDirectLink(ctx context.Context, task *Task, configPath, currentRemotePath string, isDirUpload bool) {
	linkArgs := []string{
		"link",
		"--config", configPath,
		currentRemotePath,
	}

	linkCmd := exec.CommandContext(ctx, "rclone", linkArgs...)
	linkOutput, linkErr := linkCmd.Output()
	if linkErr == nil {
		task.RemoteURL = strings.TrimSpace(string(linkOutput))
		return
	}

	slog.Warn("Failed to get direct link", "fileName", task.FileName, "error", linkErr)

	if !isDirUpload {
		slog.Debug("Skipping directory fallback for file", "fileName", task.FileName)
		return
	}

	s.generateDirectoryLink(ctx, task, configPath, currentRemotePath)
}

func (s *BotService) generateDirectoryLink(ctx context.Context, task *Task, configPath, currentRemotePath string) {
	parentPath := path.Dir(currentRemotePath)
	idArgs := []string{
		"lsjson",
		"--config", configPath,
		"--dirs-only",
		parentPath,
	}
	idCmd := exec.CommandContext(ctx, "rclone", idArgs...)
	idOutput, idErr := idCmd.Output()
	if idErr != nil {
		slog.Error("Failed to list parent directory contents", "error", idErr)
	} else {
		var files []map[string]interface{}
		if errUnmarshal := json.Unmarshal(idOutput, &files); errUnmarshal != nil {
			slog.Error("Could not parse lsjson output", "error", errUnmarshal)
		} else {
			for _, file := range files {
				if name, ok := file["Name"].(string); ok && name == task.FileName {
					if id, ok := file["ID"].(string); ok {
						task.RemoteURL = "https://drive.google.com/drive/folders/" + id
						return
					}
					if id, ok := file["Id"].(string); ok {
						task.RemoteURL = "https://drive.google.com/drive/folders/" + id
						return
					}
					slog.Warn("Found folder but no ID field in lsjson output", "fileName", task.FileName)
				}
			}
		}
	}

	linkArgsParent := []string{
		"link",
		"--config", configPath,
		parentPath,
	}
	linkCmdParent := exec.CommandContext(ctx, "rclone", linkArgsParent...)
	linkOutputParent, linkErrParent := linkCmdParent.Output()
	if linkErrParent == nil {
		baseURL := strings.TrimSpace(string(linkOutputParent))
		task.RemoteURL = baseURL + "#folders/" + task.FileName
	} else {
		slog.Warn("Failed to get parent directory link", "fileName", task.FileName, "error", linkErrParent)
		task.RemoteURL = "https://drive.google.com/drive/search?q=\"" + task.FileName + "\" in parents"
	}
}

func (s *BotService) parseRcloneProgress(task *Task, reader io.ReadCloser) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(utils.ScanLinesWithCR)

	segmentRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*([a-zA-Z]*i?[Bb]?)\s*/\s*(\d+(?:\.\d+)?)\s*([a-zA-Z]*i?[Bb]?),\s*(\d+(?:\.\d+)?)%,\s*(\d+(?:\.\d+)?)\s*([a-zA-Z]*i?[Bb]?)/s,\s*ETA\s+([^\d]*)(\d+[smhd](?:\d+[smhd])*)?`)

	lastUpdate := time.Now()
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		lineCount++
		if lineCount <= 10 {
			slog.Info("Rclone raw output", "taskID", task.ID, "line", line)
		}

		allMatches := segmentRegex.FindAllStringSubmatch(line, -1)

		if len(allMatches) > 0 {
			for i, matches := range allMatches {
				if len(matches) >= 8 {
					task.Mu.Lock()

					uploadedVal := matches[1] + matches[2]
					totalVal := matches[3] + matches[4]
					task.UploadedSize = utils.ParseBytesString(uploadedVal)
					task.TotalSize = utils.ParseBytesString(totalVal)

					if pct, err := strconv.ParseFloat(matches[5], 64); err == nil {
						task.Progress = pct
					}

					speedVal := matches[6] + matches[7]
					task.Speed = utils.ParseBytesString(speedVal)

					if len(matches) >= 10 && matches[9] != "" {
						if d, err := time.ParseDuration(matches[9]); err == nil {
							task.ETA = d
						}
					}

					task.Mu.Unlock()

					if i%10 == 0 && time.Since(lastUpdate) >= 2*time.Second {
						s.updateTaskStatus(task)
						lastUpdate = time.Now()
					}
				}
			}

			if time.Since(lastUpdate) >= 2*time.Second {
				s.updateTaskStatus(task)
				lastUpdate = time.Now()
			}

			slog.Info("Rclone progress parsed", "taskID", task.ID, "segments", len(allMatches), "progress", task.Progress)
		} else {
			simpleProgressRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)%`)
			simpleSizeRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*([a-zA-Z]*i?[Bb]?)\s*/\s*(\d+(?:\.\d+)?)\s*([a-zA-Z]*i?[Bb]?)`)
			simpleSpeedRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*([a-zA-Z]*i?[Bb]?)/s`)

			task.Mu.Lock()

			if allPct := simpleProgressRegex.FindAllStringSubmatch(line, -1); len(allPct) > 0 {
				lastPct := allPct[len(allPct)-1]
				if pct, err := strconv.ParseFloat(lastPct[1], 64); err == nil {
					task.Progress = pct
				}
			}

			if allSize := simpleSizeRegex.FindAllStringSubmatch(line, -1); len(allSize) > 0 {
				lastSize := allSize[len(allSize)-1]
				if len(lastSize) >= 5 {
					task.UploadedSize = utils.ParseBytesString(lastSize[1] + lastSize[2])
					task.TotalSize = utils.ParseBytesString(lastSize[3] + lastSize[4])
				}
			}

			if allSpeed := simpleSpeedRegex.FindAllStringSubmatch(line, -1); len(allSpeed) > 0 {
				lastSpeed := allSpeed[len(allSpeed)-1]
				if len(lastSpeed) >= 3 {
					task.Speed = utils.ParseBytesString(lastSpeed[1] + lastSpeed[2])
				}
			}

			task.Mu.Unlock()
		}
	}

	if lineCount > 0 {
		slog.Info("Rclone parser finished", "taskID", task.ID, "linesProcessed", lineCount, "finalProgress", task.Progress)
	}
}

func (s *BotService) UploadToTelegram(task *Task) error {
	task.SetStatus(StatusUploading)
	task.Mu.Lock()
	task.Progress = 0
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	filePath := task.LocalPath
	if filePath == "" {
		return fmt.Errorf("no file to upload")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	if info.IsDir() {
		return fmt.Errorf("cannot upload directory to telegram directly, please zip it first")
	}

	if info.Size() > 2*1024*1024*1024 {
		return fmt.Errorf("file too large for telegram (max 2GB)")
	}

	var msg tgbotapi.Chattable
	if organizer.IsVideoFile(filePath) {
		video := tgbotapi.NewVideo(task.ChatID, tgbotapi.FilePath(filePath))
		video.Caption = fmt.Sprintf("📄 %s", task.FileName)

		if thumb, errThumb := GenerateThumbnail(filePath, s.TaskManager.DownloadDir); errThumb == nil {
			video.Thumb = tgbotapi.FilePath(thumb)
			defer func() {
				if errRemove := os.Remove(thumb); errRemove != nil {
					slog.Warn("Failed to remove thumbnail", "error", errRemove, "path", thumb)
				}
			}()
		}

		msg = video
	} else {
		doc := tgbotapi.NewDocument(task.ChatID, tgbotapi.FilePath(filePath))
		doc.Caption = fmt.Sprintf("📄 %s", task.FileName)
		msg = doc
	}

	task.Mu.Lock()
	task.Progress = 50
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	sentMsg, err := s.Bot.Send(msg)
	if err != nil {
		return fmt.Errorf("telegram upload failed: %v", err)
	}

	task.Mu.Lock()
	task.ResultMessageID = sentMsg.MessageID
	task.Progress = 100
	task.UploadedSize = info.Size()
	task.RemotePath = "telegram"
	task.Mu.Unlock()

	return nil
}
