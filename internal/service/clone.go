package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) HandleClone(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, _, _, _, _, name, _, _ := utils.ParseFlags(args)
	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL Google Drive untuk di\\-clone\\.")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if !strings.Contains(url, "drive.google.com") && !strings.Contains(url, "docs.google.com") && !strings.Contains(url, "drive.usercontent.google.com") {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nURL bukan link Google Drive yang valid\\.")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	replyID := 0
	if message.ReplyToMessage != nil {
		replyID = message.ReplyToMessage.MessageID
	}
	fileName := name
	if fileName == "" {
		fileName = "cloning..."
	}

	task, err := s.TaskManager.CreateTask(TypeClone, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, false, false, "", "", 0, "", false)
	if err != nil {
		s.HandleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.HandleAutoDelete(task)
	s.UpdateSharedDashboard(message.Chat.ID, true)
	slog.Info("Clone task created", "taskID", task.ID, "url", url, "name", name)
}

func (s *BotService) cloneWithRclone(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	driveID, isFolderHint := extractDriveID(task.URL)
	if driveID == "" {
		task.SetError("Gagal ekstrak ID dari URL Google Drive")
		s.updateTaskStatus(task)
		return
	}

	configPath := filepath.Join(s.TaskManager.ConfigDir, "rclone.conf")
	remoteName := strings.Split(s.TaskManager.RcloneDest, ":")[0]

	driveName, isDir, driveSize, err := s.getDriveInfo(driveID, configPath, remoteName, isFolderHint, task.URL)
	if err != nil {
		task.SetError(fmt.Sprintf("Gagal mendapatkan info Google Drive: %v", err))
		s.updateTaskStatus(task)
		return
	}

	task.Mu.Lock()
	if (task.FileName == "cloning..." || task.FileName == "") && driveName != "" {
		task.FileName = driveName
	}
	if task.FileName == "cloning..." || task.FileName == "" {
		fallback := utils.GetFileNameFromURL(task.URL)
		if fallback != "" {
			task.FileName = fallback
		} else {
			task.FileName = "file_" + driveID
		}
	}
	name := task.FileName
	if driveSize > 0 {
		task.TotalSize = driveSize
	}
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	var args []string
	rcloneDest := strings.ReplaceAll(s.TaskManager.RcloneDest, "\\", "/")
	dest := rcloneDest

	commonArgs := []string{
		"--config", configPath,
		"--progress",
		"--stats", "1s",
		"--stats-one-line",
		"--transfers", "10",
		"--checkers", "20",
		"--drive-chunk-size", "256M",
		"--buffer-size", "128M",
		"--use-mmap",
		"--no-traverse",
		"--drive-pacer-min-sleep", "10ms",
		"--drive-pacer-burst", "200",
		"--drive-server-side-across-configs",
		"--drive-acknowledge-abuse",
		"--drive-description", "Mirrored by Zee-Mirror",
		"--log-level", "NOTICE",
	}

	if isDir {
		dest = path.Join(dest, name)
		src := fmt.Sprintf("%s,root_folder_id=%s:", remoteName, driveID)
		args = []string{
			"copy",
			src,
			dest,
		}
		args = append(args, commonArgs...)
	} else {
		dest = path.Join(dest, name)

		args = []string{
			"backend", "copyid",
			fmt.Sprintf("%s:", remoteName),
			driveID,
			dest,
			"--config", configPath,
			"-o", "drive-acknowledge-abuse=true",
			"-o", "drive-description=Mirrored by Zee-Mirror",
		}
	}

	task.RemotePath = dest

	slog.Debug("Running rclone clone command", "taskID", task.ID, "args", strings.Join(args, " "))

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", args...)
	stderr, _ := cmd.StderrPipe()

	slog.Info("Starting rclone clone", "taskID", task.ID, "args", strings.Join(args, " "))

	if err := cmd.Start(); err != nil {
		task.SetError(fmt.Sprintf("rclone failed to start: %v", err))
		s.updateTaskStatus(task)
		return
	}

	s.parseCloneProgress(task, stderr)

	if err := cmd.Wait(); err != nil {
		if task.Status == StatusCancelled {
			return
		}
		task.SetError(fmt.Sprintf("rclone clone failed: %v", err))
		s.updateTaskStatus(task)
		return
	}

	if task.Status == StatusCancelled {
		return
	}

	task.Mu.RLock()
	currentSize := task.TotalSize
	task.Mu.RUnlock()

	if currentSize <= 0 {
		if finalSize, err := s.getRcloneSize(ctx, dest, configPath); err == nil && finalSize > 0 {
			task.Mu.Lock()
			task.TotalSize = finalSize
			task.DownloadedSize = finalSize
			task.Mu.Unlock()
		} else {
			task.Mu.Lock()
			if task.DownloadedSize > 0 {
				task.TotalSize = task.DownloadedSize
			}
			task.Mu.Unlock()
		}
	}

	s.RcloneUploader.GenerateRcloneLink(ctx, &task.Task, configPath, isDir)
	task.SetStatus(StatusCompleted)
	s.updateTaskStatus(task)
}

