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

func uploadWithRclone(bot *tgbotapi.BotAPI, task *Task) error {
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
	progressRegex := regexp.MustCompile(`Transferred:.*?(\d+)%`)
	speedRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(B|KiB|MiB|GiB)/s`)

	lastUpdate := time.Now()

	for scanner.Scan() {
		line := scanner.Text()

		if matches := progressRegex.FindStringSubmatch(line); len(matches) >= 2 {
			if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
				task.Progress = pct
			}
		}

		if matches := speedRegex.FindStringSubmatch(line); len(matches) >= 3 {
			speed, err := strconv.ParseFloat(matches[1], 64)
			if err != nil {
				continue
			}
			switch matches[2] {
			case "KiB":
				task.Speed = int64(speed * 1024)
			case "MiB":
				task.Speed = int64(speed * 1024 * 1024)
			case "GiB":
				task.Speed = int64(speed * 1024 * 1024 * 1024)
			default:
				task.Speed = int64(speed)
			}
		}

		if time.Since(lastUpdate) >= 5*time.Second {
			updateTaskStatus(bot, task)
			lastUpdate = time.Now()
		}
	}
}
