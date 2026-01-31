package handlers

func getHelpMirror() string {
	return HelpDetailMessage(
		"📥 MIRROR",
		"Upload file dari URL langsung ke Google Drive\\.\nFile akan disimpan di cloud storage tanpa menyimpan di server\\.",
		"• `/mirror <URL>` ─ Download \\& upload ke Drive\n• Reply ke file dengan `/mirror`\n\n⚙️ *FLAGS OPSIONAL*\n• `\\-z` ─ Zip file sebelum upload\n• `\\-uz` ─ Unzip setelah download\n• `\\-p <pass>` ─ Password zip/unzip",
		"`/mirror https://example\\.com/file\\.zip`\n`/mirror \\-z https://example\\.com/folder`\n`/mirror \\-uz \\-p secret https://file\\.rar`",
		"",
	)
}

func getHelpLeech() string {
	return HelpDetailMessage(
		"📤 LEECH",
		"Download file dari URL ke server bot\\.\nFile akan dikirim langsung ke chat Telegram setelah selesai\\.",
		"• `/leech <URL>` ─ Download ke server\n• Reply ke file dengan `/leech`\n\n⚙️ *FLAGS OPSIONAL*\n• `\\-z` ─ Zip file sebelum kirim\n• `\\-uz` ─ Unzip setelah download\n• `\\-p <pass>` ─ Password zip/unzip",
		"`/leech https://example\\.com/video\\.mp4`\n`/leech \\-uz https://archive\\.rar`",
		"",
	)
}

func getHelpYTDLP() string {
	return HelpDetailMessage(
		"🎬 YT-DLP",
		"Download video/audio dari 1000\\+ situs\\.\nSupport: YouTube, TikTok, Twitter/X, Instagram, Facebook, Vimeo, dll\\.",
		"• `/ytdlp <URL>` ─ Download dengan pilihan kualitas\n\n✨ *FITUR*\n• Pilih kualitas video \\(360p\\-4K\\)\n• Download audio only \\(MP3\\)\n• Subtitle otomatis",
		"`/ytdlp https://youtube\\.com/watch?v=xxxxx`\n`/ytdlp https://tiktok\\.com/@user/video/123`",
		"",
	)
}

func getHelpTorrent() string {
	return HelpDetailMessage(
		"🧲 TORRENT",
		"Download file via magnet link atau file torrent \\(\\.torrent\\)\\.\nBot akan otomatis mendeteksi jika Anda mengirim magnet link tanpa command\\.",
		"• `/torrent <magnet>` ─ Download dari magnet\n• Reply ke \\.torrent dengan `/torrent`\n• Kirim magnet link langsung \\(Auto\\-Detect\\)\n\n✨ *FITUR*\n• Support magnet link \\& file \\.torrent\n• Multi\\-connection download\n• Resume download",
		"`/torrent magnet:?xt=urn:btih:xxxxx`",
		"💡 *AUTO\\-DETECT:* Anda bisa langsung mengirim magnet link, bot akan otomatis memprosesnya\\.",
	)
}

func getHelpClone() string {
	return HelpDetailMessage(
		"📋 CLONE",
		"Clone/salin file atau folder dari Google Drive ke Drive tujuan\\.",
		"• `/clone <drive_url>` ─ Clone ke storage aktif\n\n🔗 *URL YANG DIDUKUNG*\n• `drive\\.google\\.com/file/d/xxx`\n• `drive\\.google\\.com/drive/folders/xxx`\n• `drive\\.google\\.com/open?id=xxx`",
		"`/clone https://drive\\.google\\.com/file/d/abc123`",
		"",
	)
}

func getHelpBatch() string {
	return HelpDetailMessage(
		"📦 BATCH",
		"Download multiple URLs sekaligus dalam satu batch\\.",
		"/batch\nURL1\nURL2\nURL3\n\n📋 *COMMAND TERKAIT*\n• `/batchstatus` ─ Lihat status batch\n• `/cancelbatch` ─ Batalkan batch\n\n✨ *FITUR*\n• Download parallel\n• Progress tracking per URL\n• Auto retry jika gagal",
		"/batch\nhttps://file1\\.com/a\\.zip\nhttps://file2\\.com/b\\.zip",
		"",
	)
}

