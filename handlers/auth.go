package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	RoleAdmin      = "admin"
	RoleAuthorized = "authorized"
	RoleOwner      = "owner"
)

func (s *BotService) IsAuthorized(userID int64) bool {
	if userID == s.Config.OwnerID {
		return true
	}

	_, role, err := s.DB.GetUser(userID)
	if err != nil {
		return false
	}

	return role == RoleAdmin || role == RoleAuthorized || role == RoleOwner
}

func (s *BotService) IsAdmin(userID int64) bool {
	if userID == s.Config.OwnerID {
		return true
	}

	_, role, err := s.DB.GetUser(userID)
	if err != nil {
		return false
	}

	return role == RoleAdmin || role == RoleOwner
}

func (s *BotService) HandleAuthorize(message *tgbotapi.Message, args string) {
	if message.From.ID != s.Config.OwnerID {
		s.reply(message, GetErrorMessage("ACCESS DENIED", "Hanya Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, username := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Gunakan: /authorize ID atau reply ke user\\."))
		return
	}

	err := s.DB.UpsertUser(targetID, username, "authorized")
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menyimpan user: %v", err)))
		return
	}

	content := fmt.Sprintf("👤 *User:* %s\n🆔 *ID:* `%d`\n🔰 *Role:* `Authorized`",
		utils.EscapeMarkdownV2(username), targetID)
	s.reply(message, GetSuccessMessage("ACCESS GRANTED", content))
}

func (s *BotService) HandleUnauthorize(message *tgbotapi.Message, args string) {
	if message.From.ID != s.Config.OwnerID {
		s.reply(message, GetErrorMessage("ACCESS DENIED", "Hanya Owner yang bisa menggunakan perintah ini."))
		return
	}

	targetID, _ := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Gunakan: /unauthorize ID atau reply ke user\\."))
		return
	}

	err := s.DB.SetUserRole(targetID, "user")
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal merubah role user: %v", err)))
		return
	}

	s.reply(message, GetSuccessMessage("ACCESS REVOKED", fmt.Sprintf("User `%d` telah dikembalikan ke status user biasa\\.", targetID)))
}

func (s *BotService) HandleUsers(message *tgbotapi.Message) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	users, err := s.DB.GetAllUsers()
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", "Gagal mengambil daftar user\\."))
		return
	}

	var content strings.Builder
	for _, u := range users {
		id := u["id"].(int64)
		username := u["username"].(string)
		role := u["role"].(string)
		content.WriteString(fmt.Sprintf("• `%d` \\| %s \\(*%s*\\)\n",
			id, utils.EscapeMarkdownV2(username), strings.ToUpper(role)))
	}

	s.reply(message, ProfessionalMessage("DAFTAR PENGGUNA", content.String()))
}

func (s *BotService) parseUserArgs(message *tgbotapi.Message, args string) (int64, string) {
	if message.ReplyToMessage != nil {
		username := message.ReplyToMessage.From.UserName
		if username == "" {
			username = message.ReplyToMessage.From.FirstName
		}
		return message.ReplyToMessage.From.ID, username
	}

	if args != "" {
		parts := strings.Fields(args)
		if len(parts) > 0 {
			var id int64
			_, err := fmt.Sscanf(parts[0], "%d", &id)
			if err != nil {
				return 0, ""
			}
			username := "Unknown"
			if len(parts) > 1 {
				username = parts[1]
			}
			return id, username
		}
	}

	return 0, ""
}

func (s *BotService) reply(message *tgbotapi.Message, text string) {
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err := s.Bot.Send(msg)
	if err != nil {
		log.Printf("[Reply] Failed to send message: %v", err)
	}
}

func (s *BotService) startDiskCleanupWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.TaskManager.ShutdownChan:
			return
		case <-ticker.C:
			s.performDiskCleanup()
		}
	}
}

func (s *BotService) performDiskCleanup() {
	cutoff := time.Now().Add(-24 * time.Hour)

	entries, err := os.ReadDir(s.Config.DownloadDir)
	if err != nil {
		log.Printf("[Cleanup] Error reading download dir: %v", err)
		return
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(s.Config.DownloadDir, entry.Name())

			if s.TaskManager.GetTask(entry.Name()) != nil {
				continue
			}

			log.Printf("[Cleanup] Removing old entry: %s", entry.Name())
			_ = os.RemoveAll(path)
		}
	}

	if usage := s.getDiskUsage(); usage > 90 {
		log.Printf("[Cleanup] Disk usage critical: %.2f%%. Performing emergency cleanup.", usage)
	}
}

func (s *BotService) getDiskUsage() float64 {
	return s.getDiskUsageOS()
}
