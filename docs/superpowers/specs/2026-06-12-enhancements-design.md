# Enhancement Features Design Spec

> **Five independent feature enhancements** for Zee-Mirror Telegram bot

---

## Feature 8: TypeScheduled — Task Scheduling

### Tables

Migration `000006_add_scheduled_tasks`:

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

### Scheduler

- Goroutine in `TaskManager` ticks every 60s
- Queries `SELECT * FROM scheduled_tasks WHERE status='pending' AND scheduled_at <= datetime('now')`
- For each match, calls `CreateTask()` with the stored params, updates `status='done'` and `task_id`
- Respects `ShutdownChan` for graceful shutdown

### Bot Command

New `/schedule` command: `-at 02:00` flag parsed by existing `ParseFlags()` (extend with `-at` parser). Stores into `scheduled_tasks`. Responds with confirmation including schedule time.

### Files Changed
- Create: `migrations/000006_add_scheduled_tasks.up.sql`, `.down.sql`
- Create: `internal/repository/scheduled.go` (ScheduledTaskRepository interface)
- Modify: `internal/database/database.go` (implement + add to FullRepository)
- Modify: `internal/database/database.go` (RunMigrations picks up new migration)
- Create: `internal/service/scheduler.go` (scheduler goroutine)
- Modify: `internal/service/task_manager.go` (start scheduler)
- Modify: `handlers/download/schedule.go` (new handler)
- Modify: `cmd/zee-mirror/main.go` (register /schedule command)
- Modify: `pkg/utils/flag.go` (add -at parsing)

---

## Feature 9: Better Completion Notifications

### MD5 Checksum

- After download completes in `HandlePostDownload`, before upload, compute `md5` of the downloaded file
- Store result in `Task.MD5 string` (new field on `domain.Task`)
- Add to `TaskSnapshot` for display
- Display in completion text: `🔐 *MD5:* \`checksum\``

### Streaming Link Enhancement

- Already have `RemoteURL` with `IndexURL` support
- Add clearer formatting: `🌐 *Stream:* [Link](url)` using MarkdownV2
- Add to both `buildTaskStatusText` (final message) and `FormatTaskProfessional` (dashboard)

### Files Changed
- Modify: `internal/domain/task.go` (add MD5 field to Task + TaskSnapshot)
- Modify: `internal/service/templates.go` (add MD5 + stream link to messages)
- Modify: `internal/service/task_processor.go` (calculate MD5 after download)
- Modify: `internal/service/task_status.go` (pass MD5 through)
- Modify: `internal/database/database.go` (add md5 column to save/query)

---

## Feature 10: Task History Dashboard Page

### API Endpoint

`GET /api/tasks/history?page=1&limit=20&user_id=&status=&from=&to=`

Returns:
```json
{
  "tasks": [{...TaskRecord...}],
  "total": 1234,
  "page": 1,
  "limit": 20,
  "pages": 62
}
```

Backend: Add `GetTaskHistory(ctx, filter)` to `TaskRepository` interface and implement on `*DB` with SQL query using `LIKE`, `WHERE`, `ORDER BY created_at DESC`, `LIMIT/OFFSET`.

### Frontend Page

New files:
- `dashboard/src/pages/TaskHistory.tsx` — table with filters
- `dashboard/src/hooks/useTaskHistory.ts` — API call

Add sidebar link + route in `App.tsx`.

### Files Changed
- Modify: `internal/repository/task.go` (add GetTaskHistory method)
- Modify: `internal/database/database.go` (implement query)
- Modify: `internal/api/api.go` (add /api/tasks/history route)
- Create: `dashboard/src/pages/TaskHistory.tsx`
- Create: `dashboard/src/hooks/useTaskHistory.ts`
- Modify: `dashboard/src/App.tsx` (add route + sidebar)
- Modify: `dashboard/src/types.ts` (add TaskHistoryResponse type)

---

## Feature 11: Multi-Destination Upload

### Flag Parsing

Extend `utils.ParseFlags()` to recognize `-dest2 remote2:/path`. Store as `Task.Dest2 string`.

### Domain

Add `Dest2 string` to `domain.Task` and `TaskSnapshot`.

### Upload Logic

In `UploadWithRclone` (`upload.go`):
1. Primary upload proceeds as before
2. If `task.Dest2 != ""`, launch goroutine:
   - Create second `RcloneUploader.Upload()` call with overridden destination
   - On completion, append second `RemoteURL` to task data
   - Set `task.RemoteURL = "Primary: <url>\nSecondary: <url2>"`

### Uploader Enhancement

Add `UploadToCustomDest(ctx, task, dest, onProgress)` to `RcloneUploader` that takes an explicit destination string instead of `cfg.RcloneDest`.

### Files Changed
- Modify: `pkg/utils/flag.go` (add -dest2 parsing)
- Modify: `internal/domain/task.go` (add Dest2 field)
- Modify: `internal/service/upload.go` (add parallel upload logic)
- Modify: `internal/uploader/rclone.go` (add UploadToCustomDest)
- Modify: `internal/service/templates.go` (show both destinations in completion)

---

## Feature 12: Auto-Update yt-dlp

### API Endpoint

`POST /api/tools/update` (auth-protected, admin only):
- Runs `exec.CommandContext(ctx, 30s timeout, "pip", "install", "-U", "yt-dlp")`
- Streams stdout/stderr as newline-delimited JSON
- Returns exit code

Response format:
```
{"line": "Collecting yt-dlp...", "stream": "stdout"}
{"line": "Successfully installed yt-dlp-2024.12.6", "stream": "stdout"}
{"line": "", "stream": "exit", "code": 0}
```

### Frontend

Add "Update yt-dlp" button to Settings page. On click:
1. POST to `/api/tools/update`
2. Read streaming response
3. Display output in a terminal-style `<pre>` block
4. Show success/failure

### Files Changed
- Modify: `internal/api/api.go` (add /api/tools/update route + handler)
- Modify: `dashboard/src/pages/Settings.tsx` (add update button + output display)