func (s *BotService) downloadGDriveWithRclone(task *Task) {
	task.SetStatus(StatusDownloading)
	task.Mu.Lock()
	task.StartedAt = time.Now()
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	driveID, isFolderHint := extractDriveID(task.URL)
	if driveID == "" {
		slog.Warn("Failed to extract Drive ID from URL, falling back to Aria2", "url", task.URL)
		s.downloadWithAria2(task)
		return
	}

	configPath := filepath.Join(s.TaskManager.ConfigDir, "rclone.conf")
	remoteName := strings.Split(s.TaskManager.RcloneDest, ":")[0]

	driveName, isDir, driveSize, err := s.getDriveInfo(driveID, configPath, remoteName, isFolderHint, task.URL)
	if err != nil {
		slog.Warn("Failed to get Google Drive info", "error", err)
	}

	task.Mu.Lock()
	if driveName != "" && (task.FileName == "" || task.FileName == "download" || utils.GetFileNameFromURL(task.URL) == task.FileName) {
		task.FileName = driveName
		if driveSize > 0 {
			task.TotalSize = driveSize
		}
	}
	name := task.FileName
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	outputDir := filepath.Join(s.TaskManager.DownloadDir, task.ID)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		task.SetError(fmt.Sprintf("Failed to create download directory: %v", err))
		s.updateTaskStatus(task)
		return
	}

	commonArgs := []string{
		"--config", configPath,
		"--progress",
		"--stats", "1s",
		"--stats-one-line",
		"--transfers", "10",
		"--checkers", "20",
		"--drive-chunk-size", "256M",
		"--buffer-size", "128M",
		"--use-mmap",
		"--no-traverse",
		"--drive-pacer-min-sleep", "10ms",
		"--drive-pacer-burst", "200",
		"--drive-acknowledge-abuse",
		"--log-level", "NOTICE",
	}

	var args []string
	if isDir {
		src := fmt.Sprintf("%s,root_folder_id=%s:", remoteName, driveID)
		dest := filepath.Join(outputDir, name)
		args = []string{
			"copy",
			src,
			dest,
		}
		args = append(args, commonArgs...)
	} else {
		src := fmt.Sprintf("%s,root_folder_id=%s:%s", remoteName, driveID, name)
		dest := filepath.Join(outputDir, name)
		args = []string{
			"copyto",
			src,
			dest,
		}
		args = append(args, commonArgs...)
	}

	slog.Info("Starting local Rclone download", "taskID", task.ID, "args", strings.Join(args, " "))

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rclone", args...)
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		task.SetError(fmt.Sprintf("rclone failed to start: %v", err))
		s.updateTaskStatus(task)
		return
	}

	s.parseCloneProgress(task, stderr)

	if err := cmd.Wait(); err != nil {
		if task.Status == StatusCancelled {
			s.cleanupTask(task)
			return
		}
		task.SetError(fmt.Sprintf("Local rclone download failed: %v", err))
		s.updateTaskStatus(task)
		s.cleanupTask(task)
		return
	}

	if task.Status == StatusCancelled {
		s.cleanupTask(task)
		return
	}

	s.HandlePostDownload(task, outputDir)
}

