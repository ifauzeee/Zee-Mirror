package downloader

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/parser"
	"zee-mirror/pkg/utils"
)

type YTDLPEngine struct {
	ConfigDir string
}

type VideoFormat struct {
	Height int
	FPS    float64
}

func NewYTDLPEngine(configDir string) *YTDLPEngine {
	return &YTDLPEngine{
		ConfigDir: configDir,
	}
}

func (e *YTDLPEngine) GetFormats(ctx context.Context, url string) (map[int]float64, error) {
	args := []string{
		"-j",
		"--no-playlist",
		"--no-check-certificate",
		"--extractor-args", "youtube:player-client=web,mweb,tv",
		"--socket-timeout", "60",
		"--add-header", "Accept-Language: en-US,en;q=0.9",
		"--add-header", "Referer: https://www.youtube.com/",
		"--js-runtime", "node",
		"--remote-components", "ejs:github",
		"--playlist-items", "0",
	}

	cookiesPath := filepath.Join(e.ConfigDir, "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		args = append(args, "--cookies", cookiesPath)
		args = append(args, "--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	}

	args = append(args, url)
	output, err := e.runYTDLP(ctx, args...)
	if err != nil {
		slog.Warn("YTDLP analysis first attempt failed, retrying with fallback clients...", "error", err)
		for i, arg := range args {
			if arg == "youtube:player-client=web,mweb,tv" {
				args[i] = "youtube:player-client=web,tv"
				break
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		output, err = e.runYTDLP(ctx, args...)
		if err != nil {
			return nil, err
		}
	}

	var data struct {
		Formats []struct {
			FormatID string  `json:"format_id"`
			VCodec   string  `json:"vcodec"`
			FPS      float64 `json:"fps"`
			Height   int     `json:"height"`
		} `json:"formats"`
	}

	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}

	resMap := make(map[int]float64)
	for _, f := range data.Formats {
		if strings.HasPrefix(f.FormatID, "sb") {
			continue
		}
		if f.Height > 0 && f.VCodec != "" && f.VCodec != "none" {
			if f.FPS > resMap[f.Height] {
				resMap[f.Height] = f.FPS
			}
		}
	}
	return resMap, nil
}

func (e *YTDLPEngine) GetPlaylistMetadata(ctx context.Context, url string) (*PlaylistMetadata, error) {
	args := []string{
		"--flat-playlist",
		"-J",
		"--no-check-certificate",
		"--extractor-args", "youtube:player-client=web,mweb,tv",
		"--socket-timeout", "60",
		"--js-runtime", "node",
		"--remote-components", "ejs:github",
	}

	cookiesPath := filepath.Join(e.ConfigDir, "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		args = append(args, "--cookies", cookiesPath)
	}

	output, err := e.runYTDLP(ctx, args...)
	if err != nil {
		slog.Warn("YTDLP playlist first attempt failed, retrying...", "error", err, "url", url)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		output, err = e.runYTDLP(ctx, args...)
		if err != nil {
			return nil, err
		}
	}

	var metadata PlaylistMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		var singleEntry PlaylistEntry
		if errJSON := json.Unmarshal(output, &singleEntry); errJSON == nil && singleEntry.ID != "" {
			return &PlaylistMetadata{
				Title:   singleEntry.Title,
				Entries: []PlaylistEntry{singleEntry},
			}, nil
		}
		return nil, err
	}

	return &metadata, nil
}

func (e *YTDLPEngine) IsPlaylist(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, "playlist") || strings.Contains(lower, "list=") || strings.Contains(lower, "/channel/") || strings.Contains(lower, "/user/") || strings.Contains(lower, "/c/") || strings.Contains(lower, "@")
}

func (e *YTDLPEngine) Download(ctx context.Context, task *domain.Task, outputDir string, onProgress func(ProgressUpdate)) error {
	if errDir := os.MkdirAll(outputDir, 0750); errDir != nil {
		return &domain.StorageError{Path: outputDir, Err: errDir}
	}

	args := e.buildYTDLPArgs(task, outputDir)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &domain.NetworkError{URL: task.URL, Err: fmt.Errorf("failed to get stdout pipe: %v", err)}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return &domain.NetworkError{URL: task.URL, Err: fmt.Errorf("failed to get stderr pipe: %v", err)}
	}

	if err := cmd.Start(); err != nil {
		return &domain.NetworkError{URL: task.URL, Err: fmt.Errorf("yt-dlp failed to start: %v", err)}
	}

	errorOutput := ""
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "ERROR:") || strings.Contains(line, "WARNING:") {
				errorOutput += line + "\n"
			}
			if strings.Contains(line, "ERROR:") {
				onProgress(ProgressUpdate{Error: line})
			}
		}
	}()

	go e.parseProgress(stdout, onProgress)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		if errorOutput != "" {
			lines := strings.Split(errorOutput, "\n")
			var errors []string
			for _, l := range lines {
				if strings.Contains(l, "ERROR:") {
					errors = append(errors, l)
				}
			}
			if len(errors) > 0 {
				errorMsg := fmt.Errorf("yt-dlp failed: %s", strings.Join(errors, "\n"))
				return domain.CategorizeError(errorMsg)
			}
			errorMsg := fmt.Errorf("yt-dlp failed: %s", errorOutput)
			return domain.CategorizeError(errorMsg)
		}
		return domain.CategorizeError(fmt.Errorf("yt-dlp error: %v", err))
	}

	if task.Hardsub && task.SubtitleLangs != "" {
		onProgress(ProgressUpdate{Message: "Burning subtitles..."})
		if err := e.burnSubtitles(ctx, outputDir); err != nil {
			slog.Error("Failed to burn subtitles", "error", err)

			onProgress(ProgressUpdate{Error: "Hardsub failed: " + err.Error()})
		} else {
			onProgress(ProgressUpdate{Message: "Hardsubbing completed!"})
		}
	}

	return nil
}

