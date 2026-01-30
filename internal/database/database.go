package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func NewDB(configDir string) (*DB, error) {
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(configDir, "zee-mirror.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	instance := &DB{db}
	if err := instance.migrate(); err != nil {
		return nil, err
	}

	return instance, nil
}

func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			username TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			gid TEXT,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			url TEXT,
			file_name TEXT,
			local_path TEXT,
			remote_path TEXT,
			remote_url TEXT,
			total_size INTEGER DEFAULT 0,
			downloaded_size INTEGER DEFAULT 0,
			uploaded_size INTEGER DEFAULT 0,
			chat_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			completed_at DATETIME,
			zip BOOLEAN DEFAULT FALSE,
			unzip BOOLEAN DEFAULT FALSE,
			password TEXT,
			error TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME NOT NULL
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %v", err)
		}
	}

	return nil
}

func (db *DB) GetUser(id int64) (string, string, error) {
	var username, role string
	err := db.QueryRow("SELECT username, role FROM users WHERE id = ?", id).Scan(&username, &role)
	return username, role, err
}

func (db *DB) UpsertUser(id int64, username, role string) error {
	_, err := db.Exec(`
		INSERT INTO users (id, username, role, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			role = CASE WHEN users.role = 'owner' THEN 'owner' ELSE excluded.role END
	`, id, username, role, time.Now())
	return err
}

func (db *DB) SetUserRole(id int64, role string) error {
	_, err := db.Exec("UPDATE users SET role = ? WHERE id = ?", role, id)
	return err
}

func (db *DB) GetAllUsers() ([]map[string]interface{}, error) {
	rows, err := db.Query("SELECT id, username, role FROM users")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var users []map[string]interface{}
	for rows.Next() {
		var id int64
		var username, role string
		if err := rows.Scan(&id, &username, &role); err != nil {
			return nil, err
		}
		users = append(users, map[string]interface{}{
			"id":       id,
			"username": username,
			"role":     role,
		})
	}
	return users, nil
}

type TaskRecord struct {
	ID             string
	GID            string
	Type           string
	Status         string
	URL            string
	FileName       string
	LocalPath      string
	RemotePath     string
	RemoteURL      string
	TotalSize      int64
	DownloadedSize int64
	UploadedSize   int64
	ChatID         int64
	UserID         int64
	CreatedAt      time.Time
	CompletedAt    sql.NullTime
	Zip            bool
	Unzip          bool
	Password       string
	Error          string
}

func (db *DB) SaveTask(t TaskRecord) error {
	_, err := db.Exec(`
		INSERT INTO tasks (
			id, gid, type, status, url, file_name, local_path, remote_path, remote_url,
			total_size, downloaded_size, uploaded_size, chat_id, user_id, created_at,
			completed_at, zip, unzip, password, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			gid = excluded.gid,
			status = excluded.status,
			file_name = excluded.file_name,
			local_path = excluded.local_path,
			remote_path = excluded.remote_path,
			remote_url = excluded.remote_url,
			total_size = excluded.total_size,
			downloaded_size = excluded.downloaded_size,
			uploaded_size = excluded.uploaded_size,
			completed_at = excluded.completed_at,
			error = excluded.error
	`, t.ID, t.GID, t.Type, t.Status, t.URL, t.FileName, t.LocalPath, t.RemotePath, t.RemoteURL,
		t.TotalSize, t.DownloadedSize, t.UploadedSize, t.ChatID, t.UserID, t.CreatedAt,
		t.CompletedAt, t.Zip, t.Unzip, t.Password, t.Error)
	return err
}

func (db *DB) GetActiveTasks() ([]TaskRecord, error) {
	rows, err := db.Query("SELECT * FROM tasks WHERE status NOT IN ('completed', 'failed', 'cancelled')")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tasks []TaskRecord
	for rows.Next() {
		var t TaskRecord
		err := rows.Scan(
			&t.ID, &t.GID, &t.Type, &t.Status, &t.URL, &t.FileName, &t.LocalPath, &t.RemotePath, &t.RemoteURL,
			&t.TotalSize, &t.DownloadedSize, &t.UploadedSize, &t.ChatID, &t.UserID, &t.CreatedAt,
			&t.CompletedAt, &t.Zip, &t.Unzip, &t.Password, &t.Error,
		)
		if err != nil {
			log.Printf("Error scanning task: %v", err)
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (db *DB) GetBotStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalTasks, completedTasks, failedTasks int
	var totalBandwidth int64

	_ = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&totalTasks)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'completed'").Scan(&completedTasks)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'failed'").Scan(&failedTasks)
	_ = db.QueryRow("SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE status = 'completed'").Scan(&totalBandwidth)

	stats["total_tasks"] = totalTasks
	stats["completed_tasks"] = completedTasks
	stats["failed_tasks"] = failedTasks
	stats["total_bandwidth"] = totalBandwidth

	return stats, nil
}

type UserStats struct {
	UserID          int64
	Username        string
	TotalDownloads  int
	TotalUploads    int
	TotalBandwidth  int64
	SuccessfulTasks int
	FailedTasks     int
	LastActive      time.Time
}

