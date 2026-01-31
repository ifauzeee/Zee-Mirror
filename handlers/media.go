package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *BotService) HandleExtractAudio(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	fileID, fileName := s.getFileFromMessage(message)
	inputPath := args

	if fileID == "" && inputPath == "" {
		s.reply(message, GetErrorMessage("INVALID FORMAT", "Reply ke file video dengan /extractaudio atau gunakan:\n`/extractaudio path`"))
		return
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, "🎵 *Extracting audio\\.\\.\\.*")
	statusMsg.ParseMode = MarkdownV2
	sent, err := s.Bot.Send(statusMsg)
	if err != nil {
		log.Printf("[Send] Failed to send status message: %v", err)
		s.reply(message, "❌ *Gagal mengirim pesan status*")
		return
	}

	if fileID != "" {
		var downloadErr error
		inputPath, downloadErr = s.downloadTelegramFile(fileID, fileName)
		if downloadErr != nil {
			log.Printf("[Download] Failed: %v", downloadErr)
			s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal download file*\nError: %s", utils.EscapeMarkdownV2(downloadErr.Error())))
			return
		}
	}

	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".mp3"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if hasAudio, err := s.HasAudioStream(inputPath); err != nil || !hasAudio {
		s.editMessage(sent.Chat.ID, sent.MessageID, GetErrorMessage("NO AUDIO", "Video ini tidak memiliki track audio untuk di\\-extract\\."))
		return
	}

	inputDir := filepath.Dir(inputPath)
	inputName := filepath.Base(inputPath)
	outputName := filepath.Base(outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputName, "-vn", "-acodec", "libmp3lame", "-q:a", "2", outputName, "-y")
	cmd.Dir = inputDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, GetErrorMessage("FFMPEG ERROR", fmt.Sprintf("Gagal extract audio: %s\n\nOutput \\(tail\\):\n`%s`",
			utils.EscapeMarkdownV2(err.Error()),
			utils.EscapeMarkdownV2(utils.GetLastLines(string(output), 15)))))
		return
	}

	content := fmt.Sprintf("🎵 Output: `%s`\\!\n\nKlik tombol di bawah untuk upload ke Drive\\.", utils.EscapeMarkdownV2(filepath.Base(outputPath)))
	pathID := s.StorePath(outputPath)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Upload ke Drive", "media_m:"+pathID),
		),
	)
	s.editMessageWithMarkup(sent.Chat.ID, sent.MessageID, GetSuccessMessage("AUDIO EXTRACTED", content), &keyboard)
}

func (s *BotService) HandleCompressVideo(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	fileID, fileName := s.getFileFromMessage(message)
	inputPath := args
	quality := "medium"

	if parts := strings.Fields(args); len(parts) > 0 {
		if fileID == "" {
			inputPath = parts[0]
			if len(parts) > 1 {
				quality = parts[1]
			}
		} else {
			quality = parts[0]
		}
	}

	if fileID == "" && inputPath == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nReply ke file video dengan /compress atau gunakan:\n`/compress path/to/video [quality]`\n\nQuality: low, medium, high \\(default: medium\\)")
		return
	}

	crf := "28"
	switch quality {
	case "low":
		crf = "32"
	case "high":
		crf = "23"
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, "🗜️ *Compressing video\\.\\.\\.*")
	statusMsg.ParseMode = MarkdownV2
	sent, err := s.Bot.Send(statusMsg)
	if err != nil {
		log.Printf("[Send] Failed to send status message: %v", err)
		s.reply(message, "❌ *Gagal mengirim pesan status*")
		return
	}

	if fileID != "" {
		var downloadErr error
		inputPath, downloadErr = s.downloadTelegramFile(fileID, fileName)
		if downloadErr != nil {
			s.editMessage(sent.Chat.ID, sent.MessageID, "❌ *Gagal download file*")
			return
		}
	}

	ext := filepath.Ext(inputPath)
	outputPath := strings.TrimSuffix(inputPath, ext) + "_compressed" + ext

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	inputDir := filepath.Dir(inputPath)
	inputName := filepath.Base(inputPath)
	outputName := filepath.Base(outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputName,
		"-c:v", "libx264", "-crf", crf, "-preset", "medium",
		"-c:a", "aac", "-b:a", "128k",
		outputName, "-y")
	cmd.Dir = inputDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal compress video*\n\nError: %s\n\nOutput \\(tail\\):\n`%s`",
			utils.EscapeMarkdownV2(err.Error()),
			utils.EscapeMarkdownV2(utils.GetLastLines(string(output), 15))))
		return
	}

	content := fmt.Sprintf("🎬 Output: `%s`\\!\n📊 Quality: %s\n\nKlik tombol di bawah untuk upload ke Drive\\.",
		utils.EscapeMarkdownV2(filepath.Base(outputPath)),
		utils.EscapeMarkdownV2(quality))
	pathID := s.StorePath(outputPath)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Upload ke Drive", "media_m:"+pathID),
		),
	)
	s.editMessageWithMarkup(sent.Chat.ID, sent.MessageID, GetSuccessMessage("VIDEO COMPRESSED", content), &keyboard)
}

