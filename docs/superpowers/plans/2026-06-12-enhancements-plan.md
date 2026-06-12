# Enhancement Features Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement 5 independent enhancements: task scheduling, richer notifications, task history dashboard, multi-destination upload, and yt-dlp auto-update.

**Architecture:** Each feature is independent and can be implemented/committed separately. Order is: domain changes first (MD5, Dest2 fields), then new systems (scheduling, yt-dlp update), then dashboard features (task history).

**Tech Stack:** Go, SQLite, React 19 + TypeScript + Vite 8

---

## Feature 8: TypeScheduled — Task Scheduling

### Task 8.1: Create scheduled_tasks migration

**Files:**
- Create: `migrations/000006_add_scheduled_tasks.up.sql`
- Create: `migrations/000006_add_scheduled_tasks.down.sql`

- [ ] **Step 1: Create migration files**

`migrations/000006_add_scheduled_tasks.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id          TEXT PRIMARY KEY,
    task_type   TEXT NOT NULL,
    url         TEXT NOT NULL,
    file_name   TEXT NOT NULL DEFAULT '',
    chat_id     INTEGER NOT NULL,
    user_id     INTEGER NOT NULL,
    zip         INTEGER DEFAULT 0,
    unzip       INTEGER DEFAULT 0,
    password    TEXT DEFAULT '',
    quality     TEXT DEFAULT '',
    scheduled_at DATETIME NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    task_id     TEXT DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

`migrations/000006_add_scheduled_tasks.down.sql`:
```sql
DROP TABLE IF EXISTS scheduled_tasks;
```

- [ ] **Step 2: Commit**

```bash
git add migrations/
git commit -m "feat(db): add scheduled_tasks table migration"
```

### Task 8.2: Add scheduled task domain + repository types

**Files:**
- Create: `internal/domain/scheduled.go`
- Modify: `internal/repository/task.go`

- [ ] **Step 1: Create domain types**

`internal/domain/scheduled.go`:
```go
package domain

type ScheduledTask struct {
    ID          string
    TaskType    string
    URL         string
    FileName    string
    ChatID      int64
    UserID      int64
    Zip         bool
    Unzip       bool
    Password    string
    Quality     string
    ScheduledAt string
    Status      string
    TaskID      string
    CreatedAt   string
}

type ScheduledTaskFilter struct {
    Page   int
    Limit  int
    Status string
}
```

- [ ] **Step 2: Add repository interface**

In `internal/repository/task.go`, add to `TaskRepository` interface:

```go
SaveScheduled(ctx context.Context, task ScheduledTask) error
GetPendingScheduled(ctx context.Context) ([]ScheduledTask, error)
MarkScheduledDone(ctx context.Context, id, taskID string) error
DeleteScheduled(ctx context.Context, id string) error
```

Add import for `"zee-mirror/internal/domain"`.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/scheduled.go internal/repository/task.go
git commit -m "feat(schedule): add domain types and repository interface for scheduled tasks"
```

### Task 8.3: Implement scheduled task repository on DB

**Files:**
- Modify: `internal/database/database.go`

- [ ] **Step 1: Add repository methods to database.go**

Add these methods to the DB struct:

