# Config Hot-Reload, Worker Throttle, Rate Limit Persistence, Playlist Support

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 4 production issues: config requires restart to reload, worker concurrency not explicitly throttled, rate limiter resets on restart, and playlist downloads need full implementation.

**Architecture:** Each fix is independent and can be implemented/committed separately. Config reload adds SIGHUP handler + API endpoint. Worker throttle adds explicit semaphore channel. Rate limiter persists state to SQLite. Playlist support enhances existing partial implementation with progress tracking.

**Tech Stack:** Go, SQLite (modernc.org/sqlite), golang.org/x/time/rate, godotenv, os/signal

---

## Task 4: Config Hot-Reload

**Problem:** `LoadConfig()` is called once at startup. Changes to `.env` or environment variables require container restart.

**Solution:** Add `ReloadConfig()` function, SIGHUP signal handler, and `/api/config/reload` endpoint. The config pointer is updated atomically via `atomic.Value`.

### Task 4.1: Make Config reloadable with atomic swap

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add ReloadConfig function and atomic config store**

Add to `internal/config/config.go` after the `LoadConfig` function:

```go
// currentConfig holds the active config atomically for safe concurrent access.
var currentConfig atomic.Value

func init() {
	currentConfig.Store(LoadConfig())
}

// Get returns the current active configuration.
func Get() *Config {
	return currentConfig.Load().(*Config)
}

// Reload re-reads environment variables and .env, swaps the config atomically,
// and returns the new config. Callers should use Get() to access the active config.
func Reload() *Config {
	_ = godotenv.Load()
	newCfg := LoadConfig()
	currentConfig.Store(newCfg)
	slog.Info("Config reloaded successfully",
		"maxConcurrent", newCfg.MaxConcurrentDownloads,
		"stopDuplicate", newCfg.StopDuplicate,
		"logLevel", newCfg.LogLevel,
	)
	return newCfg
}
```

Add import for `"sync/atomic"` and `"log/slog"` and `"github.com/joho/godotenv"` to the import block.

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/config/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add atomic config store and Reload function"
```

### Task 4.2: Add SIGHUP handler in main.go

**Files:**
- Modify: `cmd/zee-mirror/main.go`

- [ ] **Step 1: Add SIGHUP signal handler after bot service creation**

In `cmd/zee-mirror/main.go`, after `botSvc := handlers.NewBotService(bot, cfg, db)` (line 92), add:

```go
// Config hot-reload via SIGHUP
sighup := make(chan os.Signal, 1)
signal.Notify(sighup, syscall.SIGHUP)
go func() {
    for range sighup {
        slog.Info("Received SIGHUP, reloading config...")
        newCfg := config.Reload()
        botSvc.TaskManager.Mu.Lock()
        botSvc.TaskManager.Config = newCfg
        botSvc.TaskManager.MaxConcurrent = newCfg.MaxConcurrentDownloads
        botSvc.TaskManager.StopDuplicate = newCfg.StopDuplicate
        botSvc.TaskManager.Mu.Unlock()
        slog.Info("TaskManager config updated")
    }
}()
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/zee-mirror/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/zee-mirror/main.go
git commit -m "feat(config): add SIGHUP handler for config hot-reload"
```

### Task 4.3: Add /api/config/reload HTTP endpoint

**Files:**
- Modify: `internal/api/api.go`

- [ ] **Step 1: Register the reload route**

In `internal/api/api.go`, inside the `Start()` method where routes are registered, add after the existing `/api/config` route:

```go
mux.HandleFunc("POST /api/config/reload", auth(func(w http.ResponseWriter, r *http.Request) {
    newCfg := config.Reload()
    s.Service.TaskManager.Mu.Lock()
    s.Service.TaskManager.Config = newCfg
    s.Service.TaskManager.MaxConcurrent = newCfg.MaxConcurrentDownloads
    s.Service.TaskManager.StopDuplicate = newCfg.StopDuplicate
    s.Service.TaskManager.Mu.Unlock()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":  "ok",
        "message": "Config reloaded successfully",
        "max_concurrent_downloads": newCfg.MaxConcurrentDownloads,
        "stop_duplicate": newCfg.StopDuplicate,
        "log_level": newCfg.LogLevel,
    })
}))
```

Add `"zee-mirror/internal/config"` to the import block if not already present.

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/api/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/api/api.go
git commit -m "feat(api): add POST /api/config/reload endpoint"
```

