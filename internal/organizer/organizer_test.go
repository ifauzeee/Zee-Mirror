package organizer

import (
	"testing"
)

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"movie.mp4", true},
		{"video.mkv", true},
		{"clip.avi", true},
		{"document.pdf", false},
		{"song.mp3", false},
		{"", false},
		{"no_extension", false},
		{"video.MP4", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsVideoFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsVideoFile(%q) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsMusicFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"song.mp3", true},
		{"track.flac", true},
		{"audio.wav", true},
		{"video.mp4", false},
		{"document.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsMusicFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsMusicFile(%q) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"photo.jpg", true},
		{"image.png", true},
		{"graphic.gif", true},
		{"picture.webp", true},
		{"video.mp4", false},
		{"song.mp3", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsImageFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsImageFile(%q) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsArchiveFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"archive.zip", true},
		{"backup.rar", true},
		{"compressed.7z", true},
		{"tarball.tar.gz", true},
		{"file.txt", false},
		{"video.mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsArchiveFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsArchiveFile(%q) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsDocumentFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"document.pdf", true},
		{"report.docx", true},
		{"sheet.xlsx", true},
		{"presentation.pptx", true},
		{"notes.txt", true},
		{"video.mp4", false},
		{"song.mp3", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsDocumentFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsDocumentFile(%q) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsTorrentFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"file.torrent", true},
		{"movie.TORRENT", true},
		{"video.mp4", false},
		{"file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsTorrentFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsTorrentFile(%q) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsSubtitleFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"subtitles.srt", true},
		{"captions.ass", true},
		{"subs.vtt", true},
		{"video.mp4", false},
		{"text.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsSubtitleFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsSubtitleFile(%q) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGetTargetFolder(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"video.mp4", "🎥 Videos"},
		{"song.mp3", "🎵 Music"},
		{"photo.jpg", "📸 Images"},
		{"document.pdf", "📄 Documents"},
		{"archive.zip", "📦 Archives"},
		{"app.exe", "🛠️ Applications"},
		{"subtitle.srt", "📜 Subtitles"},
		{"file.torrent", "🌊 Torrents"},
		{"unknown.xyz", "🔬 Scientific"},
		{"", "📂 Others"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := GetTargetFolder(tt.filename)
			if result != tt.expected {
				t.Errorf("GetTargetFolder(%q) = %q; want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsDevelopmentFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"style.css", true},
		{"index.html", true},
		{"script.jsx", true},
		{"component.tsx", true},
		{"video.mp4", false},
		{"document.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsDevelopmentFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsDevelopmentFile(%q) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}