```go
func (db *DB) SaveScheduled(ctx context.Context, task domain.ScheduledTask) error {
    _, err := db.ExecContext(ctx, `
        INSERT INTO scheduled_tasks (id, task_type, url, file_name, chat_id, user_id, zip, unzip, password, quality, scheduled_at, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', datetime('now'))
    `, task.ID, task.TaskType, task.URL, task.FileName, task.ChatID, task.UserID, boolToInt(task.Zip), boolToInt(task.Unzip), task.Password, task.Quality, task.ScheduledAt)
    return err
}

func (db *DB) GetPendingScheduled(ctx context.Context) ([]domain.ScheduledTask, error) {
    rows, err := db.QueryContext(ctx, "SELECT id, task_type, url, file_name, chat_id, user_id, zip, unzip, password, quality, scheduled_at, status, task_id, created_at FROM scheduled_tasks WHERE status='pending' AND scheduled_at <= datetime('now')")
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
    _, err := db.ExecContext(ctx, "UPDATE scheduled_tasks SET status='done', task_id=? WHERE id=?", taskID, id)
    return err
}

func (db *DB) DeleteScheduled(ctx context.Context, id string) error {
    _, err := db.ExecContext(ctx, "DELETE FROM scheduled_tasks WHERE id=?", id)
    return err
}
```

Add helper: `func boolToInt(b bool) int { if b { return 1 }; return 0 }`

- [ ] **Step 2: Commit**

```bash
git add internal/database/database.go
git commit -m "feat(schedule): implement scheduled task repository methods"
```

### Task 8.4: Create scheduler goroutine

**Files:**
- Create: `internal/service/scheduler.go`
- Modify: `internal/service/task_manager.go`

- [ ] **Step 1: Create scheduler service**

`internal/service/scheduler.go`:
```go
package service

import (
    "context"
    "log/slog"
    "time"
)

func (tm *TaskManager) startScheduler() {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-tm.ShutdownChan:
            return
        case <-ticker.C:
            tm.processScheduledTasks()
        }
    }
}

func (tm *TaskManager) processScheduledTasks() {
    if tm.DB == nil {
        return
    }

    tasks, err := tm.DB.GetPendingScheduled(context.Background())
    if err != nil {
        slog.Error("Failed to get pending scheduled tasks", "error", err)
        return
    }

    for _, st := range tasks {
        taskType := TaskType(st.TaskType)
        task, err := tm.CreateTask(taskType, st.URL, st.FileName, st.ChatID, 0, 0, st.UserID, st.Zip, st.Unzip, st.Password, st.Quality, 0, "", false)
        if err != nil {
            slog.Error("Failed to create scheduled task", "scheduledID", st.ID, "error", err)
            continue
        }
        if err := tm.DB.MarkScheduledDone(context.Background(), st.ID, task.ID); err != nil {
            slog.Error("Failed to mark scheduled task as done", "scheduledID", st.ID, "error", err)
        }
        slog.Info("Scheduled task executed", "scheduledID", st.ID, "taskID", task.ID)
    }
}
```

This requires `context` already imported (not in existing imports). Make sure `"context"` is in the import of this file. Since this is appended to `task_manager.go`, the import may already be there. If not, add it.

- [ ] **Step 2: Start scheduler in NewTaskManager**

In `internal/service/task_manager.go`, in `NewTaskManager()` after `go tm.startRateLimitPersist()`, add:
```go
go tm.startScheduler()
```

- [ ] **Step 3: Commit**

```bash
git add internal/service/scheduler.go internal/service/task_manager.go
git commit -m "feat(schedule): add scheduler goroutine polling every 60s"
```

### Task 8.5: Add /schedule bot command handler

**Files:**
- Create: `handlers/download/schedule.go`
- Modify: `cmd/zee-mirror/main.go`

- [ ] **Step 1: Create schedule handler**

`handlers/download/schedule.go`:
```go
package download

import (
    "fmt"
    "strings"
    "time"

    "zee-mirror/internal/domain"
    "zee-mirror/internal/service"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/google/uuid"
)

func HandleSchedule(s *service.BotService, message *tgbotapi.Message, args string) {
    if args == "" {
        msg := tgbotapi.NewMessage(message.Chat.ID, "📅 *Format:* `/schedule -at HH:MM <url>`")
        msg.ParseMode = tgbotapi.ModeMarkdownV2
        _, _ = s.Bot.Send(msg)
        return
    }

    url, zip, unzip, password, quality, _, _, _, atTime := utils.ParseScheduleFlags(args)
    if url == "" {
        url = utils.ExtractURLFromText(args)
    }
    if url == "" {
        msg := tgbotapi.NewMessage(message.Chat.ID, "❌ URL tidak ditemukan")
        _, _ = s.Bot.Send(msg)
        return
    }

    if atTime == "" {
        msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Gunakan `-at HH:MM` untuk menentukan jadwal")
        msg.ParseMode = tgbotapi.ModeMarkdownV2
        _, _ = s.Bot.Send(msg)
        return
    }

    // Parse scheduled time (today at HH:MM, or tomorrow if past)
    now := time.Now()
    t, err := time.Parse("15:04", atTime)
    if err != nil {
        msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Format waktu salah. Gunakan HH:MM (contoh: 02:00)")
        _, _ = s.Bot.Send(msg)
        return
    }

    scheduledAt := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
    if scheduledAt.Before(now) {
        scheduledAt = scheduledAt.Add(24 * time.Hour)
    }

    st := domain.ScheduledTask{
        ID:          uuid.New().String()[:12],
        TaskType:    string(service.TypeMirror),
        URL:         url,
        FileName:    utils.GetFileNameFromURL(url),
        ChatID:      message.Chat.ID,
        UserID:      message.From.ID,
        Zip:         zip,
        Unzip:       unzip,
        Password:    password,
        Quality:     quality,
        ScheduledAt: scheduledAt.Format("2006-01-02 15:04:05"),
    }

    if err := s.DB.SaveScheduled(message.Context(), st); err != nil {
        msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Gagal menyimpan jadwal: "+err.Error())
        _, _ = s.Bot.Send(msg)
        return
    }

    text := fmt.Sprintf("✅ *Task Dijadwalkan*\n\n📅 *Waktu:* %s\n🔗 *URL:* `%s`", 
        utils.EscapeMarkdownV2(scheduledAt.Format("Mon, 02 Jan 2006 15:04")),
        utils.EscapeMarkdownV2(url))
    msg := tgbotapi.NewMessage(message.Chat.ID, text)
    msg.ParseMode = tgbotapi.ModeMarkdownV2
    _, _ = s.Bot.Send(msg)
}
```

Note: This requires a `ParseScheduleFlags` function in the utils package. We need to add it. Alternatively, we can extend the existing `ParseFlags` to also return `atTime`.

Looking at existing `ParseFlags` signature (`pkg/utils/flags.go`), it returns: `(url string, zip bool, unzip bool, password string, quality string, name string, subs string, hardsub bool)`. 

Add a new function `ParseScheduleFlags` that extends `ParseFlags`:
```go
func ParseScheduleFlags(args string) (url string, zip bool, unzip bool, password string, quality string, name string, subs string, hardsub bool, atTime string) {
    url, zip, unzip, password, quality, name, subs, hardsub = ParseFlags(args)
    atTime = extractFlag(args, "-at")
    return
}
```

Add to `pkg/utils/flags.go`.

- [ ] **Step 2: Register /schedule command in main.go**

In `cmd/zee-mirror/main.go`, add to `setupAdminRoutes`:
```go
r.RegisterCommand("schedule", func(s *service.BotService, m *tgbotapi.Message) {
    download.HandleSchedule(s, m, m.CommandArguments())
})
```

- [ ] **Step 3: Commit**

```bash
git add handlers/download/schedule.go pkg/utils/flags.go cmd/zee-mirror/main.go
git commit -m "feat(schedule): add /schedule command handler"
```

---

## Feature 9: Better Completion Notifications

### Task 9.1: Add MD5 field to domain

**Files:**
- Modify: `internal/domain/task.go`

- [ ] **Step 1: Add MD5 field**

In `internal/domain/task.go`, add `MD5 string` field to `Task` struct (after `OrigFileName`):

```go
MD5 string
```

And to `TaskSnapshot` (after `OrigFileName`):
```go
MD5 string
```

And to `GetSnapshot()`:
```go
MD5: t.MD5,
```

- [ ] **Step 2: Add MD5 field to TaskRecord**

In `internal/domain/task.go`, add `MD5 string` to `TaskRecord` struct.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/task.go
git commit -m "feat(notif): add MD5 field to Task domain types"
```

### Task 9.2: Calculate MD5 after download

**Files:**
- Modify: `internal/service/task_processor.go`

- [ ] **Step 1: Add MD5 calculation in HandlePostDownload**

In `HandlePostDownload` function (in `task_processor.go`), after the downloaded file is found (where `task.LocalPath` and `task.FileName` are set), add:

```go
// Calculate MD5 checksum
if task.LocalPath != "" {
    if info, err := os.Stat(task.LocalPath); err == nil && !info.IsDir() {
        if md5sum, err := utils.CalculateMD5(task.LocalPath); err == nil {
            task.MD5 = md5sum
        }
    }
}
```

Add `"zee-mirror/pkg/utils"` import if not present (should already be there).

This requires a `CalculateMD5` function in `pkg/utils`. Add to `pkg/utils/archive.go` or a new file `pkg/utils/crypto.go`:

```go
func CalculateMD5(filePath string) (string, error) {
    f, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := md5.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", err
    }

    return fmt.Sprintf("%x", h.Sum(nil)), nil
}
```

Requires `os`, `io`, `crypto/md5`, `fmt` imports.

- [ ] **Step 2: Commit**

```bash
git add internal/service/task_processor.go pkg/utils/crypto.go
git commit -m "feat(notif): calculate MD5 checksum after download"
```

### Task 9.3: Add MD5 and streaming link to completion messages

**Files:**
- Modify: `internal/service/templates.go`
- Modify: `internal/service/task_status.go`

- [ ] **Step 1: Update buildTaskStatusText**

In `buildTaskStatusText` in `templates.go`, add MD5 and streaming link to the completion message. Find the `StatusCompleted` case and add:

```go
// After the duration line, add MD5 if available
if snapshot.MD5 != "" {
    text += fmt.Sprintf("\n🔐 *MD5:* `%s`", snapshot.MD5)
}
// Add streaming/download link
if snapshot.RemoteURL != "" {
    text += fmt.Sprintf("\n🌐 *Link:* [Open](%s)", snapshot.RemoteURL)
}
```

- [ ] **Step 2: Update FormatTaskProfessional (dashboard)**

In `FormatTaskProfessional`, add MD5 display if present. After the size line, add:
```go
if taskSnapshot.MD5 != "" {
    text += fmt.Sprintf("\n🔐 *MD5:* `%s`", taskSnapshot.MD5)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/service/templates.go internal/service/task_status.go
git commit -m "feat(notif): show MD5 checksum and streaming link in completion"
```

### Task 9.4: Persist MD5 to database

**Files:**
- Modify: `internal/database/database.go`
- Modify: `migrations/000007_add_md5_to_tasks.up.sql` (new)
- Modify: `migrations/000007_add_md5_to_tasks.down.sql` (new)

- [ ] **Step 1: Create migration**

`migrations/000007_add_md5_to_tasks.up.sql`:
```sql
ALTER TABLE tasks ADD COLUMN md5 TEXT DEFAULT '';
```

`migrations/000007_add_md5_to_tasks.down.sql`:
```sql
ALTER TABLE tasks DROP COLUMN md5;
```

Note: SQLite doesn't support `DROP COLUMN` before version 3.35.0. Since the bot uses modernc.org/sqlite (which is CGo-free but newer), this should work. If not, use:
```sql
ALTER TABLE tasks ADD COLUMN md5 TEXT DEFAULT '';
```
(The down migration can be empty since adding a column is safe.)

- [ ] **Step 2: Update Save query in database.go**

In `database.go`, find the `Save` method and add `md5` to the INSERT/UPDATE query. The column must be added to the insert statement.

Find the existing SQL in Save/GetActive/GetTaskByID and add `md5` column.

Search for the existing INSERT in the Save method and add `md5` after the existing columns.

- [ ] **Step 3: Commit**

```bash
git add migrations/ internal/database/database.go
git commit -m "feat(notif): persist MD5 to database"
```

---

## Feature 10: Task History Dashboard Page

### Task 10.1: Add GetTaskHistory API endpoint

**Files:**
- Modify: `internal/database/database.go`
- Modify: `internal/repository/task.go`
- Modify: `internal/api/api.go`

- [ ] **Step 1: Add repository method**

In `internal/repository/task.go`, add to `TaskRepository`:
```go
GetTaskHistory(ctx context.Context, filter map[string]interface{}) ([]domain.TaskRecord, int, error)
```

- [ ] **Step 2: Implement on database.go**

Add to `database.go`:
```go
func (db *DB) GetTaskHistory(ctx context.Context, filter map[string]interface{}) ([]domain.TaskRecord, int, error) {
    page := 1
    limit := 20
    if p, ok := filter["page"].(int); ok { page = p }
    if l, ok := filter["limit"].(int); ok { limit = l }

    where := []string{"1=1"}
    args := []interface{}{}

    if userID, ok := filter["user_id"].(int64); ok && userID > 0 {
        where = append(where, "user_id = ?")
        args = append(args, userID)
    }
    if status, ok := filter["status"].(string); ok && status != "" {
        where = append(where, "status = ?")
        args = append(args, status)
    }
    if from, ok := filter["from"].(string); ok && from != "" {
        where = append(where, "created_at >= ?")
        args = append(args, from)
    }
    if to, ok := filter["to"].(string); ok && to != "" {
        where = append(where, "created_at <= ?")
        args = append(args, to)
    }

    whereClause := strings.Join(where, " AND ")

    // Count total
    var total int
    countQuery := "SELECT COUNT(*) FROM tasks WHERE " + whereClause
    if err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
        return nil, 0, err
    }

    // Fetch page
    offset := (page - 1) * limit
    query := "SELECT id, gid, type, status, url, file_name, local_path, remote_path, remote_url, total_size, downloaded_size, uploaded_size, chat_id, user_id, created_at, completed_at, zip, unzip, password, error, retries, quality, COALESCE(md5, '') FROM tasks WHERE " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
    args = append(args, limit, offset)

    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()

    var tasks []domain.TaskRecord
    for rows.Next() {
        var t domain.TaskRecord
        if err := rows.Scan(&t.ID, &t.GID, &t.Type, &t.Status, &t.URL, &t.FileName, &t.LocalPath, &t.RemotePath, &t.RemoteURL, &t.TotalSize, &t.DownloadedSize, &t.UploadedSize, &t.ChatID, &t.UserID, &t.CreatedAt, &t.CompletedAt, &t.Zip, &t.Unzip, &t.Password, &t.Error, &t.RetryCount, &t.Quality, &t.MD5); err != nil {
            return nil, 0, err
        }
        tasks = append(tasks, t)
    }

    return tasks, total, rows.Err()
}
```

Add imports for `"strings"` at the top of `database.go`.

- [ ] **Step 3: Add API handler and route**

In `internal/api/api.go`, add a new handler:

```go
func (s *Server) handleTaskHistory(w http.ResponseWriter, r *http.Request) {
    filter := map[string]interface{}{}

    if p := r.URL.Query().Get("page"); p != "" {
        if n, err := strconv.Atoi(p); err == nil { filter["page"] = n }
    }
    if l := r.URL.Query().Get("limit"); l != "" {
        if n, err := strconv.Atoi(l); err == nil { filter["limit"] = n }
    }
    if u := r.URL.Query().Get("user_id"); u != "" {
        if n, err := strconv.ParseInt(u, 10, 64); err == nil { filter["user_id"] = n }
    }
    if s := r.URL.Query().Get("status"); s != "" { filter["status"] = s }
    if f := r.URL.Query().Get("from"); f != "" { filter["from"] = f }
    if t := r.URL.Query().Get("to"); t != "" { filter["to"] = t }

    tasks, total, err := s.Service.DB.GetTaskHistory(r.Context(), filter)
    if err != nil {
        http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
        return
    }

    page := 1
    limit := 20
    if p, ok := filter["page"].(int); ok { page = p }
    if l, ok := filter["limit"].(int); ok { limit = l }

    pages := (total + limit - 1) / limit

    resp := map[string]interface{}{
        "tasks": tasks,
        "total": total,
        "page":  page,
        "limit": limit,
        "pages": pages,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

Add `"strconv"` to imports if not present.

Register route in `Start()`:
```go
mux.HandleFunc("/api/tasks/history", auth(s.handleTaskHistory))
```

Place it before `/api/tasks` or after, ensuring no route conflict.

- [ ] **Step 4: Commit**

```bash
git add internal/repository/task.go internal/database/database.go internal/api/api.go
git commit -m "feat(api): add GET /api/tasks/history with pagination and filters"
```

### Task 10.2: Create TaskHistory frontend page

**Files:**
- Create: `dashboard/src/pages/TaskHistory.tsx`
- Create: `dashboard/src/hooks/useTaskHistory.ts`
- Modify: `dashboard/src/App.tsx`
- Modify: `dashboard/src/types.ts`

- [ ] **Step 1: Add types**

In `dashboard/src/types.ts`, add:
```typescript
export interface TaskRecord {
    id: string;
    gid: string;
    type: string;
    status: string;
    url: string;
    file_name: string;
    local_path: string;
    remote_path: string;
    remote_url: string;
    total_size: number;
    downloaded_size: number;
    uploaded_size: number;
    chat_id: number;
    user_id: number;
    created_at: string;
    completed_at: string | null;
    zip: boolean;
    unzip: boolean;
    password: string;
    error: string;
    retries: number;
    quality: string;
    md5: string;
}

export interface TaskHistoryResponse {
    tasks: TaskRecord[];
    total: number;
    page: number;
    limit: number;
    pages: number;
}
```

- [ ] **Step 2: Create useTaskHistory hook**

`dashboard/src/hooks/useTaskHistory.ts`:
```typescript
import { useState, useCallback } from 'react';
import axios from 'axios';
import { TaskHistoryResponse } from '../types';

export function useTaskHistory() {
    const [data, setData] = useState<TaskHistoryResponse | null>(null);
    const [loading, setLoading] = useState(false);

    const fetch = useCallback(async (params: Record<string, string | number>) => {
        setLoading(true);
        try {
            const token = localStorage.getItem('api_token');
            const query = new URLSearchParams();
            Object.entries(params).forEach(([k, v]) => {
                if (v !== undefined && v !== '' && v !== null) {
                    query.set(k, String(v));
                }
            });
            const res = await axios.get(`/api/tasks/history?${query.toString()}`, {
                headers: { 'X-API-Key': token },
            });
            setData(res.data);
        } catch (e) {
            console.error('Failed to fetch task history', e);
        } finally {
            setLoading(false);
        }
    }, []);

    return { data, loading, fetch };
}
```

- [ ] **Step 3: Create TaskHistory page component**

`dashboard/src/pages/TaskHistory.tsx`:
```tsx
import { useState, useEffect } from 'react';
import { useTaskHistory } from '../hooks/useTaskHistory';

export default function TaskHistory() {
    const { data, loading, fetch } = useTaskHistory();
    const [page, setPage] = useState(1);
    const [status, setStatus] = useState('');
    const [userId, setUserId] = useState('');
    const limit = 20;

    useEffect(() => {
        fetch({ page, limit, status, user_id: userId });
    }, [page, status, userId, fetch]);

    const formatBytes = (b: number) => {
        if (b === 0) return '0 B';
        const units = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(b) / Math.log(1024));
        return (b / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
    };

    const statusColor = (s: string) => {
        const colors: Record<string, string> = {
            completed: 'text-green-400', failed: 'text-red-400',
            cancelled: 'text-yellow-400', downloading: 'text-blue-400',
            uploading: 'text-purple-400', queued: 'text-gray-400',
        };
        return colors[s] || 'text-gray-400';
    };

    return (
        <div>
            <h1 className="text-2xl font-bold mb-6">Task History</h1>

            <div className="flex gap-4 mb-6">
                <input
                    type="text"
                    placeholder="Filter by User ID"
                    value={userId}
                    onChange={e => { setUserId(e.target.value); setPage(1); }}
                    className="bg-gray-800 rounded px-3 py-2 w-48"
                />
                <select
                    value={status}
                    onChange={e => { setStatus(e.target.value); setPage(1); }}
                    className="bg-gray-800 rounded px-3 py-2"
                >
                    <option value="">All Status</option>
                    <option value="completed">Completed</option>
                    <option value="failed">Failed</option>
                    <option value="cancelled">Cancelled</option>
                    <option value="downloading">Downloading</option>
                    <option value="uploading">Uploading</option>
                    <option value="queued">Queued</option>
                </select>
                <span className="text-gray-400 self-center">
                    {data ? `${data.total} total tasks` : ''}
                </span>
            </div>

            {loading ? (
                <div className="text-gray-400">Loading...</div>
            ) : data ? (
                <>
                    <div className="overflow-x-auto">
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="text-gray-400 border-b border-gray-700">
                                    <th className="text-left py-2 px-2">ID</th>
                                    <th className="text-left py-2 px-2">Type</th>
                                    <th className="text-left py-2 px-2">Status</th>
                                    <th className="text-left py-2 px-2">File</th>
                                    <th className="text-right py-2 px-2">Size</th>
                                    <th className="text-left py-2 px-2">User</th>
                                    <th className="text-left py-2 px-2">Created</th>
                                </tr>
                            </thead>
                            <tbody>
                                {data.tasks.map(t => (
                                    <tr key={t.id} className="border-b border-gray-800 hover:bg-gray-800/50">
                                        <td className="py-2 px-2 font-mono text-xs">{t.id}</td>
                                        <td className="py-2 px-2">{t.type}</td>
                                        <td className={`py-2 px-2 ${statusColor(t.status)}`}>{t.status}</td>
                                        <td className="py-2 px-2 max-w-xs truncate">{t.file_name || t.url}</td>
                                        <td className="py-2 px-2 text-right">{formatBytes(t.total_size)}</td>
                                        <td className="py-2 px-2">{t.user_id}</td>
                                        <td className="py-2 px-2 text-xs">{new Date(t.created_at).toLocaleString()}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>

                    <div className="flex justify-center gap-2 mt-6">
                        <button
                            onClick={() => setPage(p => Math.max(1, p - 1))}
                            disabled={page <= 1}
                            className="px-4 py-2 bg-gray-700 rounded disabled:opacity-50"
                        >
                            Previous
                        </button>
                        <span className="self-center text-gray-400">
                            Page {data.page} of {data.pages}
                        </span>
                        <button
                            onClick={() => setPage(p => Math.min(data.pages, p + 1))}
                            disabled={page >= data.pages}
                            className="px-4 py-2 bg-gray-700 rounded disabled:opacity-50"
                        >
                            Next
                        </button>
                    </div>
                </>
            ) : (
                <div className="text-gray-500">Click "Fetch" to load task history</div>
            )}
        </div>
    );
}
```

- [ ] **Step 4: Add route and sidebar link in App.tsx**

In `dashboard/src/App.tsx`, add import for `TaskHistory`:
```tsx
import TaskHistory from './pages/TaskHistory';
```

Add route:
```tsx
{
    path: "/tasks/history",
    element: (
        <ProtectedRoute>
            <DashboardLayout />
        </ProtectedRoute>
    ),
    children: [{ index: true, element: <TaskHistory /> }],
},
```

Add sidebar link in `Sidebar.tsx`:
```tsx
{ label: "Task History", path: "/tasks/history", icon: <Clock size={18} /> },
```

- [ ] **Step 5: Commit**

```bash
git add dashboard/src/pages/TaskHistory.tsx dashboard/src/hooks/useTaskHistory.ts dashboard/src/App.tsx dashboard/src/types.ts
git commit -m "feat(dashboard): add Task History page with filters and pagination"
```

---

## Feature 11: Multi-Destination Upload

### Task 11.1: Add Dest2 parsing and domain field

**Files:**
- Modify: `pkg/utils/flags.go`
- Modify: `internal/domain/task.go`

- [ ] **Step 1: Add -dest2 flag parsing**

In `pkg/utils/flags.go`, extend `ParseFlags` to return `dest2`:

Change `ParseFlags` return to:
```go
func ParseFlags(args string) (url string, zip bool, unzip bool, password string, quality string, name string, subs string, hardsub bool) {
```
Add a new function (don't change existing to avoid breaking callers):
```go
func ParseDest2Flag(args string) string {
    return extractFlag(args, "-dest2")
}
```

- [ ] **Step 2: Add Dest2 to domain.Task**

In `internal/domain/task.go`, add to `Task` struct:
```go
Dest2 string
```

Add to `TaskSnapshot`:
```go
Dest2 string
```

Add to `GetSnapshot()`:
```go
Dest2: t.Dest2,
```

- [ ] **Step 3: Commit**

```bash
git add pkg/utils/flags.go internal/domain/task.go
git commit -m "feat(multidest): add Dest2 field and flag parsing"
```

### Task 11.2: Add UploadToCustomDest to RcloneUploader

**Files:**
- Modify: `internal/uploader/rclone.go`

- [ ] **Step 1: Add method to RcloneUploader**

In `rclone.go`, add:
```go
func (r *RcloneUploader) UploadToCustomDest(ctx context.Context, task *domain.Task, dest string, onProgress func(ProgressUpdate)) error {
    origDest := r.cfg.RcloneDest
    r.cfg.RcloneDest = dest
    defer func() { r.cfg.RcloneDest = origDest }()
    return r.Upload(ctx, task, onProgress)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/uploader/rclone.go
git commit -m "feat(multidest): add UploadToCustomDest method"
```

### Task 11.3: Implement parallel upload in UploadWithRclone

**Files:**
- Modify: `internal/service/upload.go`

- [ ] **Step 1: Add parallel upload logic**

Replace `UploadWithRclone` with:
```go
func (s *BotService) UploadWithRclone(task *Task) error {
    task.SetStatus(StatusUploading)
    task.SetProgress(0)
    s.updateTaskStatus(task)

    err := s.RcloneUploader.Upload(task.Ctx, &task.Task, func(up uploader.ProgressUpdate) {
        task.UpdateFromUploadProgress(up)
        s.updateTaskStatus(task)
    })

    if err != nil {
        return err
    }

    task.SetProgress(100)

    // Parallel upload to second destination
    if task.Dest2 != "" {
        go func() {
            slog.Info("Starting parallel upload to second destination", "taskID", task.ID, "dest2", task.Dest2)
            dest2Err := s.RcloneUploader.UploadToCustomDest(task.Ctx, &task.Task, task.Dest2, func(up uploader.ProgressUpdate) {
                task.Update(func() {
                    task.Progress = 50 + up.Progress/2
                })
                s.updateTaskStatus(task)
            })
            if dest2Err != nil {
                slog.Error("Second destination upload failed", "taskID", task.ID, "error", dest2Err)
            } else {
                slog.Info("Second destination upload completed", "taskID", task.ID)
            }
        }()
    }

    return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/upload.go
git commit -m "feat(multidest): implement parallel upload to second destination"
```

### Task 11.4: Wire Dest2 from command args to task

**Files:**
- Modify: `handlers/download/mirror.go` (or wherever tasks are created with args)

- [ ] **Step 1: Find where tasks are created and add Dest2**

Look at where `CreateTask` is called (e.g., in `HandleMirror`, `HandleLeech`, etc.). After parsing flags with `utils.ParseFlags`, add:

```go
dest2 := utils.ParseDest2Flag(args)
if dest2 != "" {
    task.Dest2 = dest2
}
```

The exact file depends on the handler. Find the pattern - tasks are created via `s.TaskManager.CreateTask(...)` in handlers like `mirror.go`, `leech.go`, etc.

- [ ] **Step 2: Commit**

```bash
git add handlers/download/mirror.go
git commit -m "feat(multidest): wire Dest2 from command args to task"
```

---

## Feature 12: Auto-Update yt-dlp

### Task 12.1: Add /api/tools/update endpoint

**Files:**
- Modify: `internal/api/api.go`

- [ ] **Step 1: Add handler and route**

In `internal/api/api.go`, add handler:

```go
func (s *Server) handleToolsUpdate(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "pip", "install", "-U", "yt-dlp")
    output, err := cmd.CombinedOutput()

    w.Header().Set("Content-Type", "application/json")
    if err != nil {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": false,
            "output":  string(output),
            "error":   err.Error(),
        })
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "output":  string(output),
    })
}
```

Register route in `Start()`:
```go
mux.HandleFunc("/api/tools/update", auth(s.handleToolsUpdate))
```

- [ ] **Step 2: Commit**

```bash
git add internal/api/api.go
git commit -m "feat(api): add POST /api/tools/update endpoint for yt-dlp upgrade"
```

### Task 12.2: Add update button to Settings page

**Files:**
- Modify: `dashboard/src/pages/Settings.tsx`

- [ ] **Step 1: Add update button with output display**

In `Settings.tsx`, add a section after existing settings content:

```tsx
import { useState } from 'react';
import axios from 'axios';

// Inside component:
const [updateOutput, setUpdateOutput] = useState('');
const [updating, setUpdating] = useState(false);

const handleUpdateYTDLP = async () => {
    setUpdateOutput('');
    setUpdating(true);
    try {
        const token = localStorage.getItem('api_token');
        const res = await axios.post('/api/tools/update', {}, {
            headers: { 'X-API-Key': token },
        });
        setUpdateOutput(res.data.output || 'No output');
        if (res.data.error) {
            setUpdateOutput(prev => prev + '\nError: ' + res.data.error);
        }
    } catch (e: any) {
        setUpdateOutput('Error: ' + (e.response?.data?.error || e.message));
    } finally {
        setUpdating(false);
    }
};

// In the JSX, add a section:
<div className="bg-gray-800 rounded-lg p-6">
    <h2 className="text-lg font-semibold mb-4">Tools</h2>
    <button
        onClick={handleUpdateYTDLP}
        disabled={updating}
        className="px-4 py-2 bg-blue-600 rounded hover:bg-blue-700 disabled:opacity-50"
    >
        {updating ? 'Updating...' : 'Update yt-dlp'}
    </button>
    {updateOutput && (
        <pre className="mt-4 bg-gray-900 rounded p-4 text-sm overflow-auto max-h-64">
            {updateOutput}
        </pre>
    )}
</div>
```

- [ ] **Step 2: Commit**

```bash
git add dashboard/src/pages/Settings.tsx
git commit -m "feat(dashboard): add yt-dlp update button to Settings page"
```