func (db *DB) GetUserStats(userID int64) (*UserStats, error) {
	stats := &UserStats{UserID: userID}

	_ = db.QueryRow("SELECT COALESCE(username, '') FROM users WHERE id = ?", userID).Scan(&stats.Username)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ?", userID).Scan(&stats.TotalDownloads)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = 'completed'", userID).Scan(&stats.SuccessfulTasks)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = 'failed'", userID).Scan(&stats.FailedTasks)
	_ = db.QueryRow("SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE user_id = ? AND status = 'completed'", userID).Scan(&stats.TotalBandwidth)
	_ = db.QueryRow("SELECT COALESCE(MAX(created_at), datetime('now')) FROM tasks WHERE user_id = ?", userID).Scan(&stats.LastActive)

	return stats, nil
}

type DailyStats struct {
	Date           time.Time
	TotalTasks     int
	CompletedTasks int
	FailedTasks    int
	TotalBandwidth int64
	AverageSpeed   int64
	PeakConcurrent int
}

func (db *DB) GetTodayStats() (*DailyStats, error) {
	stats := &DailyStats{Date: time.Now()}
	today := time.Now().Format("2006-01-02")

	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = ?", today).Scan(&stats.TotalTasks)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = ? AND status = 'completed'", today).Scan(&stats.CompletedTasks)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = ? AND status = 'failed'", today).Scan(&stats.FailedTasks)
	_ = db.QueryRow("SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE date(created_at) = ? AND status = 'completed'", today).Scan(&stats.TotalBandwidth)

	return stats, nil
}

func (db *DB) GetUserTodayStats(userID int64) (*DailyStats, error) {
	stats := &DailyStats{Date: time.Now()}
	today := time.Now().Format("2006-01-02")

	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ? AND date(created_at) = ?", userID, today).Scan(&stats.TotalTasks)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ? AND date(created_at) = ? AND status = 'completed'", userID, today).Scan(&stats.CompletedTasks)
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE user_id = ? AND date(created_at) = ? AND status = 'failed'", userID, today).Scan(&stats.FailedTasks)
	_ = db.QueryRow("SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE user_id = ? AND date(created_at) = ? AND status = 'completed'", userID, today).Scan(&stats.TotalBandwidth)

	return stats, nil
}

func (db *DB) GetWeeklyStats() ([]DailyStats, error) {
	var stats []DailyStats

	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")

		ds := DailyStats{Date: date}
		_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = ?", dateStr).Scan(&ds.TotalTasks)
		_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = ? AND status = 'completed'", dateStr).Scan(&ds.CompletedTasks)
		_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = ? AND status = 'failed'", dateStr).Scan(&ds.FailedTasks)
		_ = db.QueryRow("SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE date(created_at) = ? AND status = 'completed'", dateStr).Scan(&ds.TotalBandwidth)

		stats = append(stats, ds)
	}

	return stats, nil
}

func (db *DB) GetMonthlyStats() ([]DailyStats, error) {
	var stats []DailyStats

	for i := 29; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")

		ds := DailyStats{Date: date}
		_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = ?", dateStr).Scan(&ds.TotalTasks)
		_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE date(created_at) = ? AND status = 'completed'", dateStr).Scan(&ds.CompletedTasks)
		_ = db.QueryRow("SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE date(created_at) = ? AND status = 'completed'", dateStr).Scan(&ds.TotalBandwidth)

		stats = append(stats, ds)
	}

	return stats, nil
}

func (db *DB) SetSetting(key, value string) error {
	_, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now())
	return err
}

func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	return value, err
}

func (db *DB) GetRecoverableTasks() ([]TaskRecord, error) {
	rows, err := db.Query(`
		SELECT * FROM tasks 
		WHERE status IN ('downloading', 'uploading', 'queued', 'processing')
		AND created_at > datetime('now', '-24 hours')
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tasks []TaskRecord
	for rows.Next() {
		var t TaskRecord
		err := rows.Scan(
			&t.ID, &t.GID, &t.Type, &t.Status, &t.URL, &t.FileName, &t.LocalPath, &t.RemotePath, &t.RemoteURL,
			&t.TotalSize, &t.DownloadedSize, &t.UploadedSize, &t.ChatID, &t.UserID, &t.CreatedAt,
			&t.CompletedAt, &t.Zip, &t.Unzip, &t.Password, &t.Error,
		)
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (db *DB) UpdateTaskStatus(taskID, status, errorMsg string) error {
	_, err := db.Exec("UPDATE tasks SET status = ?, error = ? WHERE id = ?", status, errorMsg, taskID)
	return err
}

func (db *DB) SetTaskRecoverable(taskID string, recoverable bool) error {
	return nil
}

func (db *DB) CleanupOldTasks(before time.Time) (int, error) {
	result, err := db.Exec("DELETE FROM tasks WHERE status IN ('completed', 'failed', 'cancelled') AND created_at < ?", before)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

func (db *DB) GetRecentLogs(limit int) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT id, type, status, file_name, created_at, error 
		FROM tasks 
		ORDER BY created_at DESC 
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var logs []map[string]interface{}
	for rows.Next() {
		var id, taskType, status, fileName, errorStr string
		var createdAt time.Time
		if err := rows.Scan(&id, &taskType, &status, &fileName, &createdAt, &errorStr); err != nil {
			continue
		}

		level := "info"
		message := fmt.Sprintf("[%s] %s - %s", taskType, fileName, status)
		switch status {
		case "failed":
			level = "error"
			message = fmt.Sprintf("[%s] %s - %s: %s", taskType, fileName, status, errorStr)
		case "completed":
			level = "success"
		}

		logs = append(logs, map[string]interface{}{
			"level":     level,
			"message":   message,
			"timestamp": createdAt,
		})
	}
	return logs, nil
}
