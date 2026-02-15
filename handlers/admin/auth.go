package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"zee-mirror/internal/domain"
	"zee-mirror/internal/service"
	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	UnknownSize  = "unknown"
	UnlimitedStr = "Unlimited"
)

func HandleAuth(s *service.BotService, message *tgbotapi.Message, args string) {
	if message.From.ID != s.Config.OwnerID {
		s.Reply(message, service.GetErrorMessage("ACCESS DENIED", "Hanya Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, username := parseUserArgs(message, args)
	if targetID == 0 {
		s.Reply(message, service.GetErrorMessage("INVALID FORMAT", "Gunakan: /authorize ID [username] atau reply ke user\\."))
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
		s.Reply(message, service.GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menyimpan user: %v", err)))
		return
	}

	quota := UnlimitedStr
	if s.Config.DefaultMaxDailyTasks != -1 || s.Config.DefaultMaxDailyBandwidth != -1 {
		tasks := UnlimitedStr
		if s.Config.DefaultMaxDailyTasks != -1 {
			tasks = fmt.Sprintf("%d", s.Config.DefaultMaxDailyTasks)
		}
		bw := UnlimitedStr
		if s.Config.DefaultMaxDailyBandwidth != -1 {
			bw = utils.FormatBytes(s.Config.DefaultMaxDailyBandwidth)
		}
		quota = fmt.Sprintf("Tasks: %s | BW: %s", tasks, bw)
	}

	content := fmt.Sprintf("👤 *User:* %s\n🏷️ *ID:* `%d`\n🔰 *Role:* `Authorized`\n♾️ *Kuota:* `%s`",
		utils.EscapeMarkdownV2(username), targetID, utils.EscapeMarkdownV2(quota))
	s.Reply(message, service.GetSuccessMessage("ACCESS GRANTED", content))
}

func HandleUnauth(s *service.BotService, message *tgbotapi.Message, args string) {
	if message.From.ID != s.Config.OwnerID {
		s.Reply(message, service.GetErrorMessage("ACCESS DENIED", "Hanya Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	if args == "" {
		s.Reply(message, "⚠️ *Format Salah*\n\nGunakan: `/unauth <user_id>`")
		return
	}

	userID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		s.Reply(message, "⚠️ *ID User Tidak Valid*")
		return
	}

	ctx := context.Background()
	if err := s.DB.SetRole(ctx, userID, "user"); err != nil {
		s.Reply(message, fmt.Sprintf("❌ *Gagal menghapus user:* %s", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	s.Reply(message, fmt.Sprintf("✅ *User %d berhasil dihapus dari daftar authorized*", userID))
}

func HandleUserList(s *service.BotService, message *tgbotapi.Message) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	ctx := context.Background()
	usersCount, _ := s.DB.GetCount(ctx)
	users, err := s.DB.GetAll(ctx)
	if err != nil {
		s.Reply(message, service.GetErrorMessage("DATABASE ERROR", "Gagal mengambil daftar user\\."))
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

		content.WriteString(fmt.Sprintf("%s `%d` \\| %s \\(*%s*\\) \\[%s\\]\n",
			status, u.ID, utils.EscapeMarkdownV2(u.Username), strings.ToUpper(role), utils.EscapeMarkdownV2(limits)))
	}

	s.Reply(message, service.ProfessionalMessage("DAFTAR PENGGUNA", content.String()))
}

func RemoveUserHandler(s *service.BotService, message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.Reply(message, service.GetErrorMessage("ACCESS DENIED", "Hanya Admin/Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, _ := parseUserArgs(message, args)
	if targetID == 0 {
		s.Reply(message, service.GetErrorMessage("INVALID FORMAT", "Gunakan: /removeuser ID atau reply ke user\\."))
		return
	}

	if s.IsOwner(targetID) {
		s.Reply(message, service.GetErrorMessage("ERROR", "Tidak bisa menghapus Owner\\."))
		return
	}

	ctx := context.Background()
	err := s.DB.Delete(ctx, targetID)
	if err != nil {
		s.Reply(message, service.GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menghapus user: %v", err)))
		return
	}

	s.Reply(message, service.GetSuccessMessage("USER REMOVED", fmt.Sprintf("Pengguna `%d` telah dihapus dari database\\.", targetID)))
}

func SetRoleHandler(s *service.BotService, message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.Reply(message, service.GetErrorMessage("ACCESS DENIED", "Hanya Admin/Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, _ := parseUserArgs(message, args)
	if targetID == 0 {
		s.Reply(message, service.GetErrorMessage("INVALID FORMAT", "Gunakan: /setrole ID <role> atau reply ke user dengan <role>\\."))
		return
	}

	if s.IsOwner(targetID) {
		s.Reply(message, service.GetErrorMessage("ERROR", "Tidak bisa merubah role Owner\\."))
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
		s.Reply(message, service.GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal merubah role: %v", err)))
		return
	}

	s.Reply(message, service.GetSuccessMessage("ROLE UPDATED", fmt.Sprintf("Role pengguna `%d` telah diubah menjadi `%s`\\.", targetID, strings.ToUpper(role))))
}

func SetLimitHandler(s *service.BotService, message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.Reply(message, service.GetErrorMessage("ACCESS DENIED", "Hanya Admin/Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, _ := parseUserArgs(message, args)
	if targetID == 0 {
		s.Reply(message, service.GetErrorMessage("INVALID FORMAT", "Gunakan: /setlimit ID <tasks> <bandwidth> atau reply dengan <tasks> <bandwidth>\\.\nGunakan \\-1 untuk unlimited\\."))
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
		s.Reply(message, service.GetErrorMessage("INVALID FORMAT", "Berikan jumlah task dan bandwidth (misal: 10 50GB)\\."))
		return
	}

	maxTasks, err := strconv.Atoi(remainingArgs[0])
	if err != nil {
		s.Reply(message, service.GetErrorMessage("INVALID VALUE", "Jumlah task harus angka\\."))
		return
	}

	maxBandwidth := utils.ParseBytesString(remainingArgs[1])
	if maxBandwidth == 0 && remainingArgs[1] != "0" && remainingArgs[1] != "-1" {
		s.Reply(message, service.GetErrorMessage("INVALID VALUE", "Format bandwidth tidak valid (misal: 10GB, 100MB, -1)\\."))
		return
	}

	if remainingArgs[1] == "-1" {
		maxBandwidth = -1
	}

	ctx := context.Background()
	err = s.DB.SetLimits(ctx, targetID, maxTasks, maxBandwidth)
	if err != nil {
		s.Reply(message, service.GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menyimpan limit: %v", err)))
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
	s.Reply(message, service.GetSuccessMessage("LIMITS UPDATED", content))
}

func SetExpireHandler(s *service.BotService, message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.Reply(message, service.GetErrorMessage("ACCESS DENIED", "Hanya Admin/Owner yang bisa menggunakan perintah ini\\."))
		return
	}

	targetID, _ := parseUserArgs(message, args)
	if targetID == 0 {
		s.Reply(message, service.GetErrorMessage("INVALID FORMAT", "Gunakan: /setexpire ID <days> atau reply dengan <days>\\."))
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
		s.Reply(message, service.GetErrorMessage("INVALID FORMAT", "Berikan jumlah hari masa aktif\\."))
		return
	}

	days, err := strconv.Atoi(remainingArgs[0])
	if err != nil {
		s.Reply(message, service.GetErrorMessage("INVALID VALUE", "Jumlah hari harus angka\\."))
		return
	}

	expiresAt := time.Now().AddDate(0, 0, days)
	ctx := context.Background()
	err = s.DB.SetExpiration(ctx, targetID, expiresAt)
	if err != nil {
		s.Reply(message, service.GetErrorMessage("DATABASE ERROR", fmt.Sprintf("Gagal menyimpan masa aktif: %v", err)))
		return
	}

	s.Reply(message, service.GetSuccessMessage("EXPIRATION UPDATED", fmt.Sprintf("Masa aktif pengguna `%d` diperbarui hingga *%s* \\(%d hari\\)\\.",
		targetID, expiresAt.Format("02 Jan 2006"), days)))
}

func parseUserArgs(message *tgbotapi.Message, args string) (int64, string) {
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