func (s *BotService) HandleGenerateThumbnail(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	fileID, fileName := s.getFileFromMessage(message)
	timestamp := "00:00:05"
	inputPath := ""

	if args != "" {
		parts := strings.Fields(args)
		if fileID != "" {
			timestamp = parts[0]
		} else {
			if len(parts) > 0 {
				inputPath = parts[0]
			}
			if len(parts) > 1 {
				timestamp = parts[1]
			}
		}
	}

	if fileID == "" && inputPath == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nReply ke file video dengan /thumbnail atau gunakan:\n`/thumbnail [timestamp]`\n\nContoh: `/thumbnail 00:01:30`")
		return
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, "🖼️ *Generating thumbnail\\.\\.\\.*")
	statusMsg.ParseMode = MarkdownV2
	sent, err := s.Bot.Send(statusMsg)
	if err != nil {
		log.Printf("[Send] Failed to send status message: %v", err)
		s.reply(message, "❌ *Gagal mengirim pesan status*")
		return
	}

	if fileID != "" {
		var downloadErr error
		inputPath, downloadErr = s.downloadTelegramFile(fileID, fileName)
		if downloadErr != nil {
			s.editMessage(sent.Chat.ID, sent.MessageID, "❌ *Gagal download file*")
			return
		}
	}

	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "_thumb.jpg"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	inputDir := filepath.Dir(inputPath)
	inputName := filepath.Base(inputPath)
	outputName := filepath.Base(outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputName,
		"-ss", timestamp, "-vframes", "1",
		"-q:v", "2", outputName, "-y")
	cmd.Dir = inputDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal generate thumbnail*\n\nError: %s\n\nOutput \\(tail\\):\n`%s`",
			utils.EscapeMarkdownV2(err.Error()),
			utils.EscapeMarkdownV2(utils.GetLastLines(string(output), 15))))
		return
	}

	photo := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FilePath(outputPath))
	photo.Caption = fmt.Sprintf("🖼️ Thumbnail from `%s` at %s", utils.EscapeMarkdownV2(filepath.Base(inputPath)), timestamp)
	photo.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(photo)

	_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(sent.Chat.ID, sent.MessageID))
}

