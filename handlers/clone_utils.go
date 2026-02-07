package handlers

import (
	"fmt"
)

func constructScrapeURL(id string, isFolder bool, originalURL string) string {
	if id == "" {
		return originalURL
	}

	if isFolder {
		return fmt.Sprintf("https://drive.google.com/drive/folders/%s", id)
	}
	return fmt.Sprintf("https://drive.google.com/file/d/%s/view", id)
}
