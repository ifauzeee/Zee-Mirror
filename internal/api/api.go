package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"zee-mirror/handlers"
	"zee-mirror/internal/domain"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type Server struct {
	Service *handlers.BotService
	Hub     *Hub
	Port    int
}

func NewServer(service *handlers.BotService, port int) *Server {
	return &Server{
		Service: service,
		Port:    port,
		Hub:     NewHub(),
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			expectedKey := s.Service.Config.DashboardToken

			if apiKey != expectedKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
				return
			}
			next(w, r)
		}
	}

	mux.HandleFunc("/api/stats", auth(s.handleStats))
	mux.HandleFunc("/api/tasks", auth(s.handleTasks))
	mux.HandleFunc("/api/settings", auth(s.handleSettings))
	mux.HandleFunc("/api/system", auth(s.handleSystem))
	mux.HandleFunc("/api/explorer", auth(s.handleExplorer))
	mux.HandleFunc("/api/explorer/remote", auth(s.handleRemoteExplorer))
	mux.HandleFunc("/api/explorer/remote/link", auth(s.handleRemoteLink))
	mux.HandleFunc("/api/explorer/wipe-orphans", auth(s.handleWipeOrphans))
	mux.HandleFunc("/api/analytics", auth(s.handleAnalytics))
	mux.HandleFunc("/api/logs", auth(s.handleLogs))
	mux.HandleFunc("/api/users", auth(s.handleGetUsers))
	mux.HandleFunc("/api/users/update", auth(s.handleUpdateUser))
	mux.HandleFunc("/api/users/delete", auth(s.handleDeleteUser))

	mux.HandleFunc("/api/ws", s.handleWebsocket)

	mux.HandleFunc("/api/torrent/session", auth(s.handleTorrentSession))
	mux.HandleFunc("/api/torrent/files", auth(s.handleTorrentFiles))
	mux.HandleFunc("/api/torrent/start", auth(s.handleTorrentStart))

	mux.HandleFunc("/torrent-select/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./dist/index.html")
	})

	mux.Handle("/", http.FileServer(http.Dir("./dist")))

	addr := fmt.Sprintf(":%d", s.Port)
	slog.Info("Web Dashboard API starting", "addr", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	go s.Hub.Run()
	go s.broadcastLoop()

	go func() {
		if err := server.ListenAndServe(); err != nil {
			slog.Error("API Server failed", "error", err)
		}
	}()
}

