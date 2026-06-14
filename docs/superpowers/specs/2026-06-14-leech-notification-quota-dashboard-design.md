# Leech File Link Notification & Quota Dashboard

## Feature 1: Telegram File Link untuk Leech Task

### Problem
Leech task mengupload file ke chat via Telegram, tapi tidak ada file link yang bisa disimpan/dibagikan user. Mirror task sudah punya "Cloud Link" button, leech task tidak.

### Solution
Simpan `file_id` dan `file_path` dari response upload Telegram, lalu di `sendFinalMessage()` tambahkan tombol "📥 Download File" yang mengarah ke `https://api.telegram.org/file/bot<token>/<file_path>`.

### Changes

| File | Change |
|------|--------|
| `internal/domain/task.go` | Tambah field `TelegramFileID string` dan `TelegramFilePath string` |
| `internal/service/upload.go` | Di `UploadToTelegram()`, setelah `bot.Send()` sukses, panggil `bot.GetFile()` untuk dapat `file_path`, simpan ke task via `CompleteTelegramUpload()` |
| `internal/service/task_manager.go` | Di `Task.CompleteTelegramUpload()`, simpan `fileID` dan `filePath`. Di `GetSnapshot()`, expose `TelegramFileID` dan `TelegramFilePath` |
| `internal/service/task_status.go` | Di `sendFinalMessage()`, untuk task dengan `TypeLeech`/`TypeYTDLPLeech` dan `TelegramFilePath` tidak kosong, tambahkan inline button "📥 Download File" |
| `internal/service/templates.go` | Tambah helper untuk membangun URL: `https://api.telegram.org/file/bot<botToken>/<filePath>` |

### Data Flow
```
UploadToTelegram()
  → bot.Send(video/document) sukses
  → bot.GetFile(sentMsg.Document.FileID) → dapat file_path
  → task.CompleteTelegramUpload(fileID, filePath)
      → simpan di Task struct
  → updateTaskStatus()
  → sendFinalMessage()
      → jika leech + ada TelegramFilePath
          → tambah button "📥 Download File"
          → URL = api.telegram.org/file/bot<token>/<file_path>
```

---

## Feature 2: Per-User Quota & Bandwidth Dashboard

### Problem
Config memiliki `DEFAULT_MAX_DAILY_TASKS` dan `DEFAULT_MAX_DAILY_BANDWIDTH`, backend menghitung pemakaian via `GetUserTodayStats()`, tapi dashboard tidak menampilkan sisa quota per user.

### Solution
1. Expand `/api/users` response dengan `usedTasks` dan `usedBandwidth`
2. Tambah quota summary di WebSocket broadcast untuk Overview card

### Changes

#### Backend

| File | Change |
|------|--------|
| `internal/api/api.go` | Di `handleGetUsers()`, panggil `GetUserTodayStats()` untuk setiap user, tambah `usedTasks`/`usedBandwidth` ke response JSON |
| `internal/api/api.go` | Di `broadcastLoop()`, tambah field `quota` ke broadcast payload: `{ totalTasksUsed, totalBandwidthUsed }` |

#### Frontend

| File | Change |
|------|--------|
| `dashboard/src/types.ts` | Tambah `usedTasks: number` dan `usedBandwidth: number` ke interface `User`. Tambah `quota` field ke tipe system data |
| `dashboard/src/pages/Users.tsx` | Kolom baru "Tasks" dan "Bandwidth" dengan format `used / limit` |
| `dashboard/src/pages/Overview.tsx` | Card "Quota Usage" — ringkasan pemakaian hari ini |

### Data Flow
```
GET /api/users
  → handleGetUsers()
  → db.GetAllUsers()
  → untuk setiap user: db.GetUserTodayStats(userID)
  → return { id, username, usedTasks, usedBandwidth, maxDailyTasks, maxDailyBandwidth, ... }

WebSocket broadcast (every 1s)
  → broadcastLoop()
  → hitung total tasks/bandwidth dari active tasks
  → tambah field "quota" ke payload
```

## Implementation Order
1. Backend: domain + task_manager (save file_path)
2. Backend: upload.go (get file after send)
3. Backend: task_status.go + templates.go (show button)
4. Backend: api.go (expand /api/users + websocket)
5. Frontend: types.ts
6. Frontend: Users.tsx (new columns)
7. Frontend: Overview.tsx (quota card)