func (s *BotService) HandleEmbedSubtitle(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	fileID, fileName := s.getFileFromMessage(message)
	var videoPath, subPath string

	parts := strings.Fields(args)
	if fileID != "" {
		var err error
		videoPath, err = s.downloadTelegramFile(fileID, fileName)
		if err != nil {
			s.reply(message, "❌ *Gagal download file*")
			return
		}
		if len(parts) > 0 {
			subPath = parts[0]
		}
	} else {
		if len(parts) < 2 {
			s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/subtitle video.mp4 subtitle.srt` atau reply ke video dengan `/subtitle subtitle.srt`")
			return
		}
		videoPath = parts[0]
		subPath = parts[1]
	}

	if videoPath == "" || subPath == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/subtitle video.mp4 subtitle.srt` atau reply ke video dengan `/subtitle subtitle.srt`")
		return
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, "📝 *Embedding subtitle\\.\\.\\.*")
	statusMsg.ParseMode = MarkdownV2
	sent, err := s.Bot.Send(statusMsg)
	if err != nil {
		log.Printf("[Send] Failed to send status message: %v", err)
		s.reply(message, "❌ *Gagal mengirim pesan status*")
		return
	}

	ext := filepath.Ext(videoPath)
	outputPath := strings.TrimSuffix(videoPath, ext) + "_subbed" + ext

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	inputDir := filepath.Dir(videoPath)
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", filepath.Base(videoPath),
		"-i", "file:"+subPath,
		"-c", "copy", "-c:s", "mov_text",
		filepath.Base(outputPath), "-y")
	cmd.Dir = inputDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal embed subtitle*\n\nError: %s\n\nOutput \\(tail\\):\n`%s`",
			utils.EscapeMarkdownV2(err.Error()),
			utils.EscapeMarkdownV2(utils.GetLastLines(string(output), 15))))
		return
	}

	content := fmt.Sprintf("🎬 Output: `%s`\\!\n\nKlik tombol di bawah untuk upload ke Drive\\.",
		utils.EscapeMarkdownV2(filepath.Base(outputPath)))
	pathID := s.StorePath(outputPath)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Upload ke Drive", "media_m:"+pathID),
		),
	)
	s.editMessageWithMarkup(sent.Chat.ID, sent.MessageID, GetSuccessMessage("SUBTITLE EMBEDDED", content), &keyboard)
}

