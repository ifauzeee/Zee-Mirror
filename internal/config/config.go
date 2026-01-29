package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken               string
	OwnerID                int64
	AuthorizedUsers        []int64
	RcloneDest             string
	MaxConcurrentDownloads int
	DownloadDir            string
	ConfigDir              string
}

func LoadConfig() *Config {
	cfg := &Config{
		BotToken:               os.Getenv("BOT_TOKEN"),
		RcloneDest:             os.Getenv("RCLONE_DEST"),
		DownloadDir:            "/app/downloads",
		ConfigDir:              "/app/config",
		MaxConcurrentDownloads: 3,
	}

	if ownerIDStr := os.Getenv("OWNER_ID"); ownerIDStr != "" {
		if id, err := strconv.ParseInt(ownerIDStr, 10, 64); err == nil {
			cfg.OwnerID = id
		}
	}

	if authUsers := os.Getenv("AUTHORIZED_USERS"); authUsers != "" {
		for _, idStr := range strings.Split(authUsers, ",") {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				cfg.AuthorizedUsers = append(cfg.AuthorizedUsers, id)
			}
		}
	}

	if maxDL := os.Getenv("MAX_CONCURRENT_DOWNLOADS"); maxDL != "" {
		if n, err := strconv.Atoi(maxDL); err == nil && n > 0 {
			cfg.MaxConcurrentDownloads = n
		}
	}

	if dlDir := os.Getenv("DOWNLOAD_DIR"); dlDir != "" {
		cfg.DownloadDir = dlDir
	}
	if cfgDir := os.Getenv("CONFIG_DIR"); cfgDir != "" {
		cfg.ConfigDir = cfgDir
	}

	return cfg
}

func (c *Config) Validate() bool {
	if c.BotToken == "" {
		log.Fatal("❌ BOT_TOKEN tidak ditemukan! Set environment variable BOT_TOKEN")
		return false
	}
	if c.OwnerID == 0 {
		log.Fatal("❌ OWNER_ID tidak ditemukan! Set environment variable OWNER_ID")
		return false
	}
	if c.RcloneDest == "" {
		log.Println("⚠️ RCLONE_DEST tidak di-set, menggunakan default: gdrive:/MirrorBot")
		c.RcloneDest = "gdrive:/MirrorBot"
	}
	return true
}

func (c *Config) IsAuthorized(userID int64) bool {
	if userID == c.OwnerID {
		return true
	}
	for _, id := range c.AuthorizedUsers {
		if id == userID {
			return true
		}
	}
	return false
}