func (s *Server) broadcastLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		tasks := s.Service.TaskManager.GetActiveTasks()
		var taskSnapshots []interface{}
		for _, t := range tasks {
			taskSnapshots = append(taskSnapshots, t.GetSnapshot())
		}

		v, _ := mem.VirtualMemory()
		c, _ := cpu.Percent(time.Second, false)
		d, _ := disk.Usage("/")
		h, _ := host.Info()
		cpuUsage := 0.0
		if len(c) > 0 {
			cpuUsage = c[0]
		}
		sysInfo := map[string]interface{}{
			"cpu":    cpuUsage,
			"ram":    v.UsedPercent,
			"disk":   d.UsedPercent,
			"uptime": h.Uptime,
		}

		update := map[string]interface{}{
			"type": "update",
			"data": map[string]interface{}{
				"tasks":  taskSnapshots,
				"system": sysInfo,
			},
		}

		payload, err := json.Marshal(update)
		if err == nil {
			s.Hub.Broadcast(payload)
		}
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, _ := s.Service.DB.GetBotStats(ctx)
	usersCount, _ := s.Service.DB.GetCount(ctx)
	if stats == nil {
		stats = make(map[string]interface{})
	}
	stats["users_count"] = usersCount
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		taskID := r.URL.Query().Get("id")
		if taskID != "" {
			s.Service.TaskManager.CancelTask(taskID)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	tasks := s.Service.TaskManager.GetActiveTasks()
	var snapshots []interface{}
	for _, t := range tasks {
		snapshots = append(snapshots, t.GetSnapshot())
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshots)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method == http.MethodPost {
		var settingsUpdate struct {
			DefaultMode        string `json:"DefaultMode"`
			AutoDeleteMessages bool   `json:"AutoDeleteMessages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&settingsUpdate); err != nil {
			http.Error(w, "Invalid settings", http.StatusBadRequest)
			return
		}

		s.Service.Settings.Mu.Lock()
		s.Service.Settings.AutoDeleteMessages = settingsUpdate.AutoDeleteMessages
		s.Service.Settings.DefaultMode = settingsUpdate.DefaultMode
		s.Service.Settings.Mu.Unlock()

		_ = s.Service.SettingsRepo.Set(ctx, "auto_delete_messages", fmt.Sprintf("%v", settingsUpdate.AutoDeleteMessages))
		_ = s.Service.SettingsRepo.Set(ctx, "default_mode", settingsUpdate.DefaultMode)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.Service.Settings.Mu.RLock()
	defer s.Service.Settings.Mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Service.Settings)
}

func (s *Server) handleSystem(w http.ResponseWriter, _ *http.Request) {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Percent(time.Second, false)
	d, _ := disk.Usage("/")
	h, _ := host.Info()

	cpuUsage := 0.0
	if len(c) > 0 {
		cpuUsage = c[0]
	}

	sysInfo := map[string]interface{}{
		"cpu":    cpuUsage,
		"ram":    v.UsedPercent,
		"disk":   d.UsedPercent,
		"uptime": h.Uptime,
		"os":     h.OS,
		"arch":   h.KernelArch,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sysInfo)
}

func (s *Server) handleExplorer(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	fullPath := filepath.Join(s.Service.Config.DownloadDir, path)

	if r.Method == http.MethodDelete {
		if path == "" || path == "/" || path == "." {
			http.Error(w, "Cannot delete root directory", http.StatusBadRequest)
			return
		}
		if err := os.RemoveAll(fullPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	files, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result []map[string]interface{}
	for _, f := range files {
		info, _ := f.Info()
		name := f.Name()

		if s.shouldSkipFile(path, name) {
			continue
		}

		displayName, status := s.resolveFileStatus(f, name, fullPath)
		if status == "ignore" {
			continue
		}

		result = append(result, map[string]interface{}{
			"name":        name,
			"displayName": displayName,
			"isDir":       f.IsDir(),
			"size":        info.Size(),
			"time":        info.ModTime(),
			"status":      status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) shouldSkipFile(path, name string) bool {
	if path == "" {
		if strings.Contains(name, ":") || len(name) > 40 {
			return true
		}
		if strings.HasSuffix(name, ".binlog") || name == "tqueue" || name == "webhooks" || name == "webhooks_db" {
			return true
		}
	}
	return false
}

func (s *Server) resolveFileStatus(f os.DirEntry, name, fullPath string) (string, string) {
	displayName := name
	status := "folder"
	if !f.IsDir() {
		status = "file"
	}

	if f.IsDir() && (len(name) == 8 || strings.HasPrefix(name, "batch_")) {
		if task := s.Service.TaskManager.GetTask(name); task != nil {
			return task.FileName, "active"
		}

		ctx := context.Background()
		if tr, err := s.Service.DB.GetTaskByID(ctx, name); err == nil {
			return tr.FileName, "finished"
		}

		subPath := filepath.Join(fullPath, name)
		if subFiles, errDir := os.ReadDir(subPath); errDir == nil && len(subFiles) > 0 {
			return subFiles[0].Name(), "orphan"
		}
		return "", "ignore"
	}
	return displayName, status
}

func (s *Server) handleRemoteExplorer(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	remotePath := s.Service.Config.RcloneDest
	if path != "" {
		remotePath = filepath.Join(remotePath, path)
	}
	remotePath = filepath.ToSlash(remotePath)

	configPath := filepath.Join(s.Service.Config.ConfigDir, "rclone.conf")

	if r.Method == http.MethodDelete {
		if path == "" || path == "/" || path == "." {
			http.Error(w, "Cannot delete root remote directory", http.StatusBadRequest)
			return
		}
		cmd := exec.Command("rclone", "purge", remotePath, "--config", configPath)
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("rclone", "deletefile", remotePath, "--config", configPath)
			if err := cmd.Run(); err != nil {
				http.Error(w, fmt.Sprintf("Delete failed: %v", err), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	cmd := exec.Command("rclone", "lsjson", remotePath, "--fast-list", "--config", configPath)
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Rclone failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(output)
}

func (s *Server) handleRemoteLink(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	remotePath := s.Service.Config.RcloneDest
	if path != "" {
		remotePath = filepath.Join(remotePath, path)
	}
	remotePath = filepath.ToSlash(remotePath)
	configPath := filepath.Join(s.Service.Config.ConfigDir, "rclone.conf")

	cmd := exec.Command("rclone", "link", remotePath, "--config", configPath)
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get link: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"link": strings.TrimSpace(string(output))})
}

func (s *Server) handleWipeOrphans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	baseDir := s.Service.Config.DownloadDir
	files, err := os.ReadDir(baseDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count := 0
	for _, f := range files {
		name := f.Name()
		if f.IsDir() && (len(name) == 8 || strings.HasPrefix(name, "batch_")) {
			if task := s.Service.TaskManager.GetTask(name); task == nil {
				_ = os.RemoveAll(filepath.Join(baseDir, name))
				count++
			}
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{"wiped": count})
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	weekly, _ := s.Service.DB.GetWeeklyStats(ctx)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(weekly)
}

func (s *Server) handleLogs(w http.ResponseWriter, _ *http.Request) {
	logPath := filepath.Clean(filepath.Join(s.Service.Config.ConfigDir, "zee-mirror.log"))
	if !strings.HasPrefix(logPath, filepath.Clean(s.Service.Config.ConfigDir)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		http.Error(w, "Failed to read logs", http.StatusInternalServerError)
		return
	}

	lines := strings.Split(string(data), "\n")
	start := len(lines) - 101
	if start < 0 {
		start = 0
	}
	lastLines := lines[start:]

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": strings.Join(lastLines, "\n"),
	})
}

func (s *Server) handleTorrentSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	s.Service.TaskManager.Mu.RLock()
	session, exists := s.Service.TaskManager.TorrentSessions[sessionID]
	s.Service.TaskManager.Mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found or expired", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":       sessionID,
		"url":      session.URL,
		"fileName": session.FileName,
		"zip":      session.Zip,
		"unzip":    session.Unzip,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTorrentFiles(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	s.Service.TaskManager.Mu.RLock()
	session, exists := s.Service.TaskManager.TorrentSessions[sessionID]
	s.Service.TaskManager.Mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found or expired", http.StatusNotFound)
		return
	}

	if len(session.Files) > 0 {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"files":   session.Files,
			"loading": false,
		})
		return
	}

	if session.Error != "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"files":   []domain.TorrentFile{},
			"loading": false,
			"error":   session.Error,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"files":    []domain.TorrentFile{},
		"loading":  true,
		"fetching": session.IsFetching,
		"logs":     session.StatusMessages,
		"message":  "Mengambil metadata torrent di background...",
	})
}

func (s *Server) handleTorrentStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		SessionID     string `json:"sessionId"`
		SelectedFiles []int  `json:"selectedFiles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.SessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	err := s.Service.StartTorrentWithSelectedFiles(request.SessionID, request.SelectedFiles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Torrent download started",
	})
}

