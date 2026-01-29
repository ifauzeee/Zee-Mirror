package handlers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func extractArchive(bot *tgbotapi.BotAPI, task *Task) error {
	task.SetStatus(StatusExtracting)
	updateTaskStatus(bot, task)

	extractDir := filepath.Join(filepath.Dir(task.LocalPath), "extracted")
	os.MkdirAll(extractDir, 0755)

	args := []string{
		"x",
		task.LocalPath,
		"-o" + extractDir,
		"-y",
	}

	if task.Password != "" {
		args = append(args, "-p"+task.Password)
	}

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "7z", args...)

	output, err := cmd.CombinedOutput()

	if task.Status == StatusCancelled {
		return fmt.Errorf("task cancelled")
	}

	if err != nil {
		return fmt.Errorf("7zz extract failed: %v, output: %s", err, string(output))
	}

	task.LocalPath = extractDir
	task.FileName = filepath.Base(extractDir)

	return nil
}

func createZipArchive(bot *tgbotapi.BotAPI, task *Task) error {
	task.SetStatus(StatusZipping)
	updateTaskStatus(bot, task)

	zipPath := task.LocalPath + ".zip"
	if filepath.Ext(task.LocalPath) != "" {
		zipPath = task.LocalPath[:len(task.LocalPath)-len(filepath.Ext(task.LocalPath))] + ".zip"
	}

	args := []string{
		"a",
		"-tzip",
		zipPath,
		task.LocalPath,
		"-y",
	}

	if task.Password != "" {
		args = append(args, "-p"+task.Password)
		args = append(args, "-mhe=on")
	}

	ctx, cancel := context.WithCancel(task.Ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "7z", args...)

	output, err := cmd.CombinedOutput()

	if task.Status == StatusCancelled {
		return fmt.Errorf("task cancelled")
	}

	if err != nil {
		return fmt.Errorf("7zz zip failed: %v, output: %s", err, string(output))
	}

	task.LocalPath = zipPath
	task.FileName = filepath.Base(zipPath)

	return nil
}

func cleanupTask(task *Task) {
	if task.LocalPath != "" {
		taskDir := filepath.Join(taskManager.DownloadDir, task.ID)
		os.RemoveAll(taskDir)
	}
}

func handleAutoDelete(bot *tgbotapi.BotAPI, task *Task) {
	if settings == nil || !settings.AutoDeleteMessages {
		return
	}

	go func() {
		select {
		case <-task.Ctx.Done():
			return
		case <-context.Background().Done():
			return
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 60*1000*1000*1000)
			defer cancel()
			<-ctx.Done()

			deleteMsg := tgbotapi.NewDeleteMessage(task.ChatID, task.MessageID)
			bot.Request(deleteMsg)
		}
	}()
}
