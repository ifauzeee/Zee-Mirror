package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"zee-mirror/internal/domain"
	"zee-mirror/internal/repository"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

var _ repository.TaskRepository = (*DB)(nil)
var _ repository.UserRepository = (*DB)(nil)
var _ repository.SettingsRepository = (*DB)(nil)

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

func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			username TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME NOT NULL,
			max_daily_tasks INTEGER DEFAULT -1,
			max_daily_bandwidth INTEGER DEFAULT -1,
			expires_at DATETIME
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

	_ = db.addColumnIfNotExists("users", "max_daily_tasks", "INTEGER DEFAULT -1")
	_ = db.addColumnIfNotExists("users", "max_daily_bandwidth", "INTEGER DEFAULT -1")
	_ = db.addColumnIfNotExists("users", "expires_at", "DATETIME")

	return nil
}

func (db *DB) addColumnIfNotExists(table, column, definition string) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

func (db *DB) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	u := &domain.User{ID: id}
	var expiresAt sql.NullTime

	err := db.QueryRowContext(ctx, `
		SELECT username, role, max_daily_tasks, max_daily_bandwidth, expires_at, created_at 
		FROM users WHERE id = ?
	`, id).Scan(&u.Username, &u.Role, &u.MaxDailyTasks, &u.MaxDailyBandwidth, &expiresAt, &u.CreatedAt)

	if err != nil {
		return nil, err
	}
	u.ExpiresAt = expiresAt

	u.IsActive = true
	if u.ExpiresAt.Valid && u.ExpiresAt.Time.Before(time.Now()) {
		u.IsActive = false
	}

	return u, nil
}

func (db *DB) Upsert(ctx context.Context, u domain.User) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, role, created_at, max_daily_tasks, max_daily_bandwidth, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			role = CASE WHEN users.role = 'owner' THEN 'owner' ELSE excluded.role END,
			max_daily_tasks = excluded.max_daily_tasks,
			max_daily_bandwidth = excluded.max_daily_bandwidth,
			expires_at = excluded.expires_at
	`, u.ID, u.Username, u.Role, u.CreatedAt, u.MaxDailyTasks, u.MaxDailyBandwidth, u.ExpiresAt)
	return err
}

func (db *DB) SetRole(ctx context.Context, id int64, role string) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET role = ? WHERE id = ?", role, id)
	return err
}

func (db *DB) SetLimits(ctx context.Context, id int64, maxTasks int, maxBandwidth int64) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET max_daily_tasks = ?, max_daily_bandwidth = ? WHERE id = ?", maxTasks, maxBandwidth, id)
	return err
}

func (db *DB) SetExpiration(ctx context.Context, id int64, expiresAt time.Time) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET expires_at = ? WHERE id = ?", expiresAt, id)
	return err
}

func (db *DB) GetAll(ctx context.Context) ([]domain.User, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, username, role, max_daily_tasks, max_daily_bandwidth, expires_at, created_at FROM users")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var expiresAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.MaxDailyTasks, &u.MaxDailyBandwidth, &expiresAt, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.ExpiresAt = expiresAt
		u.IsActive = true
		if u.ExpiresAt.Valid && u.ExpiresAt.Time.Before(time.Now()) {
			u.IsActive = false
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *DB) GetCount(ctx context.Context) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (db *DB) Delete(ctx context.Context, id int64) error {
	_, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}

type TaskRecord = domain.TaskRecord

func (db *DB) GetCompletedTaskByURL(ctx context.Context, url string) (*TaskRecord, error) {
	tr := &TaskRecord{}
	err := db.QueryRowContext(ctx, `
		SELECT id, gid, type, status, url, file_name, local_path, remote_path, remote_url, 
		       total_size, downloaded_size, uploaded_size, chat_id, user_id, 
		       created_at, completed_at, zip, unzip, password, error
		FROM tasks WHERE url = ? AND status = 'completed'
		ORDER BY created_at DESC LIMIT 1
	`, url).Scan(
		&tr.ID, &tr.GID, &tr.Type, &tr.Status, &tr.URL, &tr.FileName, &tr.LocalPath, &tr.RemotePath, &tr.RemoteURL,
		&tr.TotalSize, &tr.DownloadedSize, &tr.UploadedSize, &tr.ChatID, &tr.UserID,
		&tr.CreatedAt, &tr.CompletedAt, &tr.Zip, &tr.Unzip, &tr.Password, &tr.Error,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return tr, nil
}

func (db *DB) Save(ctx context.Context, t TaskRecord) error {
	_, err := db.ExecContext(ctx, `
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

func (db *DB) GetActive(ctx context.Context) ([]TaskRecord, error) {
	rows, err := db.QueryContext(ctx, "SELECT * FROM tasks WHERE status NOT IN ('completed', 'failed', 'cancelled')")
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
			slog.Error("Error scanning task", "error", err)
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (db *DB) GetBotStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalTasks, completedTasks, failedTasks int
	var totalBandwidth int64

	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks").Scan(&totalTasks)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE status = 'completed'").Scan(&completedTasks)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE status = 'failed'").Scan(&failedTasks)
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE status = 'completed'").Scan(&totalBandwidth)

	stats["total_tasks"] = totalTasks
	stats["completed_tasks"] = completedTasks
	stats["failed_tasks"] = failedTasks
	stats["total_bandwidth"] = totalBandwidth

	return stats, nil
}

