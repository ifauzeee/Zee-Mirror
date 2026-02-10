# 🚀 Zee-Mirror Telegram Bot

Bot Telegram berperforma tinggi untuk mirror dan leech file ke Google Drive, ditulis menggunakan bahasa **Go**.

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)

## ✨ Fitur Utama

- 📥 **Mirror** - Download file dari Telegram dan upload langsung ke Google Drive.
- 🔗 **Leech** - Download file dari berbagai protokol (HTTP/HTTPS/FTP/Magnet).
- 🎬 **YT-DLP Support** - Download video dari YouTube dan lebih dari 1000+ situs lainnya.
- 🧲 **Torrent Support** - Download file melalui magnet link atau file `.torrent`.
- 📊 **Real-time Progress** - Pembaruan status tugas secara *real-time* setiap 5 detik.
- 🗜️ **Manajemen Arsip** - Mendukung kompresi (Zip) dan ekstraksi (Unzip) dengan proteksi password.
- ⚡ **Performa Tinggi** - Menggunakan optimasi goroutine untuk menangani banyak tugas sekaligus.
- 🐳 **Dockerized** - Memudahkan proses deployment hanya dengan beberapa langkah.

### 🆕 Fitur Baru v2.0

- 🤖 **Userbot Integration** - Download from private channels/groups.
- 📊 **Dashboard Analytics** - Statistik lengkap: harian, mingguan, bulanan, per-user.
- 💾 **Multi Storage** - Support multiple cloud storage (Google Drive, OneDrive, Mega, dll).
- 📂 **File Manager** - Kelola file di Google Drive langsung dari Telegram.
- 🎵 **Media Processing** - Extract audio, compress video, generate thumbnails.
- 🔄 **Task Recovery** - Otomatis recover task yang terinterupsi saat restart.
- 🖥️ **Resource Monitor** - Monitoring CPU, RAM, Disk dengan custom alerts.
- 📢 **Channel Logging** - Log aktivitas ke channel Telegram khusus.
- 🌐 **Web Dashboard** - Panel kontrol berbasis web untuk memantau performa, mengelola task, dan eksplorasi file.

## 🛠️ Teknologi & Tools

| Tool | Fungsi Utama |
|------|--------------|
| **aria2c** | Engine download untuk protokol HTTP, FTP, dan BitTorrent. |
| **yt-dlp** | Engine khusus untuk mengunduh video streaming. |
| **gotd** | Telegram Client untuk fitur Userbot (MTProto). |
| **rclone** | Alat transfer file untuk mengunggah hasil ke cloud storage. |
| **7zz** | Alat kompresi dan ekstraksi arsip dengan performa tinggi. |
| **ffmpeg** | Media processing: konversi, kompresi, extract audio. |

## 📋 Persyaratan Sistem

