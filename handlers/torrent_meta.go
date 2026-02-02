package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"zee-mirror/internal/domain"
)

func (s *BotService) fetchTorrentMetadataBackground(sessionID string) {
	s.TaskManager.Mu.RLock()
	session, exists := s.TaskManager.TorrentSessions[sessionID]
	s.TaskManager.Mu.RUnlock()

	if !exists {
		return
	}

	s.TaskManager.Mu.Lock()
	session.IsFetching = true
	s.TaskManager.Mu.Unlock()

	slog.Info("Starting background metadata fetch", "sessionID", sessionID, "url", session.URL)

	tmpDir, err := os.MkdirTemp("", "torrent_meta_"+sessionID+"_")
	if err != nil {
		s.updateSessionError(sessionID, fmt.Sprintf("failed to create temp dir: %v", err))
		return
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	url := strings.TrimPrefix(session.URL, "file://")
	isMagnet := strings.HasPrefix(session.URL, "magnet:")

	var metaFilePath string

	if isMagnet {
		fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer fetchCancel()

		args := []string{
			"--dir=" + tmpDir,
			"--seed-time=0",
			"--bt-stop-timeout=0",
			"--bt-save-metadata=true",
			"--bt-metadata-only=true",
			"--enable-dht=true",
			"--enable-peer-exchange=true",
			"--dht-entry-point=dht.transmissionbt.com:6881",
			"--dht-entry-point=router.bittorrent.com:6881",
			"--summary-interval=10",
			url,
		}

		cmd := exec.CommandContext(fetchCtx, "aria2c", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.Warn("Aria2c metadata fetch finished with error", "sessionID", sessionID, "error", err)
		}

		metaFiles, _ := filepath.Glob(filepath.Join(tmpDir, "*.torrent"))
		if len(metaFiles) > 0 {
			metaFilePath = metaFiles[0]
			slog.Info("Found downloaded metadata file", "sessionID", sessionID, "path", metaFilePath)
		} else {
			slog.Warn("No .torrent file found after aria2c run", "sessionID", sessionID, "output", string(output))
		}
	} else {
		metaFilePath = url
		slog.Info("Using local torrent file", "sessionID", sessionID, "path", metaFilePath)
	}

	var files []domain.TorrentFile
	if metaFilePath != "" {
		files = s.parseTorrentMetadataFile(metaFilePath)
	}

	s.TaskManager.Mu.Lock()
	defer s.TaskManager.Mu.Unlock()

	session, exists = s.TaskManager.TorrentSessions[sessionID]
	if !exists {
		return
	}

	session.IsFetching = false
	if len(files) > 0 {
		session.Files = files
		session.Error = ""
		slog.Info("Successfully fetched torrent metadata", "sessionID", sessionID, "files", len(files))
	} else {
		session.Files = nil
		session.Error = "Gagal mengambil daftar file. Torrent mungkin tidak memiliki seeder atau link tidak valid."
		slog.Warn("Failed to parse torrent metadata", "sessionID", sessionID)
	}
}

func (s *BotService) updateSessionError(sessionID, errMsg string) {
	s.TaskManager.Mu.Lock()
	defer s.TaskManager.Mu.Unlock()
	if session, exists := s.TaskManager.TorrentSessions[sessionID]; exists {
		session.IsFetching = false
		session.Error = errMsg
	}
}

func (s *BotService) parseTorrentMetadataFile(torrentPath string) []domain.TorrentFile {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slog.Info("Parsing torrent file", "path", torrentPath)

	if _, err := os.Stat(torrentPath); os.IsNotExist(err) {
		slog.Error("Torrent file does not exist", "path", torrentPath)
		return nil
	}

	cmd := exec.CommandContext(ctx, "aria2c", "--show-files=true", torrentPath)
	output, err := cmd.CombinedOutput()

	cleanOutput := stripANSI(string(output))

	slog.Debug("aria2c --show-files output", "output", cleanOutput)

	if err != nil {
		slog.Error("aria2c --show-files failed", "path", torrentPath, "error", err, "output", cleanOutput)
		return nil
	}

	files := parseTorrentOutput(cleanOutput)
	slog.Info("Parsed torrent files", "count", len(files))

	return files
}

func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}

func parseTorrentOutput(output string) []domain.TorrentFile {
	var files []domain.TorrentFile
	lines := strings.Split(output, "\n")

	filePattern := regexp.MustCompile(`^\s*(\d+)\|(.+?)\|(\d+)\|`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := filePattern.FindStringSubmatch(line); len(matches) >= 4 {
			idx, _ := strconv.Atoi(matches[1])
			size, _ := strconv.ParseInt(matches[3], 10, 64)
			path := matches[2]
			name := filepath.Base(path)

			files = append(files, domain.TorrentFile{
				Index: idx,
				Name:  name,
				Path:  path,
				Size:  size,
			})
			slog.Debug("Parsed file", "index", idx, "name", name, "size", size)
		}
	}

	return files
}
