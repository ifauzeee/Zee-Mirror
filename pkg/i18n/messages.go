package i18n

import (
	"fmt"
	"strings"
)

var translations = map[string]map[string]string{
	"id": {
		"error_header":            "❌ *Error*",
		"access_denied":           "🚫 *Akses Ditolak*\n\nAnda tidak memiliki izin untuk menggunakan fitur ini\\.",
		"quota_exceeded":          "⚠️ *Kuota Terlampaui*\n\n%s",
		"invalid_url":             "❌ *URL Tidak Valid*\n\nBerikan URL yang benar\\.",
		"reply_required":          "❌ *Error*\n\nReply ke file atau berikan URL\\.",
		"ytdlp_analysis":          "🎬 Menganalisa kualitas video…",
		"ytdlp_analysis_failed":   "❌ *Gagal menganalisa video:* %s\n\n_Pastikan URL valid atau coba lagi nanti\\._",
		"ytdlp_select_quality":    "📽️ *Pilih Kualitas Video*\n\nVideo ini mendukung resolusi berikut:",
		"ytdlp_no_resolution":     "📽️ *Pilih Kualitas Video*\n\nResolusi tidak terdeteksi, gunakan kualitas terbaik:",
		"ytdlp_session_expired":   "❌ Sesi kadaluarsa",
		"ytdlp_session_not_found": "❌ Sesi tidak ditemukan",
		"ytdlp_playlist_error":    "❌ *Error*\n\nURL yang Anda berikan adalah link Channel atau Playlist\\. Bot ini hanya mendukung download video tunggal untuk saat ini\\.",
		"download_started":        "✅ *Download dimulai*",
		"torrent_mirror_title":    "🧲 *TORRENT MIRROR*",
		"torrent_select_all":      "📦 Select All",
		"torrent_select_files":    "📋 Select Files \\(Web\\)",
		"torrent_cancel":          "❌ Cancel",
		"torrent_selected_start":  "✅ *Download dimulai*\n\nMendownload %d file yang dipilih\\.\\.\\.",
		"telegram_file_limit":     "\n\n⚠️ *Limitasi Telegram:* Bot hanya dapat mengunduh file hingga 20MB melalui server resmi\\. Gunakan *Local Bot API Server* untuk mengunduh file hingga 2GB\\.",
		"no_active_tasks":         "❌ *Tidak ada task aktif\\.*",
		"dashboard_refreshed":     "🔄 Dashboard direfresh",
		"paging_error":            "Halaman tidak ditemukan\\.",
		"task_cancelled":          "Task `%s` telah dibatalkan\\.",
		"batch_cancelled":         "Batch `%s` dibatalkan",
		"sub_task_cancelled":      "Sub\\-Task `%s` dibatalkan",
		"task_not_found":          "Task/Batch `%s` tidak ditemukan",
		"admin_only":              "Fitur ini hanya untuk Admin/Owner\\.",
		"all_tasks_cancelled":     "%d tugas/batch aktif telah dibatalkan\\.",
		"status_completed":        "✅ *Selesai\\!*",
		"status_failed":           "❌ *Gagal\\!*",
		"queuing":                 "⌛ Antre",
		"downloading":             "📥 Mendownload",
		"uploading":               "📤 Mengupload",
		"cloning":                 "📋 Mengkloning",
		"waiting":                 "💤 Menunggu",
		"processing":              "⚙️ Memproses",
		"completed":               "✅ Selesai",
		"failed":                  "❌ Gagal",
		"cancelled":               "🚫 Dibatalkan",
		"invalid_magnet":          "❌ *Error*\n\nBerikan magnet link\\.",
		"torrent_select_web":      "🧲 *TORRENT SELECTION*\n\nSilakan pilih file yang ingin di-download melalui Dashboard Web:",
		"torrent_menu_text":       "🧲 *TORRENT MIRROR*\n━━━━━━━━━━━━━━━━━━━━━━\n\nPilih opsi download untuk torrent ini:\n\n📦 *Select All* \\- Download semua file dalam torrent\n📋 *Select Files* \\- Pilih file tertentu via Web\n\n_Torrent biasanya berisi banyak file dalam folder\\. Gunakan Select Files untuk memilih file yang ingin didownload\\._",
		"welcome_title":           "ZEE-MIRROR BOT",
		"welcome_content":         "👋 Hai, *%s*\\!\n\nSelamat datang di *ZEE\\-MIRROR*\\.\nBot serbaguna untuk Mirror, Leech, dan Media Tools\\.\n\n🚀 *Didesain untuk kecepatan dan kemudahan\\.*",
		"help_title":              "PANDUAN BANTUAN",
		"help_content":            "Silakan pilih kategori bantuan di bawah untuk melihat detail fungsi dan cara penggunaan\\.\n\n💡 *Klik tombol untuk membuka sub\\-menu\\.*",
		"help_download":           "📥 DOWNLOAD",
		"help_monitor":            "📊 MONITOR",
		"help_files":              "📁 FILES",
		"help_media":              "🎵 MEDIA",
		"help_task":               "📋 TASK",
		"help_storage":            "💾 STORAGE",
		"help_admin":              "👑 ADMIN",
		"help_recovery":           "🔧 RECOVERY",
		"help_settings":           "⚙️ SETTINGS",
		"help_all":                "📋 ALL COMMANDS",
		"help_home":               "🔙 HOME",
		"help_close":              "✖️ Close",
		"help_back":               "🔙 Kembali",
		"language_set":            "✅ Bahasa diatur ke: *Bahasa Indonesia*",
	},
	"ja": {
		"error_header":            "❌ *エラー*",
		"access_denied":           "🚫 *アクセス拒否*\n\nこの機能を使用する権限がありません。",
		"quota_exceeded":          "⚠️ *クォータ超過*\n\n%s",
		"invalid_url":             "❌ *無効なURL*\n\n正しいURLを提供してください。",
		"reply_required":          "❌ *エラー*\n\nファイルに返信するか、URLを提供してください。",
		"ytdlp_analysis":          "🎬 ビデオ品質を分析中...",
		"ytdlp_analysis_failed":   "❌ *ビデオの分析に失敗しました:* %s\n\n_URLが有効であることを確認するか、後で再試行してください。_",
		"ytdlp_select_quality":    "📽️ *ビデオ品質を選択*\n\nこのビデオは以下の解像度をサポートしています:",
		"ytdlp_no_resolution":     "📽️ *ビデオ品質を選択*\n\n解像度が検出されませんでした。最高の品質を使用します:",
		"ytdlp_session_expired":   "❌ セッションが期限切れです",
		"ytdlp_session_not_found": "❌ セッションが見つかりません",
		"download_started":        "✅ *ダウンロード開始*",
		"status_completed":        "✅ *完了！*",
		"status_failed":           "❌ *失敗！*",
		"queuing":                 "⌛ 待機中",
		"downloading":             "📥 ダウンロード中",
		"uploading":               "📤 アップロード中",
		"cloning":                 "📋 クローニング中",
		"completed":               "✅ 完了",
		"failed":                  "❌ 失敗",
		"cancelled":               "🚫 キャンセル済み",
		"welcome_title":           "ZEE-MIRROR ボット",
		"welcome_content":         "👋 こんにちは、*%s*！\n\n*ZEE-MIRROR*へようこそ。\nミラー、リーチ、メディアツールのための多機能ボットです。\n\n🚀 *スピードと使いやすさを追求した設計。*",
		"help_title":              "ヘルプガイド",
		"help_home":               "🔙 ホーム",
		"help_close":              "✖️ 閉じる",
		"help_back":               "🔙 戻る",
		"language_set":            "✅ 言語が設定されました: *日本語*",
	},
	"zh": {
		"error_header":            "❌ *错误*",
		"access_denied":           "🚫 *拒绝访问*\n\n您没有权限使用此功能。",
		"quota_exceeded":          "⚠️ *超过配额*\n\n%s",
		"invalid_url":             "❌ *无效URL*\n\n请提供正确的URL。",
		"reply_required":          "❌ *错误*\n\n请回复文件或提供URL。",
		"ytdlp_analysis":          "🎬 正在分析视频质量...",
		"ytdlp_analysis_failed":   "❌ *分析视频失败:* %s\n\n_请确保URL有效或稍后重试。_",
		"ytdlp_select_quality":    "📽️ *选择视频质量*\n\n此视频支持以下分辨率:",
		"ytdlp_no_resolution":     "📽️ *选择视频质量*\n\n未检测到分辨率，使用最佳质量:",
		"ytdlp_session_expired":   "❌ 会话已过期",
		"ytdlp_session_not_found": "❌ 会话未找到",
		"download_started":        "✅ *下载已开始*",
		"status_completed":        "✅ *已完成！*",
		"status_failed":           "❌ *已失败！*",
		"queuing":                 "⌛ 排队中",
		"downloading":             "📥 下载中",
		"uploading":               "📤 上传中",
		"cloning":                 "📋 克隆中",
		"completed":               "✅ 已完成",
		"failed":                  "✅ 已失败",
		"cancelled":               "🚫 已取消",
		"welcome_title":           "ZEE-MIRROR 机器人",
		"welcome_content":         "👋 你好，*%s*！\n\n欢迎使用 *ZEE-MIRROR*。\n一款用于镜像、离线转换和媒体工具的多功能机器人。\n\n🚀 *专为速度和易用性打造。*",
		"help_title":              "帮助指南",
		"help_home":               "🔙 首页",
		"help_close":              "✖️ 关闭",
		"help_back":               "🔙 返回",
		"language_set":            "✅ 语言已设置为: *简体中文*",
	},
	"en": {
		"error_header":            "❌ *Error*",
		"access_denied":           "🚫 *Access Denied*\n\nYou do not have permission to use this feature\\.",
		"quota_exceeded":          "⚠️ *Quota Exceeded*\n\n%s",
		"invalid_url":             "❌ *Invalid URL*\n\nPlease provide a correct URL\\.",
		"reply_required":          "❌ *Error*\n\nReply to a file or provide a URL\\.",
		"ytdlp_analysis":          "🎬 Analyzing video quality...",
		"ytdlp_analysis_failed":   "❌ *Failed to analyze video:* %s\n\n_Make sure the URL is valid or try again later\\._",
		"ytdlp_select_quality":    "📽️ *Select Video Quality*\n\nThis video supports the following resolutions:",
		"ytdlp_no_resolution":     "📽️ *Select Video Quality*\n\nResolution not detected, using best quality:",
		"ytdlp_session_expired":   "❌ Session expired",
		"ytdlp_session_not_found": "❌ Session not found",
		"ytdlp_playlist_error":    "❌ *Error*\n\nThe URL provided is a Channel or Playlist link\\. This bot only supports single video downloads for now\\.",
		"download_started":        "✅ *Download started*",
		"torrent_mirror_title":    "🧲 *TORRENT MIRROR*",
		"torrent_select_all":      "📦 Select All",
		"torrent_select_files":    "📋 Select Files \\(Web\\)",
		"torrent_cancel":          "❌ Cancel",
		"torrent_selected_start":  "✅ *Download started*\n\nDownloading %d selected files\\.\\.\\.",
		"telegram_file_limit":     "\n\n⚠️ *Telegram Limitation:* Regular bot only supports files up to 20MB\\. Use *Local Bot API Server* to download files up to 2GB\\.",
		"no_active_tasks":         "❌ *No active tasks\\.*",
		"dashboard_refreshed":     "🔄 Dashboard refreshed",
		"paging_error":            "Page not found\\.",
		"task_cancelled":          "Task `%s` has been cancelled\\.",
		"batch_cancelled":         "Batch `%s` cancelled",
		"sub_task_cancelled":      "Sub\\-Task `%s` cancelled",
		"task_not_found":          "Task/Batch `%s` not found",
		"admin_only":              "This feature is for Admin/Owner only\\.",
		"all_tasks_cancelled":     "%d active tasks/batches have been cancelled\\.",
		"status_completed":        "✅ *Completed\\!*",
		"status_failed":           "❌ *Failed\\!*",
		"queuing":                 "⌛ Queuing",
		"downloading":             "📥 Downloading",
		"uploading":               "📤 Uploading",
		"cloning":                 "📋 Cloning",
		"waiting":                 "💤 Waiting",
		"processing":              "⚙️ Processing",
		"completed":               "✅ Completed",
		"failed":                  "❌ Failed",
		"cancelled":               "🚫 Cancelled",
		"invalid_magnet":          "❌ *Error*\n\nPlease provide a magnet link\\.",
		"torrent_select_web":      "🧲 *TORRENT SELECTION*\n\nPlease select the files you want to download via the Web Dashboard:",
		"torrent_menu_text":       "🧲 *TORRENT MIRROR*\n━━━━━━━━━━━━━━━━━━━━━━\n\nChoose download options for this torrent:\n\n📦 *Select All* \\- Download all files in the torrent\n📋 *Select Files* \\- Select specific files via Web\n\n_Torrents usually contain many files in a folder\\. Use Select Files to choose specific files to download\\._",
		"welcome_title":           "ZEE-MIRROR BOT",
		"welcome_content":         "👋 Hi, *%s*\\!\n\nWelcome to *ZEE\\-MIRROR*\\.\nA versatile bot for Mirror, Leech, and Media Tools\\.\n\n🚀 *Designed for speed and ease of use\\.*",
		"help_title":              "HELP GUIDE",
		"help_content":            "Please select a help category below to see function details and usage guide\\.\n\n💡 *Click a button to open a sub\\-menu\\.*",
		"help_download":           "📥 DOWNLOAD",
		"help_monitor":            "📊 MONITOR",
		"help_files":              "📁 FILES",
		"help_media":              "🎵 MEDIA",
		"help_task":               "📋 TASK",
		"help_storage":            "💾 STORAGE",
		"help_admin":              "👑 ADMIN",
		"help_recovery":           "🔧 RECOVERY",
		"help_settings":           "⚙️ SETTINGS",
		"help_all":                "📋 ALL COMMANDS",
		"help_home":               "🔙 HOME",
		"help_close":              "✖️ Close",
		"help_back":               "🔙 Back",
		"language_set":            "✅ Language set to: *English*",
	},
}

func T(lang, key string, args ...any) string {
	if lang == "" {
		lang = "id"
	}
	lang = strings.ToLower(lang)

	t, ok := translations[lang]
	if !ok {
		t = translations["id"]
	}

	msg, ok := t[key]
	if !ok {
		if lang != "id" {
			msg = translations["id"][key]
		}
		if msg == "" {
			return key
		}
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

const (
	MsgErrorHeader   = "❌ *Error*"
	MsgAccessDenied  = "🚫 *Akses Ditolak*\n\nAnda tidak memiliki izin untuk menggunakan fitur ini\\."
	MsgReplyRequired = "❌ *Error*\n\nReply ke file atau berikan URL\\."
	MsgInvalidURL    = "❌ *URL Tidak Valid*\n\nBerikan URL yang benar\\."
)
