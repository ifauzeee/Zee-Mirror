package handlers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ProgressBar(progress float64, width int) string {
	if width <= 0 {
		width = 10
	}

	filled := int(progress / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	empty := width - filled

	filledChar := "■"
	emptyChar := "□"

	bar := strings.Repeat(filledChar, filled) + strings.Repeat(emptyChar, empty)
	return fmt.Sprintf("[%s] %.1f%%", bar, progress)
}

func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func FormatSpeed(bytesPerSec int64) string {
	return FormatBytes(bytesPerSec) + "/s"
}

func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "∞"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func EscapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

func SanitizePath(path string) string {
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "..", "")
	reg := regexp.MustCompile(`[^a-zA-Z0-9._\-\/\\]`)
	path = reg.ReplaceAllString(path, "_")
	return path
}

func SanitizeFileName(filename string) string {
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	filename = reg.ReplaceAllString(filename, "_")

	if len(filename) > 200 {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		if len(name) > 200-len(ext) {
			name = name[:200-len(ext)]
		}
		filename = name + ext
	}

	return filename
}

func ExtractURLFromText(text string) string {
	urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	matches := urlRegex.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func ExtractMagnetFromText(text string) string {
	magnetRegex := regexp.MustCompile(`magnet:\?xt=urn:[^\s]+`)
	matches := magnetRegex.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func ParseFlags(args string) (url string, zip bool, unzip bool, password string, quality string) {
	parts := strings.Fields(args)

	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "-z":
			zip = true
		case "-uz":
			unzip = true
		case "-p":
			if i+1 < len(parts) {
				password = parts[i+1]
				i++
			}
		case "-q":
			if i+1 < len(parts) {
				quality = parts[i+1]
				i++
			}
		default:
			if url == "" && (strings.HasPrefix(parts[i], "http") || strings.HasPrefix(parts[i], "magnet:")) {
				url = parts[i]
			}
		}
	}

	return
}

func IsValidURL(s string) bool {
	urlRegex := regexp.MustCompile(`^(https?|ftp|magnet):`)
	return urlRegex.MatchString(s)
}

func IsMagnetLink(s string) bool {
	return strings.HasPrefix(s, "magnet:?")
}

func IsTorrentFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".torrent")
}

func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	return strings.ToLower(ext)
}

func IsArchiveFile(filename string) bool {
	ext := GetFileExtension(filename)
	archiveExts := []string{".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz", ".tgz"}
	for _, e := range archiveExts {
		if ext == e || strings.HasSuffix(strings.ToLower(filename), e) {
			return true
		}
	}
	return false
}

func IsVideoFile(filename string) bool {
	ext := GetFileExtension(filename)
	videoExts := []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v"}
	for _, e := range videoExts {
		if ext == e {
			return true
		}
	}
	return false
}

func GenerateThumbnail(videoPath string) (string, error) {
	videoPath = filepath.Clean(videoPath)
	if !filepath.IsAbs(videoPath) {
		return "", fmt.Errorf("video path must be absolute")
	}

	thumbnailPath := videoPath + ".jpg"

	if taskManager == nil {
		return "", fmt.Errorf("task manager not initialized")
	}
	allowedBaseDir := filepath.Clean(taskManager.DownloadDir)
	videoDir := filepath.Dir(videoPath)

	if !strings.HasPrefix(videoDir, allowedBaseDir) {
		return "", fmt.Errorf("video path is not within allowed directory")
	}

	cmd := exec.Command("ffmpeg", "-i", videoPath, "-ss", "00:00:05", "-vframes", "1", "-q:v", "2", thumbnailPath, "-y")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("ffmpeg", "-i", videoPath, "-ss", "00:00:00", "-vframes", "1", "-q:v", "2", thumbnailPath, "-y")
		if err := cmd.Run(); err != nil {
			return "", err
		}
	}
	return thumbnailPath, nil
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}

func StatusEmoji(status string) string {
	switch status {
	case "queued":
		return "⏳"
	case "downloading":
		return "📥"
	case "extracting":
		return "📂"
	case "zipping":
		return "🗜️"
	case "uploading":
		return "📤"
	case "completed":
		return "✅"
	case "failed":
		return "❌"
	case "cancelled":
		return "🚫"
	default:
		return "❓"
	}
}

func FormatStatus(status string) string {
	if status == "" {
		return ""
	}
	return strings.ToUpper(status[:1]) + status[1:]
}

func GetFileNameFromURL(url string) string {
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		if name != "" {
			return SanitizeFileName(name)
		}
	}

	return "unknown_file"
}

func ParseBytesString(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	mult := int64(1)
	suffixes := map[string]int64{
		"k":   1000,
		"m":   1000 * 1000,
		"g":   1000 * 1000 * 1000,
		"t":   1000 * 1000 * 1000 * 1000,
		"kb":  1000,
		"mb":  1000 * 1000,
		"gb":  1000 * 1000 * 1000,
		"tb":  1000 * 1000 * 1000 * 1000,
		"kib": 1024,
		"mib": 1024 * 1024,
		"gib": 1024 * 1024 * 1024,
		"tib": 1024 * 1024 * 1024 * 1024,
	}

	lowerS := strings.ToLower(s)
	lowerS = strings.TrimSuffix(lowerS, "/s")
	lowerS = strings.TrimSuffix(lowerS, "b/s")
	lowerS = strings.TrimSuffix(lowerS, "/sec")
	lowerS = strings.TrimSpace(lowerS)

	for suffix, m := range suffixes {
		if strings.HasSuffix(lowerS, suffix) {
			mult = m
			s = s[:len(s)-len(suffix)]
			break
		}
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	return int64(val * float64(mult))
}

func ParseSizeString(s string) int64 {
	return ParseBytesString(s)
}

func calculateDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