func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := s.Service.DB.GetAll(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Role              string `json:"role"`
		ExpiresAt         string `json:"expiresAt"`
		ID                int64  `json:"id"`
		MaxDailyBandwidth int64  `json:"maxDailyBandwidth"`
		MaxDailyTasks     int    `json:"maxDailyTasks"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	ctx := r.Context()

	if req.Role != "" {
		if err := s.Service.DB.SetRole(ctx, req.ID, req.Role); err != nil {
			slog.Error("Failed to update role", "error", err)
		}
	}

	if err := s.Service.DB.SetLimits(ctx, req.ID, req.MaxDailyTasks, req.MaxDailyBandwidth); err != nil {
		slog.Error("Failed to update limits", "error", err)
	}

	if req.ExpiresAt != "" {
		if exp, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
			if err := s.Service.DB.SetExpiration(ctx, req.ID, exp); err != nil {
				slog.Error("Failed to update expiration", "error", err)
			}
		} else if exp, err := time.Parse("2006-01-02", req.ExpiresAt); err == nil {
			if err := s.Service.DB.SetExpiration(ctx, req.ID, exp); err != nil {
				slog.Error("Failed to update expiration", "error", err)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID int64 `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if s.Service.IsOwner(req.ID) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Cannot delete owner"})
		return
	}

	ctx := r.Context()
	if err := s.Service.DB.Delete(ctx, req.ID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
