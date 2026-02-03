package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zee-mirror/internal/domain"
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

	ctx := context.Background()
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return false
	}

	if !user.IsActive {
		return false
	}

	return user.Role == RoleAdmin || user.Role == RoleAuthorized || user.Role == RoleOwner
}

func (s *BotService) IsOwner(userID int64) bool {
	return userID == s.Config.OwnerID
}

func (s *BotService) IsAdmin(userID int64) bool {
	if userID == s.Config.OwnerID {
		return true
	}

	ctx := context.Background()
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return false
	}

	if !user.IsActive {
		return false
	}

	return user.Role == RoleAdmin || user.Role == RoleOwner
}

func (s *BotService) CheckQuota(userID int64) error {
	if s.IsOwner(userID) {
		return nil
	}

	ctx := context.Background()
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if !user.IsActive {
		if user.ExpiresAt.Valid {
			return fmt.Errorf("akses anda telah berakhir pada %s", user.ExpiresAt.Time.Format("02 Jan 2006"))
		}
		return fmt.Errorf("akun anda tidak aktif")
	}

	if user.MaxDailyTasks == -1 && user.MaxDailyBandwidth == -1 {
		return nil
	}

	stats, err := s.DB.GetUserTodayStats(ctx, userID)
	if err != nil {
		return nil
	}

	if user.MaxDailyTasks != -1 && stats.TotalTasks >= user.MaxDailyTasks {
		return fmt.Errorf("kuota task harian habis (%d/%d)", stats.TotalTasks, user.MaxDailyTasks)
	}

	if user.MaxDailyBandwidth != -1 && stats.TotalBandwidth >= user.MaxDailyBandwidth {
		return fmt.Errorf("kuota bandwidth harian habis (%s/%s)", utils.FormatBytes(stats.TotalBandwidth), utils.FormatBytes(user.MaxDailyBandwidth))
	}

	return nil
}

func (s *BotService) HandleAuthorize(message *tgbotapi.Message, args string) {
	if message.From.ID != s.Config.OwnerID {
		s.reply(message, GetErrorMessage("ACCESS DENIED", "Hanya Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, username := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Gunakan: /authorize ID [username] atau reply ke user\\."))
		return
	}

	if username == "" || username == UnknownSize {
		username = "User"
	}

	ctx := context.Background()
	user := domain.User{
		ID:                targetID,
		Username:          username,
		Role:              "authorized",
		CreatedAt:         time.Now(),
		MaxDailyTasks:     s.Config.DefaultMaxDailyTasks,
		MaxDailyBandwidth: s.Config.DefaultMaxDailyBandwidth,
	}

	err := s.DB.Upsert(ctx, user)
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menyimpan user: %v", err)))
		return
	}

	quota := UnlimitedStr
	if s.Config.DefaultMaxDailyTasks != -1 || s.Config.DefaultMaxDailyBandwidth != -1 {
		tasks := "∞"
		if s.Config.DefaultMaxDailyTasks != -1 {
			tasks = fmt.Sprintf("%d", s.Config.DefaultMaxDailyTasks)
		}
		bw := "∞"
		if s.Config.DefaultMaxDailyBandwidth != -1 {
			bw = utils.FormatBytes(s.Config.DefaultMaxDailyBandwidth)
		}
		quota = fmt.Sprintf("Tasks: %s | BW: %s", tasks, bw)
	}

	content := fmt.Sprintf("👤 *User:* %s\n🏷️ *ID:* `%d`\n🔰 *Role:* `Authorized`\n♾️ *Kuota:* `%s`",
		utils.EscapeMarkdownV2(username), targetID, utils.EscapeMarkdownV2(quota))
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

	ctx := context.Background()

	err := s.DB.SetRole(ctx, targetID, "user")
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

	ctx := context.Background()
	usersCount, _ := s.DB.GetCount(ctx)
	users, err := s.DB.GetAll(ctx)
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", "Gagal mengambil daftar user\\."))
		return
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Total Pengguna: %d\n\n", usersCount))
	for _, u := range users {
		status := "✅"
		if !u.IsActive {
			status = "❌"
		}

		role := u.Role

		limits := ""
		if u.MaxDailyTasks != -1 {
			limits += fmt.Sprintf("T:%d ", u.MaxDailyTasks)
		}
		if u.MaxDailyBandwidth != -1 {
			limits += fmt.Sprintf("B:%s ", utils.FormatBytes(u.MaxDailyBandwidth))
		}
		if limits == "" {
			limits = "♾️"
		}

		content.WriteString(fmt.Sprintf("%s `%d` | %s (*%s*) [%s]\n",
			status, u.ID, utils.EscapeMarkdownV2(u.Username), strings.ToUpper(role), limits))
	}

	s.reply(message, ProfessionalMessage("DAFTAR PENGGUNA", content.String()))
}

func (s *BotService) HandleRemoveUser(message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.reply(message, GetErrorMessage("ACCESS DENIED", "Hanya Admin/Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, _ := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Gunakan: /removeuser ID atau reply ke user\\."))
		return
	}

	if s.IsOwner(targetID) {
		s.reply(message, GetErrorMessage("ERROR", "Tidak bisa menghapus Owner\\."))
		return
	}

	ctx := context.Background()
	err := s.DB.Delete(ctx, targetID)
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menghapus user: %v", err)))
		return
	}

	s.reply(message, GetSuccessMessage("USER REMOVED", fmt.Sprintf("Pengguna `%d` telah dihapus dari database\\.", targetID)))
}

