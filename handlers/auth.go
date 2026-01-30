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

func (s *BotService) IsAuthorized(userID int64) bool {
	if userID == s.Config.OwnerID {
		return true
	}

	_, role, err := s.DB.GetUser(userID)
	if err != nil {
		return false
	}

	return role == "admin" || role == "authorized" || role == "owner"
}

func (s *BotService) IsAdmin(userID int64) bool {
	if userID == s.Config.OwnerID {
		return true
	}

	_, role, err := s.DB.GetUser(userID)
	if err != nil {
		return false
	}

	return role == "admin" || role == "owner"
}

func (s *BotService) HandleAuthorize(message *tgbotapi.Message, args string) {
	if message.From.ID != s.Config.OwnerID {
		s.reply(message, "❌ *Akses Ditolak*\nHanya Owner yang bisa menggunakan perintah ini.")
		return
	}

	targetID, username := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, "⚠️ *Format Salah*\nGunakan: `/authorize ID` atau reply ke user.")
		return
	}

	err := s.DB.UpsertUser(targetID, username, "authorized")
	if err != nil {
		s.reply(message, fmt.Sprintf("❌ *Gagal:* %v", err))
		return
	}

	text := fmt.Sprintf("✅ *Akses Diberikan*\n\n"+
		"👤 *User:* %s\n"+
		"🆔 *ID:* `%d`\n"+
		"🔰 *Role:* `Authorized`",
		utils.EscapeMarkdownV2(username), targetID)
	s.reply(message, text)
}

func (s *BotService) HandleUnauthorize(message *tgbotapi.Message, args string) {
	if message.From.ID != s.Config.OwnerID {
		s.reply(message, "❌ *Akses Ditolak*\nHanya Owner yang bisa menggunakan perintah ini.")
		return
	}

	targetID, _ := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, "⚠️ *Format Salah*\nGunakan: `/unauthorize ID` atau reply ke user.")
		return
	}

	err := s.DB.SetUserRole(targetID, "user")
	if err != nil {
		s.reply(message, fmt.Sprintf("❌ *Gagal:* %v", err))
		return
	}

	s.reply(message, fmt.Sprintf("✅ *Akses Dicabut*\nUser `%d` telah dikembalikan ke status user biasa.", targetID))
}

func (s *BotService) HandleUsers(message *tgbotapi.Message) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	users, err := s.DB.GetAllUsers()
	if err != nil {
		s.reply(message, "❌ *Gagal mengambil daftar user.*")
		return
	}

	var text strings.Builder
	text.WriteString("👥 *Daftar Pengguna Zee\\-Mirror*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	for _, u := range users {
		id := u["id"].(int64)
		username := u["username"].(string)
		role := u["role"].(string)
		text.WriteString(fmt.Sprintf("• `%d` \\| %s \\(*%s*\\)\n",
			id, utils.EscapeMarkdownV2(username), strings.ToUpper(role)))
	}
	text.WriteString("━━━━━━━━━━━━━━━━━━━━")

	s.reply(message, text.String())
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
	_, _ = s.Bot.Send(msg)
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