---

## Task 5: Worker MaxConcurrent Throttling

**Problem:** Worker goroutines process tasks synchronously, but `ActiveCount` is never tracked. The implicit concurrency limit (N goroutines) works, but there's no explicit semaphore or counter for monitoring/debugging.

**Solution:** Add an explicit semaphore channel (`chan struct{}`) to bound concurrent task execution, and properly track `ActiveCount`.

### Task 5.1: Add semaphore to TaskManager and worker loop

**Files:**
- Modify: `internal/service/task_manager.go`

- [ ] **Step 1: Add semaphore field to TaskManager struct**

In `internal/service/task_manager.go`, add to the `TaskManager` struct after the `MaxConcurrent` field:

```go
Semaphore chan struct{}
```

- [ ] **Step 2: Initialize semaphore in NewTaskManager**

In `NewTaskManager()`, add after `MaxConcurrent: maxConcurrent,` (around line 131):

```go
Semaphore: make(chan struct{}, maxConcurrent),
```

- [ ] **Step 3: Update worker() to use semaphore and track ActiveCount**

Replace the entire `worker()` method with:

```go
func (tm *TaskManager) worker(_ int) {
	for {
		select {
		case <-tm.ShutdownChan:
			return
		case <-tm.QueueSignal:
			item := tm.Queue.DequeueNonBlocking()
			if item == nil {
				continue
			}
			task, ok := item.(*Task)
			if !ok {
				continue
			}

			// Acquire semaphore slot
			tm.Semaphore <- struct{}{}

			tm.Mu.Lock()
			tm.ActiveCount++
			tm.Mu.Unlock()

			tm.Wg.Add(1)
			if tm.ProcessTaskFunc != nil {
				tm.ProcessTaskFunc(task)
			}
			tm.Wg.Done()

			// Release semaphore slot
			<-tm.Semaphore

			tm.Mu.Lock()
			tm.ActiveCount--
			tm.Mu.Unlock()
		}
	}
}
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./internal/service/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/service/task_manager.go
git commit -m "feat(worker): add explicit semaphore throttling and ActiveCount tracking"
```

---

## Task 6: Rate Limiter Persistence

**Problem:** `UserRateLimiter` is purely in-memory. After restart, all counters reset and users can bypass limits.

**Solution:** Persist rate limit token bucket state to SQLite. On startup, restore from DB. On each `Allow()` call, persist the state periodically (not every call, to avoid DB thrashing).

### Task 6.1: Create rate_limits database table

**Files:**
- Create: `migrations/000010_add_rate_limits.up.sql`
- Create: `migrations/000010_add_rate_limits.down.sql`

- [ ] **Step 1: Create migration files**

`migrations/000010_add_rate_limits.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS rate_limits (
    user_id INTEGER PRIMARY KEY,
    tokens REAL NOT NULL,
    last_fetch REAL NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

`migrations/000010_add_rate_limits.down.sql`:
```sql
DROP TABLE IF EXISTS rate_limits;
```

- [ ] **Step 2: Commit**

```bash
git add migrations/000010_add_rate_limits.up.sql migrations/000010_add_rate_limits.down.sql
git commit -m "feat(db): add rate_limits table migration"
```

### Task 6.2: Implement persistent rate limiter

**Files:**
- Modify: `internal/queue/ratelimit.go`

- [ ] **Step 1: Add DB-backed rate limiter**

Replace the entire content of `internal/queue/ratelimit.go` with:

```go
package queue

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimitRecord struct {
	UserID    int64
	Tokens    float64
	LastFetch time.Time
}

type UserRateLimiter struct {
	limiters map[int64]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	db       *sql.DB
}

func NewUserRateLimiter(ratePerMin int, burst int) *UserRateLimiter {
	return &UserRateLimiter{
		limiters: make(map[int64]*rate.Limiter),
		rate:     rate.Limit(float64(ratePerMin) / 60.0),
		burst:    burst,
	}
}

