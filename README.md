# 🚀 Zee-Mirror Telegram Bot

Bot Telegram untuk mirror/leech file ke Google Drive dengan performa tinggi, ditulis dalam **Go**.

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)

## ✨ Fitur

- 📥 **Mirror** - Download file dari Telegram dan upload ke Google Drive
- 🔗 **Leech** - Download dari URL (HTTP/HTTPS/FTP/Magnet)
- 🎬 **YT-DLP** - Download video dari YouTube dan 1000+ situs
- 🧲 **Torrent** - Download via magnet link atau file .torrent
- 📊 **Real-time Progress** - Update status setiap 5 detik
- 🗜️ **Archive Support** - Zip/Unzip dengan password
- ⚡ **Super Fast** - Concurrency optimal dengan goroutine
- 🐳 **Dockerized** - Siap deploy dalam hitungan menit

## 🛠️ Tools yang Digunakan

| Tool | Fungsi |
|------|--------|
| **aria2c** | Download HTTP/Torrent dengan multi-connection |
| **yt-dlp** | Download video streaming |
| **rclone** | Upload ke cloud storage |
| **7zz** | Compress/extract archive |

## 📋 Prerequisites

- Docker & Docker Compose
- Telegram Bot Token (dari [@BotFather](https://t.me/BotFather))
- Google Drive dengan rclone sudah dikonfigurasi

## 🚀 Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/ifauzeee/Zee-Mirror
cd zee-mirror
```

### 2. Konfigurasi Environment

```bash
# Copy template environment
cp .env.example .env

# Edit file .env dengan editor favorit
nano .env
```

Isi nilai berikut:
```env
BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ
OWNER_ID=123456789
RCLONE_DEST=gdrive:/MirrorBot
```

### 3. Setup Rclone

```bash
# Buat folder config
mkdir -p config

# Copy rclone.conf dari komputer Anda
# Windows: %APPDATA%\rclone\rclone.conf
# Linux/Mac: ~/.config/rclone/rclone.conf
cp ~/.config/rclone/rclone.conf config/rclone.conf
```

Atau setup rclone baru:
```bash
docker run --rm -it -v $(pwd)/config:/config rclone/rclone config --config /config/rclone.conf
```

Opsional: Jika Anda perlu login ke situs tertentu untuk download (misalnya cookie authentication), tambahkan file `cookies.txt` ke folder config:
```bash
# Contoh struktur cookies.txt (format Netscape)
# .site.com	TRUE	/	FALSE	9999999999	cookie_name	cookie_value
echo ".example.com	TRUE	/	FALSE	9999999999	session_id	abc123" > config/cookies.txt
```

### 4. Build & Run

```bash
# Build dan jalankan
docker-compose up -d --build

# Lihat logs
docker-compose logs -f
```

## 📱 Penggunaan

### Perintah Dasar

| Command | Deskripsi |
|---------|-----------|
| `/start` | Dashboard utama |
| `/help` | Bantuan lengkap |
| `/mirror <URL>` | Mirror ke Drive |
| `/leech <URL>` | Download dari URL |
| `/ytdlp <URL>` | Download video |
| `/torrent <magnet>` | Download torrent |
| `/search <keyword>` | Cari torrent |
| `/status` | Status task aktif |
| `/cancel <ID>` | Batalkan task |
| `/settings` | Pengaturan bot |

### Batch Download

| Command | Deskripsi |
|---------|-----------|
| `/batch` | Download multiple URLs sekaligus |
| `/batchstatus` | Status batch aktif |
| `/cancelbatch <ID>` | Batalkan batch download |

### Flags Opsional

| Flag | Deskripsi |
|------|-----------|
| `-z` | Zip file sebelum upload |
| `-uz` | Extract archive setelah download |
| `-p PASSWORD` | Password untuk zip |
| `-name NAME` | Nama batch (untuk /batch) |
| `-priority 1-10` | Prioritas batch (default: 5) |

### Contoh Penggunaan

```
# Download dan upload ke Drive
/mirror https://example.com/file.zip

# Download dan extract
/leech https://example.com/archive.rar -uz

# Download, zip dengan password, upload
/mirror -z -p rahasia123
(reply ke file di Telegram)

# Download video YouTube
/ytdlp https://youtube.com/watch?v=xxxxx

# Download torrent
/torrent magnet:?xt=urn:btih:xxxxx

# Batch download (multiple URLs)
/batch -name MyDownloads -z
https://example.com/file1.zip
https://example.com/file2.mp4
https://example.com/file3.rar
```


## 🔧 Konfigurasi Lanjutan

### Environment Variables

| Variable | Deskripsi | Default |
|----------|-----------|---------|
| `BOT_TOKEN` | Token dari BotFather | **Required** |
| `OWNER_ID` | Telegram User ID owner | **Required** |
| `AUTHORIZED_USERS` | User ID yang diizinkan (comma-separated) | - |
| `RCLONE_DEST` | Destinasi rclone | `gdrive:/MirrorBot` |
| `MAX_CONCURRENT_DOWNLOADS` | Maks download bersamaan | `3` |
| `TZ` | Timezone | `Asia/Jakarta` |

### Volume Mounts

| Path | Deskripsi |
|------|-----------|
| `/app/downloads` | Temporary download files |
| `/app/config` | Config files (rclone.conf, cookies.txt) |

## 📁 Struktur Project

```
zee-mirror/
├── main.go              # Entry point
├── handlers/
│   ├── archive.go       # Zip/unzip handler (7zz)
│   ├── batch.go         # Batch download handler
│   ├── download.go      # Download handlers (aria2, yt-dlp)
│   ├── handlers.go      # Core task management
│   ├── search.go        # Torrent search handler
│   ├── settings.go      # Settings handler
│   ├── start.go         # /start dan /help handler
│   ├── status.go        # Status dan cancel handler
│   ├── upload.go        # Upload handler (rclone)
│   └── utils.go         # Utility functions
├── config/
│   └── rclone.conf.example
├── downloads/           # Temporary download files
├── scripts/
│   └── (shell scripts)
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── go.mod
├── go.sum
├── LICENSE
├── Makefile
├── Taskfile.yml
└── README.md
```

## 🐛 Troubleshooting

### Bot tidak merespon
1. Cek apakah `BOT_TOKEN` sudah benar
2. Pastikan `OWNER_ID` sesuai dengan Telegram ID Anda
3. Lihat logs: `docker-compose logs -f`

### Upload gagal
1. Pastikan `rclone.conf` sudah di-copy ke `config/`
2. Cek nama remote di `RCLONE_DEST` sesuai dengan rclone.conf
3. Test rclone manual: `docker exec zee-mirror-bot rclone lsd gdrive:`

### Download lambat
1. Tingkatkan `MAX_CONCURRENT_DOWNLOADS`
2. Pastikan server memiliki bandwidth yang cukup

## 🤝 Contributing

Pull requests welcome! Untuk perubahan besar, silakan buka issue terlebih dahulu.

## 📄 License

MIT License - lihat [LICENSE](LICENSE) untuk detail.

---

Made with ❤️ by Zee-Mirror Team