type UserStats = domain.UserStats

func (db *DB) GetTaskByID(ctx context.Context, id string) (*TaskRecord, error) {
	tr := &TaskRecord{}
	err := db.QueryRowContext(ctx, `
		SELECT id, gid, type, status, url, file_name, local_path, remote_path, remote_url, 
		       total_size, downloaded_size, uploaded_size, chat_id, user_id, 
		       created_at, completed_at, zip, unzip, password, error
		FROM tasks WHERE id = ?
	`, id).Scan(
		&tr.ID, &tr.GID, &tr.Type, &tr.Status, &tr.URL, &tr.FileName, &tr.LocalPath, &tr.RemotePath, &tr.RemoteURL,
		&tr.TotalSize, &tr.DownloadedSize, &tr.UploadedSize, &tr.ChatID, &tr.UserID,
		&tr.CreatedAt, &tr.CompletedAt, &tr.Zip, &tr.Unzip, &tr.Password, &tr.Error,
	)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (db *DB) GetUserStats(ctx context.Context, userID int64) (*UserStats, error) {
	stats := &UserStats{UserID: userID}

	_ = db.QueryRowContext(ctx, "SELECT COALESCE(username, '') FROM users WHERE id = ?", userID).Scan(&stats.Username)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = ?", userID).Scan(&stats.TotalDownloads)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = 'completed'", userID).Scan(&stats.SuccessfulTasks)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = 'failed'", userID).Scan(&stats.FailedTasks)
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE user_id = ? AND status = 'completed'", userID).Scan(&stats.TotalBandwidth)
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(MAX(created_at), datetime('now')) FROM tasks WHERE user_id = ?", userID).Scan(&stats.LastActive)

	return stats, nil
}

type DailyStats = domain.DailyStats

func (db *DB) GetTodayStats(ctx context.Context) (*DailyStats, error) {
	stats := &DailyStats{Date: time.Now()}
	today := time.Now().Format("2006-01-02")

	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at LIKE ?", today+"%").Scan(&stats.TotalTasks)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at LIKE ? AND status = 'completed'", today+"%").Scan(&stats.CompletedTasks)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at LIKE ? AND status = 'failed'", today+"%").Scan(&stats.FailedTasks)
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE created_at LIKE ? AND status = 'completed'", today+"%").Scan(&stats.TotalBandwidth)

	return stats, nil
}

func (db *DB) GetUserTodayStats(ctx context.Context, userID int64) (*DailyStats, error) {
	stats := &DailyStats{Date: time.Now()}
	today := time.Now().Format("2006-01-02")

	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = ? AND created_at LIKE ?", userID, today+"%").Scan(&stats.TotalTasks)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = ? AND created_at LIKE ? AND status = 'completed'", userID, today+"%").Scan(&stats.CompletedTasks)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = ? AND created_at LIKE ? AND status = 'failed'", userID, today+"%").Scan(&stats.FailedTasks)
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE user_id = ? AND created_at LIKE ? AND status = 'completed'", userID, today+"%").Scan(&stats.TotalBandwidth)

	return stats, nil
}

func (db *DB) GetWeeklyStats(ctx context.Context) ([]DailyStats, error) {
	var stats []DailyStats

	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")

		ds := DailyStats{Date: date}
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at LIKE ?", dateStr+"%").Scan(&ds.TotalTasks)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at LIKE ? AND status = 'completed'", dateStr+"%").Scan(&ds.CompletedTasks)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at LIKE ? AND status = 'failed'", dateStr+"%").Scan(&ds.FailedTasks)
		_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE created_at LIKE ? AND status = 'completed'", dateStr+"%").Scan(&ds.TotalBandwidth)

		stats = append(stats, ds)
	}

	return stats, nil
}

func (db *DB) GetMonthlyStats(ctx context.Context) ([]DailyStats, error) {
	var stats []DailyStats

	for i := 29; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")

		ds := DailyStats{Date: date}
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at LIKE ?", dateStr+"%").Scan(&ds.TotalTasks)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at LIKE ? AND status = 'completed'", dateStr+"%").Scan(&ds.CompletedTasks)
		_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE created_at LIKE ? AND status = 'completed'", dateStr+"%").Scan(&ds.TotalBandwidth)

		stats = append(stats, ds)
	}

	return stats, nil
}

func (db *DB) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	return value, err
}

func (db *DB) Set(ctx context.Context, key, value string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now())
	return err
}

func (db *DB) GetRecoverable(ctx context.Context) ([]TaskRecord, error) {
	rows, err := db.QueryContext(ctx, `
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

func (db *DB) UpdateStatus(ctx context.Context, taskID, status, errorMsg string) error {
	_, err := db.ExecContext(ctx, "UPDATE tasks SET status = ?, error = ? WHERE id = ?", status, errorMsg, taskID)
	return err
}

func (db *DB) SetTaskRecoverable(ctx context.Context, taskID string, recoverable bool) error {
	return nil
}

func (db *DB) DeleteOld(ctx context.Context, before string) (int, error) {
	result, err := db.ExecContext(ctx, "DELETE FROM tasks WHERE status IN ('completed', 'failed', 'cancelled') AND created_at < ?", before)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

func (db *DB) GetRecentLogs(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
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
