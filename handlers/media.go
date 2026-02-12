package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type FFProbeOutput struct {
	Streams []FFStream `json:"streams"`
	Format  FFFormat   `json:"format"`
}

type FFStream struct {
	Tags               map[string]string `json:"tags,omitempty"`
	CodecName          string            `json:"codec_name"`
	CodecType          string            `json:"codec_type"`
	DisplayAspectRatio string            `json:"display_aspect_ratio,omitempty"`
	PixFmt             string            `json:"pix_fmt,omitempty"`
	RFrameRate         string            `json:"r_frame_rate,omitempty"`
	BitRate            string            `json:"bit_rate,omitempty"`
	ChannelLayout      string            `json:"channel_layout,omitempty"`
	SampleRate         string            `json:"sample_rate,omitempty"`
	Index              int               `json:"index"`
	Width              int               `json:"width,omitempty"`
	Height             int               `json:"height,omitempty"`
	Channels           int               `json:"channels,omitempty"`
}

type FFFormat struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
	NbStreams  int    `json:"nb_streams"`
}

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
		slog.Error("Failed to send status message", "error", err)
		s.reply(message, "❌ *Gagal mengirim pesan status*")
		return
	}

	if fileID != "" {
		var downloadErr error
		inputPath, downloadErr = s.downloadTelegramFile(fileID, fileName)
		if downloadErr != nil {
			slog.Error("Telegram download failed during audio extraction", "error", downloadErr)
			s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal download file*\nError: %s", utils.EscapeMarkdownV2(downloadErr.Error())))
			return
		}
	}

	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".mp3"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if hasAudio, errAudio := s.HasAudioStream(inputPath); errAudio != nil || !hasAudio {
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
			utils.EscapeMarkdownV2Code(utils.GetLastLines(string(output), 15)))))
		return
	}

	content := fmt.Sprintf("🎵 Output: `%s`\\!\n\nKlik tombol di bawah untuk upload ke Drive\\.", utils.EscapeMarkdownV2Code(filepath.Base(outputPath)))
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
		slog.Error("Failed to send status message", "error", err)
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
			utils.EscapeMarkdownV2Code(utils.GetLastLines(string(output), 15))))
		return
	}

	content := fmt.Sprintf("🎬 Output: `%s`\\!\n📊 Quality: %s\n\nKlik tombol di bawah untuk upload ke Drive\\.",
		utils.EscapeMarkdownV2Code(filepath.Base(outputPath)),
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
		slog.Error("Failed to send status message", "error", err)
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
			utils.EscapeMarkdownV2Code(utils.GetLastLines(string(output), 15))))
		return
	}

	photo := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FilePath(outputPath))
	photo.Caption = fmt.Sprintf("🖼️ Thumbnail from `%s` at %s", utils.EscapeMarkdownV2Code(filepath.Base(inputPath)), utils.EscapeMarkdownV2(timestamp))
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
		slog.Error("Failed to send status message", "error", err)
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
			utils.EscapeMarkdownV2Code(utils.GetLastLines(string(output), 15))))
		return
	}

	content := fmt.Sprintf("🎬 Output: `%s`\\!\n\nKlik tombol di bawah untuk upload ke Drive\\.",
		utils.EscapeMarkdownV2Code(filepath.Base(outputPath)))
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
		slog.Error("Failed to send status message", "error", err)
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
			utils.EscapeMarkdownV2Code(utils.GetLastLines(string(output), 15))))
		return
	}

	content := fmt.Sprintf("📄 Output: `%s`\\!\n\nKlik tombol di bawah untuk upload ke Drive\\.",
		utils.EscapeMarkdownV2Code(filepath.Base(outputPath)))
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

	var probe FFProbeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		slog.Error("FFProbe JSON unmarshal failed", "error", err)
		s.reply(message, "❌ *Gagal menguraikan metadata file*")
		return
	}

	text := s.formatMediaInfo(filepath.Base(inputPath), probe)
	s.reply(message, text)
}

