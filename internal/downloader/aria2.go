package downloader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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
	//nolint:gosec
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		return fmt.Errorf("failed to create output dir: %v", err)
	}

	args := e.buildAria2Args(task, outputDir)
	cmd := exec.CommandContext(ctx, "aria2c", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("aria2c failed to start: %v", err)
	}

	go e.parseProgress(stdout, onProgress)

	err = cmd.Wait()
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("aria2c execution failed: %v", err)
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

func (e *Aria2Engine) parseProgress(stdout interface{}, onProgress func(ProgressUpdate)) {
	reader, ok := stdout.(interface {
		Close() error
		Read(p []byte) (n int, err error)
	})
	if !ok {
		return
	}
	defer func() { _ = reader.Close() }()

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		p := parser.ParseAria2Line(line)
		if p.Found {
			onProgress(ProgressUpdate{
				Downloaded:  p.Downloaded,
				Total:       p.Total,
				Speed:       p.Speed,
				Progress:    p.Progress,
				ETA:         p.ETA,
				Connections: p.Connections,
			})
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
