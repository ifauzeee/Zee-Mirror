package utils

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1.00 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.input)
		if result != test.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", test.input, result, test.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{10 * time.Second, "10s"},
		{65 * time.Second, "1m 5s"},
		{3665 * time.Second, "1h 1m 5s"},
	}

	for _, test := range tests {
		result := FormatDuration(test.input)
		if result != test.expected {
			t.Errorf("FormatDuration(%v) = %s; want %s", test.input, result, test.expected)
		}
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"file.txt", "file.txt"},
		{"file/with/slash.txt", "file_with_slash.txt"},
		{"file*with*star.txt", "file_with_star.txt"},
	}

	for _, test := range tests {
		result := SanitizeFileName(test.input)
		if result != test.expected {
			t.Errorf("SanitizeFileName(%s) = %s; want %s", test.input, result, test.expected)
		}
	}
}
func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"file_1.txt", "file\\_1\\.txt"},
		{"[progress]", "\\[progress\\]"},
		{"already\\_escaped", "already\\_escaped"},
		{"lone\\backslash", "lone\\\\backslash"},
		{"mixed *bold* and _italic_", "mixed \\*bold\\* and \\_italic\\_"},
		{"| pipes |", "\\| pipes \\|"},
	}

	for _, test := range tests {
		result := EscapeMarkdownV2(test.input)
		if result != test.expected {
			t.Errorf("EscapeMarkdownV2(%s) = %s; want %s", test.input, result, test.expected)
		}
	}
}
