package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

	url, _, _, _, _, name := utils.ParseFlags(args)
	if url == "" {
		url = utils.ExtractURLFromText(args)
	}

	if url == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL Google Drive untuk di\\-clone\\.")
		msg.ParseMode = MarkdownV2
		_, _ = s.Bot.Send(msg)
		return
	}

	if !strings.Contains(url, "drive.google.com") && !strings.Contains(url, "docs.google.com") {
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

	task, err := s.TaskManager.CreateTask(TypeClone, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, false, false, "", "", 0)
	if err != nil {
		s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
		return
	}
	s.handleAutoDelete(task)
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

	driveName, isDir, err := s.getDriveInfo(driveID, configPath, remoteName, isFolderHint, task.URL)
	if err != nil {
		task.SetError(fmt.Sprintf("Gagal mendapatkan info Google Drive: %v", err))
		s.updateTaskStatus(task)
		return
	}

	task.Mu.Lock()
	if task.FileName == "cloning..." || task.FileName == "" {
		task.FileName = driveName
	}
	name := task.FileName
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	var args []string
	rcloneDest := strings.ReplaceAll(s.TaskManager.RcloneDest, "\\", "/")
	dest := rcloneDest

	if isDir {
		dest = path.Join(dest, name)
		src := fmt.Sprintf("%s,root_folder_id=%s:", remoteName, driveID)
		args = []string{
			"copy",
			src,
			dest,
		}
	} else {
		src := fmt.Sprintf("%s,id=%s:", remoteName, driveID)
		dest = path.Join(dest, name)
		args = []string{
			"copyto",
			src,
			dest,
		}
	}

	task.RemotePath = dest

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
		"-v",
	}
	args = append(args, commonArgs...)

	slog.Debug("Running rclone clone command", "taskID", task.ID, "args", strings.Join(args, " "))

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
		if finalSize, err := s.getRcloneSize(ctx, dest, configPath); err == nil {
			task.Mu.Lock()
			task.TotalSize = finalSize
			task.DownloadedSize = finalSize
			task.Mu.Unlock()
		}
	}

	s.generateRcloneLink(ctx, task, configPath, isDir)
	task.SetStatus(StatusCompleted)
	s.updateTaskStatus(task)
}

func (s *BotService) parseCloneProgress(task *Task, reader io.ReadCloser) {
	scanner := bufio.NewScanner(reader)
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
	pctRegex := regexp.MustCompile(`(\d+)%`)
	if matches := pctRegex.FindStringSubmatch(line); len(matches) >= 2 {
		if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
			task.Progress = pct
		}
	}

	sizeRegex := regexp.MustCompile(`(\d+(?:\.\d+)?\s*[a-zA-Z]+i?B)\s*/\s*(\d+(?:\.\d+)?\s*[a-zA-Z]+i?B)`)
	if matches := sizeRegex.FindStringSubmatch(line); len(matches) >= 3 {
		task.Mu.Lock()
		task.DownloadedSize = utils.ParseBytesString(matches[1])
		task.TotalSize = utils.ParseBytesString(matches[2])
		task.Mu.Unlock()
	}

	speedRegex := regexp.MustCompile(`,\s*(\d+(?:\.\d+)?\s*[a-zA-Z]+i?B/s)`)
	if matches := speedRegex.FindStringSubmatch(line); len(matches) >= 2 {
		speedStr := matches[1]
		task.Mu.Lock()
		task.Speed = utils.ParseBytesString(speedStr)
		task.Mu.Unlock()
	}
}

func extractDriveID(urlStr string) (string, bool) {
	folderRegex := regexp.MustCompile(`folders/([a-zA-Z0-9_-]{25,})`)
	if matches := folderRegex.FindStringSubmatch(urlStr); len(matches) >= 2 {
		return matches[1], true
	}

	fileRegex := regexp.MustCompile(`(?:d/|id=)([a-zA-Z0-9_-]{25,})`)
	if matches := fileRegex.FindStringSubmatch(urlStr); len(matches) >= 2 {
		return matches[1], false
	}

	return "", false
}

func (s *BotService) getDriveInfo(id, configPath, remoteName string, isFolder bool, urlStr string) (string, bool, error) {
	name := getDriveNameFromURL(urlStr)
	if name != "" {
		slog.Debug("Resolved GDrive name from title", "id", id, "name", name)
		return name, isFolder, nil
	}

	args := []string{
		"lsjson",
		fmt.Sprintf("%s:", remoteName),
		"--drive-root-folder-id", id,
		"--max-depth", "0",
		"--config", configPath,
	}

	cmd := exec.Command("rclone", args...)
	output, err := cmd.Output()
	if err != nil {
		if isFolder {
			return "Folder_" + id, true, nil
		}
		return "File_" + id, false, nil
	}

	var infos []map[string]interface{}
	if err := json.Unmarshal(output, &infos); err != nil || len(infos) == 0 {
		if isFolder {
			return "Folder_" + id, true, nil
		}
		return "File_" + id, false, nil
	}

	if len(infos) > 0 {
		info := infos[0]
		isDir, _ := info["IsDir"].(bool)
		size, _ := info["Size"].(float64)
		if !isDir && size > 0 {
			slog.Debug("Detected GDrive file size", "id", id, "size", size)
		}
		return name, isFolder || isDir, nil
	}

	return "Item_" + id, isFolder, nil
}

func getDriveNameFromURL(urlStr string) string {
	client := &http.Client{
		Timeout: 10 * time.Second,
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
	if strings.Contains(title, "Sign-in") || strings.Contains(title, "Masuk") {
		return ""
	}

	return strings.TrimSpace(title)
}