func getHelpSearch() string {
	return HelpDetailMessage(
		"🔍 SEARCH",
		"Cari torrent dari berbagai sumber/tracker\\.",
		"• `/search <keyword>` ─ Cari torrent\n\n✨ *FITUR*\n• Multi\\-tracker search\n• Filter hasil\n• Download langsung dari hasil",
		"`/search ubuntu 22\\.04`\n`/search archlinux iso`",
		"",
	)
}

func getHelpStatus() string {
	return HelpDetailMessage(
		"📊 STATUS",
		"Melihat semua task yang sedang aktif \\(download/upload\\)\\.",
		"• `/status` ─ Tampilkan semua task aktif\n\n📋 *INFO DITAMPILKAN*\n• Nama file\n• Progress \\(%\\)\n• Kecepatan download/upload\n• ETA \\(estimasi waktu selesai\\)\n• Status \\(downloading/uploading\\)\n\n🔘 *TOMBOL AKSI*\n• Refresh ─ Update status\n• Cancel ─ Batalkan task",
		"`/status`",
		"",
	)
}

func getHelpStats() string {
	return HelpDetailMessage(
		"📈 STATS",
		"Melihat statistik penggunaan bot\\.",
		"• `/stats` ─ Tampilkan statistik\n\n📋 *INFO DITAMPILKAN*\n• Total download\n• Total upload\n• Data yang diproses\n• Uptime bot\n• Task completed/failed",
		"`/stats`",
		"",
	)
}

func getHelpSystem() string {
	return HelpDetailMessage(
		"🖥️ SYSTEM",
		"Melihat informasi lengkap sistem server\\.",
		"• `/system` ─ Tampilkan info sistem\n\n📋 *INFO DITAMPILKAN*\n• CPU usage\n• RAM usage\n• Disk space\n• Network stats\n• Uptime server\n• OS info",
		"`/system`",
		"",
	)
}

func getHelpHealth() string {
	return HelpDetailMessage(
		"🏥 HEALTH",
		"Mengecek kesehatan semua komponen bot\\.",
		"• `/health` ─ Cek kesehatan sistem\n\n🔍 *KOMPONEN YANG DICEK*\n• Aria2 daemon\n• Rclone\n• Database\n• Disk space\n• Memory\n• Network connectivity",
		"`/health`",
		"",
	)
}

func getHelpLogs() string {
	return HelpDetailMessage(
		"📜 LOGS",
		"Melihat log aktivitas bot\\.",
		"• `/logs` ─ Lihat 50 baris terakhir\n• `/logs <n>` ─ Lihat n baris terakhir",
		"`/logs` ─ Default 50 baris\n`/logs 100` ─ 100 baris terakhir\n`/logs 20` ─ 20 baris terakhir",
		"",
	)
}

func getHelpPing() string {
	return HelpDetailMessage(
		"🏓 PING",
		"Mengecek latency/response time bot\\.",
		"• `/ping` ─ Cek latency\n\n📋 *INFO DITAMPILKAN*\n• Response time \\(ms\\)\n• Bot status",
		"`/ping`",
		"",
	)
}

func getHelpSpeed() string {
	return HelpDetailMessage(
		"🚀 SPEED",
		"Test kecepatan internet server\\.",
		"• `/speed` ─ Jalankan speedtest\n\n📋 *INFO DITAMPILKAN*\n• Download speed\n• Upload speed\n• Ping\n• Server lokasi",
		"`/speed`",
		"",
	)
}

func getHelpLs() string {
	return HelpDetailMessage(
		"📂 LIST (LS / DIR)",
		"Melihat isi folder di cloud storage\\.\nPerintah ini mendukung navigasi interaktif melalui tombol\\.",
		"• `/ls` ─ Lihat root folder\n• `/dir` ─ Alias dari /ls\n• `/ls <path>` ─ Lihat folder tertentu",
		"`/ls` ─ Root folder\n`/ls Movies` ─ Folder Movies\n`/ls Movies/2024` ─ Subfolder",
		"💡 *TIP:* Anda bisa klik nama folder di tombol untuk masuk ke dalam folder tersebut\\.",
	)
}

func getHelpMkdir() string {
	return HelpDetailMessage(
		"📁 MKDIR",
		"Membuat folder baru di cloud storage\\.",
		"• `/mkdir <nama_folder>` ─ Buat folder",
		"`/mkdir Movies`\n`/mkdir Backup/2024`",
		"",
	)
}

