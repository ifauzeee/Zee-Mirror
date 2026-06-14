# Leech File Link & Quota Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Telegram file download link to leech task completion messages, and display per-user quota usage in the dashboard.

**Architecture:** Two independent features. Feature 1 captures the Telegram file path during leech upload and exposes a "📥 Download File" button in the completion message. Feature 2 expands `/api/users` with today's `usedTasks`/`usedBandwidth` and displays them in the Users dashboard page.

**Tech Stack:** Go (backend), React/TypeScript/Tailwind (frontend)

---

## File Map

| File | Change |
|------|--------|
| `internal/domain/task.go` | Add `TelegramFileID`/`TelegramFilePath` to `Task` and `TaskSnapshot` |
| `internal/service/task_manager.go` | Extend `CompleteTelegramUpload()` signature to accept fileID/filePath |
| `internal/service/upload.go` | After `bot.Send()`, call `bot.GetFile()` to capture file path |
| `internal/service/task_status.go` | Add helper `addLeechDownloadButton()` and `isLeechType()`. Wire into `sendFinalMessage()` |
| `internal/service/templates.go` | Add helper `buildTelegramFileURL()` |
| `internal/api/api.go` | Expand `handleGetUsers()` to include `usedTasks`/`usedBandwidth` |
| `dashboard/src/types.ts` | Add `usedTasks`/`usedBandwidth` to `User` interface |
| `dashboard/src/pages/Users.tsx` | Display `used / limit` columns with progress bars |

---

### Task 1: Add TelegramFileID/TelegramFilePath to domain

**Files:**
- Modify: `internal/domain/task.go`

- [ ] **Step 1: Add fields to Task struct** (after `RemoteURL string` at line 57)
```go
TelegramFileID   string
TelegramFilePath string
```

- [ ] **Step 2: Add fields to TaskSnapshot struct** (after `RemoteURL string` at line 149)
```go
TelegramFileID   string `json:"telegramFileID,omitempty"`
TelegramFilePath string `json:"telegramFilePath,omitempty"`
```

- [ ] **Step 3: Populate in GetSnapshot()** (after `RemoteURL: t.RemoteURL`)
```go
TelegramFileID:   t.TelegramFileID,
TelegramFilePath: t.TelegramFilePath,
```

- [ ] **Step 4: Commit**
```bash
git add internal/domain/task.go
git commit -m "feat: add TelegramFileID and TelegramFilePath to task domain"
```

---

### Task 2: Capture Telegram file path on leech upload

**Files:**
- Modify: `internal/service/task_manager.go:783-790`
- Modify: `internal/service/upload.go:83-92`

- [ ] **Step 1: Extend CompleteTelegramUpload signature**
```go
func (t *Task) CompleteTelegramUpload(msgID int, uploadedSize int64, fileID, filePath string) {
	t.Update(func() {
		t.ResultMessageID = msgID
		t.Progress = 100
		t.UploadedSize = uploadedSize
		t.RemotePath = "telegram"
		t.TelegramFileID = fileID
		t.TelegramFilePath = filePath
	})
}
```

- [ ] **Step 2: Capture file info in UploadToTelegram** (replace lines 83-92)
```go
	sentMsg, err := s.Bot.Send(msg)
	if err != nil {
		metrics.UploadDuration.WithLabelValues("telegram", "failed").Observe(time.Since(startTime).Seconds())
		return fmt.Errorf("%w: telegram upload failed: %v", domain.ErrExternal, err)
	}
	metrics.UploadDuration.WithLabelValues("telegram", "success").Observe(time.Since(startTime).Seconds())

	var fileID, filePath string
	if sentMsg.Document != nil {
		fileID = sentMsg.Document.FileID
	} else if sentMsg.Video != nil {
		fileID = sentMsg.Video.FileID
	}
	if fileID != "" {
		if tgFile, err := s.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID}); err == nil {
			filePath = tgFile.FilePath
		}
	}

	task.CompleteTelegramUpload(sentMsg.MessageID, info.Size(), fileID, filePath)
```

- [ ] **Step 3: Commit**
```bash
git add internal/service/task_manager.go internal/service/upload.go
git commit -m "feat: capture Telegram file path on leech upload"
```

---

### Task 3: Add download file button in completion message

**Files:**
- Modify: `internal/service/templates.go`
- Modify: `internal/service/task_status.go`

- [ ] **Step 1: Add buildTelegramFileURL in templates.go** (after line 273)
```go
func buildTelegramFileURL(botToken, filePath string) string {
	return "https://api.telegram.org/file/bot" + botToken + "/" + filePath
}
```

