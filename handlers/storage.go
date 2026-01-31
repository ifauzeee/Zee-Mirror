package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StorageProvider struct {
	Name       string
	RemoteName string
	Type       string
	Icon       string
}

func (s *BotService) GetAvailableStorages() ([]StorageProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configPath := s.TaskManager.ConfigDir + "/rclone.conf"
	cmd := exec.CommandContext(ctx, "rclone", "listremotes", "--config", configPath, "--long")
	output, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %v", err)
	}

	var providers []StorageProvider
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := strings.TrimSuffix(parts[0], ":")
			remoteType := parts[1]

			provider := StorageProvider{
				Name:       name,
				RemoteName: parts[0],
				Type:       remoteType,
				Icon:       getStorageIcon(remoteType),
			}
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

func getStorageIcon(storageType string) string {
	switch strings.ToLower(storageType) {
	case "drive":
		return "📁"
	case "onedrive":
		return "☁️"
	case "dropbox":
		return "📦"
	case "mega":
		return "🔷"
	case "s3", "b2":
		return "🪣"
	case "ftp", "sftp":
		return "🖥️"
	case "webdav":
		return "🌐"
	default:
		return "💾"
	}
}

func (s *BotService) HandleStorages(message *tgbotapi.Message) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	providers, err := s.GetAvailableStorages()
	if err != nil {
		s.reply(message, fmt.Sprintf("❌ *Gagal mendapatkan daftar storage*\n\n%s", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	if len(providers) == 0 {
		s.reply(message, "📭 *Tidak ada storage yang dikonfigurasi*\n\nTambahkan remote di rclone\\.conf")
		return
	}

	var text strings.Builder
	text.WriteString("💾 *Available Storage Providers*\n")
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range providers {
		text.WriteString(fmt.Sprintf("%s *%s* \\(%s\\)\n", p.Icon, utils.EscapeMarkdownV2(p.Name), utils.EscapeMarkdownV2(p.Type)))

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %s", p.Icon, p.Name),
				fmt.Sprintf("storage:select:%s", p.Name),
			),
		))
	}

	text.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━\n")
	text.WriteString("*Current:* `" + utils.EscapeMarkdownV2(s.TaskManager.RcloneDest) + "`\n\n")
	text.WriteString("Pilih storage untuk upload\\:")

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "storage:close:none"),
	))
	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	msg := tgbotapi.NewMessage(message.Chat.ID, text.String())
	msg.ParseMode = MarkdownV2
	msg.ReplyMarkup = keyboard
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) HandleStorageCallback(callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	action := parts[1]
	storageName := parts[2]

	switch action {
	case "select":
		s.handleStorageSelect(callback, storageName)
	case "browse":
		s.handleStorageBrowse(callback, storageName, "")
	case "info":
		s.handleStorageInfo(callback, storageName)
	case CmdClose:
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		_, _ = s.Bot.Request(deleteMsg)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "Closed"))
	}
}

func (s *BotService) handleStorageSelect(callback *tgbotapi.CallbackQuery, storageName string) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📂 Browse", fmt.Sprintf("storage:browse:%s", storageName)),
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Info", fmt.Sprintf("storage:info:%s", storageName)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Set as Default", fmt.Sprintf("storage:setdefault:%s", storageName)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "storage:list"),
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "storage:close:none"),
		),
	)

	text := fmt.Sprintf("💾 *Storage: %s*\n\n━━━━━━━━━━━━━━━━━━━━━━━━\n\nPilih aksi:", utils.EscapeMarkdownV2(storageName))

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) handleStorageBrowse(callback *tgbotapi.CallbackQuery, storageName, path string) {
	remotePath := storageName + ":"
	if path != "" {
		remotePath = storageName + ":" + path
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configPath := s.TaskManager.ConfigDir + "/rclone.conf"
	cmd := exec.CommandContext(ctx, "rclone", "lsjson", remotePath, "--config", configPath)
	output, err := cmd.Output()

	if err != nil {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Failed to browse"))
		return
	}

	var files []DriveFile
	_ = json.Unmarshal(output, &files)

	var text strings.Builder
	text.WriteString(fmt.Sprintf("📂 *%s*\n", utils.EscapeMarkdownV2(remotePath)))
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	for i, f := range files {
		if i >= 15 {
			text.WriteString(fmt.Sprintf("\n_\\.\\.\\. dan %d item lainnya_", len(files)-15))
			break
		}
		icon := "📄"
		if f.IsDir {
			icon = "📁"
		}
		text.WriteString(fmt.Sprintf("%s `%s`\n", icon, utils.EscapeMarkdownV2(utils.TruncateString(f.Name, 40))))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back", fmt.Sprintf("storage:select:%s", storageName)),
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "storage:close:none"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text.String())
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) handleStorageInfo(callback *tgbotapi.CallbackQuery, storageName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configPath := s.TaskManager.ConfigDir + "/rclone.conf"
	cmd := exec.CommandContext(ctx, "rclone", "about", storageName+":", "--config", configPath, "--json")
	output, err := cmd.Output()

	var text strings.Builder
	text.WriteString(fmt.Sprintf("ℹ️ *Storage Info: %s*\n", utils.EscapeMarkdownV2(storageName)))
	text.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if err != nil {
		text.WriteString("_Info tidak tersedia_\n")
	} else {
		var info map[string]interface{}
		if jsonErr := json.Unmarshal(output, &info); jsonErr == nil {
			if total, ok := info["total"].(float64); ok {
				text.WriteString(fmt.Sprintf("📊 *Total:* `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(int64(total)))))
			}
			if used, ok := info["used"].(float64); ok {
				text.WriteString(fmt.Sprintf("💾 *Used:* `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(int64(used)))))
			}
			if free, ok := info["free"].(float64); ok {
				text.WriteString(fmt.Sprintf("✅ *Free:* `%s`\n", utils.EscapeMarkdownV2(utils.FormatBytes(int64(free)))))
			}
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back", fmt.Sprintf("storage:select:%s", storageName)),
			tgbotapi.NewInlineKeyboardButtonData("✖️ Close", "storage:close:none"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text.String())
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = &keyboard
	_, _ = s.Bot.Send(editMsg)
}

func (s *BotService) HandleSetStorage(message *tgbotapi.Message, args string) {
	if !s.IsAdmin(message.From.ID) {
		s.reply(message, "❌ *Akses Ditolak*")
		return
	}

	if args == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/setstorage remote:/path`\n\nContoh: `/setstorage gdrive:/MyMirror`")
		return
	}

	remoteName := strings.Split(args, ":")[0]
	providers, err := s.GetAvailableStorages()
	if err != nil {
		s.reply(message, "❌ *Gagal validasi storage*")
		return
	}

	found := false
	for _, p := range providers {
		if p.Name == remoteName {
			found = true
			break
		}
	}

	if !found {
		s.reply(message, fmt.Sprintf("❌ *Remote tidak ditemukan:* `%s`", utils.EscapeMarkdownV2(remoteName)))
		return
	}

	s.TaskManager.RcloneDest = args
	_ = s.DB.SetSetting("rclone_dest", args)

	s.reply(message, fmt.Sprintf("✅ *Storage Updated*\n\n📂 Destination: `%s`", utils.EscapeMarkdownV2(args)))
}
