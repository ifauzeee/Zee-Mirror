package utils

import (
	"strings"
	"testing"
)

func TestGetFileNameFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "Simple filename in query",
			url:  "http://example.com/file?filename=cool_video.mp4",
			want: "cool_video.mp4",
		},
		{
			name: "Signed URL with content disposition",
			url:  `https://vikingfile.com/abc?response-content-disposition=attachment%3B%20filename%3D%22Grenland.2.Migration.2026.PROPER.1080p.mkv%22&X-Amz-Content-Sha256=UNSIGNED-PAYLOAD`,
			want: "Grenland.2.Migration.2026.PROPER.1080p.mkv",
		},
		{
			name: "Magnet link with DN",
			url:  "magnet:?xt=urn:btih:12345&dn=Big%20Buck%20Bunny.mp4",
			want: "Big_Buck_Bunny.mp4",
		},
		{
			name: "Path based filename",
			url:  "https://example.com/videos/awesome_movie.mkv",
			want: "awesome_movie.mkv",
		},
		{
			name: "PixelDrain API",
			url:  "https://pixeldrain.com/api/file/12345/MyFile.zip",
			want: "MyFile.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFileNameFromURL(tt.url)
			got = strings.ReplaceAll(got, " ", "_")
			tt.want = strings.ReplaceAll(tt.want, " ", "_")

			if got != tt.want {
				t.Errorf("GetFileNameFromURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
