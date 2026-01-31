package downloader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/parser"
)

type YTDLPEngine struct {
	ConfigDir string
}

func NewYTDLPEngine(configDir string) *YTDLPEngine {
	return &YTDLPEngine{
		ConfigDir: configDir,
	}
}

func (e *YTDLPEngine) Download(ctx context.Context, task *domain.Task, outputDir string, onProgress func(ProgressUpdate)) error {
	//nolint:gosec
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		return fmt.Errorf("failed to create output dir: %v", err)
	}

	args := e.buildYTDLPArgs(task, outputDir)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("yt-dlp failed to start: %v", err)
	}

	go e.parseProgress(stdout, onProgress)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("yt-dlp error: %v", err)
	}

	return nil
}

func (e *YTDLPEngine) buildYTDLPArgs(task *domain.Task, outputDir string) []string {
	args := []string{
		"-o", filepath.Join(outputDir, "%(title)s.%(ext)s"),
		"--newline", "--no-playlist",
		"--continue",
		"--merge-output-format", "mp4",
		"--no-check-certificate",
		"--format-sort", "res,fps,codec:vp9,vcodec,br",
		"--extractor-args", "youtube:player-client=web,web_embedded,ios,mweb,tv",
		"--socket-timeout", "60",
		"--concurrent-fragments", "16",
		"--buffer-size", "1M",
		"--add-header", "Accept-Language: en-US,en;q=0.9",
		"--add-header", "Referer: https://www.youtube.com/",
		"--remote-components", "ejs:github",
		"--js-runtime", "node",
		"--cache-dir", "/home/botuser/.cache/yt-dlp-final",
	}

	cookiesPath := filepath.Join(e.ConfigDir, "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		args = append(args, "--cookies", cookiesPath)
		args = append(args, "--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	}

	if task.Quality != "" {
		format := fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", task.Quality, task.Quality)
		args = append(args, "-f", format)
	}

	args = append(args, task.URL)
	return args
}

func (e *YTDLPEngine) parseProgress(stdout interface{}, onProgress func(ProgressUpdate)) {
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
		update := ProgressUpdate{}

		if strings.Contains(line, "ERROR:") {
			update.Error = line
		}

		if strings.HasPrefix(line, "[download] Destination:") {
			title := strings.TrimPrefix(line, "[download] Destination:")
			update.FileName = filepath.Base(strings.TrimSpace(title))
		} else if strings.HasPrefix(line, "[download]") && strings.Contains(line, "has already been downloaded") {
			title := strings.TrimPrefix(line, "[download]")
			title = strings.TrimSuffix(title, " has already been downloaded")
			update.FileName = filepath.Base(strings.TrimSpace(title))
		}

		p := parser.ParseYTDLPLine(line)
		if p.Found {
			update.Progress = p.Progress
			update.Total = p.Total
			update.Downloaded = p.Downloaded
			update.Speed = p.Speed
			update.ETA = p.ETA
			update.Connections = 16
		}

		if update.FileName != "" || update.Progress != 0 || update.Error != "" {
			onProgress(update)
		}
	}
}