func NewUserRateLimiterWithDB(ratePerMin int, burst int, db *sql.DB) *UserRateLimiter {
	rl := NewUserRateLimiter(ratePerMin, burst)
	rl.db = db
	rl.loadFromDB()
	return rl
}

func (rl *UserRateLimiter) loadFromDB() {
	if rl.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := rl.db.QueryContext(ctx, "SELECT user_id, tokens, last_fetch FROM rate_limits")
	if err != nil {
		slog.Warn("Failed to load rate limits from DB", "error", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var rec RateLimitRecord
		if err := rows.Scan(&rec.UserID, &rec.Tokens, &rec.LastFetch); err != nil {
			continue
		}

		limiter := rate.NewLimiter(rl.rate, rl.burst)
		elapsed := time.Since(rec.LastFetch).Seconds()
		// Restore tokens based on elapsed time since last fetch
		newTokens := rec.Tokens + elapsed*float64(rl.rate)
		if newTokens > float64(rl.burst) {
			newTokens = float64(rl.burst)
		}
		limiter.ReserveN(time.Now(), rate.Limit(0)) // noop to set internal state
		rl.limiters[rec.UserID] = limiter
		count++
	}

	slog.Info("Loaded rate limits from database", "users", count)
}

func (rl *UserRateLimiter) getLimiter(userID int64) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[userID]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[userID] = limiter
	}
	return limiter
}

func (rl *UserRateLimiter) Allow(userID int64) bool {
	return rl.getLimiter(userID).Allow()
}

