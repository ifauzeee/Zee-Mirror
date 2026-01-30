//go:build !linux

package handlers

func (s *BotService) getDiskUsageOS() float64 {
	return 0
}
