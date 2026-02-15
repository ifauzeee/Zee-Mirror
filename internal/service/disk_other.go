//go:build !linux

package service

func (s *BotService) getDiskUsageOS() float64 {
	return 0
}
