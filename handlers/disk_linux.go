//go:build linux

package handlers

import (
	"os"
	"syscall"
)

func (s *BotService) getDiskUsageOS() float64 {
	var stat syscall.Statfs_t
	wd, _ := os.Getwd()
	if err := syscall.Statfs(wd, &stat); err != nil {
		return 0
	}
	bsize := stat.Bsize
	if bsize < 0 {
		bsize = 0
	}
	total := stat.Blocks * uint64(bsize)
	free := stat.Bfree * uint64(bsize)
	if total == 0 {
		return 0
	}
	used := total - free
	return float64(used) / float64(total) * 100
}