- [ ] **Step 2: Add helpers in task_status.go** (after imports)
```go
func isLeechType(t domain.TaskType) bool {
	return t == domain.TypeLeech || t == domain.TypeYTDLPLeech
}

func addLeechDownloadButton(keyboard tgbotapi.InlineKeyboardMarkup, snapshot domain.TaskSnapshot, botToken string) tgbotapi.InlineKeyboardMarkup {
	if isLeechType(snapshot.Type) && snapshot.TelegramFilePath != "" {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📥 Download File", buildTelegramFileURL(botToken, snapshot.TelegramFilePath)),
		)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}
	return keyboard
}
```

- [ ] **Step 3: Wire into sendFinalMessage — edit caption path** (after keyboard creation, before `send`)
```go
			keyboard = addLeechDownloadButton(keyboard, snapshot, s.Bot.Token)
```

- [ ] **Step 4: Wire into sendFinalMessage — edit text path** (same location pattern)
```go
			keyboard = addLeechDownloadButton(keyboard, snapshot, s.Bot.Token)
```

- [ ] **Step 5: Wire into sendFinalMessage — new message path** (after keyboard creation inside `if` block)
```go
			keyboard = addLeechDownloadButton(*keyboard, snapshot, s.Bot.Token)
```

Note: In the new message path, `msg.ReplyMarkup` is `*tgbotapi.InlineKeyboardMarkup`, so this dereferences the pointer.

- [ ] **Step 6: Build and verify**
```bash
go build ./...
```
Expected: no errors

- [ ] **Step 7: Commit**
```bash
git add internal/service/task_status.go internal/service/templates.go
git commit -m "feat: add download file button in leech task completion messages"
```

---

### Task 4: Expand /api/users with usedTasks/usedBandwidth

**Files:**
- Modify: `internal/api/api.go:850-863`

- [ ] **Step 1: Replace handleGetUsers**
```go
func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := s.Service.DB.GetAll(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	type userWithStats struct {
		domain.User
		UsedTasks     int   `json:"usedTasks"`
		UsedBandwidth int64 `json:"usedBandwidth"`
	}

	var result []userWithStats
	for _, u := range users {
		stats, _ := s.Service.DB.GetUserTodayStats(ctx, u.ID)
		result = append(result, userWithStats{
			User:          u,
			UsedTasks:     stats.TotalTasks,
			UsedBandwidth: stats.TotalBandwidth,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("Failed to encode users response", "error", err)
	}
}
```

- [ ] **Step 2: Build and verify**
```bash
go build ./...
```
Expected: no errors

- [ ] **Step 3: Commit**
```bash
git add internal/api/api.go
git commit -m "feat: add usedTasks and usedBandwidth to user API response"
```

---

### Task 5: Update frontend types and Users page

**Files:**
- Modify: `dashboard/src/types.ts`
- Modify: `dashboard/src/pages/Users.tsx`

- [ ] **Step 1: Add usedTasks/usedBandwidth to User type**
```tsx
export interface User {
    id: number
    username: string
    role: 'admin' | 'user' | 'owner' | 'authorized'
    status: 'active' | 'banned'
    maxDailyTasks: number
    maxDailyBandwidth: number
    expiresAt?: { Valid: boolean; Time: string }
    usedTasks?: number
    usedBandwidth?: number
}
```

- [ ] **Step 2: Replace Load Limits column in Users.tsx** (lines 217-228)

Replace with:
```tsx
                    <td className="px-8 py-6">
                      <div className="flex flex-col items-center gap-1">
                        {user.maxDailyTasks === -1 ? (
                          <span className="text-[10px] font-bold text-slate-600 dark:text-slate-300">
                            Tasks: {user.usedTasks ?? 0} / ∞
                          </span>
                        ) : (
                          <div className="flex flex-col items-center gap-0.5 w-full">
                            <span className="text-[10px] font-bold text-slate-600 dark:text-slate-300">
                              Tasks: {user.usedTasks ?? 0} / {user.maxDailyTasks}
                            </span>
                            <div className="w-24 h-1.5 bg-slate-200 dark:bg-white/10 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-primary rounded-full transition-all"
                                style={{ width: `${Math.min(100, ((user.usedTasks ?? 0) / user.maxDailyTasks) * 100)}%` }}
                              />
                            </div>
                          </div>
                        )}
                        {user.maxDailyBandwidth === -1 ? (
                          <span className="text-[10px] font-bold text-slate-400">
                            BW: {formatBytes(user.usedBandwidth ?? 0)} / ∞
                          </span>
                        ) : (
                          <div className="flex flex-col items-center gap-0.5 w-full">
                            <span className="text-[10px] font-bold text-slate-400">
                              BW: {formatBytes(user.usedBandwidth ?? 0)} / {formatBytes(user.maxDailyBandwidth)}
                            </span>
                            <div className="w-24 h-1.5 bg-slate-200 dark:bg-white/10 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-indigo-500 rounded-full transition-all"
                                style={{ width: `${Math.min(100, ((user.usedBandwidth ?? 0) / user.maxDailyBandwidth) * 100)}%` }}
                              />
                            </div>
                          </div>
                        )}
                      </div>
                    </td>
```

