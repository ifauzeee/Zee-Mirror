package downloader

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/parser"
	"zee-mirror/pkg/utils"
)

type Aria2Engine struct {
	ConfigDir string
}

func NewAria2Engine(configDir string) *Aria2Engine {
	return &Aria2Engine{
		ConfigDir: configDir,
	}
}

func (e *Aria2Engine) Download(ctx context.Context, task *domain.Task, outputDir string, onProgress func(ProgressUpdate)) error {
	if errDir := os.MkdirAll(outputDir, 0750); errDir != nil {
		return &domain.StorageError{Path: outputDir, Err: errDir}
	}

	args := e.buildAria2Args(task, outputDir)
	cmd := exec.CommandContext(ctx, "aria2c", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &domain.NetworkError{URL: task.URL, Err: fmt.Errorf("failed to get stdout pipe: %v", err)}
	}

	if errStart := cmd.Start(); errStart != nil {
		return &domain.NetworkError{URL: task.URL, Err: fmt.Errorf("aria2c failed to start: %v", errStart)}
	}

	var lastLines []string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.parseProgress(stdout, onProgress, &lastLines)
	}()

	err = cmd.Wait()
	wg.Wait()

	if err != nil && ctx.Err() == nil {
		var errorDetails []string
		for _, line := range lastLines {
			if strings.Contains(line, "ERROR") ||
				strings.Contains(line, "Exception") ||
				strings.Contains(line, "ErrorCode") ||
				strings.Contains(line, "status=") {
				errorDetails = append(errorDetails, strings.TrimSpace(line))
			}
		}

		if len(errorDetails) > 0 {
			detailErr := fmt.Errorf("aria2c failed: %v, details: %s", err, strings.Join(errorDetails, " | "))
			return domain.CategorizeError(detailErr)
		}

		tail := ""
		if len(lastLines) > 0 {
			start := 0
			if len(lastLines) > 3 {
				start = len(lastLines) - 3
			}
			tail = strings.Join(lastLines[start:], " | ")
		}
		combinedErr := fmt.Errorf("aria2c failed: %v, stderr: %s, tail: %s", err, stderr.String(), tail)
		return domain.CategorizeError(combinedErr)
	}

	return nil
}

func (e *Aria2Engine) buildAria2Args(task *domain.Task, outputDir string) []string {
	args := []string{
		"--dir=" + outputDir,
		"--allow-overwrite=true",
		"--max-connection-per-server=16",
		"--split=32",
		"--min-split-size=1M",
		"--max-overall-download-limit=0",
		"--max-resume-failure-tries=0",
		"--retry-wait=1",
		"--connect-timeout=30",
		"--timeout=30",
		"--console-log-level=notice",
		"--summary-interval=1",
		"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"--header=Accept-Encoding: gzip, deflate",
		"--async-dns=true",
		"--file-allocation=none",
		"--disk-cache=128M",
		"--enable-mmap=true",
		"--check-certificate=false",
		"--optimize-concurrent-downloads=true",
		"--max-file-not-found=2",
		"--disable-ipv6=true",
		"--enable-http-pipelining=true",
		"--peer-id-prefix=-AZ2060-",
		"--peer-agent=Transmission/2.94",
		"--content-disposition-default-utf8=true",
		"--remote-time=true",
		"--check-integrity=true",
	}

	cookiesPath := filepath.Join(e.ConfigDir, "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		args = append(args, "--load-cookies="+cookiesPath)
	}

	if task.FileName != "" && task.FileName != "unknown_file" && !isGenericName(task.FileName) {
		args = append(args, "--out="+task.FileName)
	}

	url := task.URL
	selectFiles := ""
	if idx := strings.Index(url, "#select="); idx != -1 {
		selectFiles = url[idx+8:]
		url = url[:idx]
	}

	args = append(args, url)

	if selectFiles != "" {
		args = append(args, "--select-file="+selectFiles)
	}

	if utils.IsMagnetLink(url) {
		args = append(args, "--seed-time=0")
	}

	if task.Type == domain.TypeTorrent && len(url) > 7 && url[:7] == "file://" {
		localPath := url[7:]
		for i, arg := range args {
			if arg == url {
				args[i] = localPath
				break
			}
		}
		args = append(args, "--seed-time=0")
	}

	return args
}

func (e *Aria2Engine) parseProgress(reader io.ReadCloser, onProgress func(ProgressUpdate), lastLines *[]string) {
	defer func() { _ = reader.Close() }()

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		if lastLines != nil {
			*lastLines = append(*lastLines, line)
			if len(*lastLines) > 15 {
				*lastLines = (*lastLines)[1:]
			}
		}

		update := ProgressUpdate{}

		if strings.Contains(line, "ERROR") || strings.Contains(line, "Exception") || strings.Contains(line, "ErrorCode") {
			update.Error = line
		}

		p := parser.ParseAria2Line(line)
		if p.Found {
			update.Downloaded = p.Downloaded
			update.Total = p.Total
			update.Speed = p.Speed
			update.Progress = p.Progress
			update.ETA = p.ETA
			update.Connections = p.Connections
		}

		if update.Found() || update.Error != "" {
			onProgress(update)
		}
	}
}

func isGenericName(name string) bool {
	uuidRegex := regexp.MustCompile(`^[a-fA-F0-9]{8}(-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}$`)
	if uuidRegex.MatchString(name) {
		return true
	}

	hexRegex := regexp.MustCompile(`^[a-fA-F0-9]{16,}$`)
	return hexRegex.MatchString(name)
}
