<div align="center">

# 🚀 Zee-Mirror

### High-Performance Telegram Mirror & Leech Bot

*Download anything. Upload anywhere. Control everything.*

<br>

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black)](https://reactjs.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://docker.com)
[![SQLite](https://img.shields.io/badge/SQLite-Embedded-003B57?style=for-the-badge&logo=sqlite&logoColor=white)](https://sqlite.org)
[![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-E6522C?style=for-the-badge&logo=prometheus&logoColor=white)](https://prometheus.io)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)

<br>

[🚀 Quick Start](#-quick-start) •
[✨ Features](#-features-at-a-glance) •
[📖 Commands](#-bot-commands) •
[🌐 Dashboard](#-web-dashboard) •
[📡 API](#-rest-api) •
[🤝 Contributing](#-contributing)

<br>

</div>

---

## 💡 Why Zee-Mirror?

<table>
<tr>
<td width="50%">

### 🏎️ Built for Speed
Written in **Go** — compiles to a single binary with goroutine-powered concurrency. Handles dozens of simultaneous tasks with minimal memory footprint.

### 🧩 All-in-One Solution
Mirror, leech, torrent, video download, media processing, file management, analytics — everything in one bot. No need for multiple tools.

### 🛡️ Production Ready
Role-based access control, daily task/bandwidth limits, health checks, Prometheus metrics, structured logging, task recovery, and graceful shutdown. Built for real-world 24/7 operation.

</td>
<td width="50%">

### 🌐 Modern Web Dashboard
A full React-powered control panel with real-time WebSocket updates, interactive charts, file explorer, torrent browser, and user management — all accessible from your browser.

### 📦 Zero-Config Deploy
Multi-stage Docker build bundles everything — aria2, yt-dlp, rclone, ffmpeg, 7zz, Local Bot API. One command to deploy, fully operational.

### 🔌 Extensible Storage
Google Drive, OneDrive, Mega, S3, Dropbox, and **40+ cloud providers** via rclone. Switch destinations on the fly, manage multiple remotes simultaneously.

</td>
</tr>
</table>

<br>

---

## ✨ Features at a Glance

<div align="center">

| | Feature | Description |
|:---:|---------|-------------|
| 📥 | **Mirror / Leech** | Download from HTTP/HTTPS/FTP and upload to any cloud storage |
| 🧲 | **Torrent Engine** | Magnet links & `.torrent` files with per-file selective download |
| 🎬 | **Video Downloader** | yt-dlp integration — YouTube + 1000 sites with quality picker UI |
| 🤖 | **Userbot (MTProto)** | Download from private Telegram channels & groups via [gotd](https://github.com/gotd/td) |
| ⚔️ | **Viking Upload** | Upload to Viking File storage with optional account binding |
| 🔄 | **Cloud Clone** | Server-side copy between any two cloud remotes via rclone |
| 📋 | **Batch Downloads** | Queue multiple URLs in a single command |
| 🗜️ | **Archive Operations** | Zip/Unzip with password protection powered by 7zz |
| 🎵 | **Media Processing** | Extract audio, compress video, burn subtitles, rescale, convert formats |
| 📂 | **Drive File Manager** | Browse, create, delete, move, search, and share files on Google Drive |
| 💾 | **Multi Storage** | Switch between multiple configured cloud remotes instantly |
| 📊 | **Live Progress** | Real-time progress bars with automatic 5-second refresh |
| 🔄 | **Task Recovery** | Checkpoint-based auto-recovery for interrupted tasks after restart |
| 🚫 | **Duplicate Prevention** | Configurable detection to block duplicate URL downloads |
| 👥 | **User Management** | Roles (owner/admin/user), per-user limits, access expiration |
| 📈 | **Analytics Dashboard** | Daily, weekly, monthly, and per-user statistics with charts |
| 🖥️ | **System Monitor** | CPU, RAM, Disk metrics with custom Telegram alert channels |
| 📡 | **Prometheus Metrics** | Export all metrics to Grafana or external monitoring systems |
| 🏥 | **Health Checks** | Automated Docker health check + `/health` command |
| 📢 | **Channel Logging** | Forward activity logs and alerts to dedicated Telegram channels |
| 🔍 | **Torrent Search** | Search for torrents by keyword directly from Telegram |
| 🌐 | **Speed Test** | Built-in network speed testing via `/speed` command |
| 🌍 | **Internationalization** | User-selectable interface language (i18n support) |
| 📁 | **Smart Organization** | Auto-organize files into categories (Movies, Music, etc.) |
| 🌐 | **Web Dashboard** | Full React control panel with 7 pages and WebSocket real-time updates |

</div>

<br>

---

## 🛠️ Tech Stack

<details open>
<summary><b>🔧 Backend — Go 1.25+</b></summary>
<br>

| Package | Purpose |
|---------|---------|
| [`go-telegram-bot-api/v5`](https://github.com/go-telegram-bot-api/telegram-bot-api) | Telegram Bot API client for handling messages, callbacks, and media |
| [`gotd/td`](https://github.com/gotd/td) | Native MTProto implementation for Userbot features (private channels) |
| [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) | Pure Go SQLite driver — no CGO required, zero external dependencies |
| [`golang-migrate/v4`](https://github.com/golang-migrate/migrate) | Database schema migrations with versioned SQL files |
| [`gopsutil/v3`](https://github.com/shirou/gopsutil) | Cross-platform system metrics (CPU, RAM, Disk, Host info) |
| [`prometheus/client_golang`](https://github.com/prometheus/client_golang) | Prometheus metrics export for Grafana integration |
| [`gorilla/websocket`](https://github.com/gorilla/websocket) | WebSocket hub for pushing real-time updates to the dashboard |
| [`joho/godotenv`](https://github.com/joho/godotenv) | `.env` file loading for local development |
| [`google/uuid`](https://github.com/google/uuid) | Unique task ID generation |
| [`PuerkitoBio/goquery`](https://github.com/PuerkitoBio/goquery) | HTML parsing for torrent search scraping |
| [`cenkalti/backoff`](https://github.com/cenkalti/backoff) | Exponential backoff for retrying failed operations |
| [`stretchr/testify`](https://github.com/stretchr/testify) | Test assertions and mocking |
| `log/slog` *(stdlib)* | Structured logging with configurable levels |
| `golang.org/x/time` | Rate limiting for API requests |

</details>

<details>
<summary><b>🎨 Frontend — React Web Dashboard</b></summary>
<br>

| Package | Purpose |
|---------|---------|
| [React 18](https://reactjs.org) | Component-based UI framework |
| [Vite 4](https://vitejs.dev) | Next-gen build tool with instant HMR |
| [Tailwind CSS 3](https://tailwindcss.com) | Utility-first CSS for rapid UI styling |
| [Recharts 3](https://recharts.org) | Composable data visualization (charts, graphs) |
| [Framer Motion 12](https://www.framer.com/motion/) | Fluid animations and page transitions |
| [Lucide React](https://lucide.dev) | Beautiful, consistent icon library |
| [Axios](https://axios-http.com) | Promise-based HTTP client for API calls |
| [clsx](https://github.com/lukeed/clsx) + [tailwind-merge](https://github.com/dcastil/tailwind-merge) | Dynamic class name utility |
| [ESLint](https://eslint.org) + [Prettier](https://prettier.io) | Code linting and formatting |

</details>

<details>
<summary><b>⚙️ External Tools (Bundled in Docker Image)</b></summary>
<br>

| Tool | Version | Purpose |
|------|---------|---------|
| [aria2c](https://aria2.github.io) | Latest | Multi-connection download engine for HTTP/HTTPS/FTP/BitTorrent with segmented transfer |
| [yt-dlp](https://github.com/yt-dlp/yt-dlp) | Latest | Video/audio downloader supporting YouTube + 1000 websites |
| [rclone](https://rclone.org) | Latest | Cloud storage transfer and management for 40+ providers |
| [ffmpeg](https://ffmpeg.org) | Latest | Media processing — video compression, audio extraction, format conversion, subtitle operations |
| [7zz (p7zip)](https://7-zip.org) | Latest | High-performance archive compression and extraction (zip, 7z, rar, tar…) |
| [Local Bot API](https://github.com/aiogram/telegram-bot-api) | Latest | Self-hosted Telegram Bot API server for handling files larger than 2GB |
| [speedtest-cli](https://github.com/sivel/speedtest-cli) | Latest | Network bandwidth testing |
| [Python 3](https://python.org) | Alpine | Runtime for yt-dlp and speedtest-cli |
| [Node.js 18](https://nodejs.org) | Alpine | Dashboard build toolchain (build-time only) |

</details>

<br>

---

## 📋 Prerequisites

> **💡 You only need Docker installed.** All tools (aria2, yt-dlp, rclone, ffmpeg, 7zz) are bundled inside the container image.

| Requirement | Where to get it |
|-------------|-----------------|
| Docker & Docker Compose | [Install Docker](https://docs.docker.com/get-docker/) |
| Telegram Bot Token | Create a bot via [@BotFather](https://t.me/BotFather) on Telegram |
| Telegram API ID & Hash | Register at [my.telegram.org](https://my.telegram.org) → API Development Tools |
| Rclone remote configured | [Rclone Setup Guide](https://rclone.org/docs/) — run `rclone config` to set up a remote |
| Your Telegram User ID | Send `/start` to [@userinfobot](https://t.me/userinfobot) to find your numeric ID |

<br>

---

## 🚀 Quick Start

> **From zero to running in under 5 minutes.**

### Step 1 — Clone the Repository

```bash
git clone https://github.com/ifauzeee/Zee-Mirror.git
cd Zee-Mirror
```

### Step 2 — Configure Environment Variables

```bash
cp .env.example .env
```

Open `.env` in your editor and fill in the **required** values:

```env
# ─── REQUIRED ──────────────────────────────────────────
BOT_TOKEN=your_bot_token_here           # From @BotFather
OWNER_ID=your_telegram_user_id          # Your numeric Telegram ID
TELEGRAM_API_ID=your_api_id             # From my.telegram.org
TELEGRAM_API_HASH=your_api_hash         # From my.telegram.org

# ─── RECOMMENDED ───────────────────────────────────────
RCLONE_DEST=gdrive:/Zee-Mirror          # Default upload destination
WEB_DASHBOARD_TOKEN=your_secure_pass    # Dashboard login password
WEB_DASHBOARD_URL=http://your-ip        # Your server's public IP/domain
```

> 📝 See the [Full Configuration Reference](#%EF%B8%8F-configuration-reference) for all available variables.

### Step 3 — Set Up Rclone

Place your `rclone.conf` file in the `config/` directory:

```bash
mkdir -p config
cp ~/.config/rclone/rclone.conf config/rclone.conf
```

> 💡 **Don't have rclone configured yet?** Run `rclone config` to create a remote interactively, then copy the generated config file.

### Step 4 — Deploy with Docker

```bash
docker compose up -d --build
```

Monitor the startup logs:

```bash
docker compose logs -f --no-log-prefix zee-mirror
```

✅ **Success!** You should see the Zee-Mirror banner and `Authorized on account @YourBotName`.

### Step 5 — Start Using the Bot

Open your bot on Telegram → Send **`/start`** → The main dashboard appears → You're live! 🎉

<br>

<details>
<summary><b>🤖 Optional: Set Up Userbot (Download from Private Channels)</b></summary>
<br>

If you want the bot to download files from **private channels** or **join groups** via invite links, you need to set up the Userbot feature.

A built-in interactive tool generates the required Telegram session string:

```bash
go run cmd/session-gen/main.go
```

**The wizard will:**

1. Read `APP_ID` and `APP_HASH` from your `.env` file (or prompt you to enter them)
2. Connect to Telegram and request your phone number
3. Send a login code to your Telegram app
4. Ask for 2FA password (if enabled)
5. Generate a base64-encoded session string
6. **Automatically update your `.env` file** with `APP_ID`, `APP_HASH`, and `USER_SESSION_STRING`

After running the tool, restart your bot for the changes to take effect:

```bash
docker compose down && docker compose up -d --build
```

> ⚠️ **Security Warning:** The session string grants **full access** to your Telegram account. Never share it publicly.

</details>

<details>
<summary><b>🖥️ Optional: Deploy Without Docker (Bare Metal)</b></summary>
<br>

**System Requirements:**
- Go 1.25+
- Node.js 18+
- aria2c, yt-dlp, rclone, ffmpeg, 7zz installed on the system
- SQLite (built into Go driver, no separate install needed)

```bash
# 1. Build the dashboard
cd dashboard && npm install && npm run build && cd ..

# 2. Build the Go binary
go mod download
go build -o zee-mirror ./cmd/zee-mirror

# 3. Run
./zee-mirror
```

The binary reads configuration from `.env` in the current directory. Make sure `migrations/` folder is present for database auto-migration.

</details>

<br>

---

## ⚙️ Configuration Reference

All configuration is managed through environment variables in the `.env` file.

<details open>
<summary><b>🔴 Required Settings</b></summary>
<br>

| Variable | Description |
|----------|-------------|
| `BOT_TOKEN` | Telegram Bot API token obtained from [@BotFather](https://t.me/BotFather) |
| `OWNER_ID` | Telegram User ID of the bot owner (numeric). The owner has full admin privileges |
| `TELEGRAM_API_ID` | API ID from [my.telegram.org](https://my.telegram.org). Required for Local Bot API (>2GB files) |
| `TELEGRAM_API_HASH` | API Hash from [my.telegram.org](https://my.telegram.org). Required for Local Bot API (>2GB files) |

</details>

<details>
<summary><b>🌐 Dashboard & Networking</b></summary>
<br>

| Variable | Default | Description |
|----------|---------|-------------|
| `DASHBOARD_PORT` | `80` | External port for the web dashboard (the port you access in your browser) |
| `DASHBOARD_PORT_INTERNAL` | `8080` | Internal container listen port (typically unchanged) |
| `WEB_DASHBOARD_URL` | `http://localhost` | Public URL of the dashboard. Used for generating links in bot messages |
| `WEB_DASHBOARD_TOKEN` | `zee-mirror-secret` | Password for dashboard login. **Change this for production!** |

</details>

<details>
<summary><b>📂 Storage & Download Paths</b></summary>
<br>

| Variable | Default | Description |
|----------|---------|-------------|
| `RCLONE_DEST` | `gdrive:/MirrorBot` | Default rclone upload destination. Format: `remote_name:/path` |
| `INDEX_URL` | — | Google Drive index worker URL for generating direct download links. Format: `https://your-index.workers.dev` |
| `DOWNLOAD_DIR` | `/app/downloads` | Internal download directory inside the container |
| `CONFIG_DIR` | `/app/config` | Internal config directory inside the container |
| `VIKING_USER_HASH` | — | Viking File user hash for authenticated uploads. Leave empty for anonymous |

</details>

<details>
<summary><b>⚙️ Bot Behavior</b></summary>
<br>

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTHORIZED_USERS` | — | Comma-separated list of Telegram User IDs allowed to use the bot (in addition to the owner) |
| `MAX_CONCURRENT_DOWNLOADS` | `3` | Maximum number of downloads running simultaneously. Increase for faster servers |
| `STOP_DUPLICATE` | `false` | When `true`, prevents downloading the same URL if it's already being processed |
| `DEFAULT_MAX_DAILY_TASKS` | `-1` | Maximum tasks per user per day. `-1` = unlimited |
| `DEFAULT_MAX_DAILY_BANDWIDTH` | `-1` | Maximum download bandwidth per user per day. Accepts human-readable values like `10GB`. `-1` = unlimited |
| `SMART_AUTO_ORGANIZATION` | `false` | When `true`, auto-organizes uploaded files into categories (Movies, Music, Documents, etc.) |
| `MAX_RETRIES` | `3` | Number of retry attempts for failed download/upload operations |

</details>

<details>
<summary><b>📝 Logging & Debug</b></summary>
<br>

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Logging verbosity. Options: `debug`, `info`, `warn`, `error` |
| `TZ` | `Asia/Jakarta` | Timezone for log timestamps, scheduled tasks, and statistics |

Log output is written to both **stdout** and the file `config/zee-mirror.log` simultaneously.

</details>

<details>
<summary><b>🤖 Userbot / MTProto (Optional)</b></summary>
<br>

| Variable | Description |
|----------|-------------|
| `APP_ID` | Telegram MTProto application ID. Can be the same as `TELEGRAM_API_ID` |
| `APP_HASH` | Telegram MTProto application hash. Can be the same as `TELEGRAM_API_HASH` |
| `USER_SESSION_STRING` | Base64-encoded Telegram session. Generate with `go run cmd/session-gen/main.go` |

> When all three are set, the bot can download files from private channels/groups and join groups via `/join` command.

</details>

<details>
<summary><b>🐳 Docker Compose Services</b></summary>
<br>

The `docker-compose.yml` defines two services:

| Service | Image | Purpose |
|---------|-------|---------|
| `telegram-bot-api` | `aiogram/telegram-bot-api` | Self-hosted Telegram Bot API server. Enables file downloads >2GB and faster local file handling |
| `zee-mirror` | Built from `Dockerfile` | The main bot application. Depends on `telegram-bot-api` |

Both services share a Docker network (`zee-network`) and a named volume (`downloads`) for file storage.

**Container security:**
- `no-new-privileges` security option enabled
- Log rotation: max 10MB per file, 3 files retained
- Health check: every 30s against `/api/health`
- Automatic restart on failure (`unless-stopped`)

</details>

<br>

---

## 🤖 Bot Commands

> **50+ commands** with short aliases for power users. All commands support inline keyboard navigation.

<details open>
<summary><b>📌 General Commands</b></summary>
<br>

| Command | Alias | Description |
|---------|:-----:|-------------|
| `/start` | — | Open the main bot dashboard with quick-access buttons |
| `/help` | — | Interactive help guide with categorized command list |
| `/settings` | — | Open bot settings menu (upload destination, preferences) |
| `/ping` | — | Check bot responsiveness and measure round-trip latency |
| `/speed` | — | Run a full network speed test (download + upload) |
| `/stats` | — | Open analytics dashboard with global, daily, and per-user stats |
| `/lang` | `/language` | Switch the bot's interface language |

</details>

<details open>
<summary><b>📥 Download Commands</b></summary>
<br>

| Command | Alias | Description |
|---------|:-----:|-------------|
| `/mirror <URL>` | `/m` | Download file from URL → upload to default cloud storage |
| `/leech <URL>` | `/l` | Download file from URL → send back to Telegram chat |
| `/viking <URL>` | `/v` | Download file from URL → upload to Viking File storage |
| `/ytdlp <URL>` | `/y` | Download video via yt-dlp → upload to cloud storage |
| `/ytdlpleech <URL>` | `/yl` | Download video via yt-dlp → send back to Telegram chat |
| `/torrent <magnet/file>` | `/t` | Download via magnet link or `.torrent` file with file selection UI |
| `/clone <URL>` | `/cl` | Server-side copy between cloud remotes (no local download) |
| `/batch <URLs>` | — | Queue multiple URLs for sequential download |
| `/search <keyword>` | — | Search for torrents by keyword with paginated results |

**Download Flags** — Append these to any download command:

| Flag | Example | Description |
|------|---------|-------------|
| `-z` | `/mirror URL -z` | Compress download to ZIP before uploading |
| `-uz` | `/mirror URL -uz` | Extract archive after download completes |
| `-p PASSWORD` | `/mirror URL -z -p secret` | Set password on the ZIP archive |
| `-name NAME` | `/batch URLs -name "My Batch"` | Assign a custom display name to the task |

**Telegram File Support:**
- Reply to any document, video, or audio message with `/mirror` or `/leech` (no URL needed)
- Reply to a `.torrent` file with `/torrent` to start downloading
- The userbot can forward files from private channels if configured

**Cookies Support (Gofile, YouTube, etc.):**
To download from sites that require login or are strictly rate-limited (like **Gofile** or age-restricted YouTube videos):
1. Extract your cookies from your browser using an extension like **Get cookies.txt LOCALLY**.
2. Save the file as `cookies.txt`.
3. Send the `cookies.txt` file directly to the bot in Telegram.
4. The bot will automatically save and use these cookies for subsequent `/mirror`, `/leech`, and `/ytdlp` tasks!

</details>

<details>
<summary><b>📋 Task Management</b></summary>
<br>

| Command | Alias | Description |
|---------|:-----:|-------------|
| `/status` | `/st` | List all active tasks with real-time progress bars, speed, and ETA |
| `/cancel <ID>` | `/c` | Cancel a specific task by its ID. Cleans up partial downloads |
| `/cancelall` | — | Cancel **all** running tasks at once (owner/admin only) |

**Task Progress Display:**
- Progress bars update every **5 seconds** automatically
- Shows: file name, percentage, speed, downloaded/total size, ETA
- Inline buttons for cancel and refresh

</details>

<details>
<summary><b>🔄 Task Recovery</b></summary>
<br>

| Command | Description |
|---------|-------------|
| `/recover` | Attempt to recover all tasks that were interrupted by a restart or crash |
| `/recoverystatus` | View all recoverable task checkpoints with their status |

**How it works:**
- Tasks create checkpoints in the SQLite database at key stages (downloading, uploaded, etc.)
- On restart, checkpoints are preserved and can be resumed
- Recovery attempts to re-queue interrupted downloads from the last known state

</details>

<details>
<summary><b>📂 File Manager (Google Drive)</b></summary>
<br>

| Command | Alias | Description |
|---------|:-----:|-------------|
| `/ls [path]` | `/dir` | List files and folders in the current or specified Drive path |
| `/mkdir <name>` | — | Create a new folder in the current Drive path |
| `/rm <file>` | — | Delete a file or folder (with confirmation prompt) |
| `/mv <src> <dst>` | — | Move or rename a file/folder |
| `/share <file>` | — | Generate a public shareable link for a file |
| `/find <keyword>` | — | Search for files across your entire Drive by name |

**Navigation:**
- Inline keyboard buttons for browsing directories
- Pagination for large directory listings
- File size and modification date displayed

</details>

<details>
<summary><b>💾 Multi-Storage Management</b></summary>
<br>

| Command | Description |
|---------|-------------|
| `/storages` | List all configured rclone remotes with their types and paths |
| `/setstorage <remote:/path>` | Set the default upload destination for all subsequent tasks |

**Supported Storage Providers** (via rclone):
Google Drive, OneDrive, Mega, Amazon S3, Dropbox, Box, Backblaze B2, Azure Blob, Google Cloud Storage, FTP/SFTP, and [40+ more](https://rclone.org/overview/).

</details>

<details>
<summary><b>🎵 Media Processing</b></summary>
<br>

All media commands use **ffmpeg** under the hood. Reply to a video/audio message to process it.

| Command | Description |
|---------|-------------|
| `/extractaudio` | Extract the audio track from a video file and send as MP3/AAC |
| `/compress [quality]` | Compress video with quality presets: `low`, `medium`, `high` |
| `/thumbnail [timestamp]` | Generate a single thumbnail image at the specified timestamp (e.g., `00:01:30`) |
| `/screenshots <video> [count]` | Generate N evenly-spaced screenshots from a video |
| `/subtitle <video> <sub>` | Embed a subtitle file (.srt, .ass) as a selectable track |
| `/hardsub` | Hardcode (burn-in) subtitles permanently into the video |
| `/rescale` | Rescale video resolution (e.g., 1080p → 720p) |
| `/convert <file> <format>` | Convert between media formats: `mp4`, `mkv`, `mp3`, `aac`, `flac`, `wav`, etc. |
| `/mediainfo` | Display detailed metadata: codec, resolution, bitrate, duration, streams |

</details>

<details>
<summary><b>🖥️ System & Monitoring</b></summary>
<br>

| Command | Description |
|---------|-------------|
| `/system` | Detailed system status: CPU usage, RAM (used/total), Disk (used/total), Uptime, OS info |
| `/health` | Health check across all components: bot, database, aria2, rclone, ffmpeg, disk space |
| `/logs [lines]` | View the most recent log entries (default: last 50 lines) |

**Monitoring Features:**
- Resource usage metrics are exported via **Prometheus** at `/metrics`
- Configure alert thresholds and get notified via Telegram channels
- Storage usage metrics updated every 5 minutes automatically

</details>

<details>
<summary><b>🔐 Admin & User Management</b></summary>
<br>

| Command | Description |
|---------|-------------|
| `/authorize <ID>` | Grant bot access to a user by their Telegram User ID |
| `/unauthorize <ID>` | Revoke a user's access (they can no longer use the bot) |
| `/removeuser <ID>` | Permanently delete a user and all their records from the database |
| `/setrole <ID> <role>` | Set a user's role. Roles: `admin` (can manage users), `user` (standard access) |
| `/setlimit <ID> <count>` | Set a user's maximum daily task limit |
| `/setexpire <ID> <date>` | Set an expiration date for a user's access (format: `YYYY-MM-DD`) |
| `/users` | List all registered users with their roles, limits, usage stats, and expiration dates |
| `/join <link>` | **(Userbot)** Join a private channel or group via invite link |
| `/setalertchannel [ID]` | Set a Telegram channel to receive system alerts (CPU/RAM/Disk warnings) |

**Access Control Flow:**
1. Unknown user sends a message → Bot shows "Access Denied" and logs the attempt
2. Owner runs `/authorize <ID>` → User is added to the database with default limits
3. User can now use the bot within their role and limit constraints
4. Owner can adjust limits, set expiration, or revoke access at any time

</details>

<br>

---

## 🌐 Web Dashboard

> A full-featured **React 18** control panel with **Tailwind CSS** styling, **Framer Motion** animations, and real-time **WebSocket** updates.

### 🔑 Accessing the Dashboard

| | |
|---|---|
| **URL** | `http://your-server-ip:DASHBOARD_PORT` (default port `80`) |
| **Login** | Use the `WEB_DASHBOARD_TOKEN` value from your `.env` file |

<br>

### 📑 Dashboard Pages

<table>
<tr>
<td align="center" width="14%"><b>📊<br>Overview</b></td>
<td align="center" width="14%"><b>📈<br>Analytics</b></td>
<td align="center" width="14%"><b>📁<br>Explorer</b></td>
<td align="center" width="14%"><b>🧲<br>Torrents</b></td>
<td align="center" width="14%"><b>👥<br>Users</b></td>
<td align="center" width="14%"><b>⚙️<br>Settings</b></td>
<td align="center" width="14%"><b>📋<br>Logs</b></td>
</tr>
<tr>
<td><sub>System metrics, active tasks, resource utilization graphs</sub></td>
<td><sub>Interactive Recharts with download/upload trends over time</sub></td>
<td><sub>Browse local & remote files with download/delete actions</sub></td>
<td><sub>Browse torrent contents and selectively download specific files</sub></td>
<td><sub>Create, edit roles, set limits, manage expiration, delete users</sub></td>
<td><sub>View and update bot configuration values in real-time</sub></td>
<td><sub>Stream, filter, and search application logs live</sub></td>
</tr>
</table>

<br>

### ⚡ Real-time Updates

The dashboard maintains a persistent **WebSocket** connection (`/ws`) to the backend, providing instant push updates for:

- 📊 Task progress changes (percentage, speed, ETA)
- ✅ Task completion / failure notifications
- 🖥️ System resource utilization (CPU, RAM, Disk)
- 📋 New log entries as they are written

No manual page refresh needed — everything updates automatically.

<br>

### 🔧 Dashboard Tech Details

| Aspect | Detail |
|--------|--------|
| **Framework** | React 18 with functional components and hooks |
| **Build** | Vite 4 with fast HMR for development |
| **Styling** | Tailwind CSS 3 with custom configuration |
| **Charts** | Recharts 3 for responsive data visualization |
| **Animations** | Framer Motion 12 for page transitions and micro-interactions |
| **Icons** | Lucide React for consistent, scalable icons |
| **State** | React Context for global state management |
| **HTTP** | Axios for API calls with interceptors |
| **Custom Hooks** | 7 specialized hooks for data fetching, WebSocket, etc. |
| **Linting** | ESLint + Prettier with React-specific rules |

<br>

---

## 📡 REST API

> The backend exposes a comprehensive RESTful API on port `:8080` (internal). All endpoints require `WEB_DASHBOARD_TOKEN` authentication.

<details open>
<summary><b>📋 All API Endpoints</b></summary>
<br>

**System & Monitoring**

| Method | Endpoint | Description |
|:------:|----------|-------------|
| `GET` | `/api/health` | Health check — returns status of all components |
| `GET` | `/api/stats` | Aggregated system statistics |
| `GET` | `/api/system` | Real-time CPU, RAM, Disk usage, and uptime |
| `GET` | `/api/analytics` | Download/upload analytics data for chart display |
| `GET` | `/api/logs` | Retrieve recent application log entries |
| `GET` | `/metrics` | Prometheus metrics endpoint (for Grafana/external scraping) |

**Task Management**

| Method | Endpoint | Description |
|:------:|----------|-------------|
| `GET` | `/api/tasks` | List all active and recently completed tasks |

**File Management**

| Method | Endpoint | Description |
|:------:|----------|-------------|
| `GET` | `/api/explorer` | Browse local server files with status, size, and type info |
| `GET` | `/api/remote-explorer` | Browse remote cloud storage files via rclone |
| `GET` | `/api/remote-link` | Generate a direct download link for a remote file |
| `POST` | `/api/upload` | Upload a file from your browser to the server |
| `POST` | `/api/wipe-orphans` | Clean up orphaned files from the download directory |

**User Management**

| Method | Endpoint | Description |
|:------:|----------|-------------|
| `GET` | `/api/users` | List all registered users with roles, limits, and stats |
| `POST` | `/api/users` | Create a new user with specified role and limits |
| `PUT` | `/api/users` | Update user details (role, limits, expiration) |
| `DELETE` | `/api/users` | Delete a user from the database |

**Configuration**

| Method | Endpoint | Description |
|:------:|----------|-------------|
| `GET` | `/api/settings` | Read current bot settings |
| `PUT` | `/api/settings` | Update bot settings (applies immediately) |
| `GET` | `/api/config` | Read runtime configuration |
| `PUT` | `/api/config` | Update runtime configuration |

**Torrent Management**

| Method | Endpoint | Description |
|:------:|----------|-------------|
| `GET` | `/api/torrent/session` | Current torrent session info |
| `GET` | `/api/torrent/files` | List files within a torrent for selective download |
| `POST` | `/api/torrent/start` | Start downloading selected torrent files |

**Real-time**

| Method | Endpoint | Description |
|:------:|----------|-------------|
| `GET` | `/ws` | WebSocket connection for live task/system updates |

</details>

<br>

---

## 📁 Project Structure

<details open>
<summary><b>📂 Full Project Tree</b></summary>
<br>

```
Zee-Mirror/
│
├── cmd/                               # Application entry points
│   ├── zee-mirror/
│   │   └── main.go                   # Main entry point: bot init, route setup, signal handling
│   └── session-gen/
│       └── main.go                   # Interactive Telegram session string generator
│
├── handlers/                          # Telegram bot command handlers (36 files)
│   ├── handlers.go                   # BotService struct, initialization, shared utilities
│   ├── service.go                    # Service lifecycle (init, shutdown, cleanup)
│   ├── task_processor.go             # Core task processing pipeline (download → process → upload)
│   ├── task_status.go                # Task progress tracking and status formatting
│   ├── mirror_handler.go             # /mirror command: URL download + cloud upload
│   ├── torrent_handler.go            # /torrent command: magnet/torrent file handling
│   ├── torrent_meta.go               # Torrent metadata parsing and file selection
│   ├── ytdlp_handler.go              # /ytdlp command: video download with quality selection
│   ├── clone.go                      # /clone command: cloud-to-cloud server-side copy
│   ├── clone_utils.go                # Clone helper functions
│   ├── clone_test.go                 # Clone unit tests
│   ├── viking.go                     # /viking command: Viking File storage upload
│   ├── batch.go                      # /batch command: multi-URL sequential download
│   ├── upload.go                     # Rclone upload logic with progress tracking
│   ├── archive.go                    # Zip/Unzip operations with password support
│   ├── search.go                     # Torrent search with paginated results
│   ├── filemanager.go                # Google Drive file manager (ls/mkdir/rm/mv/share/find)
│   ├── storage.go                    # Multi-storage management and remote switching
│   ├── media.go                      # Media processing (ffmpeg): extract, compress, convert
│   ├── statistics.go                 # Analytics: daily/weekly/monthly/per-user stats
│   ├── monitor.go                    # System resource monitoring and alerting
│   ├── recovery.go                   # Task checkpoint recovery after restarts
│   ├── notifications.go              # Telegram channel logging and alert forwarding
│   ├── auth.go                       # User auth, role management, limits, expiration
│   ├── settings.go                   # Bot settings management with inline keyboard
│   ├── start.go                      # /start command: main dashboard with action buttons
│   ├── help_handler.go               # Interactive categorized help system
│   ├── help_texts.go                 # Help text content and formatting
│   ├── status.go                     # /status command: active task list with progress
│   ├── system.go                     # /system and /health commands
│   ├── join.go                       # /join command: userbot group/channel joining
│   ├── templates.go                  # Message templates and MarkdownV2 formatting
│   ├── utils.go                      # Handler utility functions
│   ├── handlers_test.go              # Handler unit tests
│   ├── disk_linux.go                 # Linux-specific disk usage implementation
│   └── disk_other.go                 # Cross-platform disk usage fallback
│
├── internal/                          # Internal packages (not importable by external code)
│   ├── api/                          # REST API & WebSocket server
│   │   ├── api.go                    # HTTP server, route registration, 20+ API handlers
│   │   └── websocket.go             # WebSocket hub: client management, broadcast loop
│   ├── config/                       # Configuration management
│   │   └── config.go                # Load env vars, type conversion, validation
│   ├── database/                     # SQLite database layer
│   │   └── *.go                     # Connection management, migrations, query helpers
│   ├── domain/                       # Domain entities and business types
│   │   ├── task.go                  # Task entity: ID, status, progress, timestamps
│   │   ├── user.go                  # User entity: ID, role, limits, expiration
│   │   ├── session.go               # Session management types
│   │   └── errors.go               # Domain-specific error types and codes
│   ├── downloader/                   # Download engine abstractions
│   │   ├── engine.go                # Download engine interface definition
│   │   ├── aria2.go                 # aria2c RPC integration: start, monitor, cancel
│   │   ├── ytdlp.go                # yt-dlp subprocess: format listing, download, progress
│   │   └── userbot.go              # Userbot file download via MTProto
│   ├── metrics/                      # Prometheus metrics
│   │   └── *.go                     # Counter/gauge/histogram definitions
│   ├── organizer/                    # Smart file organization
│   │   └── *.go                     # Category detection, auto-sorting rules
│   ├── parser/                       # Input parsing utilities
│   │   └── *.go                     # URL parsing, flag extraction, torrent parsing
│   ├── queue/                        # Download queue management
│   │   └── *.go                     # Concurrency control, priority queue, rate limiting
│   ├── recovery/                     # Task recovery system
│   │   └── *.go                     # Checkpoint creation, state persistence, resume logic
│   ├── repository/                   # Data access layer
│   │   └── *.go                     # SQLite CRUD operations for users, tasks, settings
│   ├── router/                       # Command & callback routing
│   │   └── *.go                     # Command registration, prefix matching, dispatch
│   └── userbot/                      # MTProto client
│       └── *.go                     # Singleton gotd client, session management, media download
│
├── pkg/                               # Public utility packages
│   ├── utils/                        # Helper functions
│   │   └── *.go                     # File size formatting, dir size calc, string utils
│   └── i18n/                         # Internationalization
│       └── *.go                     # Language detection, translation loading, fallbacks
│
├── dashboard/                         # Web Dashboard (React SPA)
│   ├── src/
│   │   ├── App.jsx                  # Root component: routing, layout, auth gate
│   │   ├── main.jsx                 # React DOM entry point
│   │   ├── index.css                # Global styles and Tailwind imports
│   │   ├── pages/                   # 7 page components
│   │   │   ├── Overview.jsx         # System dashboard: metrics cards, task list
│   │   │   ├── Analytics.jsx        # Charts: download/upload trends
│   │   │   ├── Explorer.jsx         # Dual file browser: local + remote
│   │   │   ├── TorrentSelect.jsx    # Torrent content browser with checkboxes
│   │   │   ├── Users.jsx            # User CRUD with role/limit editing
│   │   │   ├── Settings.jsx         # Configuration editor with save/reset
│   │   │   └── Logs.jsx             # Log viewer with auto-scroll
│   │   ├── components/              # Reusable UI components
│   │   │   ├── Sidebar/            # Navigation sidebar
│   │   │   ├── Stats/              # Statistics display cards (4 components)
│   │   │   ├── Task/               # Task progress card
│   │   │   └── Popups/             # Modal dialogs and confirmations
│   │   ├── hooks/                   # 7 custom React hooks
│   │   │   └── *.js                # useWebSocket, useFetch, useAuth, etc.
│   │   ├── context/                 # React context providers
│   │   │   └── *.jsx               # AuthContext, ThemeContext
│   │   └── utils/                   # Frontend utilities
│   │       └── *.js                # API helpers, formatters, constants
│   ├── package.json                 # Dependencies and build scripts
│   ├── vite.config.js               # Vite build configuration
│   ├── tailwind.config.js           # Tailwind CSS customization
│   ├── postcss.config.js            # PostCSS plugins
│   ├── eslint.config.js             # ESLint rules
│   └── .prettierrc                  # Prettier formatting config
│
├── migrations/                        # SQLite schema migrations (auto-applied on startup)
│   ├── 000001_init_schema.up.sql    # Users, tasks, settings tables + indexes
│   ├── 000002_add_user_language.up.sql  # Add language column to users
│   └── 000003_create_task_checkpoints.up.sql  # Task checkpoint table for recovery
│
├── config/                            # Runtime config directory (mounted as Docker volume)
│                                     # Contains: rclone.conf, cookies, zee-mirror.db, zee-mirror.log
│
├── .github/                           # GitHub configuration
│   ├── workflows/                   # CI/CD pipelines (4 workflows)
│   │   ├── ci-cd.yml               # Lint → Security → Test → Frontend Build → Docker Build
│   │   ├── quality.yml             # Tidy → Vet → Test → Security → Lint (via Taskfile)
│   │   ├── docker-publish.yml      # Build & push Docker image to registry
│   │   └── release.yml             # Automated release creation
│   └── dependabot.yml               # Automated dependency update PRs
│
├── Dockerfile                         # Multi-stage build: Node (dashboard) → Go (binary) → Alpine (runtime)
├── docker-compose.yml                 # 2 services: telegram-bot-api + zee-mirror
├── Taskfile.yml                       # 13 dev tasks: build, run, test, lint, fmt, vet, security, etc.
├── .golangci.yml                      # GolangCI-Lint: 12 linters enabled with custom rules
├── .gitignore                         # Ignored files and directories
├── .dockerignore                      # Docker build exclusions
├── go.mod                             # Go module definition (Go 1.25.7)
├── go.sum                             # Go dependency checksums
└── LICENSE                            # MIT License
```

</details>

<br>

---

## 🗄️ Database Schema

The bot uses **SQLite** with automatic migration on startup. Three migration files define the schema:

<details>
<summary><b>View Database Tables</b></summary>
<br>

**`users` table** — Registered bot users

| Column | Type | Description |
|--------|------|-------------|
| `id` | `INTEGER` (PK) | Telegram User ID |
| `username` | `TEXT` | Telegram username |
| `role` | `TEXT` | `user` or `admin` (default: `user`) |
| `language` | `TEXT` | Preferred interface language |
| `created_at` | `DATETIME` | Registration timestamp |
| `max_daily_tasks` | `INTEGER` | Daily task limit (`-1` = unlimited) |
| `max_daily_bandwidth` | `INTEGER` | Daily bandwidth limit in bytes |
| `expires_at` | `DATETIME` | Access expiration date (NULL = never) |

**`tasks` table** — Download/upload task records

| Column | Type | Description |
|--------|------|-------------|
| `id` | `TEXT` (PK) | UUID task identifier |
| `gid` | `TEXT` | aria2 GID or external download ID |
| `type` | `TEXT` | Task type: `mirror`, `leech`, `torrent`, `ytdlp`, `clone`, `viking` |
| `status` | `TEXT` | Current status: `pending`, `downloading`, `uploading`, `completed`, `failed`, `cancelled` |
| `url` | `TEXT` | Source URL |
| `file_name` | `TEXT` | Downloaded file name |
| `local_path` | `TEXT` | Local file system path |
| `remote_path` | `TEXT` | Cloud storage destination path |
| `remote_url` | `TEXT` | Generated share/index URL |
| `total_size` | `INTEGER` | Total file size in bytes |
| `downloaded_size` | `INTEGER` | Downloaded bytes so far |
| `uploaded_size` | `INTEGER` | Uploaded bytes so far |
| `chat_id` | `INTEGER` | Telegram chat ID |
| `user_id` | `INTEGER` | Owner's Telegram User ID |
| `created_at` | `DATETIME` | Task creation timestamp |
| `completed_at` | `DATETIME` | Task completion timestamp |
| `zip` | `BOOLEAN` | Compress before upload |
| `unzip` | `BOOLEAN` | Extract after download |
| `password` | `TEXT` | Archive password |
| `error` | `TEXT` | Error message (if failed) |
| `retries` | `INTEGER` | Retry attempt counter |

**`settings` table** — Key-value bot settings

| Column | Type | Description |
|--------|------|-------------|
| `key` | `TEXT` (PK) | Setting name |
| `value` | `TEXT` | Setting value |
| `updated_at` | `DATETIME` | Last modification timestamp |

**`task_checkpoints` table** — Recovery checkpoints

| Column | Type | Description |
|--------|------|-------------|
| `task_id` | `TEXT` | Reference to task ID |
| `stage` | `TEXT` | Checkpoint stage name |
| `data` | `TEXT` | Serialized checkpoint data |
| `created_at` | `DATETIME` | Checkpoint timestamp |

**Indexes:**
- `idx_tasks_user_id` — for per-user task queries
- `idx_tasks_status` — for active task filtering

</details>

<br>

---

## 🔨 Development

<details>
<summary><b>🖥️ Local Development (Without Docker)</b></summary>
<br>

**System Requirements:**
- Go 1.25+
- Node.js 18+
- [Task](https://taskfile.dev) (task runner)
- aria2c, yt-dlp, rclone, ffmpeg, 7zz installed locally

```bash
# ─── Backend ───────────────────────────────
go mod download           # Install Go dependencies
task run                  # Run with go run
task build                # Build binary → bin/zee-mirror
task test                 # Run all tests with verbose output
task lint                 # Run GolangCI-Lint
task check                # Run ALL quality checks (tidy → fmt → vet → security → lint → test)
task all                  # Quality checks + build

# ─── Frontend ──────────────────────────────
cd dashboard
npm install               # Install npm dependencies
npm run dev               # Start Vite dev server with HMR
npm run build             # Production build → dist/
npm run lint              # Run ESLint
```

</details>

<details>
<summary><b>🐳 Docker Commands</b></summary>
<br>

```bash
# Build & start all services
task up
# Or: docker compose up -d --build

# Stop all services
task down
# Or: docker compose down

# View live logs
task logs
# Or: docker compose logs -f --no-log-prefix zee-mirror

# Rebuild after code changes
docker compose up -d --build

# Enter container shell (for debugging)
docker exec -it zee-mirror-bot sh

# Check container status
docker ps
```

</details>

<details>
<summary><b>📋 Taskfile Commands Reference</b></summary>
<br>

| Command | Description |
|---------|-------------|
| `task build` | Compile Go binary to `bin/zee-mirror` |
| `task run` | Run application directly with `go run` |
| `task test` | Run all unit tests with verbose output |
| `task lint` | Run GolangCI-Lint (12 linters) |
| `task fmt` | Auto-format code with `gofmt` + `goimports` |
| `task vet` | Static analysis with `go vet` |
| `task security` | Vulnerability scan with `govulncheck` |
| `task tidy` | Clean up `go.mod` and `go.sum` |
| `task check` | Run **all** quality checks sequentially |
| `task all` | Quality checks + build |
| `task pre-push` | Same as `check` — run before pushing to remote |
| `task up` | Build & start Docker containers |
| `task down` | Stop Docker containers |
| `task logs` | Tail Docker logs (no prefix) |
| `task clean` | Remove build artifacts and binaries |

</details>

<details>
<summary><b>🔍 Code Quality Tools</b></summary>
<br>

The project uses **GolangCI-Lint** with 12 linters enabled:

| Linter | Purpose |
|--------|---------|
| `errcheck` | Detect unchecked errors |
| `gosimple` | Simplify code suggestions |
| `govet` | Report suspicious constructs |
| `staticcheck` | Advanced static analysis |
| `unused` | Find unused code |
| `revive` | Fast, configurable linter |
| `gosec` | Security-focused checks |
| `gocritic` | Opinionated style checks |
| `misspell` | Spelling corrections |
| `bodyclose` | Detect unclosed HTTP bodies |
| `gofmt` | Formatting enforcement |
| `goimports` | Import ordering |

Configuration: `.golangci.yml` with 5-minute timeout and custom exclusion rules.

</details>

<br>

---

## 🔄 CI/CD Pipeline

Four **GitHub Actions** workflows ensure code quality on every push and PR:

| Workflow | Trigger | What it does |
|----------|---------|-------------|
| **CI/CD** | Push & PR to `main` | Go lint → `govulncheck` security audit → Tests with `-race` → Frontend lint & build → Docker image build with Buildx cache |
| **Quality** | Push & PR to `main` | `go mod tidy` check → `go vet` → Test suite → Security scan → GolangCI-Lint (via Taskfile) |
| **Docker Publish** | Push to `main` | Build and publish Docker image to container registry |
| **Release** | Tag push | Automated release creation and artifact publishing |

> 🤖 **Dependabot** is configured for automated Go module (`gomod`) and npm (`npm`) dependency update PRs.

<br>

---

## 🐛 Troubleshooting

<details>
<summary><b>🔇 Bot not responding</b></summary>

1. Verify `BOT_TOKEN` is correct and matches your bot
2. Ensure the bot is started in Telegram (send `/start`)
3. Check container is running: `docker ps`
4. Check logs for errors: `docker compose logs zee-mirror`
5. Verify the `telegram-bot-api` container is also running
</details>

<details>
<summary><b>❌ Upload failing to cloud storage</b></summary>

1. Ensure `config/rclone.conf` exists and is properly configured
2. Test rclone connectivity: `docker exec zee-mirror-bot rclone lsd your_remote:`
3. Verify `RCLONE_DEST` format: `remote_name:/path`
4. Check disk space — downloads need temporary local storage
</details>

<details>
<summary><b>🐢 Slow download speeds</b></summary>

1. Increase `MAX_CONCURRENT_DOWNLOADS` in `.env`
2. Run `/speed` to check server bandwidth
3. Check disk I/O with `/system`
4. For torrents, ensure port forwarding is configured if behind NAT
</details>

<details>
<summary><b>📦 Files larger than 2GB not working</b></summary>

1. Verify `TELEGRAM_API_ID` and `TELEGRAM_API_HASH` are set in `.env`
2. Ensure the `telegram-bot-api` container is running: `docker ps | grep telegram-bot-api`
3. Check local bot API logs: `docker compose logs telegram-bot-api`
</details>

<details>
<summary><b>🤖 Userbot not connecting / private channel download fails</b></summary>

1. Regenerate session string: `go run cmd/session-gen/main.go`
2. Ensure **all three** variables are set: `APP_ID`, `APP_HASH`, `USER_SESSION_STRING`
3. Check if the session string is expired (re-run the generator)
4. Review logs: look for "Userbot failed to start" messages
</details>

<details>
<summary><b>🌐 Dashboard not loading</b></summary>

1. Check `DASHBOARD_PORT` is not blocked by a firewall
2. Verify port mapping: `docker ps` should show `0.0.0.0:80->8080/tcp`
3. Try accessing directly: `curl http://localhost:DASHBOARD_PORT/api/health`
4. Check browser console for any CORS or auth errors
</details>

<details>
<summary><b>🗄️ Database errors on startup</b></summary>

1. SQLite auto-migrates on startup — check logs for migration errors
2. If the database is corrupted, remove it: `rm config/zee-mirror.db`
3. Restart the container — a fresh database will be created automatically
</details>

<details>
<summary><b>🎬 Media processing errors (ffmpeg)</b></summary>

1. Verify ffmpeg is installed: `docker exec zee-mirror-bot ffmpeg -version`
2. Check available disk space for temporary processing files
3. Some media formats may not be supported — check ffmpeg codec list
4. Review logs for specific ffmpeg error messages
</details>

<details>
<summary><b>📋 How to view logs</b></summary>

```bash
# Docker container logs (live)
docker compose logs -f zee-mirror

# Application log file (inside container)
docker exec zee-mirror-bot cat /app/config/zee-mirror.log

# Via Telegram bot
/logs

# Via web dashboard
# Navigate to the "Logs" page
```
</details>

<br>

---

## 🤝 Contributing

Contributions are welcome! Here's how to get started:

1. **Fork** the repository
2. **Create** your branch: `git checkout -b feature/amazing-feature`
3. **Commit** with [Conventional Commits](https://www.conventionalcommits.org): `git commit -m 'feat: add amazing feature'`
4. **Push** to your fork: `git push origin feature/amazing-feature`
5. **Open** a Pull Request

**Before pushing**, always run the full quality check:

```bash
task check
```

### Development Guidelines

- Follow Go conventions and idiomatic patterns
- Write tests for new features and bug fixes
- Use `slog` for all logging (structured, with key-value pairs)
- Keep handlers focused — one file per feature area
- Use the `internal/` package for code not intended for external import
- Use `pkg/` only for truly reusable utilities
- Frontend: follow existing component patterns and use Tailwind utility classes

<br>

---

## 📄 License

Distributed under the **MIT License**. See [`LICENSE`](LICENSE) for details.

<br>

---

<div align="center">

**Built with ❤️ by Zee**

⭐ **Star this repo if you find it useful!** ⭐

<br>

<sub>
<a href="#-zee-mirror">Back to Top ↑</a>
</sub>

</div>