package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"zee-mirror/pkg/utils"
)

var ytdlpProgressRegex = regexp.MustCompile(`\[download\]\s+([\d\.]+)%\s+of\s+(?:~)?(\S+)\s+at\s+(\S+)\s+ETA\s+(\S+)`)

type YTDLPProgress struct {
	Progress   float64
	Total      int64
	Downloaded int64
	Speed      int64
	ETA        time.Duration
	Found      bool
}

func ParseYTDLPLine(line string) YTDLPProgress {
	matches := ytdlpProgressRegex.FindStringSubmatch(line)
	if len(matches) < 5 {
		return YTDLPProgress{Found: false}
	}

	p := YTDLPProgress{Found: true}

	pctStr := matches[1]
	totalStr := matches[2]
	speedStr := matches[3]
	etaStr := matches[4]

	if pct, err := strconv.ParseFloat(pctStr, 64); err == nil {
		p.Progress = pct
	}
	p.Total = utils.ParseBytesString(totalStr)
	p.Speed = utils.ParseBytesString(speedStr)

	if p.Total > 0 && p.Progress > 0 {
		p.Downloaded = int64(float64(p.Total) * p.Progress / 100)
	}

	if d, err := ParseYTDLPDuration(etaStr); err == nil {
		p.ETA = d
	}

	return p
}

func ParseYTDLPDuration(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	var h, m, sec int
	var err error

	switch len(parts) {
	case 3:
		h, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		m, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		sec, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, err
		}
	case 2:
		m, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		sec, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
	case 1:
		sec, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
	default:
		return 0, strconv.ErrSyntax
	}

	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second, nil
}
