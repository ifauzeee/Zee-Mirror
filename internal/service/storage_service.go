package service

import (
	"fmt"
	"os/exec"
	"strings"
)

type StorageProvider struct {
	Name string
	Type string
	Icon string
}

func (s *BotService) GetAvailableStorages() ([]StorageProvider, error) {
	cmd := exec.Command("rclone", "listremotes")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list rclone remotes: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var providers []StorageProvider
	for _, line := range lines {
		name := strings.TrimSuffix(line, ":")
		if name == "" {
			continue
		}

		providers = append(providers, StorageProvider{
			Name: name,
			Type: "rclone",
			Icon: "📁",
		})
	}

	return providers, nil
}