func (s *BotService) HandleSetRole(message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.reply(message, GetErrorMessage("ACCESS DENIED", "Hanya Admin/Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, _ := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Gunakan: /setrole ID <role> atau reply ke user dengan <role>\\."))
		return
	}

	if s.IsOwner(targetID) {
		s.reply(message, GetErrorMessage("ERROR", "Tidak bisa merubah role Owner\\."))
		return
	}

	parts := strings.Fields(args)
	role := "authorized"
	if len(parts) > 1 {
		role = parts[1]
	} else if message.ReplyToMessage != nil && len(parts) > 0 {
		role = parts[0]
	}

	ctx := context.Background()
	err := s.DB.SetRole(ctx, targetID, role)
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal merubah role: %v", err)))
		return
	}

	s.reply(message, GetSuccessMessage("ROLE UPDATED", fmt.Sprintf("Role pengguna `%d` telah diubah menjadi `%s`\\.", targetID, strings.ToUpper(role))))
}

func (s *BotService) HandleSetLimit(message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.reply(message, GetErrorMessage("ACCESS DENIED", "Hanya Admin/Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, _ := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Gunakan: /setlimit ID <tasks> <bandwidth> atau reply dengan <tasks> <bandwidth>\\.\nGunakan \\-1 untuk unlimited\\."))
		return
	}

	parts := strings.Fields(args)
	remainingArgs := parts
	if len(parts) > 0 {
		var maybeID int64
		_, err := fmt.Sscanf(parts[0], "%d", &maybeID)
		if err == nil && maybeID == targetID {
			remainingArgs = parts[1:]
		}
	}

	if len(remainingArgs) < 2 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Berikan jumlah task dan bandwidth (misal: 10 50GB)\\."))
		return
	}

	maxTasks, err := strconv.Atoi(remainingArgs[0])
	if err != nil {
		s.reply(message, GetErrorMessage("INVALID VALUE", "Jumlah task harus angka\\."))
		return
	}

	maxBandwidth := utils.ParseBytesString(remainingArgs[1])
	if maxBandwidth == 0 && remainingArgs[1] != "0" && remainingArgs[1] != "-1" {
		s.reply(message, GetErrorMessage("INVALID VALUE", "Format bandwidth tidak valid (misal: 10GB, 100MB, -1)\\."))
		return
	}

	if remainingArgs[1] == "-1" {
		maxBandwidth = -1
	}

	ctx := context.Background()
	err = s.DB.SetLimits(ctx, targetID, maxTasks, maxBandwidth)
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menyimpan limit: %v", err)))
		return
	}

	bwStr := UnlimitedStr
	if maxBandwidth != -1 {
		bwStr = utils.FormatBytes(maxBandwidth)
	}
	taskStr := UnlimitedStr
	if maxTasks != -1 {
		taskStr = strconv.Itoa(maxTasks)
	}

	content := fmt.Sprintf("👤 *User ID:* `%d`\n📋 *Daily Tasks:* `%s`\n📊 *Daily Bandwidth:* `%s`",
		targetID, taskStr, bwStr)
	s.reply(message, GetSuccessMessage("LIMITS UPDATED", content))
}

func (s *BotService) HandleSetExpire(message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.reply(message, GetErrorMessage("ACCESS DENIED", "Hanya Admin/Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, _ := s.parseUserArgs(message, args)
	if targetID == 0 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Gunakan: /setexpire ID <days> atau reply dengan <days>\\."))
		return
	}

	parts := strings.Fields(args)
	remainingArgs := parts
	if len(parts) > 0 {
		var maybeID int64
		_, err := fmt.Sscanf(parts[0], "%d", &maybeID)
		if err == nil && maybeID == targetID {
			remainingArgs = parts[1:]
		}
	}

	if len(remainingArgs) < 1 {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Berikan jumlah hari masa aktif\\."))
		return
	}

	days, err := strconv.Atoi(remainingArgs[0])
	if err != nil {
		s.reply(message, GetErrorMessage("INVALID VALUE", "Jumlah hari harus angka\\."))
		return
	}

	expiresAt := time.Now().AddDate(0, 0, days)
	ctx := context.Background()
	err = s.DB.SetExpiration(ctx, targetID, expiresAt)
	if err != nil {
		s.reply(message, GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menyimpan masa aktif: %v", err)))
		return
	}

	s.reply(message, GetSuccessMessage("EXPIRATION UPDATED", fmt.Sprintf("Masa aktif pengguna `%d` diperbarui hingga *%s* \\(%d hari\\)\\.",
		targetID, expiresAt.Format("02 Jan 2006"), days)))
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
			username := UnknownSize
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
	msg.ParseMode = MarkdownV2
	if _, err := s.Bot.Send(msg); err != nil {
		slog.Error("Failed to send message", "error", err, "userID", message.From.ID)
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
		slog.Error("Error reading download dir", "error", err, "path", s.Config.DownloadDir)
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

			slog.Info("Removing old entry", "name", entry.Name())
			_ = os.RemoveAll(path)
		}
	}

	if usage := s.getDiskUsage(); usage > 90 {
		slog.Warn("Disk usage critical", "usage", usage)
	}
}

func (s *BotService) getDiskUsage() float64 {
	return s.getDiskUsageOS()
}