func getHelpRm() string {
	return HelpDetailMessage(
		"🗑️ REMOVE (RM)",
		"Menghapus file atau folder di cloud storage\\.",
		"• `/rm <path>` ─ Hapus file/folder\n\n⚠️ *PERINGATAN*\nPenghapusan bersifat permanen\\!",
		"`/rm old_file\\.zip`\n`/rm Temp/cache`",
		"",
	)
}

func getHelpMv() string {
	return HelpDetailMessage(
		"📦 MOVE (MV)",
		"Memindahkan file/folder ke lokasi lain\\.",
		"• `/mv <source> <destination>`",
		"`/mv file\\.zip Backup/`\n`/mv OldFolder NewFolder`",
		"",
	)
}

func getHelpShare() string {
	return HelpDetailMessage(
		"🔗 SHARE",
		"Membuat link berbagi untuk file/folder\\.",
		"• `/share <path>` ─ Generate share link",
		"`/share movie\\.mp4`\n`/share SharedFolder`",
		"",
	)
}

func getHelpFind() string {
	return HelpDetailMessage(
		"🔍 FIND",
		"Mencari file di cloud storage\\.",
		"• `/find <keyword>` ─ Cari file",
		"`/find movie`\n`/find \\.mp4` ─ Cari semua MP4",
		"",
	)
}

func getHelpExtractAudio() string {
	return HelpDetailMessage(
		"🎵 EXTRACT AUDIO",
		"Mengekstrak audio dari file video\\.",
		"• Reply ke video dengan `/extractaudio`\n\n📋 *FORMAT OUTPUT*\nMP3, AAC, FLAC, WAV",
		"Reply ke video lalu ketik `/extractaudio`",
		"",
	)
}

func getHelpCompress() string {
	return HelpDetailMessage(
		"🗜️ COMPRESS",
		"Mengkompresi ukuran file video\\.",
		"• Reply ke video dengan `/compress`\n\n✨ *FITUR*\n• Kurangi ukuran file\n• Pertahankan kualitas\n• Support berbagai format",
		"Reply ke video lalu ketik `/compress`",
		"",
	)
}

func getHelpThumbnail() string {
	return HelpDetailMessage(
		"🖼️ THUMBNAIL",
		"Generate thumbnail dari video\\.",
		"• Reply ke video dengan `/thumbnail`\n\n✨ *FITUR*\n• Auto generate dari frame terbaik\n• Output format JPG/PNG",
		"Reply ke video lalu ketik `/thumbnail`",
		"",
	)
}

func getHelpScreenshots() string {
	return HelpDetailMessage(
		"📸 SCREENSHOTS",
		"Membuat multiple screenshot dari video\\.",
		"• Reply ke video dengan `/screenshots`\n• `/screenshots <jumlah>`\n\n✨ *FITUR*\n• Auto interval\n• Pilih jumlah screenshot\n• High quality output",
		"`/screenshots` ─ Default 9 screenshot\n`/screenshots 6` ─ 6 screenshot",
		"",
	)
}

func getHelpSubtitle() string {
	return HelpDetailMessage(
		"💬 SUBTITLE",
		"Embed/hardcode subtitle ke video\\.",
		"1\\. Reply ke video dengan `/subtitle`\n2\\. Kirim file subtitle \\(\\.srt/\\.ass\\)\n\n📋 *FORMAT SUBTITLE*\nSRT, ASS, SSA, VTT",
		"Reply ke video lalu ketik `/subtitle`",
		"",
	)
}

func getHelpConvert() string {
	return HelpDetailMessage(
		"🔄 CONVERT",
		"Konversi format file video/audio\\.",
		"• Reply ke file dengan `/convert <format>`\n\n🎬 *FORMAT VIDEO*\nMP4, MKV, AVI, WEBM, MOV\n\n🎵 *FORMAT AUDIO*\nMP3, AAC, FLAC, WAV, OGG",
		"`/convert mp4`\n`/convert mkv`\n`/convert mp3`",
		"",
	)
}

