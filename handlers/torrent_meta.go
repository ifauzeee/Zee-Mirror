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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	trackers := []string{
		"http://tracker.opentrackr.org:1337/announce",
		"udp://tracker.openbittorrent.com:80/announce",
		"udp://tracker.coppersurfer.tk:6969/announce",
		"udp://9.rarbg.to:2710/announce",
		"udp://tracker.leechers-paradise.org:6969/announce",
		"udp://explodie.org:6969/announce",
		"udp://p4p.arenabg.com:1337/announce",
	}

	args := []string{
		"--dir=" + tmpDir,
		"--seed-time=0",
		"--quiet=true",
		"--show-files=true",
		"--bt-stop-timeout=60",
		"--dht-entry-point=dht.transmissionbt.com:6881",
		"--dht-entry-point=router.bittorrent.com:6881",
		"--dht-entry-point=router.utorrent.com:6881",
	}

	if strings.HasPrefix(session.URL, "magnet:") {
		args = append(args, "--bt-metadata-only=true", "--bt-save-metadata=true")
		for _, t := range trackers {
			args = append(args, "--bt-tracker="+t)
		}
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, "aria2c", args...)
	output, _ := cmd.CombinedOutput()

	files := parseTorrentOutput(string(output))

	if len(files) == 0 {
		metaFiles, _ := filepath.Glob(filepath.Join(tmpDir, "*.torrent"))
		for _, metaFile := range metaFiles {
			files = s.parseTorrentMetadataFile(metaFile)
			if len(files) > 0 {
				break
			}
		}
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
		slog.Info("Successfully fetched metadata in background", "sessionID", sessionID, "fileCount", len(files))
	} else {
		session.Error = "Metadata fetching timed out or failed. Please try again or use 'Select All'."
		slog.Warn("Failed to fetch metadata in background", "sessionID", sessionID)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "aria2c", "--show-files", torrentPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	return parseTorrentOutput(string(output))
}

func parseTorrentOutput(output string) []domain.TorrentFile {
	var files []domain.TorrentFile
	lines := strings.Split(output, "\n")

	filePattern := regexp.MustCompile(`^(\d+)\|(.+?)\|(\d+)\|`)

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
		}
	}

	return files
}
