package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zee-mirror/internal/domain"
	"zee-mirror/internal/repository"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver
)

type DB struct {
	*sql.DB
	driver string
}

var _ repository.TaskRepository = (*DB)(nil)
var _ repository.UserRepository = (*DB)(nil)
var _ repository.SettingsRepository = (*DB)(nil)
var _ repository.ScheduledTaskRepository = (*DB)(nil)

func NewDB(driverName, configDir, dsn, migrationsDir string) (*DB, error) {
	switch driverName {
	case "postgres":
		return newPostgresDB(dsn, migrationsDir)
	default:
		return newSQLiteDB(configDir, migrationsDir)
	}
}

func newSQLiteDB(configDir, migrationsDir string) (*DB, error) {
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(configDir, "zee-mirror.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	instance := &DB{DB: db, driver: "sqlite"}
	if err := instance.RunMigrations(migrationsDir); err != nil {
		slog.Error("Database migration failed", "error", err)
		return nil, err
	}

	return instance, nil
}

func newPostgresDB(dsn, migrationsDir string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is required for postgres driver")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	instance := &DB{DB: db, driver: "postgres"}
	if err := instance.RunMigrations(migrationsDir); err != nil {
		slog.Error("Database migration failed", "error", err)
		return nil, err
	}

	return instance, nil
}

func (db *DB) nowExpr() string {
	if db.driver == "postgres" {
		return "NOW()"
	}
	return "datetime('now')"
}

func (db *DB) nowMinusExpr(duration string) string {
	if db.driver == "postgres" {
		return fmt.Sprintf("NOW() - INTERVAL '%s'", duration)
	}
	return fmt.Sprintf("datetime('now', '-%s')", duration)
}

func (db *DB) RunMigrations(migrationsDir string) error {
	var driver database.Driver
	var err error

	switch db.driver {
	case "postgres":
		driver, err = postgres.WithInstance(db.DB, &postgres.Config{})
		if err != nil {
			return fmt.Errorf("failed to create postgres migrate driver: %w", err)
		}
	default:
		driver, err = sqlite.WithInstance(db.DB, &sqlite.Config{})
		if err != nil {
			return fmt.Errorf("failed to create sqlite migrate driver: %w", err)
		}
	}

	migrateDir := migrationsDir
	if db.driver == "postgres" {
		migrateDir = filepath.Join(migrationsDir, "postgres")
		absMigrateDir, absErr := filepath.Abs(migrateDir)
		if absErr == nil {
			migrateDir = strings.ReplaceAll(absMigrateDir, "\\", "/")
		}
		if _, statErr := os.Stat(migrateDir); os.IsNotExist(statErr) {
			return fmt.Errorf("postgres migrations directory not found: %s", migrateDir)
		}
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrateDir,
		db.driver, driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		if strings.Contains(err.Error(), "Dirty database version 1") {
			slog.Warn("Database is dirty at version 1, attempting to force version 1 and retry...")
			if errForce := m.Force(1); errForce != nil {
				return fmt.Errorf("failed to force migration: %w", errForce)
			}
			if errRetry := m.Up(); errRetry != nil && errRetry != migrate.ErrNoChange {
				return fmt.Errorf("migration failed after force: %w", errRetry)
			}
			slog.Info("Database migration fixed and completed successfully")
			return nil
		}
		return err
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

func (db *DB) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	u := &domain.User{ID: id}
	var expiresAt sql.NullTime

	err := db.QueryRowContext(ctx, `
		SELECT username, role, language, max_daily_tasks, max_daily_bandwidth, expires_at, created_at 
		FROM users WHERE id = $1
	`, id).Scan(&u.Username, &u.Role, &u.Language, &u.MaxDailyTasks, &u.MaxDailyBandwidth, &expiresAt, &u.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
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
		INSERT INTO users (id, username, role, language, created_at, max_daily_tasks, max_daily_bandwidth, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			role = CASE WHEN users.role = 'owner' THEN 'owner' ELSE excluded.role END,
			language = excluded.language,
			max_daily_tasks = excluded.max_daily_tasks,
			max_daily_bandwidth = excluded.max_daily_bandwidth,
			expires_at = excluded.expires_at
	`, u.ID, u.Username, u.Role, u.Language, u.CreatedAt, u.MaxDailyTasks, u.MaxDailyBandwidth, u.ExpiresAt)
	return err
}

func (db *DB) SetRole(ctx context.Context, id int64, role string) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET role = $1 WHERE id = $2", role, id)
	return err
}

func (db *DB) SetLimits(ctx context.Context, id int64, maxTasks int, maxBandwidth int64) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET max_daily_tasks = $1, max_daily_bandwidth = $2 WHERE id = $3", maxTasks, maxBandwidth, id)
	return err
}

func (db *DB) SetExpiration(ctx context.Context, id int64, expiresAt time.Time) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET expires_at = $1 WHERE id = $2", expiresAt, id)
	return err
}

func (db *DB) SetLanguage(ctx context.Context, id int64, lang string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, language, created_at, role, max_daily_tasks, max_daily_bandwidth)
		VALUES ($1, $2, $3, 'user', 0, 0)
		ON CONFLICT(id) DO UPDATE SET language = excluded.language
	`, id, lang, time.Now())
	return err
}

func (db *DB) GetAll(ctx context.Context) ([]domain.User, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, username, role, language, max_daily_tasks, max_daily_bandwidth, expires_at, created_at FROM users")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var expiresAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Language, &u.MaxDailyTasks, &u.MaxDailyBandwidth, &expiresAt, &u.CreatedAt); err != nil {
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
	_, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

type TaskRecord = domain.TaskRecord

const taskColumns = "id, gid, type, status, url, file_name, local_path, remote_path, remote_url, total_size, downloaded_size, uploaded_size, chat_id, user_id, created_at, completed_at, zip, unzip, password, error, retries, quality, md5"

func (db *DB) ListTasks(ctx context.Context, filter domain.TaskFilter) ([]TaskRecord, error) {
	query := "SELECT " + taskColumns + " FROM tasks WHERE 1=1"
	var args []interface{}
	paramIdx := 1

	if filter.UserID > 0 {
		query += fmt.Sprintf(" AND user_id = $%d", paramIdx)
		args = append(args, filter.UserID)
		paramIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", paramIdx)
		args = append(args, filter.Status)
		paramIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", paramIdx)
		args = append(args, filter.Limit)
		paramIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", paramIdx)
		args = append(args, filter.Offset)
		paramIdx++
	}

	rows, err := db.QueryContext(ctx, query, args...)
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
			&t.CompletedAt, &t.Zip, &t.Unzip, &t.Password, &t.Error, &t.RetryCount, &t.Quality, &t.MD5,
		)
		if err != nil {
			slog.Error("Error scanning task in ListTasks", "error", err)
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (db *DB) GetCompletedTaskByURL(ctx context.Context, url, quality string) (*TaskRecord, error) {
	tr := &TaskRecord{}
	err := db.QueryRowContext(ctx, `
		SELECT `+taskColumns+` FROM tasks WHERE url = $1 AND quality = $2 AND status = 'completed'
		ORDER BY created_at DESC LIMIT 1
	`, url, quality).Scan(
		&tr.ID, &tr.GID, &tr.Type, &tr.Status, &tr.URL, &tr.FileName, &tr.LocalPath, &tr.RemotePath, &tr.RemoteURL,
		&tr.TotalSize, &tr.DownloadedSize, &tr.UploadedSize, &tr.ChatID, &tr.UserID,
		&tr.CreatedAt, &tr.CompletedAt, &tr.Zip, &tr.Unzip, &tr.Password, &tr.Error, &tr.RetryCount, &tr.Quality, &tr.MD5,
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
			completed_at, zip, unzip, password, error, retries, quality
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
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
			error = excluded.error,
			retries = excluded.retries,
			quality = excluded.quality
	`, t.ID, t.GID, t.Type, t.Status, t.URL, t.FileName, t.LocalPath, t.RemotePath, t.RemoteURL,
		t.TotalSize, t.DownloadedSize, t.UploadedSize, t.ChatID, t.UserID, t.CreatedAt,
		t.CompletedAt, t.Zip, t.Unzip, t.Password, t.Error, t.RetryCount, t.Quality)
	return err
}

const allTaskColumns = "id, gid, type, status, url, file_name, local_path, remote_path, remote_url, total_size, downloaded_size, uploaded_size, chat_id, user_id, created_at, completed_at, zip, unzip, password, error, retries, quality, md5"

func (db *DB) GetActive(ctx context.Context) ([]TaskRecord, error) {
	rows, err := db.QueryContext(ctx, "SELECT "+allTaskColumns+" FROM tasks WHERE status NOT IN ('completed', 'failed', 'cancelled')")
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
			&t.CompletedAt, &t.Zip, &t.Unzip, &t.Password, &t.Error, &t.RetryCount, &t.Quality, &t.MD5,
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

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks").Scan(&totalTasks); err != nil {
		slog.Error("Database error in GetBotStats count", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE status = 'completed'").Scan(&completedTasks); err != nil {
		slog.Error("Database error in GetBotStats completed", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE status = 'failed'").Scan(&failedTasks); err != nil {
		slog.Error("Database error in GetBotStats failed", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE status = 'completed'").Scan(&totalBandwidth); err != nil {
		slog.Error("Database error in GetBotStats bandwidth", "error", err)
	}

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
		SELECT `+taskColumns+` FROM tasks WHERE id = $1
	`, id).Scan(
		&tr.ID, &tr.GID, &tr.Type, &tr.Status, &tr.URL, &tr.FileName, &tr.LocalPath, &tr.RemotePath, &tr.RemoteURL,
		&tr.TotalSize, &tr.DownloadedSize, &tr.UploadedSize, &tr.ChatID, &tr.UserID,
		&tr.CreatedAt, &tr.CompletedAt, &tr.Zip, &tr.Unzip, &tr.Password, &tr.Error, &tr.RetryCount, &tr.Quality, &tr.MD5,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return tr, nil
}

func (db *DB) GetUserStats(ctx context.Context, userID int64) (*UserStats, error) {
	stats := &UserStats{UserID: userID}

	if err := db.QueryRowContext(ctx, "SELECT COALESCE(username, '') FROM users WHERE id = $1", userID).Scan(&stats.Username); err != nil {
		slog.Error("Database error in GetUserStats username", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = $1", userID).Scan(&stats.TotalDownloads); err != nil {
		slog.Error("Database error in GetUserStats total", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND status = 'completed'", userID).Scan(&stats.SuccessfulTasks); err != nil {
		slog.Error("Database error in GetUserStats successful", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND status = 'failed'", userID).Scan(&stats.FailedTasks); err != nil {
		slog.Error("Database error in GetUserStats failed", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE user_id = $1 AND status = 'completed'", userID).Scan(&stats.TotalBandwidth); err != nil {
		slog.Error("Database error in GetUserStats bandwidth", "error", err)
	}
	var lastActiveStr sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT MAX(created_at) FROM tasks WHERE user_id = $1", userID).Scan(&lastActiveStr); err != nil {
		slog.Error("Database error in GetUserStats last active", "error", err)
	}
	if lastActiveStr.Valid {
		stats.LastActive, _ = time.Parse("2006-01-02 15:04:05", lastActiveStr.String)
	}
	if stats.LastActive.IsZero() {
		stats.LastActive = time.Now()
	}

	return stats, nil
}

type DailyStats = domain.DailyStats

func todayRange() (string, string) {
	now := time.Now()
	start := now.Format("2006-01-02 00:00:00")
	end := now.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
	return start, end
}

func (db *DB) GetTodayStats(ctx context.Context) (*DailyStats, error) {
	stats := &DailyStats{Date: time.Now()}
	start, end := todayRange()

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at >= $1 AND created_at < $2", start, end).Scan(&stats.TotalTasks); err != nil {
		slog.Error("Database error in GetTodayStats total", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at >= $1 AND created_at < $2 AND status = 'completed'", start, end).Scan(&stats.CompletedTasks); err != nil {
		slog.Error("Database error in GetTodayStats completed", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE created_at >= $1 AND created_at < $2 AND status = 'failed'", start, end).Scan(&stats.FailedTasks); err != nil {
		slog.Error("Database error in GetTodayStats failed", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE created_at >= $1 AND created_at < $2 AND status = 'completed'", start, end).Scan(&stats.TotalBandwidth); err != nil {
		slog.Error("Database error in GetTodayStats", "error", err)
	}

	return stats, nil
}

func (db *DB) GetUserTodayStats(ctx context.Context, userID int64) (*DailyStats, error) {
	stats := &DailyStats{Date: time.Now()}
	start, end := todayRange()

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND created_at >= $2 AND created_at < $3", userID, start, end).Scan(&stats.TotalTasks); err != nil {
		slog.Error("Database error in GetUserTodayStats total", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND created_at >= $2 AND created_at < $3 AND status = 'completed'", userID, start, end).Scan(&stats.CompletedTasks); err != nil {
		slog.Error("Database error in GetUserTodayStats completed", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = $1 AND created_at >= $2 AND created_at < $3 AND status = 'failed'", userID, start, end).Scan(&stats.FailedTasks); err != nil {
		slog.Error("Database error in GetUserTodayStats failed", "error", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_size), 0) FROM tasks WHERE user_id = $1 AND created_at >= $2 AND created_at < $3 AND status = 'completed'", userID, start, end).Scan(&stats.TotalBandwidth); err != nil {
		slog.Error("Database error in GetUserTodayStats bandwidth", "error", err)
	}

	return stats, nil
}

func (db *DB) GetWeeklyStats(ctx context.Context) ([]DailyStats, error) {
	now := time.Now()
	start := now.AddDate(0, 0, -6).Format("2006-01-02")
	end := now.AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := db.QueryContext(ctx, `
		SELECT DATE(created_at) as day,
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN total_size ELSE 0 END), 0) as bandwidth
		FROM tasks WHERE created_at >= $1 AND created_at < $2
		GROUP BY DATE(created_at) ORDER BY day
	`, start, end)
	if err != nil {
		slog.Error("Database error in GetWeeklyStats", "error", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	resultMap := make(map[string]*DailyStats)
	for i := 6; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")
		resultMap[dateStr] = &DailyStats{Date: d}
	}

	for rows.Next() {
		var day string
		var ds DailyStats
		if err := rows.Scan(&day, &ds.TotalTasks, &ds.CompletedTasks, &ds.FailedTasks, &ds.TotalBandwidth); err != nil {
			slog.Error("Database error scanning weekly stats row", "error", err)
			continue
		}
		if existing, ok := resultMap[day]; ok {
			existing.TotalTasks = ds.TotalTasks
			existing.CompletedTasks = ds.CompletedTasks
			existing.FailedTasks = ds.FailedTasks
			existing.TotalBandwidth = ds.TotalBandwidth
		}
	}

	var stats []DailyStats
	for i := 6; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")
		stats = append(stats, *resultMap[dateStr])
	}

	return stats, nil
}

func (db *DB) GetMonthlyStats(ctx context.Context) ([]DailyStats, error) {
	now := time.Now()
	start := now.AddDate(0, 0, -29).Format("2006-01-02")
	end := now.AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := db.QueryContext(ctx, `
		SELECT DATE(created_at) as day,
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN total_size ELSE 0 END), 0) as bandwidth
		FROM tasks WHERE created_at >= $1 AND created_at < $2
		GROUP BY DATE(created_at) ORDER BY day
	`, start, end)
	if err != nil {
		slog.Error("Database error in GetMonthlyStats", "error", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	resultMap := make(map[string]*DailyStats)
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")
		resultMap[dateStr] = &DailyStats{Date: d}
	}

	for rows.Next() {
		var day string
		var ds DailyStats
		if err := rows.Scan(&day, &ds.TotalTasks, &ds.CompletedTasks, &ds.FailedTasks, &ds.TotalBandwidth); err != nil {
			slog.Error("Database error scanning monthly stats row", "error", err)
			continue
		}
		if existing, ok := resultMap[day]; ok {
			existing.TotalTasks = ds.TotalTasks
			existing.CompletedTasks = ds.CompletedTasks
			existing.FailedTasks = ds.FailedTasks
			existing.TotalBandwidth = ds.TotalBandwidth
		}
	}

	var stats []DailyStats
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")
		stats = append(stats, *resultMap[dateStr])
	}

	return stats, nil
}

func (db *DB) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = $1", key).Scan(&value)
	return value, err
}

func (db *DB) Set(ctx context.Context, key, value string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, $3)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now())
	return err
}

func (db *DB) GetRecoverable(ctx context.Context) ([]TaskRecord, error) {
	query := fmt.Sprintf(`
		SELECT `+allTaskColumns+` FROM tasks 
		WHERE status IN ('downloading', 'uploading', 'queued', 'processing')
		AND created_at > %s
	`, db.nowMinusExpr("24 hours"))

	rows, err := db.QueryContext(ctx, query)
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
			&t.CompletedAt, &t.Zip, &t.Unzip, &t.Password, &t.Error, &t.RetryCount, &t.Quality, &t.MD5,
		)
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (db *DB) UpdateStatus(ctx context.Context, taskID, status, errorMsg string) error {
	_, err := db.ExecContext(ctx, "UPDATE tasks SET status = $1, error = $2 WHERE id = $3", status, errorMsg, taskID)
	return err
}

func (db *DB) UpdateMD5(ctx context.Context, id, md5 string) error {
	_, err := db.ExecContext(ctx, "UPDATE tasks SET md5=$1 WHERE id=$2", md5, id)
	return err
}

func (db *DB) DeleteOld(ctx context.Context, before string) (int, error) {
	result, err := db.ExecContext(ctx, "DELETE FROM tasks WHERE status IN ('completed', 'failed', 'cancelled') AND created_at < $1", before)
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
		LIMIT $1
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

func (db *DB) SaveCheckpoint(ctx context.Context, cp domain.TaskCheckpoint) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO task_checkpoints (task_id, downloaded_bytes, total_bytes, progress, last_update)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(task_id) DO UPDATE SET
			downloaded_bytes = excluded.downloaded_bytes,
			total_bytes = excluded.total_bytes,
			progress = excluded.progress,
			last_update = excluded.last_update
	`, cp.TaskID, cp.DownloadedBytes, cp.TotalBytes, cp.Progress, cp.LastUpdate.Unix())
	return err
}

func (db *DB) GetCheckpoint(ctx context.Context, taskID string) (*domain.TaskCheckpoint, error) {
	var cp domain.TaskCheckpoint
	var lastUpdate int64
	err := db.QueryRowContext(ctx, `
		SELECT task_id, downloaded_bytes, total_bytes, progress, last_update
		FROM task_checkpoints WHERE task_id = $1
	`, taskID).Scan(&cp.TaskID, &cp.DownloadedBytes, &cp.TotalBytes, &cp.Progress, &lastUpdate)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	cp.LastUpdate = time.Unix(lastUpdate, 0)
	return &cp, nil
}

func (db *DB) DeleteCheckpoint(ctx context.Context, taskID string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM task_checkpoints WHERE task_id = $1", taskID)
	return err
}

func (db *DB) SaveScheduled(ctx context.Context, task domain.ScheduledTask) error {
	query := fmt.Sprintf(`
		INSERT INTO scheduled_tasks (id, task_type, url, file_name, chat_id, user_id, zip, unzip, password, quality, scheduled_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'pending', %s)
	`, db.nowExpr())

	_, err := db.ExecContext(ctx, query,
		task.ID, task.TaskType, task.URL, task.FileName, task.ChatID, task.UserID,
		boolToInt(task.Zip), boolToInt(task.Unzip), task.Password, task.Quality, task.ScheduledAt)
	return err
}

func (db *DB) GetPendingScheduled(ctx context.Context) ([]domain.ScheduledTask, error) {
	query := fmt.Sprintf(`
		SELECT id, task_type, url, file_name, chat_id, user_id, zip, unzip, password, quality, scheduled_at, status, task_id, created_at 
		FROM scheduled_tasks WHERE status='pending' AND scheduled_at <= %s
	`, db.nowExpr())

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.ScheduledTask
	for rows.Next() {
		var t domain.ScheduledTask
		var zipInt, unzipInt int
		if err := rows.Scan(&t.ID, &t.TaskType, &t.URL, &t.FileName, &t.ChatID, &t.UserID, &zipInt, &unzipInt, &t.Password, &t.Quality, &t.ScheduledAt, &t.Status, &t.TaskID, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Zip = zipInt == 1
		t.Unzip = unzipInt == 1
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (db *DB) MarkScheduledDone(ctx context.Context, id, taskID string) error {
	_, err := db.ExecContext(ctx, "UPDATE scheduled_tasks SET status='done', task_id=$1 WHERE id=$2", taskID, id)
	return err
}

func (db *DB) DeleteScheduled(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM scheduled_tasks WHERE id=$1", id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