func getHelpMediaInfo() string {
	return HelpDetailMessage(
		"ℹ️ MEDIA INFO",
		"Melihat informasi detail file media\\.",
		"• Reply ke file dengan `/mediainfo`\n\n📋 *INFO DITAMPILKAN*\n• Format/Codec\n• Resolusi\n• Bitrate\n• Duration\n• Audio tracks\n• Subtitle tracks",
		"Reply ke video lalu ketik `/mediainfo`",
		"",
	)
}

func getHelpCancel() string {
	return HelpDetailMessage(
		"❌ CANCEL",
		"Membatalkan task tertentu\\.",
		"• `/cancel <task_id>` ─ Batalkan task\n• Klik tombol Cancel di status",
		"`/cancel abc123`",
		"💡 *TIP:* Lihat task ID dengan `/status`",
	)
}

func getHelpCancelAll() string {
	return HelpDetailMessage(
		"🚫 CANCEL ALL",
		"Membatalkan SEMUA task yang sedang berjalan\\.",
		"• `/cancelall` ─ Batalkan semua task",
		"`/cancelall`",
		"⚠️ Semua download/upload aktif akan dihentikan\\!",
	)
}

func getHelpStorages() string {
	return HelpDetailMessage(
		"📋 STORAGES",
		"Melihat daftar cloud storage yang tersedia\\.",
		"• `/storages` ─ Lihat semua storage\n\n📋 *INFO DITAMPILKAN*\n• Nama storage\n• Type \\(GDrive, OneDrive, dll\\)\n• Space tersedia\n• Status \\(aktif/tidak\\)",
		"`/storages`",
		"",
	)
}

func getHelpSetStorage() string {
	return HelpDetailMessage(
		"⚙️ SET STORAGE",
		"Mengatur storage aktif untuk upload\\.",
		"• `/setstorage <nama>` ─ Set storage",
		"`/setstorage gdrive`\n`/setstorage onedrive`",
		"💡 *TIP:* Lihat nama storage dengan `/storages`",
	)
}

func getHelpAuthorize() string {
	return HelpDetailMessage(
		"✅ AUTHORIZE",
		"Menambahkan user baru yang diizinkan menggunakan bot\\.",
		"• `/authorize <user_id>` ─ Tambah user\n• `/authorize <user_id> <role>` ─ Dengan role\n\n👤 *ROLE*\n• `user` ─ User biasa \\(default\\)\n• `admin` ─ Administrator",
		"`/authorize 123456789`\n`/authorize 123456789 admin`",
		"",
	)
}

func getHelpUnauthorize() string {
	return HelpDetailMessage(
		"❌ UNAUTHORIZE",
		"Menghapus akses user dari bot\\.",
		"• `/unauthorize <user_id>` ─ Hapus akses",
		"`/unauthorize 123456789`",
		"⚠️ User tidak akan bisa menggunakan bot lagi",
	)
}

func getHelpUsers() string {
	return HelpDetailMessage(
		"👥 USERS",
		"Melihat daftar semua user yang terdaftar\\.",
		"• `/users` ─ Lihat semua user\n\n📋 *INFO DITAMPILKAN*\n• User ID\n• Username\n• Role\n• Status",
		"`/users`",
		"",
	)
}

func getHelpSetAlertChannel() string {
	return HelpDetailMessage(
		"🚨 SET ALERT CHANNEL",
		"Mengatur channel untuk menerima alert/notifikasi penting\\.",
		"• `/setalertchannel <channel_id>` ─ Set channel\n\n🔔 *ALERT YANG DIKIRIM*\n• Task completed\n• Task failed\n• System warnings",
		"`/setalertchannel \\-1001234567890`",
		"",
	)
}

func getHelpRecover() string {
	return HelpDetailMessage(
		"🔄 RECOVER",
		"Memulihkan task yang gagal atau terhenti\\.",
		"• `/recover` ─ Pulihkan semua task gagal\n\n✨ *FITUR*\n• Resume download yang gagal\n• Retry upload yang error\n• Restore dari checkpoint",
		"`/recover`",
		"",
	)
}

func getHelpRecoveryStatus() string {
	return HelpDetailMessage(
		"📊 RECOVERY STATUS",
		"Melihat status proses recovery\\.",
		"• `/recoverystatus` ─ Lihat status\n\n📋 *INFO DITAMPILKAN*\n• Task yang bisa di\\-recover\n• Progress recovery\n• Error messages",
		"`/recoverystatus`",
		"",
	)
}