func (s *BotService) formatMediaInfo(filename string, info FFProbeOutput) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("📄 *File:* `%s`\n", utils.EscapeMarkdownV2Code(filename)))
	size, _ := strconv.ParseInt(info.Format.Size, 10, 64)
	content.WriteString(fmt.Sprintf("⚖️ *Size:* `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(size))))

	durationSec, _ := strconv.ParseFloat(info.Format.Duration, 64)
	duration := time.Duration(durationSec * float64(time.Second))
	content.WriteString(fmt.Sprintf("🕒 *Duration:* `%s`\n", utils.EscapeMarkdownV2Code(utils.FormatDuration(duration))))

	bitrate, _ := strconv.ParseInt(info.Format.BitRate, 10, 64)
	content.WriteString(fmt.Sprintf("📊 *Overall Bitrate:* `%s/s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(bitrate/8))))
	content.WriteString(fmt.Sprintf("📦 *Format:* `%s`\n", utils.EscapeMarkdownV2Code(info.Format.FormatName)))

	videoCount := 0
	audioCount := 0
	for _, st := range info.Streams {
		switch st.CodecType {
		case "video":
			videoCount++
			content.WriteString(fmt.Sprintf("\n🎬 *VIDEO STREAM \\#%d*\n", videoCount))
			content.WriteString(fmt.Sprintf(" • *Codec:* `%s`\n", utils.EscapeMarkdownV2Code(st.CodecName)))
			content.WriteString(fmt.Sprintf(" • *Resolution:* `%dx%d`\n", st.Width, st.Height))
			if st.DisplayAspectRatio != "" {
				content.WriteString(fmt.Sprintf(" • *Aspect Ratio:* `%s`\n", utils.EscapeMarkdownV2Code(st.DisplayAspectRatio)))
			}
			content.WriteString(fmt.Sprintf(" • *Pixel Format:* `%s`\n", utils.EscapeMarkdownV2Code(st.PixFmt)))
			fpsParts := strings.Split(st.RFrameRate, "/")
			if len(fpsParts) == 2 {
				f1, _ := strconv.ParseFloat(fpsParts[0], 64)
				f2, _ := strconv.ParseFloat(fpsParts[1], 64)
				if f2 > 0 {
					content.WriteString(fmt.Sprintf(" • *Frame Rate:* `%.2f fps`\n", f1/f2))
				}
			}
		case "audio":
			audioCount++
			content.WriteString(fmt.Sprintf("\n🎵 *AUDIO STREAM \\#%d*\n", audioCount))
			content.WriteString(fmt.Sprintf(" • *Codec:* `%s`\n", utils.EscapeMarkdownV2Code(st.CodecName)))
			content.WriteString(fmt.Sprintf(" • *Channels:* `%d` \\(%s\\)\n", st.Channels, utils.EscapeMarkdownV2(st.ChannelLayout)))
			sampleRate, _ := strconv.ParseInt(st.SampleRate, 10, 64)
			content.WriteString(fmt.Sprintf(" • *Sample Rate:* `%.1f kHz`\n", float64(sampleRate)/1000.0))
			if st.BitRate != "" {
				abr, _ := strconv.ParseInt(st.BitRate, 10, 64)
				content.WriteString(fmt.Sprintf(" • *Bitrate:* `%s/s`\n", utils.EscapeMarkdownV2Code(utils.FormatBytes(abr/8))))
			}
		}
	}

	content.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")

	return ProfessionalMessage("✨ MEDIA INFORMATION ✨", content.String())
}

func (s *BotService) editMessage(chatID int64, msgID int, text string) {
	editMsg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	editMsg.ParseMode = MarkdownV2
	_, err := s.Bot.Send(editMsg)
	if err != nil {
		slog.Warn("Failed to edit message", "error", err)
	}
}

func (s *BotService) editMessageWithMarkup(chatID int64, msgID int, text string, markup *tgbotapi.InlineKeyboardMarkup) {
	editMsg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	editMsg.ParseMode = MarkdownV2
	editMsg.ReplyMarkup = markup
	_, err := s.Bot.Send(editMsg)
	if err != nil {
		slog.Warn("Failed to edit message with markup", "error", err)
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

	replyID := 0
	if cb.Message.ReplyToMessage != nil {
		replyID = cb.Message.ReplyToMessage.MessageID
	}

	task, err := s.TaskManager.CreateTask(TypeMirror, url, fileName, cb.Message.Chat.ID, 0, replyID, cb.From.ID, false, false, "", "", 0)
	if err != nil {
		s.handleCreateTaskError(cb.Message.Chat.ID, cb.Message.MessageID, err)
		return
	}
	s.handleAutoDelete(task)
	s.UpdateSharedDashboard(cb.Message.Chat.ID, false)
	slog.Info("Local media mirror task created", "taskID", task.ID, "path", fullPath)
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
		slog.Error("Failed to send status message during screenshots", "error", err)
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
			slog.Warn("Failed to generate screenshot", "index", i, "error", err)
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
	file, err := s.GetFileWithFallback(fileID)
	if err != nil {
		return "", err
	}

	if fileName == "" {
		fileName = filepath.Base(file.FilePath)
	}

	if filepath.IsAbs(file.FilePath) {
		translatedPath := strings.Replace(file.FilePath, "/var/lib/telegram-bot-api", s.Config.DownloadDir, 1)
		if _, err := os.Stat(translatedPath); err == nil {
			slog.Info("Using local telegram file", "path", translatedPath)
			return translatedPath, nil
		}
	}

	inputPath := filepath.Join(s.Config.DownloadDir, fileName)
	downloadURL := s.getFileLink(file)
	slog.Debug("Telegram file download URL", "url", downloadURL)

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

func (s *BotService) HandleHardsub(message *tgbotapi.Message, args string) {
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
			s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/hardsub video.mp4 subtitle.srt` atau reply ke video dengan `/hardsub subtitle.srt`")
			return
		}
		videoPath = parts[0]
		subPath = parts[1]
	}

	if videoPath == "" || subPath == "" {
		s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/hardsub video.mp4 subtitle.srt` atau reply ke video dengan `/hardsub subtitle.srt`")
		return
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, "🔥 *Burning subtitle (Hard-sub)...*\n_Proses ini memakan waktu lebih lama karena harus re-encoding._")
	statusMsg.ParseMode = MarkdownV2
	sent, err := s.Bot.Send(statusMsg)
	if err != nil {
		slog.Error("Failed to send status message", "error", err)
		return
	}

	ext := filepath.Ext(videoPath)
	outputPath := strings.TrimSuffix(videoPath, ext) + "_hardsub" + ext

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	absSubPath, _ := filepath.Abs(subPath)
	ffmpegSubPath := strings.ReplaceAll(absSubPath, "\\", "/")
	ffmpegSubPath = strings.ReplaceAll(ffmpegSubPath, ":", "\\:")

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", videoPath,
		"-vf", fmt.Sprintf("subtitles='%s'", ffmpegSubPath),
		"-c:v", "libx264", "-crf", "23", "-preset", "medium",
		"-c:a", "copy",
		outputPath, "-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal burn subtitle*\n\nError: %s\n\nOutput (tail):\n`%s`",
			utils.EscapeMarkdownV2(err.Error()),
			utils.EscapeMarkdownV2Code(utils.GetLastLines(string(output), 15))))
		return
	}

	content := fmt.Sprintf("🎬 Output: `%s`\\!\n\nKlik tombol di bawah untuk upload ke Drive\\.",
		utils.EscapeMarkdownV2Code(filepath.Base(outputPath)))
	pathID := s.StorePath(outputPath)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Upload ke Drive", "media_m:"+pathID),
		),
	)
	s.editMessageWithMarkup(sent.Chat.ID, sent.MessageID, GetSuccessMessage("SUBTITLE BURNED", content), &keyboard)
}

