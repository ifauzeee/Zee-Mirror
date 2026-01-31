package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken               string
	TelegramAPI            string
	OwnerID                int64
	AuthorizedUsers        []int64
	RcloneDest             string
	MaxConcurrentDownloads int
	DownloadDir            string
	ConfigDir              string
	DashboardToken         string
	DashboardPort          int
	LogLevel               string
}

func LoadConfig() *Config {
	cfg := &Config{
		BotToken:               os.Getenv("BOT_TOKEN"),
		TelegramAPI:            os.Getenv("TELEGRAM_API"),
		RcloneDest:             os.Getenv("RCLONE_DEST"),
		DownloadDir:            getEnv("DOWNLOAD_DIR", "/app/downloads"),
		ConfigDir:              getEnv("CONFIG_DIR", "/app/config"),
		MaxConcurrentDownloads: getEnvInt("MAX_CONCURRENT_DOWNLOADS", 3),
		DashboardToken:         getEnv("WEB_DASHBOARD_TOKEN", "zee-mirror-secret"),
		DashboardPort:          getEnvInt("DASHBOARD_PORT_INTERNAL", 8080),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
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

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	str := getEnv(key, "")
	if n, err := strconv.Atoi(str); err == nil {
		return n
	}
	return fallback
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