func (s *BotService) HandleConvertFormat(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	fileID, fileName := s.getFileFromMessage(message)
	var inputPath, targetFormat string

	parts := strings.Fields(args)
	if fileID != "" {
		var err error
		inputPath, err = s.downloadTelegramFile(fileID, fileName)
		if err != nil {
			s.reply(message, "❌ *Gagal download file*")
			return
		}
		if len(parts) > 0 {
			targetFormat = strings.ToLower(parts[0])
		}
	} else {
		if len(parts) < 2 {
			s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/convert input.mp4 mkv` atau reply ke file with `/convert mkv`")
			return
		}
		inputPath = parts[0]
		targetFormat = strings.ToLower(parts[1])
	}

	validFormats := map[string]bool{
		"mp4": true, "mkv": true, "avi": true, "mov": true, "webm": true,
		"mp3": true, "aac": true, "flac": true, "wav": true,
	}

	if !validFormats[targetFormat] {
		s.reply(message, "❌ *Format tidak didukung*\n\nSupported: mp4, mkv, avi, mov, webm, mp3, aac, flac, wav")
		return
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("🔄 *Converting to %s\\.\\.\\.*", targetFormat))
	statusMsg.ParseMode = MarkdownV2
	sent, err := s.Bot.Send(statusMsg)
	if err != nil {
		log.Printf("[Send] Failed to send status message: %v", err)
		s.reply(message, "❌ *Gagal mengirim pesan status*")
		return
	}

	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "." + targetFormat

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	inputDir := filepath.Dir(inputPath)
	var cmd *exec.Cmd
	switch targetFormat {
	case "mp3", "aac", "flac", "wav":
		cmd = exec.CommandContext(ctx, "ffmpeg", "-i", filepath.Base(inputPath), "-vn", filepath.Base(outputPath), "-y")
	default:
		cmd = exec.CommandContext(ctx, "ffmpeg", "-i", filepath.Base(inputPath), "-c:v", "copy", "-c:a", "copy", filepath.Base(outputPath), "-y")
	}
	cmd.Dir = inputDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal convert format*\n\nError: %s\n\nOutput \\(tail\\):\n`%s`",
			utils.EscapeMarkdownV2(err.Error()),
			utils.EscapeMarkdownV2(utils.GetLastLines(string(output), 15))))
		return
	}

	content := fmt.Sprintf("📄 Output: `%s`\\!\n\nKlik tombol di bawah untuk upload ke Drive\\.",
		utils.EscapeMarkdownV2(filepath.Base(outputPath)))
	pathID := s.StorePath(outputPath)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Upload ke Drive", "media_m:"+pathID),
		),
	)
	s.editMessageWithMarkup(sent.Chat.ID, sent.MessageID, GetSuccessMessage("MEDIA CONVERTED", content), &keyboard)
}

func (s *BotService) HandleMediaInfo(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	fileID, fileName := s.getFileFromMessage(message)
	inputPath := args

	if fileID != "" {
		var err error
		inputPath, err = s.downloadTelegramFile(fileID, fileName)
		if err != nil {
			s.reply(message, "❌ *Gagal download file*")
			return
		}
	}

	if inputPath == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nReply ke file media atau gunakan:\n`/mediainfo path/to/file`")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", inputPath)
	output, err := cmd.Output()
	if err != nil {
		s.reply(message, fmt.Sprintf("❌ *Gagal mendapatkan info*\n\nError: %s", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	text := s.formatMediaInfo(filepath.Base(inputPath), string(output))
	s.reply(message, text)
}

func (s *BotService) formatMediaInfo(filename, jsonOutput string) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("📄 *File:* `%s`\n\n", utils.EscapeMarkdownV2(filename)))

	if strings.Contains(jsonOutput, "video") {
		content.WriteString("🎬 *Type:* `Video`\n")
	} else if strings.Contains(jsonOutput, "audio") {
		content.WriteString("🎵 *Type:* `Audio`\n")
	}

	content.WriteString("\n_Gunakan ffprobe untuk detail lengkap_")

	return ProfessionalMessage("MEDIA INFORMATION", content.String())
}

func (s *BotService) editMessage(chatID int64, msgID int, text string) {
	editMsg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	editMsg.ParseMode = MarkdownV2
	_, err := s.Bot.Send(editMsg)
	if err != nil {
		log.Printf("[EditMessage] Failed to edit message: %v", err)
	}
}

func (s *BotService) editMessageWithMarkup(chatID int64, msgID int, text string, markup *tgbotapi.InlineKeyboardMarkup) {
	editMsg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = markup
	_, err := s.Bot.Send(editMsg)
	if err != nil {
		log.Printf("[EditMessage] Failed to edit message with markup: %v", err)
	}
}

func (s *BotService) HandleMediaMirrorCallback(cb *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	pathID := parts[1]
	fullPath, ok := s.GetPath(pathID)
	if !ok {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(cb.ID, "❌ Link kadaluarsa atau tidak valid!"))
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(cb.ID, "❌ File sudah dihapus dari server!"))
		return
	}

	_, _ = s.Bot.Request(tgbotapi.NewCallback(cb.ID, "📤 Memulai mirror ke Drive..."))

	s.TaskManager.Mu.Lock()
	s.TaskManager.LastStatusMsg[cb.Message.Chat.ID] = cb.Message.MessageID
	s.TaskManager.Mu.Unlock()

	url := "file://" + fullPath
	fileName := filepath.Base(fullPath)

	task := s.TaskManager.CreateTask(TypeMirror, url, fileName, cb.Message.Chat.ID, 0, cb.From.ID, false, false, "", "")
	s.UpdateSharedDashboard(cb.Message.Chat.ID, false)
	log.Printf("[Mirror] Local media task created: %s for %s", task.ID, fullPath)
}

func downloadFile(url, destPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-L", "-o", destPath, url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("curl failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}

func (s *BotService) getFileLink(file tgbotapi.File) string {
	if s.Config.TelegramAPI != "" {
		apiFormat := s.Config.TelegramAPI
		apiFormat = strings.Replace(apiFormat, "bot%s/%s", "file/bot%s/%s", 1)

		filePath := strings.TrimPrefix(file.FilePath, "/")

		return fmt.Sprintf(apiFormat, s.Config.BotToken, filePath)
	}
	return file.Link(s.Config.BotToken)
}

func (s *BotService) HandleScreenshots(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	fileID, fileName := s.getFileFromMessage(message)
	count := 4
	inputPath := args

	parts := strings.Fields(args)
	if fileID != "" {
		var err error
		inputPath, err = s.downloadTelegramFile(fileID, fileName)
		if err != nil {
			s.reply(message, "❌ *Gagal download file*")
			return
		}
		if len(parts) > 0 {
			_, _ = fmt.Sscanf(parts[0], "%d", &count)
		}
	} else if len(parts) > 0 {
		inputPath = parts[0]
		if len(parts) > 1 {
			_, _ = fmt.Sscanf(parts[1], "%d", &count)
		}
	}

	if count > 10 {
		count = 10
	} else if count < 1 {
		count = 1
	}

	if inputPath == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/screenshots video.mp4 [count]` atau reply ke video dengan `/screenshots [count]`")
		return
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("📸 *Generating %d screenshots\\.\\.\\.*", count))
	statusMsg.ParseMode = MarkdownV2
	sent, err := s.Bot.Send(statusMsg)
	if err != nil {
		log.Printf("[Send] Failed to send status message: %v", err)
		s.reply(message, "❌ *Gagal mengirim pesan status*")
		return
	}

	screenshots, err := s.generateScreenshotsList(inputPath, count)
	if err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal generate screenshots*\n\nError: %s", utils.EscapeMarkdownV2(err.Error())))
		return
	}

	var mediaGroup []interface{}
	for _, ss := range screenshots {
		photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(ss))
		mediaGroup = append(mediaGroup, photo)
	}

	mediaMsg := tgbotapi.NewMediaGroup(message.Chat.ID, mediaGroup)
	_, _ = s.Bot.Send(mediaMsg)

	_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(sent.Chat.ID, sent.MessageID))
}