func (s *BotService) parseCloneProgress(task *Task, reader io.ReadCloser) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(utils.ScanLinesWithCR)
	lastUpdate := time.Now()

	for scanner.Scan() {
		line := scanner.Text()
		slog.Debug("Rclone clone progress", "taskID", task.ID, "line", line)

		s.handleRcloneLine(task, line)

		if time.Since(lastUpdate) >= 2*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
		}
	}
}

func (s *BotService) getRcloneSize(ctx context.Context, remotePath, configPath string) (int64, error) {
	args := []string{"size", "--json", remotePath, "--config", configPath}
	cmd := exec.CommandContext(ctx, "rclone", args...)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var res struct {
		TotalBytes int64 `json:"bytes"`
	}
	if err := json.Unmarshal(output, &res); err != nil {
		return 0, err
	}
	return res.TotalBytes, nil
}

func (s *BotService) handleRcloneLine(task *Task, line string) {
	sizeRegex := regexp.MustCompile(`(\d+(?:\.\d+)?\s*[a-zA-Z]+i?B)\s*/\s*(\d+(?:\.\d+)?\s*[a-zA-Z]+i?B)`)
	progressRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)%`)
	speedRegex := regexp.MustCompile(`(?i)(?:,|\s)(\d+(?:\.\d+)?\s*[a-zA-Z]+i?B/s)`)
	etaRegex := regexp.MustCompile(`(?i)(?:ETA\s+|,\s+)(\d+[smhd])`)

	task.Mu.Lock()
	defer task.Mu.Unlock()

	if matches := sizeRegex.FindStringSubmatch(line); len(matches) >= 3 {
		task.DownloadedSize = utils.ParseBytesString(matches[1])
		task.TotalSize = utils.ParseBytesString(matches[2])
	}

	if matches := progressRegex.FindStringSubmatch(line); len(matches) >= 2 {
		if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
			task.Progress = pct
		}
	}

	if matches := speedRegex.FindStringSubmatch(line); len(matches) >= 2 {
		task.Speed = utils.ParseBytesString(matches[1])
	}

	if matches := etaRegex.FindStringSubmatch(line); len(matches) >= 2 {
		if d, err := time.ParseDuration(matches[1]); err == nil {
			task.ETA = d
		}
	}
}

func extractDriveID(urlStr string) (string, bool) {
	folderRegex := regexp.MustCompile(`folders/([a-zA-Z0-9_-]+)`)
	if matches := folderRegex.FindStringSubmatch(urlStr); len(matches) >= 2 {
		return matches[1], true
	}

	fileRegex := regexp.MustCompile(`(?:d/|id=)([a-zA-Z0-9_-]+)`)
	if matches := fileRegex.FindStringSubmatch(urlStr); len(matches) >= 2 {
		return matches[1], false
	}

	return "", false
}

func (s *BotService) getDriveInfo(id, configPath, remoteName string, isFolder bool, urlStr string) (string, bool, int64, error) {
	idArgs := []string{
		"lsjson",
		fmt.Sprintf("%s,root_folder_id=%s:", remoteName, id),
		"--max-depth", "0",
		"--stat",
		"--config", configPath,
		"--no-mimetype",
		"--no-modtime",
	}
	idCmd := exec.Command("rclone", idArgs...)
	if out, err := idCmd.Output(); err == nil {
		var info map[string]interface{}
		if json.Unmarshal(out, &info) == nil {
			fetchedID, _ := info["ID"].(string)
			if fetchedID == "" {
				fetchedID, _ = info["Id"].(string)
			}
			if fetchedID == id {
				fileName, _ := info["Name"].(string)
				isDirVal, _ := info["IsDir"].(bool)
				sizeVal := int64(0)
				if s, ok := info["Size"].(float64); ok {
					sizeVal = int64(s)
				}
				if fileName != "" {
					slog.Debug("Resolved GDrive info from lsjson --stat", "id", id, "name", fileName, "isDir", isDirVal)
					return fileName, isDirVal, sizeVal, nil
				}
			} else {
				slog.Warn("Fetched ID (lsjson) does not match requested ID", "expected", id, "got", fetchedID)
			}
		} else {
			var infos []map[string]interface{}
			if json.Unmarshal(out, &infos) == nil && len(infos) > 0 {
				info := infos[0]
				fetchedID, _ := info["ID"].(string)
				if fetchedID == "" {
					fetchedID, _ = info["Id"].(string)
				}
				if fetchedID == id {
					fileName, _ := info["Name"].(string)
					isDirVal, _ := info["IsDir"].(bool)
					sizeVal := int64(0)
					if s, ok := info["Size"].(float64); ok {
						sizeVal = int64(s)
					}
					if fileName != "" {
						slog.Debug("Resolved GDrive info from lsjson list", "id", id, "name", fileName, "isDir", isDirVal)
						return fileName, isDirVal, sizeVal, nil
					}
				}
			}
		}
	}

	args := []string{
		"backend", "info",
		fmt.Sprintf("%s:", remoteName),
		"-o", fmt.Sprintf("root_folder_id=%s", id),
		"--config", configPath,
		"-o", "drive-acknowledge-abuse=true",
		"--json",
	}

	cmd := exec.Command("rclone", args...)
	output, err := cmd.Output()
	if err == nil {
		var info map[string]interface{}
		if unmarshalErr := json.Unmarshal(output, &info); unmarshalErr == nil {
			fetchedID, _ := info["id"].(string)
			if fetchedID == id {
				fileName, _ := info["name"].(string)
				mimeType, _ := info["mimeType"].(string)
				sizeVal := int64(0)
				if s, ok := info["size"].(float64); ok {
					sizeVal = int64(s)
				} else if sStr, ok := info["size"].(string); ok {
					sizeVal, _ = strconv.ParseInt(sStr, 10, 64)
				}

				isDir := strings.Contains(mimeType, "folder")

				if fileName != "" {
					slog.Debug("Resolved GDrive info from backend info", "id", id, "name", fileName, "isDir", isDir, "size", sizeVal)
					return fileName, isDir, sizeVal, nil
				}
			} else {
				slog.Warn("Fetched ID (backend info) does not match requested ID", "expected", id, "got", fetchedID)
			}
		}
	}

	scrapeURL := ConstructScrapeURL(id, isFolder, urlStr)
	name := getDriveNameFromURL(scrapeURL)
	if name != "" {
		slog.Debug("Resolved GDrive name from title", "id", id, "name", name)
		return name, isFolder, 0, nil
	}

	if isFolder {
		return "Folder_" + id, true, 0, nil
	}

	return "File_" + id, false, 0, nil
}

func getDriveNameFromURL(urlStr string) string {
	if strings.Contains(urlStr, "drive.usercontent.google.com") {
		return ""
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(urlStr)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*100))
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return ""
	}

	title := matches[1]
	if idx := strings.Index(title, " - Google "); idx != -1 {
		title = title[:idx]
	}
	if strings.Contains(title, "Sign-in") || strings.Contains(title, "Masuk") || strings.Contains(title, "Virus scan") {
		return ""
	}

	return strings.TrimSpace(title)
}

func ConstructScrapeURL(id string, isFolder bool, originalURL string) string {
	if id == "" {
		return originalURL
	}

	if isFolder {
		return fmt.Sprintf("https://drive.google.com/drive/folders/%s", id)
	}
	return fmt.Sprintf("https://drive.google.com/file/d/%s/view", id)
}