func (e *YTDLPEngine) burnSubtitles(ctx context.Context, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var videoFile, subFile string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".mp4", ".mkv", ".webm":
			videoFile = filepath.Join(dir, name)
		case ".srt", ".vtt", ".ass":
			subFile = filepath.Join(dir, name)
		}
	}

	if videoFile == "" || subFile == "" {
		return fmt.Errorf("video or subtitle file not found for burning")
	}

	outputFile := strings.TrimSuffix(videoFile, filepath.Ext(videoFile)) + "_hardsub.mp4"

	escapedSubFile := strings.ReplaceAll(subFile, "\\", "/")
	escapedSubFile = strings.ReplaceAll(escapedSubFile, ":", "\\:")

	args := []string{
		"-i", videoFile,
		"-vf", fmt.Sprintf("subtitles='%s'", escapedSubFile),
		"-c:a", "copy",
		"-y",
		outputFile,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %v, output: %s", err, string(out))
	}

	_ = os.Remove(videoFile)
	_ = os.Remove(subFile)
	return os.Rename(outputFile, videoFile)
}

func (e *YTDLPEngine) buildYTDLPArgs(task *domain.Task, outputDir string) []string {
	outputTemplate := "%(title)s.%(ext)s"
	if task.FileName != "" && task.FileName != "video" && task.FileName != "unknown_file" && task.FileName != "watch" {
		ext := filepath.Ext(task.FileName)
		if ext != "" {
			if strings.ToLower(ext) == ".m3u8" {
				outputTemplate = strings.TrimSuffix(task.FileName, ext) + ".mp4"
			} else {
				outputTemplate = task.FileName
			}
		} else {
			outputTemplate = task.FileName + ".%(ext)s"
		}
	}

	args := []string{
		"-o", filepath.Join(outputDir, outputTemplate),
		"--newline", "--no-playlist",
		"--continue",
		"--merge-output-format", "mp4",
		"--no-check-certificate",
		"--ignore-errors",
		"--format-sort", "res,fps,codec:vp9,vcodec,br",
		"--extractor-args", "youtube:player-client=web,mweb,tv",
		"--socket-timeout", "120",
		"--concurrent-fragments", "16",
		"--buffer-size", "1M",
		"--add-header", "Accept-Language: en-US,en;q=0.9",
		"--add-header", "Referer: https://www.youtube.com/",
		"--js-runtime", "node",
		"--remote-components", "ejs:github",
		"--no-cache-dir",
	}

	if task.SubtitleLangs != "" {
		args = append(args, "--write-subs", "--write-auto-subs")
		args = append(args, "--sub-langs", task.SubtitleLangs)
		if !task.Hardsub {
			args = append(args, "--embed-subs")
		}
	}

	cookiesPath := filepath.Join(e.ConfigDir, "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		args = append(args, "--cookies", cookiesPath)
		args = append(args, "--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	}

	if task.Quality != "" {
		if task.Quality == "audio" {
			args = append(args, "-f", "bestaudio/best")
			args = append(args, "-x", "--audio-format", "mp3", "--audio-quality", "0")
		} else {
			format := fmt.Sprintf("bestvideo[height<=?%[1]s]+bestaudio/best[height<=?%[1]s]/best[height<=?%[1]s]", task.Quality)
			args = append(args, "-f", format)
		}
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
	scanner.Split(utils.ScanLinesWithCR)
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
func (e *YTDLPEngine) runYTDLP(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderrStr))
		}
		return nil, err
	}
	return []byte(stdout.String()), nil
}
