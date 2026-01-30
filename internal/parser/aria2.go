package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"zee-mirror/pkg/utils"
)

var aria2StatusRegex = regexp.MustCompile(`\[#\w+\s+(\S+)/(\S+)\((\d+)%\)(?:\s+CN:(\d+))?.*DL:(\S+)(?:\s+ETA:(\S+))?\]`)

type Aria2Progress struct {
	Downloaded  int64
	Total       int64
	Progress    float64
	Speed       int64
	Connections int
	ETA         time.Duration
	Found       bool
}

func ParseAria2Line(line string) Aria2Progress {
	matches := aria2StatusRegex.FindStringSubmatch(line)
	if len(matches) < 5 {
		return Aria2Progress{Found: false}
	}

	p := Aria2Progress{
		Found: true,
	}

	downloadedStr := matches[1]
	totalStr := matches[2]
	pctStr := matches[3]
	cnStr := matches[4]
	speedStr := matches[5]
	etaStr := ""
	if len(matches) >= 7 {
		etaStr = matches[6]
	}

	p.Downloaded = utils.ParseBytesString(downloadedStr)
	p.Total = utils.ParseBytesString(totalStr)
	p.Speed = utils.ParseBytesString(speedStr)

	if cn, err := strconv.Atoi(cnStr); err == nil {
		p.Connections = cn
	}
	if pct, err := strconv.ParseFloat(pctStr, 64); err == nil {
		p.Progress = pct
	}

	if etaStr != "" {
		etaStr = strings.TrimRight(etaStr, "]")
		if d, err := time.ParseDuration(etaStr); err == nil {
			p.ETA = d
		}
	}

	return p
}
