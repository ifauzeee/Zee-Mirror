package utils

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		expected string
		input    int64
	}{
		{"100 B", 100},
		{"1.00 KB", 1024},
		{"1.00 MB", 1048576},
		{"1.00 GB", 1073741824},
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
		expected string
		input    time.Duration
	}{
		{"10s", 10 * time.Second},
		{"1m 5s", 65 * time.Second},
		{"1h 1m 5s", 3665 * time.Second},
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

func TestParseBytesString(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1.5GB", 1500000000},
		{"500MB", 500000000},
		{"100KB", 100000},
		{"2048B", 2048},
		{"1.25GiB", 1342177280},
		{"256MiB", 268435456},
		{"512KiB", 524288},
		{"invalid", 0},
		{"", 0},
		{"10.5KB", 10500},
		{"0.5MB", 500000},
	}

	for _, test := range tests {
		result := ParseBytesString(test.input)
		if result != test.expected {
			t.Errorf("ParseBytesString(%s) = %d; want %d", test.input, result, test.expected)
		}
	}
}

func TestExtractURLFromText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Check out https://example.com for more", "https://example.com"},
		{"Visit http://test.org/path", "http://test.org/path"},
		{"No URL here", ""},
		{"Multiple https://first.com and https://second.com URLs", "https://first.com"},
		{"", ""},
	}

	for _, test := range tests {
		result := ExtractURLFromText(test.input)
		if result != test.expected {
			t.Errorf("ExtractURLFromText(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestExtractMagnetFromText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Download: magnet:?xt=urn:btih:abc123", "magnet:?xt=urn:btih:abc123"},
		{"No magnet here", ""},
		{"", ""},
		{"Text before magnet:?xt=urn:btih:xyz789 text after", "magnet:?xt=urn:btih:xyz789"},
	}

	for _, test := range tests {
		result := ExtractMagnetFromText(test.input)
		if result != test.expected {
			t.Errorf("ExtractMagnetFromText(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		url      string
		password string
		quality  string
		fileName string
		zip      bool
		unzip    bool
	}{
		{
			name:     "URL with zip flag",
			input:    "https://example.com/file.zip -z",
			url:      "https://example.com/file.zip",
			zip:      true,
			unzip:    false,
			password: "",
			quality:  "",
			fileName: "",
		},
		{
			name:     "URL with unzip flag",
			input:    "https://example.com/file.zip -uz",
			url:      "https://example.com/file.zip",
			zip:      false,
			unzip:    true,
			password: "",
			quality:  "",
			fileName: "",
		},
		{
			name:     "URL with password",
			input:    "https://example.com/file.zip -p secret123",
			url:      "https://example.com/file.zip",
			zip:      false,
			unzip:    false,
			password: "secret123",
			quality:  "",
			fileName: "",
		},
		{
			name:     "URL with quality",
			input:    "https://youtube.com/watch?v=123 -q 1080",
			url:      "https://youtube.com/watch?v=123",
			zip:      false,
			unzip:    false,
			password: "",
			quality:  "1080",
			fileName: "",
		},
		{
			name:     "URL with custom name",
			input:    "https://example.com/file.zip -n MyFile.zip",
			url:      "https://example.com/file.zip",
			zip:      false,
			unzip:    false,
			password: "",
			quality:  "",
			fileName: "MyFile.zip",
		},
		{
			name:     "Multiple flags combined",
			input:    "https://example.com/file.zip -z -p secret -n CustomName.zip",
			url:      "https://example.com/file.zip",
			zip:      true,
			unzip:    false,
			password: "secret",
			quality:  "",
			fileName: "CustomName.zip",
		},
		{
			name:     "No flags",
			input:    "https://example.com/file.zip",
			url:      "https://example.com/file.zip",
			zip:      false,
			unzip:    false,
			password: "",
			quality:  "",
			fileName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, zip, unzip, password, quality, fileName, _, _ := ParseFlags(tt.input)

			if url != tt.url {
				t.Errorf("URL = %q; want %q", url, tt.url)
			}
			if zip != tt.zip {
				t.Errorf("Zip = %v; want %v", zip, tt.zip)
			}
			if unzip != tt.unzip {
				t.Errorf("Unzip = %v; want %v", unzip, tt.unzip)
			}
			if password != tt.password {
				t.Errorf("Password = %q; want %q", password, tt.password)
			}
			if quality != tt.quality {
				t.Errorf("Quality = %q; want %q", quality, tt.quality)
			}
			if fileName != tt.fileName {
				t.Errorf("FileName = %q; want %q", fileName, tt.fileName)
			}
		})
	}
}

func TestScanLinesWithCR(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		token    []byte
		advance  int
		atEOF    bool
		hasError bool
	}{
		{
			name:     "Line ending with LF",
			input:    []byte("hello\nworld"),
			atEOF:    false,
			advance:  6,
			token:    []byte("hello"),
			hasError: false,
		},
		{
			name:     "Line ending with CRLF",
			input:    []byte("hello\r\nworld"),
			atEOF:    false,
			advance:  6,
			token:    []byte("hello"),
			hasError: false,
		},
		{
			name:     "Line ending with CR only",
			input:    []byte("hello\rworld"),
			atEOF:    false,
			advance:  6,
			token:    []byte("hello"),
			hasError: false,
		},
		{
			name:     "At EOF with data",
			input:    []byte("hello"),
			atEOF:    true,
			advance:  5,
			token:    []byte("hello"),
			hasError: false,
		},
		{
			name:     "Empty at EOF",
			input:    []byte(""),
			atEOF:    true,
			advance:  0,
			token:    nil,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advance, token, err := ScanLinesWithCR(tt.input, tt.atEOF)

			if tt.hasError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if advance != tt.advance {
				t.Errorf("Advance = %d; want %d", advance, tt.advance)
			}
			if string(token) != string(tt.token) {
				t.Errorf("Token = %q; want %q", string(token), string(tt.token))
			}
		})
	}
}
