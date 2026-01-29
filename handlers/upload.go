package handlers

import (
	"bufio"
	"context"
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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func UploadWithRclone(bot *tgbotapi.BotAPI, task *Task) error {
	task.SetStatus(StatusUploading)
	task.Progress = 0
	updateTaskStatus(bot, task)

	uploadPath := task.LocalPath
	if uploadPath == "" {
		return fmt.Errorf("no file to upload")
	}

	if info, err := os.Stat(uploadPath); err == nil {
		task.Mu.Lock()
		task.TotalSize = info.Size()
		task.Mu.Unlock()
	}

	remotePath := filepath.Join(taskManager.RcloneDest, task.FileName)
	task.RemotePath = remotePath

	configPath := filepath.Join(taskManager.ConfigDir, "rclone.conf")
	args := []string{
		"copy",
		uploadPath,
		taskManager.RcloneDest,
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

	go parseRcloneProgress(bot, task, stderr)

	err = cmd.Wait()

	if task.Status == StatusCancelled {
		return fmt.Errorf("task cancelled")
	}

	if err != nil {
		return fmt.Errorf("rclone failed: %v", err)
	}

	linkArgs := []string{
		"link",
		"--config", configPath,
		filepath.Join(taskManager.RcloneDest, task.FileName),
	}
	//nolint:gosec
	linkCmd := exec.CommandContext(ctx, "rclone", linkArgs...)
	linkOutput, linkErr := linkCmd.Output()
	if linkErr == nil {
		task.RemoteURL = strings.TrimSpace(string(linkOutput))
	} else {
		log.Printf("[RcloneLink] Failed to get link for %s: %v", task.FileName, linkErr)
	}

	task.Progress = 100
	return nil
}

func parseRcloneProgress(bot *tgbotapi.BotAPI, task *Task, reader io.ReadCloser) {
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
			task.Speed = ParseBytesString(speedStr)
			task.Mu.Unlock()
		}

		if time.Since(lastUpdate) >= 2*time.Second {
			updateTaskStatus(bot, task)
			lastUpdate = time.Now()
		}
	}
}