func (s *BotService) generateScreenshotsList(inputPath string, count int) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	durationCmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", "file:"+inputPath)
	durationOutput, err := durationCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get duration: %w", err)
	}

	var duration float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(durationOutput)), "%f", &duration); err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}

	if duration <= 0 {
		return nil, fmt.Errorf("invalid video duration")
	}

	interval := duration / float64(count+1)
	baseName := strings.TrimSuffix(inputPath, filepath.Ext(inputPath))

	var screenshots []string
	for i := 1; i <= count; i++ {
		timestamp := interval * float64(i)
		outputPath := fmt.Sprintf("%s_ss%d.jpg", baseName, i)

		shotCtx, shotCancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(shotCtx, "ffmpeg", "-ss", fmt.Sprintf("%.2f", timestamp),
			"-i", "file:"+inputPath, "-vframes", "1", "-q:v", "2", "file:"+outputPath, "-y")

		if err := cmd.Run(); err != nil {
			log.Printf("Failed to generate screenshot %d: %v", i, err)
			shotCancel()
			continue
		}
		shotCancel()
		screenshots = append(screenshots, outputPath)
	}

	if len(screenshots) == 0 {
		return nil, fmt.Errorf("no screenshots generated")
	}

	return screenshots, nil
}

func (s *BotService) getFileFromMessage(message *tgbotapi.Message) (fileID, fileName string) {
	if message.ReplyToMessage == nil {
		return "", ""
	}
	msg := message.ReplyToMessage
	switch {
	case msg.Video != nil:
		return msg.Video.FileID, msg.Video.FileName
	case msg.Audio != nil:
		return msg.Audio.FileID, msg.Audio.FileName
	case msg.Document != nil:
		return msg.Document.FileID, msg.Document.FileName
	case msg.Animation != nil:
		return msg.Animation.FileID, msg.Animation.FileName
	case msg.Voice != nil:
		return msg.Voice.FileID, ""
	case msg.VideoNote != nil:
		return msg.VideoNote.FileID, ""
	}
	return "", ""
}

func (s *BotService) downloadTelegramFile(fileID, fileName string) (string, error) {
	file, err := s.Bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", err
	}

	if fileName == "" {
		fileName = filepath.Base(file.FilePath)
	}

	if filepath.IsAbs(file.FilePath) {
		translatedPath := strings.Replace(file.FilePath, "/var/lib/telegram-bot-api", s.Config.DownloadDir, 1)
		if _, err := os.Stat(translatedPath); err == nil {
			log.Printf("[Download] Using local file: %s", translatedPath)
			return translatedPath, nil
		}
	}

	inputPath := filepath.Join(s.Config.DownloadDir, fileName)
	downloadURL := s.getFileLink(file)
	log.Printf("[Download] URL: %s", downloadURL)

	if err := downloadFile(downloadURL, inputPath); err != nil {
		return "", err
	}
	return inputPath, nil
}

func (s *BotService) HasAudioStream(inputPath string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inputDir := filepath.Dir(inputPath)
	inputName := filepath.Base(inputPath)

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", inputName)
	cmd.Dir = inputDir
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}
