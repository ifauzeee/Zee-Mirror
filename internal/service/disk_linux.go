//go:build linux

package service

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
	if stat.Bsize < 0 {
		return 0
	}
	bsize := uint64(stat.Bsize)
	total := stat.Blocks * bsize
	free := stat.Bfree * bsize
	if total == 0 {
		return 0
	}
	used := total - free
	return float64(used) / float64(total) * 100
}
