package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"zee-mirror/pkg/utils"
)

func (s *BotService) UploadWithRclone(task *Task) error {
	task.SetStatus(StatusUploading)
	task.Progress = 0
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
				log.Printf("[Upload] Warning: Could not calculate directory size: %v", err)
				task.TotalSize = info.Size()
			} else {
				task.TotalSize = dirSize
			}
		} else {
			task.TotalSize = info.Size()
		}
		task.Mu.Unlock()
	}

	remotePath := filepath.Join(s.TaskManager.RcloneDest, task.FileName)
	task.RemotePath = remotePath

	rcloneDest := s.TaskManager.RcloneDest
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
		"-v",
	}

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	//nolint:gosec
	cmd := exec.CommandContext(ctx, "rclone", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start rclone: %v", err)
	}

	go s.parseRcloneProgress(task, stderr)

	if err := cmd.Wait(); err != nil {
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

	task.Progress = 100
	return nil
}

func (s *BotService) generateRcloneLink(ctx context.Context, task *Task, configPath string, isDirUpload bool) {

	linkArgs := []string{
		"link",
		"--config", configPath,
		filepath.Join(s.TaskManager.RcloneDest, task.FileName),
	}
	//nolint:gosec
	linkCmd := exec.CommandContext(ctx, "rclone", linkArgs...)
	linkOutput, linkErr := linkCmd.Output()
	if linkErr == nil {
		task.RemoteURL = strings.TrimSpace(string(linkOutput))
		return
	}

	log.Printf("[RcloneLink] Failed to get direct link for %s: %v", task.FileName, linkErr)

	if !isDirUpload {
		log.Printf("[RcloneLink] Skipping directory fallback for file: %s", task.FileName)
		return
	}

	idArgs := []string{
		"lsjson",
		"--config", configPath,
		"--dirs-only",
		s.TaskManager.RcloneDest,
	}
	idCmd := exec.CommandContext(ctx, "rclone", idArgs...)
	idOutput, idErr := idCmd.Output()
	if idErr == nil {
		var files []map[string]interface{}
		if err := json.Unmarshal(idOutput, &files); err == nil {
			for _, file := range files {
				if name, ok := file["Name"].(string); ok && name == task.FileName {
					if id, ok := file["ID"].(string); ok {
						task.RemoteURL = "https://drive.google.com/drive/folders/" + id
						return
					} else if id, ok := file["Id"].(string); ok {
						task.RemoteURL = "https://drive.google.com/drive/folders/" + id
						return
					} else {
						log.Printf("[RcloneLink] Found folder %s but no ID field in lsjson output", task.FileName)
					}
				}
			}
		} else {
			log.Printf("[RcloneLink] Could not parse lsjson output for parent directory: %v", err)
		}
	} else {
		log.Printf("[RcloneLink] Failed to list parent directory contents: %v", idErr)
	}

	parentPath := s.TaskManager.RcloneDest
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
		log.Printf("[RcloneLink] Also failed to get parent directory link for %s: %v", task.FileName, linkErrParent)
		task.RemoteURL = "https://drive.google.com/drive/search?q=\"" + task.FileName + "\" in parents"
	}
}

func (s *BotService) parseRcloneProgress(task *Task, reader io.ReadCloser) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		for i := 0; i < len(data); i++ {
			if data[i] == '\n' || data[i] == '\r' {
				return i + 1, data[0:i], nil
			}
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	progressRegex := regexp.MustCompile(`Transferred:.*?(\d+)%`)
	speedRegex := regexp.MustCompile(`,\s*(\d+(?:\.\d+)?)\s*([a-zA-Z]+/s)`)

	lastUpdate := time.Now()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.Contains(line, "Transferred:") {
			log.Printf("[Rclone %s] %s", task.ID, line)
		}

		if matches := progressRegex.FindStringSubmatch(line); len(matches) >= 2 {
			if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
				task.Progress = pct
			}
		}

		if matches := speedRegex.FindStringSubmatch(line); len(matches) >= 3 {
			speedStr := matches[1] + matches[2]
			task.Mu.Lock()
			task.Speed = utils.ParseBytesString(speedStr)
			task.Mu.Unlock()
		}

		if time.Since(lastUpdate) >= 2*time.Second {
			s.updateTaskStatus(task)
			lastUpdate = time.Now()
		}
	}
}