func (s *BotService) HandleRescale(message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	fileID, fileName := s.getFileFromMessage(message)
	var inputPath, resolution string

	parts := strings.Fields(args)
	if fileID != "" {
		if len(parts) == 0 {
			s.reply(message, "⚠️ *Format Salah*\n\nReply ke video dengan `/rescale 1280x720` atau `/rescale 720p`")
			return
		}
		var err error
		inputPath, err = s.downloadTelegramFile(fileID, fileName)
		if err != nil {
			s.reply(message, "❌ *Gagal download file*")
			return
		}
		resolution = parts[0]
	} else {
		if len(parts) < 2 {
			s.reply(message, "⚠️ *Format Salah*\n\nGunakan: `/rescale video.mp4 1280x720` atau `/rescale video.mp4 720p`")
			return
		}
		inputPath = parts[0]
		resolution = parts[1]
	}

	scale := ""
	switch strings.ToLower(resolution) {
	case "4k":
		scale = "3840:2160"
	case "2k":
		scale = "2560:1440"
	case "1080p":
		scale = "1920:1080"
	case "720p":
		scale = "1280:720"
	case "480p":
		scale = "854:480"
	case "360p":
		scale = "640:360"
	default:
		switch {
		case strings.Contains(resolution, "x"):
			scale = strings.Replace(resolution, "x", ":", 1)
		case strings.Contains(resolution, ":"):
			scale = resolution
		default:
			s.reply(message, "❌ *Resolusi tidak valid*\n\nGunakan format: `1280x720` atau `720p`")
			return
		}
	}

	statusMsg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("📐 *Rescaling video to %s\\.\\.\\.*", resolution))
	statusMsg.ParseMode = MarkdownV2
	sent, err := s.Bot.Send(statusMsg)
	if err != nil {
		slog.Error("Failed to send status message", "error", err)
		return
	}

	ext := filepath.Ext(inputPath)
	outputPath := strings.TrimSuffix(inputPath, ext) + "_" + resolution + ext

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputPath,
		"-vf", fmt.Sprintf("scale=%s:force_original_aspect_ratio=decrease,pad=%s:(ow-iw)/2:(oh-ih)/2", scale, scale),
		"-c:v", "libx264", "-crf", "23", "-preset", "medium",
		"-c:a", "copy",
		outputPath, "-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.editMessage(sent.Chat.ID, sent.MessageID, fmt.Sprintf("❌ *Gagal rescale video*\n\nError: %s\n\nOutput (tail):\n`%s`",
			utils.EscapeMarkdownV2(err.Error()),
			utils.EscapeMarkdownV2Code(utils.GetLastLines(string(output), 15))))
		return
	}

	content := fmt.Sprintf("🎬 Output: `%s`\\!\n📐 Resolution: %s\n\nKlik tombol di bawah untuk upload ke Drive\\.",
		utils.EscapeMarkdownV2Code(filepath.Base(outputPath)),
		utils.EscapeMarkdownV2(resolution))
	pathID := s.StorePath(outputPath)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Upload ke Drive", "media_m:"+pathID),
		),
	)
	s.editMessageWithMarkup(sent.Chat.ID, sent.MessageID, GetSuccessMessage("VIDEO RESCALED", content), &keyboard)
}
