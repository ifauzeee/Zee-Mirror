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

## 🛠️ Teknologi & Tools

| Tool | Fungsi Utama |
|------|--------------|
| **aria2c** | Engine download untuk protokol HTTP, FTP, dan BitTorrent. |
| **yt-dlp** | Engine khusus untuk mengunduh video streaming. |
| **rclone** | Alat transfer file untuk mengunggah hasil ke cloud storage. |
| **7zz** | Alat kompresi dan ekstraksi arsip dengan performa tinggi. |

## 📋 Persyaratan Sistem

- Docker & Docker Compose terinstal.
- Telegram Bot Token (Dapatkan dari [@BotFather](https://t.me/BotFather)).
- Akun Google Drive yang sudah dikonfigurasi melalui Rclone.

## 🚀 Panduan Instalasi Cepat

### 1. Klon Repositori
```bash
git clone https://github.com/ifauzeee/Zee-Mirror
cd Zee-Mirror
```

### 2. Konfigurasi Environment
Salin file template `.env.example` menjadi `.env` dan sesuaikan nilainya:
```bash
cp .env.example .env
nano .env
```
Lengkapi data berikut:
```env
BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ
OWNER_ID=123456789
RCLONE_DEST=gdrive:/MirrorBot
```

### 3. Konfigurasi Rclone
Buat direktori konfigurasi dan masukkan file `rclone.conf` Anda:
```bash
mkdir -p config
# Salin rclone.conf yang sudah ada ke folder config/
cp ~/.config/rclone/rclone.conf config/rclone.conf
```

### 4. Jalankan Aplikasi
```bash
# Build dan jalankan container
docker-compose up -d --build

# Pantau log aktivitas
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
| `/ytdlp <URL>` | Mengunduh video menggunakan yt-dlp. |
| `/torrent <magnet>` | Mengunduh file melalui torrent/magnet. |
| `/search <keyword>` | Mencari file torrent berdasarkan kata kunci. |
| `/status` | Menampilkan daftar tugas yang sedang berjalan. |
| `/cancel <ID>` | Membatalkan tugas yang sedang diproses. |
| `/settings` | Membuka menu pengaturan bot. |

### Fitur Batch
| Perintah | Deskripsi |
|----------|-----------|
| `/batch` | Mengunduh banyak URL sekaligus dalam satu antrean. |
| `/batchstatus` | Menampilkan status tugas batch yang aktif. |
| `/cancelbatch <ID>` | Membatalkan tugas batch tertentu. |

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
│       └── main.go      # Titik masuk utama aplikasi (Entry Point)
├── handlers/            # Logika penanganan perintah Telegram
│   ├── archive.go       # Operasi Zip/Unzip
│   ├── batch.go         # Logika Batch Download
│   ├── download.go      # Integrasi aria2 & yt-dlp
│   ├── search.go        # Pencarian Torrent
│   ├── upload.go        # Integrasi Rclone
│   └── ...
├── internal/
│   └── config/          # Manajemen konfigurasi aplikasi
├── pkg/
│   └── utils/           # Fungsi pembantu (Helper functions)
├── config/              # Tempat penyimpanan rclone.conf & cookies
├── Dockerfile           # Konfigurasi build Docker
└── docker-compose.yml   # Konfigurasi deployment Docker Compose
```

## 🐛 Troubleshooting

- **Bot Tidak Merespons**: Pastikan `BOT_TOKEN` benar dan bot sudah di-*start* di Telegram. Periksa log dengan `docker-compose logs -f`.
- **Upload Gagal**: Pastikan file `config/rclone.conf` sudah tersedia dan `RCLONE_DEST` sesuai dengan nama remote di konfigurasi.
- **Download Lambat**: Cek koneksi server atau coba tingkatkan nilai `MAX_CONCURRENT_DOWNLOADS`.

---
Dibuat dengan ❤️ oleh **Zee-Mirror Team**. Lisensi [MIT](LICENSE).