- [ ] **Step 3: Commit**
```bash
git add dashboard/src/types.ts dashboard/src/pages/Users.tsx
git commit -m "feat: show used task and bandwidth quota in Users page"
```

---

---

### Task 6: Add quota usage overview card

**Files:**
- Modify: `dashboard/src/pages/Overview.tsx`
- Modify: `dashboard/src/hooks/useUsers.ts`

The `/api/users` endpoint now returns `usedTasks`/`usedBandwidth`. Fetch all users in Overview and show a summary card.

- [ ] **Step 1: Add useUsers to Overview** (pass `apiToken` prop)

Update `OverviewProps` and add a fetch for users data at the top of the component.

- [ ] **Step 2: Add quota summary card after the 4 stats cards**

After the existing StatsCard grid (after line 58), add:
```tsx
      {/* Quota summary card */}
      <div className="bg-white/80 dark:bg-zinc-900/60 backdrop-blur-xl rounded-3xl border border-slate-200 dark:border-white/5 shadow-2xl shadow-black/5 p-8">
        <h3 className="text-lg font-black text-slate-900 dark:text-white tracking-tight mb-6">
          Today's Quota Usage
        </h3>
        <div className="space-y-4">
          {users.filter(u => u.maxDailyTasks !== -1 || u.maxDailyBandwidth !== -1).length > 0 ? (
            users.filter(u => u.maxDailyTasks !== -1 || u.maxDailyBandwidth !== -1).map((user) => (
              <div key={user.id} className="space-y-2">
                <div className="flex justify-between text-xs font-bold">
                  <span className="text-slate-600 dark:text-slate-300">{user.username || `User ${user.id}`}</span>
                  <span className="text-slate-400">
                    Tasks: {user.usedTasks ?? 0}/{user.maxDailyTasks === -1 ? '∞' : user.maxDailyTasks}
                    {user.maxDailyBandwidth !== -1 && ` • BW: ${formatBytes(user.usedBandwidth ?? 0)}/${formatBytes(user.maxDailyBandwidth)}`}
                  </span>
                </div>
                {user.maxDailyTasks !== -1 && (
                  <div className="w-full h-2 bg-slate-200 dark:bg-white/10 rounded-full overflow-hidden">
                    <div className="h-full bg-primary rounded-full transition-all" style={{ width: `${Math.min(100, ((user.usedTasks ?? 0) / user.maxDailyTasks) * 100)}%` }} />
                  </div>
                )}
              </div>
            ))
          ) : (
            <p className="text-sm text-slate-400">All users have unlimited quota.</p>
          )}
        </div>
      </div>
```

- [ ] **Step 3: Fetch users in Overview**

The Overview component needs `apiToken` prop. Update usage in `App.tsx` and add a fetch for users.

In `App.tsx`, pass `apiToken` to Overview:
```tsx
<Overview
  tasks={tasks}
  stats={stats}
  system={system}
  onCancelTask={cancelTask}
  setActiveTab={() => {}}
  apiToken={apiToken}
/>
```

In `Overview.tsx`, add the fetch:
```tsx
import { useState, useEffect } from 'react'
import { User } from '../types'

// Inside component
const [users, setUsers] = useState<User[]>([])
useEffect(() => {
  if (!apiToken) return
  fetch('/api/users', { headers: { 'X-API-Key': apiToken } })
    .then(res => res.json())
    .then(data => setUsers(data || []))
    .catch(() => {})
}, [apiToken])
```

- [ ] **Step 4: Commit**
```bash
git add dashboard/src/pages/Overview.tsx dashboard/src/App.tsx
git commit -m "feat: add quota usage overview card in dashboard"
```

---

## Summary

| Task | Feature | Files Changed |
|------|---------|--------------|
| 1 | Domain fields | 1 |
| 2 | Capture file path | 2 |
| 3 | Download button | 2 |
| 4 | API expansion | 1 |
| 5 | Frontend Users table | 2 |
| 6 | Frontend Overview card | 2 |
