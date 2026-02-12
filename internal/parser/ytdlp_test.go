package parser

import (
	"testing"
	"time"
)

func TestParseYTDLPLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected YTDLPProgress
	}{
		{
			name:  "valid progress line with GB",
			input: "[download]  45.5% of 1.25GiB at 2.5MiB/s ETA 01:23:45",
			expected: YTDLPProgress{
				Progress:   45.5,
				Total:      1342177280,
				Downloaded: 610820659,
				Speed:      2621440,
				ETA:        time.Hour + 23*time.Minute + 45*time.Second,
				Found:      true,
			},
		},
		{
			name:  "valid progress with approximate size",
			input: "[download]  10.0% of ~500MiB at 1.0MiB/s ETA 00:07:30",
			expected: YTDLPProgress{
				Progress:   10.0,
				Total:      524288000,
				Downloaded: 52428800,
				Speed:      1048576,
				ETA:        7*time.Minute + 30*time.Second,
				Found:      true,
			},
		},
		{
			name:  "progress with MB",
			input: "[download]  75.3% of 256.00MiB at 5.12MiB/s ETA 00:01:20",
			expected: YTDLPProgress{
				Progress:   75.3,
				Total:      268435456,
				Downloaded: 202103738,
				Speed:      5368709,
				ETA:        time.Minute + 20*time.Second,
				Found:      true,
			},
		},
		{
			name:  "no regex match - random output",
			input: "Some random output from yt-dlp",
			expected: YTDLPProgress{
				Found: false,
			},
		},
		{
			name:  "no regex match - empty string",
			input: "",
			expected: YTDLPProgress{
				Found: false,
			},
		},
		{
			name:  "malformed progress line",
			input: "[download] invalid format",
			expected: YTDLPProgress{
				Found: false,
			},
		},
		{
			name:  "100% complete",
			input: "[download]  100.0% of 1.00GiB at 10.0MiB/s ETA 00:00:00",
			expected: YTDLPProgress{
				Progress:   100.0,
				Total:      1073741824,
				Downloaded: 1073741824,
				Speed:      10485760,
				ETA:        0,
				Found:      true,
			},
		},
		{
			name:  "small file with KB",
			input: "[download]  50.0% of 2048.00KiB at 512.00KiB/s ETA 00:00:04",
			expected: YTDLPProgress{
				Progress:   50.0,
				Total:      2097152,
				Downloaded: 1048576,
				Speed:      524288,
				ETA:        4 * time.Second,
				Found:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseYTDLPLine(tt.input)

			if result.Found != tt.expected.Found {
				t.Errorf("Found = %v, want %v", result.Found, tt.expected.Found)
			}

			if !result.Found {
				return
			}

			if abs(result.Progress-tt.expected.Progress) > 0.01 {
				t.Errorf("Progress = %v, want %v", result.Progress, tt.expected.Progress)
			}

			if !withinPercent(result.Total, tt.expected.Total, 1) {
				t.Errorf("Total = %v, want %v", result.Total, tt.expected.Total)
			}

			if !withinPercent(result.Downloaded, tt.expected.Downloaded, 1) {
				t.Errorf("Downloaded = %v, want %v", result.Downloaded, tt.expected.Downloaded)
			}

			if !withinPercent(result.Speed, tt.expected.Speed, 1) {
				t.Errorf("Speed = %v, want %v", result.Speed, tt.expected.Speed)
			}

			if result.ETA != tt.expected.ETA {
				t.Errorf("ETA = %v, want %v", result.ETA, tt.expected.ETA)
			}
		})
	}
}

func TestParseYTDLPDuration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  time.Duration
		expectErr bool
	}{
		{
			name:     "HH:MM:SS format",
			input:    "01:23:45",
			expected: time.Hour + 23*time.Minute + 45*time.Second,
		},
		{
			name:     "MM:SS format",
			input:    "07:30",
			expected: 7*time.Minute + 30*time.Second,
		},
		{
			name:     "SS format",
			input:    "45",
			expected: 45 * time.Second,
		},
		{
			name:     "zero seconds",
			input:    "00:00:00",
			expected: 0,
		},
		{
			name:     "single digit seconds",
			input:    "5",
			expected: 5 * time.Second,
		},
		{
			name:     "leading zeros",
			input:    "00:05:03",
			expected: 5*time.Minute + 3*time.Second,
		},
		{
			name:      "invalid - too many parts",
			input:     "01:02:03:04",
			expectErr: true,
		},
		{
			name:      "invalid - not a number",
			input:     "abc",
			expectErr: true,
		},
		{
			name:      "invalid - mixed content",
			input:     "01:ab:30",
			expectErr: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
		{
			name:     "large duration",
			input:    "99:59:59",
			expected: 99*time.Hour + 59*time.Minute + 59*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseYTDLPDuration(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ParseYTDLPDuration(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func withinPercent(a, b int64, percent float64) bool {
	if b == 0 {
		return a == 0
	}
	diff := float64(abs(float64(a - b)))
	return (diff/float64(b))*100 <= percent
}
