package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"zee-mirror/pkg/utils"
)

type Config struct {
	IndexURL                 string
	VikingUserHash           string
	TelegramAPI              string
	RcloneDest               string
	DownloadDir              string
	ConfigDir                string
	DashboardToken           string
	LogLevel                 string
	BotToken                 string
	DashboardURL             string
	AppHash                  string
	UserSessionString        string
	AuthorizedUsers          []int64
	OwnerID                  int64
	DefaultMaxDailyBandwidth int64
	MaxConcurrentDownloads   int
	DashboardPort            int
	DefaultMaxDailyTasks     int
	AppID                    int
	SmartAutoOrganization    bool
	StopDuplicate            bool
	MaxRetries               int
}

func LoadConfig() *Config {
	cfg := &Config{
		BotToken:                 os.Getenv("BOT_TOKEN"),
		TelegramAPI:              os.Getenv("TELEGRAM_API"),
		RcloneDest:               os.Getenv("RCLONE_DEST"),
		DownloadDir:              getEnv("DOWNLOAD_DIR", "/app/downloads"),
		ConfigDir:                getEnv("CONFIG_DIR", "/app/config"),
		MaxConcurrentDownloads:   getEnvInt("MAX_CONCURRENT_DOWNLOADS", 3),
		DashboardToken:           getEnv("WEB_DASHBOARD_TOKEN", "zee-mirror-secret"),
		DashboardPort:            getEnvInt("DASHBOARD_PORT_INTERNAL", 8080),
		LogLevel:                 getEnv("LOG_LEVEL", "info"),
		SmartAutoOrganization:    getEnvBool("SMART_AUTO_ORGANIZATION", false),
		IndexURL:                 os.Getenv("INDEX_URL"),
		DashboardURL:             getEnv("WEB_DASHBOARD_URL", "127.0.0.1"),
		DefaultMaxDailyTasks:     getEnvInt("DEFAULT_MAX_DAILY_TASKS", -1),
		DefaultMaxDailyBandwidth: utils.ParseBytesString(getEnv("DEFAULT_MAX_DAILY_BANDWIDTH", "-1")),
		StopDuplicate:            getEnvBool("STOP_DUPLICATE", false),
		MaxRetries:               getEnvInt("MAX_RETRIES", 3),
		VikingUserHash:           os.Getenv("VIKING_USER_HASH"),
		AppID:                    getEnvInt("APP_ID", 0),
		AppHash:                  os.Getenv("APP_HASH"),
		UserSessionString:        os.Getenv("USER_SESSION_STRING"),
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

func getEnvBool(key string, fallback bool) bool {
	str := getEnv(key, "")
	if str == "" {
		return fallback
	}
	b, err := strconv.ParseBool(str)
	if err != nil {
		return fallback
	}
	return b
}

func (c *Config) Validate() bool {
	if c.BotToken == "" {
		log.Fatal("❌ BOT_TOKEN tidak ditemukan! Set environment variable BOT_TOKEN")
	}
	if c.OwnerID == 0 {
		log.Fatal("❌ OWNER_ID tidak ditemukan! Set environment variable OWNER_ID")
	}
	if c.RcloneDest == "" {
		log.Println("⚠️ RCLONE_DEST tidak di-set, menggunakan default: gdrive:/MirrorBot")
		c.RcloneDest = "gdrive:/MirrorBot"
	}
	return true
}
