package organizer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeArchive(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"zip without ext", write("noext-zip", []byte("PK\x03\x04payload")), true},
		{"rar without ext", write("noext-rar", []byte("Rar!\x1a\x07\x01\x00")), true},
		{"7z without ext", write("noext-7z", []byte("7z\xbc\xaf'\x1cpayload")), true},
		{"plain text", write("plain.txt", []byte("just text, not an archive")), false},
		{"too short", write("tiny", []byte("PK")), false},
		{"missing file", filepath.Join(dir, "nope"), false},
	}
	for _, tc := range tests {
		if got := LooksLikeArchive(tc.path); got != tc.want {
			t.Errorf("%s: LooksLikeArchive(%q) = %v, want %v", tc.name, tc.path, got, tc.want)
		}
	}
}
