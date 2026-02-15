package service

import (
	"testing"
)

func TestConstructScrapeURL(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		originalURL string
		want        string
		isFolder    bool
	}{
		{
			name:        "Empty ID",
			id:          "",
			originalURL: "https://example.com",
			want:        "https://example.com",
			isFolder:    false,
		},
		{
			name:        "File ID",
			id:          "12345",
			originalURL: "https://drive.usercontent.google.com/...",
			want:        "https://drive.google.com/file/d/12345/view",
			isFolder:    false,
		},
		{
			name:        "Folder ID",
			id:          "folder123",
			originalURL: "https://drive.google.com/drive/folders/folder123",
			want:        "https://drive.google.com/drive/folders/folder123",
			isFolder:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConstructScrapeURL(tt.id, tt.isFolder, tt.originalURL)
			if got != tt.want {
				t.Errorf("ConstructScrapeURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractDriveID(t *testing.T) {
	tests := []struct {
		urlStr       string
		wantID       string
		wantIsFolder bool
	}{
		{"https://drive.google.com/drive/folders/1abcDEfg", "1abcDEfg", true},
		{"https://drive.google.com/file/d/1xyzABC/view", "1xyzABC", false},
		{"https://drive.usercontent.google.com/download?id=1xyzABC&export=download", "1xyzABC", false},
		{"https://docs.google.com/uc?id=1xyzABC", "1xyzABC", false},
		{"invalid_url", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.urlStr, func(t *testing.T) {
			gotID, gotIsFolder := extractDriveID(tt.urlStr)
			if gotID != tt.wantID {
				t.Errorf("extractDriveID() gotID = %v, want %v", gotID, tt.wantID)
			}
			if gotIsFolder != tt.wantIsFolder {
				t.Errorf("extractDriveID() gotIsFolder = %v, want %v", gotIsFolder, tt.wantIsFolder)
			}
		})
	}
}
