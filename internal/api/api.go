package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"zee-mirror/handlers"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type APIServer struct {
	Service *handlers.BotService
	Port    int
}

func NewAPIServer(service *handlers.BotService, port int) *APIServer {
	return &APIServer{
		Service: service,
		Port:    port,
	}
}

func (s *APIServer) Start() {
	mux := http.NewServeMux()

	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			expectedKey := os.Getenv("WEB_DASHBOARD_TOKEN")
			if expectedKey == "" {
				expectedKey = "zee-mirror-secret"
			}

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

	mux.Handle("/", http.FileServer(http.Dir("./dist")))

	addr := fmt.Sprintf(":%d", s.Port)
	log.Printf("🌐 Web Dashboard API starting on %s", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("❌ API Server failed: %v", err)
		}
	}()
}

func (s *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, _ := s.Service.DB.GetBotStats()
	usersCount, _ := s.Service.DB.GetUsersCount()
	if stats == nil {
		stats = make(map[string]interface{})
	}
	stats["users_count"] = usersCount
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) handleTasks(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var newSettings struct {
			AutoDeleteMessages bool   `json:"AutoDeleteMessages"`
			DefaultMode        string `json:"DefaultMode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&newSettings); err == nil {
			s.Service.Settings.Mu.Lock()
			s.Service.Settings.AutoDeleteMessages = newSettings.AutoDeleteMessages
			s.Service.Settings.DefaultMode = newSettings.DefaultMode
			s.Service.Settings.Mu.Unlock()

			_ = s.Service.DB.SetSetting("auto_delete_messages", fmt.Sprintf("%v", newSettings.AutoDeleteMessages))
			_ = s.Service.DB.SetSetting("default_mode", newSettings.DefaultMode)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	s.Service.Settings.Mu.RLock()
	defer s.Service.Settings.Mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Service.Settings)
}

func (s *APIServer) handleSystem(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) handleExplorer(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) shouldSkipFile(path, name string) bool {
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

func (s *APIServer) resolveFileStatus(f os.DirEntry, name, fullPath string) (string, string) {
	displayName := name
	status := "folder"
	if !f.IsDir() {
		status = "file"
	}

	if f.IsDir() && (len(name) == 8 || strings.HasPrefix(name, "batch_")) {
		if task := s.Service.TaskManager.GetTask(name); task != nil {
			return task.FileName, "active"
		}

		if tr, err := s.Service.DB.GetTask(name); err == nil {
			return tr.FileName, "finished"
		}

		subPath := filepath.Join(fullPath, name)
		if subFiles, err := os.ReadDir(subPath); err == nil && len(subFiles) > 0 {
			return subFiles[0].Name(), "orphan"
		} else {
			return "", "ignore"
		}
	}
	return displayName, status
}

func (s *APIServer) handleRemoteExplorer(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) handleRemoteLink(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) handleWipeOrphans(w http.ResponseWriter, r *http.Request) {
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

func (s *APIServer) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	weekly, _ := s.Service.DB.GetWeeklyStats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(weekly)
}

func (s *APIServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	logPath := filepath.Join(s.Service.Config.ConfigDir, "zee-mirror.log")
	// #nosec G304
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
