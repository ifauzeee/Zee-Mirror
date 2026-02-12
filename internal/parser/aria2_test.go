package parser

import (
	"testing"
	"time"
)

func TestParseAria2Line(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Aria2Progress
	}{
		{
			name:  "valid progress with all fields",
			input: "[#1a2b3c 512.0MiB/1.0GiB(50%) CN:4 DL:2.5MiB ETA:3m25s]",
			expected: Aria2Progress{
				Downloaded:  536870912,
				Total:       1073741824,
				Progress:    50.0,
				Speed:       2621440,
				Connections: 4,
				ETA:         3*time.Minute + 25*time.Second,
				Found:       true,
			},
		},
		{
			name:  "valid progress without connections",
			input: "[#abc123 256MiB/512MiB(50%) DL:1.5MiB ETA:2m30s]",
			expected: Aria2Progress{
				Downloaded:  268435456,
				Total:       536870912,
				Progress:    50.0,
				Speed:       1572864,
				Connections: 0,
				ETA:         2*time.Minute + 30*time.Second,
				Found:       true,
			},
		},
		{
			name:  "valid progress without ETA",
			input: "[#xyz789 100MiB/200MiB(50%) CN:2 DL:5.0MiB]",
			expected: Aria2Progress{
				Downloaded:  104857600,
				Total:       209715200,
				Progress:    50.0,
				Speed:       5242880,
				Connections: 2,
				ETA:         0,
				Found:       true,
			},
		},
		{
			name:  "progress with GB download",
			input: "[#data01 2.5GiB/5.0GiB(50%) CN:8 DL:10.0MiB ETA:4m15s]",
			expected: Aria2Progress{
				Downloaded:  2684354560,
				Total:       5368709120,
				Progress:    50.0,
				Speed:       10485760,
				Connections: 8,
				ETA:         4*time.Minute + 15*time.Second,
				Found:       true,
			},
		},
		{
			name:  "progress with KB download",
			input: "[#small1 512KiB/1024KiB(50%) CN:1 DL:256KiB ETA:2s]",
			expected: Aria2Progress{
				Downloaded:  524288,
				Total:       1048576,
				Progress:    50.0,
				Speed:       262144,
				Connections: 1,
				ETA:         2 * time.Second,
				Found:       true,
			},
		},
		{
			name:  "99% complete",
			input: "[#final99 990MiB/1000MiB(99%) CN:4 DL:5.0MiB ETA:2s]",
			expected: Aria2Progress{
				Downloaded:  1038090240,
				Total:       1048576000,
				Progress:    99.0,
				Speed:       5242880,
				Connections: 4,
				ETA:         2 * time.Second,
				Found:       true,
			},
		},
		{
			name:  "no regex match - random output",
			input: "Some random aria2 output",
			expected: Aria2Progress{
				Found: false,
			},
		},
		{
			name:  "no regex match - empty string",
			input: "",
			expected: Aria2Progress{
				Found: false,
			},
		},
		{
			name:  "malformed progress line",
			input: "[#abc incomplete data",
			expected: Aria2Progress{
				Found: false,
			},
		},
		{
			name:  "starting download (0%)",
			input: "[#start00 0B/100MiB(0%) CN:1 DL:0B ETA:0s]",
			expected: Aria2Progress{
				Downloaded:  0,
				Total:       104857600,
				Progress:    0.0,
				Speed:       0,
				Connections: 1,
				ETA:         0,
				Found:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseAria2Line(tt.input)

			if result.Found != tt.expected.Found {
				t.Errorf("Found = %v, want %v", result.Found, tt.expected.Found)
			}

			if !result.Found {
				return
			}

			if !withinPercent(result.Downloaded, tt.expected.Downloaded, 1) {
				t.Errorf("Downloaded = %v, want %v", result.Downloaded, tt.expected.Downloaded)
			}

			if !withinPercent(result.Total, tt.expected.Total, 1) {
				t.Errorf("Total = %v, want %v", result.Total, tt.expected.Total)
			}

			if !withinPercent(result.Speed, tt.expected.Speed, 1) {
				t.Errorf("Speed = %v, want %v", result.Speed, tt.expected.Speed)
			}

			if abs(result.Progress-tt.expected.Progress) > 0.01 {
				t.Errorf("Progress = %v, want %v", result.Progress, tt.expected.Progress)
			}

			if result.Connections != tt.expected.Connections {
				t.Errorf("Connections = %v, want %v", result.Connections, tt.expected.Connections)
			}

			if result.ETA != tt.expected.ETA {
				t.Errorf("ETA = %v, want %v", result.ETA, tt.expected.ETA)
			}
		})
	}
}