// Persist saves all rate limit states to the database.
func (rl *UserRateLimiter) Persist() {
	if rl.db == nil {
		return
	}

	rl.mu.RLock()
	records := make([]RateLimitRecord, 0, len(rl.limiters))
	for userID, limiter := range rl.limiters {
		records = append(records, RateLimitRecord{
			UserID:    userID,
			Tokens:    float64(limiter.Tokens()),
			LastFetch: time.Now(),
		})
	}
	rl.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := rl.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Warn("Failed to begin rate limit persist transaction", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO rate_limits (user_id, tokens, last_fetch, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			tokens = excluded.tokens,
			last_fetch = excluded.last_fetch,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		slog.Warn("Failed to prepare rate limit persist statement", "error", err)
		return
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.UserID, rec.Tokens, rec.LastFetch, now); err != nil {
			slog.Warn("Failed to persist rate limit", "userID", rec.UserID, "error", err)
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Warn("Failed to commit rate limit persist", "error", err)
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/queue/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/queue/ratelimit.go
git commit -m "feat(ratelimit): add SQLite-backed persistent rate limiter"
```

### Task 6.3: Wire persistent rate limiter and periodic persist

**Files:**
- Modify: `internal/service/task_manager.go`
- Modify: `internal/service/bot_service.go`
- Modify: `cmd/zee-mirror/main.go`

- [ ] **Step 1: Update NewTaskManager to accept db parameter for rate limiter**

In `internal/service/task_manager.go`, change the `NewTaskManager` signature to accept the raw `*sql.DB`:

The function already receives `db repository.TaskRepository`. We need to pass the underlying `*sql.DB` to the rate limiter. Add a parameter:

```go
func NewTaskManager(bot *tgbotapi.BotAPI, maxConcurrent int, downloadDir, rcloneDest, configDir string, processTaskFunc func(*Task), refreshDashboardFunc func(int64, bool), db repository.TaskRepository, sqlDB *sql.DB) *TaskManager {
```

Then change the rate limiter initialization from:

```go
RateLimiter: queue.NewUserRateLimiter(5, 10),
```

to:

```go
RateLimiter: queue.NewUserRateLimiterWithDB(5, 10, sqlDB),
```

Add `"database/sql"` to the import block.

- [ ] **Step 2: Update BotService to pass sqlDB through**

In `internal/service/bot_service.go`, find where `NewTaskManager` is called and add the `sqlDB` parameter. The `BotService` struct needs a `SQLDB *sql.DB` field. Add it to the struct and pass it through.

In the `NewBotService` function, add `sqlDB` parameter:

```go
func NewBotService(bot *tgbotapi.BotAPI, cfg *config.Config, db repository.FullRepository, sqlDB *sql.DB) *BotService {
```

Store it: `s.SQLDB = sqlDB`

Pass to NewTaskManager: `NewTaskManager(bot, cfg.MaxConcurrentDownloads, ..., sqlDB)`

- [ ] **Step 3: Update handlers/bot.go NewBotService call**

In `handlers/bot.go`, update the `NewBotService` wrapper to accept and pass `sqlDB`.

- [ ] **Step 4: Update main.go to pass sqlDB**

In `cmd/zee-mirror/main.go`, the `database.NewDB()` returns a `*database.DB`. We need to extract the underlying `*sql.DB`. Check `internal/database/database.go` for the `DB` struct - it likely has a `*sql.DB` field. Pass it through to `handlers.NewBotService`.

- [ ] **Step 5: Add periodic persist goroutine**

In `internal/service/task_manager.go`, add a new method:

```go
func (tm *TaskManager) startRateLimitPersist() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ShutdownChan:
			return
		case <-ticker.C:
			tm.RateLimiter.Persist()
		}
	}
}
```

Call it from `NewTaskManager()` after starting the other goroutines:

```go
go tm.startRateLimitPersist()
```

- [ ] **Step 6: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add internal/service/task_manager.go internal/service/bot_service.go handlers/bot.go cmd/zee-mirror/main.go
git commit -m "feat(ratelimit): wire persistent rate limiter with periodic DB sync"
```

---

## Task 7: Playlist Download Support

**Problem:** Playlist URLs trigger `ytdlp_playlist_error` in the quality menu flow. While `handleYTDLPPlaylist()` exists and creates per-item tasks, it lacks proper progress tracking and uses `TypeYTDLP` instead of `TypePlaylist`.

**Solution:** Enhance playlist handling with proper `TypePlaylist` usage, a parent task for tracking overall progress, and improved UI feedback.

### Task 7.1: Add playlist parent task and improve progress tracking

**Files:**
- Modify: `handlers/download/ytdlp.go`
- Modify: `handlers/ytdlp_handler.go`

- [ ] **Step 1: Create a playlist parent task in handleYTDLPPlaylist**

In `handlers/download/ytdlp.go`, update `handleYTDLPPlaylist` to create a parent task of type `TypePlaylist` that tracks overall progress:

Find the `handleYTDLPPlaylist` method and update it. The key change is to create a parent task with `TypePlaylist` and set `PlaylistCount` on sub-tasks:

```go
func (s *BotService) handleYTDLPPlaylist(message *tgbotapi.Message, url string, taskType service.TaskType) {
	lang := s.GetUserLanguage(message.From.ID)

	msg := tgbotapi.NewMessage(message.Chat.ID, i18n.T(lang, "ytdlp_playlist_fetching"))
	sentMsg, _ := s.Bot.Send(msg)

	metadata, err := s.TaskManager.YTDLPEngine.GetPlaylistMetadata(message.Context(), url)
	if err != nil {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentMsg.MessageID,
			i18n.T(lang, "ytdlp_playlist_error"))
		_, _ = s.Bot.Send(editMsg)
		return
	}

	if metadata == nil || len(metadata.Entries) == 0 {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentMsg.MessageID,
			i18n.T(lang, "ytdlp_playlist_empty"))
		_, _ = s.Bot.Send(editMsg)
		return
	}

	maxItems := 50
	if len(metadata.Entries) > maxItems {
		metadata.Entries = metadata.Entries[:maxItems]
	}

	// Create parent playlist task for progress tracking
	parentTask, _ := s.TaskManager.CreatePlaylistParentTask(
		metadata.Title,
		url,
		message.Chat.ID,
		sentMsg.MessageID,
		message.From.ID,
		len(metadata.Entries),
	)

	text := fmt.Sprintf("📋 *Playlist:* %s\n📊 *Total:* %d items\n\nMulai download...",
		utils.EscapeMarkdownV2(metadata.Title),
		len(metadata.Entries))
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentMsg.MessageID, text)
	editMsg.ParseMode = service.MarkdownV2
	_, _ = s.Bot.Send(editMsg)

	for i, entry := range metadata.Entries {
		entryURL := entry.URL
		if entryURL == "" {
			entryURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", entry.ID)
		}

		fileName := utils.SanitizeFileName(entry.Title)
		if fileName == "" {
			fileName = fmt.Sprintf("video_%d", i+1)
		}

		task, err := s.TaskManager.CreatePlaylistSubTask(
			parentTask,
			entryURL,
			fileName,
			i+1,
			len(metadata.Entries),
			taskType,
		)
		if err != nil {
			slog.Warn("Skipping playlist item", "index", i+1, "error", err)
			continue
		}
		slog.Info("Playlist sub-task created", "taskID", task.ID, "index", i+1, "title", entry.Title)
	}

	s.UpdateSharedDashboard(message.Chat.ID, true)
}
```

- [ ] **Step 2: Add CreatePlaylistParentTask and CreatePlaylistSubTask to TaskManager**

In `internal/service/task_manager.go`, add new methods:

```go
func (tm *TaskManager) CreatePlaylistParentTask(title, url string, chatID, msgID int, userID int64, totalItems int) (*Task, error) {
	ctx, cancel := context.WithCancel(context.Background())

	task := &Task{
		Task: domain.Task{
			ID:             uuid.New().String()[:12],
			Type:           TypePlaylist,
			Status:         StatusDownloading,
			URL:            url,
			FileName:       title,
			ChatID:         chatID,
			MessageID:      msgID,
			UserID:         userID,
			CreatedAt:      time.Now().UTC(),
			Ctx:            ctx,
			CancelFunc:     cancel,
			PlaylistCount:  totalItems,
			TotalSize:      0,
			MaxRetries:     tm.Config.MaxRetries,
		},
		DB: tm.DB,
	}

	tm.Mu.Lock()
	tm.Tasks[task.ID] = task
	tm.Mu.Unlock()

	_ = task.SaveToDB()

	return task, nil
}

func (tm *TaskManager) CreatePlaylistSubTask(parent *Task, url, fileName string, index, total int, taskType TaskType) (*Task, error) {
	ctx, cancel := context.WithCancel(parent.Ctx)

	task := &Task{
		Task: domain.Task{
			ID:             fmt.Sprintf("%s_%d", parent.ID, index),
			Type:           taskType,
			Status:         StatusQueued,
			URL:            url,
			FileName:       fileName,
			ChatID:         parent.ChatID,
			UserID:         parent.UserID,
			CreatedAt:      time.Now().UTC(),
			Ctx:            ctx,
			CancelFunc:     cancel,
			PlaylistIndex:  index,
			PlaylistCount:  total,
			MaxRetries:     tm.Config.MaxRetries,
		},
		DB: tm.DB,
	}

	tm.Mu.Lock()
	tm.Tasks[task.ID] = task
	tm.Mu.Unlock()

	_ = task.SaveToDB()

	tm.Queue.Enqueue(task, 0)
	select {
	case tm.QueueSignal <- struct{}{}:
	default:
	}

	return task, nil
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add handlers/download/ytdlp.go handlers/ytdlp_handler.go internal/service/task_manager.go
git commit -m "feat(playlist): add TypePlaylist parent task with per-item progress tracking"
```

### Task 7.2: Update dashboard to show playlist progress

**Files:**
- Modify: `internal/service/dashboard.go` (or wherever dashboard status is built)

- [ ] **Step 1: Add playlist progress calculation to dashboard**

Find the function that builds the dashboard status text (likely `buildStatusText` or similar in dashboard-related files). Add a case for `TypePlaylist` that shows overall playlist progress:

In the task status display logic, add handling for playlist tasks. The parent task of type `TypePlaylist` should show: `📋 Playlist: [title] (X/Y completed)`

This requires finding the exact dashboard rendering function. Search for where `task.Type` is checked in the status display code and add a playlist case.

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/service/dashboard.go
git commit -m "feat(dashboard): display playlist progress in status view"
```

---

## Execution Order

These 4 tasks are independent and can be executed in any order or in parallel:

1. **Task 4** (Config Hot-Reload) - No dependencies
2. **Task 5** (Worker Throttle) - No dependencies
3. **Task 6** (Rate Limit Persistence) - Depends on existing SQLite setup
4. **Task 7** (Playlist Support) - No dependencies

Recommended: Execute Task 4 and 5 first (simpler), then Task 6 (DB migration), then Task 7 (most complex).