- Docker & Docker Compose terinstal.
- Telegram Bot Token (Dapatkan dari [@BotFather](https://t.me/BotFather)).
- Akun Google Drive yang sudah dikonfigurasi melalui Rclone.
- Telegram API ID & Hash (untuk Userbot).

## 🚀 Panduan Instalasi Cepat

### 1. Klon Repositori
```bash
git clone https://github.com/ifauzeee/Zee-Mirror
cd Zee-Mirror
```

### 2. Userbot Setup (Optional)
Jika Anda ingin bot bisa download dari **Private Channel** atau **Join Group**, Anda perlu mengaktifkan Userbot.

Jalankan tool generator session yang sudah disediakan:
```bash
go run cmd/session-gen/main.go
```
Ikuti instruksi di layar (Login Telegram). Tool ini otomatis akan menambahkan `APP_ID`, `APP_HASH`, dan `USER_SESSION_STRING` ke file `.env` Anda.

### 3. Konfigurasi Environment
Salin file template `.env.example` menjadi `.env` dan sesuaikan nilainya (jika belum dibuat oleh tool session-gen):
```bash
cp .env.example .env
nano .env
```

### 4. Konfigurasi Rclone
Buat direktori konfigurasi dan masukkan file `rclone.conf` Anda:
```bash
mkdir -p config
cp ~/.config/rclone/rclone.conf config/rclone.conf
```

### 5. Jalankan Aplikasi
```bash
docker-compose up -d --build
docker-compose logs -f
```

## 📱 Panduan Penggunaan

### Perintah Dasar
| Perintah | Deskripsi |
|----------|-----------|
| `/start` | Membuka dashboard utama bot. |
| `/help` | Menampilkan panduan bantuan lengkap. |
| `/mirror <URL>` | Mengunduh file dan mengunggahnya ke Drive. |
| `/leech <URL>` | Mengunduh file dari URL ke server. |
| `/viking <URL>` | Mirror file ke Viking File storage. |
| `/ytdlp <URL>` | Mengunduh video menggunakan yt-dlp. |
| `/torrent <magnet>` | Mengunduh file melalui torrent/magnet. |
| `/search <keyword>` | Mencari file torrent berdasarkan kata kunci. |
| `/status` | Menampilkan daftar tugas yang sedang berjalan. |
| `/cancel <ID>` | Membatalkan tugas yang sedang diproses. |
| `/settings` | Membuka menu pengaturan bot. |

### 📊 Statistik & Analytics
| Perintah | Deskripsi |
|----------|-----------|
| `/stats` | Dashboard statistik lengkap (global, harian, per-user). |

### 📂 File Manager (Google Drive)
| Perintah | Deskripsi |
|----------|-----------|
| `/ls [path]` | List file/folder di Google Drive. |
| `/mkdir <name>` | Buat folder baru. |
| `/rm <file>` | Hapus file/folder. |

## 🌐 Web Dashboard
Zee-Mirror kini dilengkapi dengan dashboard web modern untuk monitoring sistem.
- **URL**: `http://localhost:8080` (Default Docker) atau port yang dikonfigurasi.
- **Akses**: Memerlukan `WEB_DASHBOARD_TOKEN` untuk login.
- **Fitur**:
  - Grafik utilisasi CPU, RAM, Disk.
  - List task aktif dengan progress bar real-time.
  - Explorer file server (Unduh/Hapus file).
  - Update konfigurasi bot secara langsung.
| `/mv <src> <dst>` | Pindahkan/rename file. |
| `/share <file>` | Generate share link. |
| `/find <keyword>` | Cari file di Drive. |

### 💾 Multi Storage
| Perintah | Deskripsi |
|----------|-----------|
| `/storages` | Lihat daftar storage yang dikonfigurasi. |
| `/setstorage <remote:/path>` | Set default storage destination. |

### 🎵 Media Processing
| Perintah | Deskripsi |
|----------|-----------|
| `/extractaudio` | Extract audio dari video (reply ke video). |
| `/compress [quality]` | Kompres video (low/medium/high). |
| `/thumbnail [timestamp]` | Generate thumbnail dari video. |
| `/screenshots <video> [count]` | Generate multiple screenshots. |
| `/subtitle <video> <sub>` | Embed subtitle ke video. |
| `/convert <file> <format>` | Convert format (mp4, mkv, mp3, dll). |
| `/mediainfo` | Tampilkan informasi media file. |

### 🖥️ System Monitoring
| Perintah | Deskripsi |
|----------|-----------|
| `/system` | Status sistem (CPU, RAM, Disk, Uptime). |
| `/health` | Health check semua komponen. |
| `/logs` | Tampilkan log terbaru. |

### 🔄 Task Recovery
| Perintah | Deskripsi |
|----------|-----------|
| `/recover` | Recovery task yang terinterupsi. |
| `/recoverystatus` | Status task yang bisa di-recover. |

### Admin Commands
| Perintah | Deskripsi |
|----------|-----------|
| `/join <link>` | **(Userbot)** Join private channel/group via link invite. |
| `/authorize <ID>` | Izinkan user baru. |
| `/unauthorize <ID>` | Cabut izin user. |
| `/users` | Lihat daftar user. |
| `/setlogchannel [ID]` | Set channel untuk logging aktivitas. |
| `/setalertchannel [ID]` | Set channel untuk custom alerts. |

### Flags (Opsi Tambahan)
Gunakan flag berikut di akhir perintah (misal: `/mirror URL -z`):
- `-z` : Kompres hasil download ke format Zip sebelum diunggah.
- `-uz` : Ekstrak file arsip setelah proses download selesai.
- `-p PASSWORD` : Memberikan password pada file Zip.
- `-name NAMA` : Memberikan nama khusus pada tugas batch.

## 🔧 Konfigurasi Lanjutan

### Variabel Lingkungan (.env)
| Variabel | Deskripsi | Default |
|----------|-----------|---------|
| `BOT_TOKEN` | Token API Bot Telegram. | **Wajib Diisi** |
| `OWNER_ID` | User ID Telegram pemilik bot. | **Wajib Diisi** |
| `AUTHORIZED_USERS` | Daftar User ID yang diizinkan (pisahkan dengan koma). | - |
| `RCLONE_DEST` | Destinasi penyimpanan di Rclone. | `gdrive:/MirrorBot` |
| `MAX_CONCURRENT_DOWNLOADS` | Jumlah maksimal download bersamaan. | `3` |
| `TZ` | Zona waktu aplikasi. | `Asia/Jakarta` |

## 📁 Struktur Proyek

```text
Zee-Mirror/
├── cmd/
│   └── zee-mirror/
│       └── main.go          # Entry Point
├── handlers/                 # Logika penanganan perintah
│   ├── archive.go           # Operasi Zip/Unzip
│   ├── batch.go             # Batch Download
│   ├── download.go          # Integrasi aria2 & yt-dlp
│   ├── filemanager.go       # 🆕 File Manager Google Drive
│   ├── media.go             # 🆕 Media Processing
│   ├── monitor.go           # 🆕 Resource Monitoring
│   ├── notifications.go     # 🆕 Channel Logging & Alerts
│   ├── recovery.go          # 🆕 Task Recovery
│   ├── statistics.go        # 🆕 Analytics Dashboard
│   ├── storage.go           # 🆕 Multi Storage Support
│   ├── upload.go            # Integrasi Rclone
│   └── ...
├── internal/
│   ├── config/              # Manajemen konfigurasi
│   └── database/            # SQLite database
├── pkg/
│   └── utils/               # Helper functions
├── config/                   # rclone.conf & cookies
├── Dockerfile
└── docker-compose.yml
```

## 🐛 Troubleshooting

- **Bot Tidak Merespons**: Pastikan `BOT_TOKEN` benar dan bot sudah di-*start* di Telegram.
- **Upload Gagal**: Pastikan file `config/rclone.conf` sudah tersedia.
- **Download Lambat**: Tingkatkan nilai `MAX_CONCURRENT_DOWNLOADS`.
- **Media Processing Error**: Pastikan ffmpeg terinstal di container.

---
Dibuat dengan ❤️ oleh **Zee-Mirror Team**. Lisensi [MIT](LICENSE).