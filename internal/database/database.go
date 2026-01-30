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