func getHelpSettings() string {
	return HelpDetailMessage(
		"⚙️ SETTINGS",
		"Mengatur preferensi dan konfigurasi bot\\.",
		"• `/settings` ─ Buka menu pengaturan\n\n📋 *PENGATURAN TERSEDIA*\n• **Auto Delete Messages**\n  Hapus otomatis pesan perintah dan bot setelah 60s\\.\n• **Default Mode**\n  Pilih antara Mirror atau Leech sebagai mode default\\.",
		"`/settings`",
		"💡 Pengaturan ini bersifat global dan mempengaruhi cara bot berinteraksi dengan Anda\\.",
	)
}

func getHelpSetLogChannel() string {
	return HelpDetailMessage(
		"📜 SET LOG CHANNEL",
		"Mengatur channel khusus untuk menerima log real-time aktivitas bot\\.",
		"• `/setlogchannel <channel_id>` ─ Set channel log",
		"`/setlogchannel -1001234567890`",
		"⚠️ Pastikan bot sudah menjadi admin di channel tersebut\\.",
	)
}

func getHelpAllCommands() string {
	content := "📋 *DAFTAR SELURUH PERINTAH BOT*\n\n" +
		"📥 *DOWNLOAD*\n" +
		"• `/mirror` ─ Mirror ke Drive\n" +
		"• `/leech` ─ Leech ke Telegram\n" +
		"• `/ytdlp` ─ YT\\-DLP Download\n" +
		"• `/torrent` ─ Torrent Download\n" +
		"• `/clone` ─ Clone GDrive\n" +
		"• `/batch` ─ Batch Download\n" +
		"• `/batchstatus` ─ Status Batch\n" +
		"• `/cancelbatch` ─ Batal Batch\n" +
		"• `/search` ─ Cari Torrent\n\n" +
		"📊 *MONITOR & SYSTEM*\n" +
		"• `/status` ─ Status Aktif\n" +
		"• `/stats` ─ Statistik Bot\n" +
		"• `/system` ─ Info System\n" +
		"• `/health` ─ Cek Kesehatan\n" +
		"• `/logs` ─ Lihat Log Bot\n" +
		"• `/ping` ─ Cek Latency\n" +
		"• `/speed` ─ Test Kecepatan\n\n" +
		"📁 *STORAGE & FILES*\n" +
		"• `/ls` / `/dir` ─ List File \\(Drive\\)\n" +
		"• `/mkdir` ─ Buat Folder\n" +
		"• `/rm` ─ Hapus File/Folder\n" +
		"• `/mv` ─ Pindah/Rename\n" +
		"• `/share` ─ Share Link\n" +
		"• `/find` ─ Cari File Drive\n" +
		"• `/storages` ─ Daftar Storage\n" +
		"• `/setstorage` ─ Aktifkan Storage\n\n" +
		"🎞️ *MEDIA TOOLS*\n" +
		"• `/extractaudio` ─ Ambil Audio\n" +
		"• `/compress` ─ Kompres Video\n" +
		"• `/thumbnail` ─ Buat Thumb\n" +
		"• `/screenshots` ─ Screenshot\n" +
		"• `/subtitle` ─ Embed Sub\n" +
		"• `/convert` ─ Ganti Format\n" +
		"• `/mediainfo` ─ Detail Media\n\n" +
		"👑 *ADMINISTRATION*\n" +
		"• `/authorize` ─ Beri Akses\n" +
		"• `/unauthorize` ─ Cabut Akses\n" +
		"• `/users` ─ Daftar User\n" +
		"• `/setlogchannel` ─ Set Log\n" +
		"• `/setalertchannel` ─ Set Alert\n\n" +
		"🛠️ *SETTINGS & GENERAL*\n" +
		"• `/settings` ─ Menu Pengaturan\n" +
		"• `/start` ─ Pesan Awal\n" +
		"• `/help` ─ Menu Bantuan\n" +
		"• `/recover` ─ Pulihkan Task\n" +
		"• `/recoverystatus` ─ Status Recovery\n" +
		"• `/cancel` ─ Batal Task ID\n" +
		"• `/cancelall` ─ Batal Semua"

	return ProfessionalMessage("ALL COMMANDS", content)
}
