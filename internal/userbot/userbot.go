package userbot

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"zee-mirror/internal/config"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
)

type UserBot struct {
	Client  *telegram.Client
	Config  *config.Config
	Manager *peers.Manager
	Sender  *message.Sender
	Context context.Context
	Cancel  context.CancelFunc
	Mu      sync.RWMutex
	Started bool
}

var (
	instance *UserBot
	once     sync.Once
)

func GetInstance(cfg *config.Config) *UserBot {
	once.Do(func() {
		instance = &UserBot{
			Config: cfg,
		}
	})
	return instance
}

func (u *UserBot) Start() error {
	u.Mu.Lock()
	defer u.Mu.Unlock()

	if u.Started {
		return nil
	}

	if u.Config.AppID == 0 || u.Config.AppHash == "" || u.Config.UserSessionString == "" {
		slog.Warn("🚫 Userbot credentials missing (APP_ID, APP_HASH, or USER_SESSION_STRING). Userbot disabled.")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	u.Context = ctx
	u.Cancel = cancel

	storage := &session.StorageMemory{}
	if u.Config.UserSessionString != "" {
		data, decodeErr := base64.StdEncoding.DecodeString(u.Config.UserSessionString)
		if decodeErr == nil {
			if storeErr := storage.StoreSession(context.Background(), data); storeErr != nil {
				slog.Warn("Failed to store session to memory", "error", storeErr)
			}
		} else {
			slog.Warn("Invalid base64 session string", "error", decodeErr)
		}
	}

	client := telegram.NewClient(u.Config.AppID, u.Config.AppHash, telegram.Options{
		SessionStorage: storage,
		Logger:         nil,
	})

	u.Client = client

	go func() {
		if err := client.Run(ctx, func(ctx context.Context) error {
			status, authErr := client.Auth().Status(ctx)
			if authErr != nil {
				return authErr
			}
			if !status.Authorized {
				slog.Error("❌ Userbot session is invalid or expired. Please generate a new session string using gotd.")
				return fmt.Errorf("userbot unauthorized")
			}

			u.Manager = peers.Options{}.Build(client.API())

			slog.Info("🔄 Userbot syncing dialogs...")

			if _, err := client.API().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
				Limit:      100,
				OffsetPeer: &tg.InputPeerEmpty{},
			}); err != nil {
				slog.Warn("Failed to sync dialogs", "error", err)
			}

			self, _ := client.Self(ctx)
			slog.Info("✅ Userbot Connected", "user", self.Username, "id", self.ID)

			u.Mu.Lock()
			u.Started = true
			u.Mu.Unlock()

			<-ctx.Done()
			return ctx.Err()
		}); err != nil {
			slog.Error("Userbot stopped", "error", err)
			u.Mu.Lock()
			u.Started = false
			u.Mu.Unlock()
		}
	}()

	return nil
}

func (u *UserBot) Stop() {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	if u.Started && u.Cancel != nil {
		slog.Info("Stopping Userbot session gracefully...")
		u.Cancel()
		u.Started = false
	}
}

func (u *UserBot) DownloadFile(link string, outputDir string, onProgress func(downloaded, total int64)) (string, error) {
	u.Mu.RLock()
	if !u.Started {
		u.Mu.RUnlock()
		return "", fmt.Errorf("userbot not started")
	}
	u.Mu.RUnlock()

	isPrivate := strings.Contains(link, "/c/")
	parts := strings.Split(link, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid link format")
	}

	msgIDStr := parts[len(parts)-1]
	msgID, err := strconv.Atoi(msgIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid message ID")
	}

	var identifier string
	if len(parts) >= 2 {
		identifier = parts[len(parts)-2]
	}

	ctx, cancel := context.WithTimeout(u.Context, 10*time.Minute)
	defer cancel()

	var inputPeer tg.InputPeerClass
	var peer peers.Peer

	if isPrivate {
		p, resolveErr := u.Manager.Resolve(ctx, identifier)
		if resolveErr != nil {
			return "", fmt.Errorf("could not resolve private channel: %v (make sure userbot has joined)", resolveErr)
		}
		peer = p
	} else {
		p, resolveErr := u.Manager.Resolve(ctx, "@"+identifier)
		if resolveErr != nil {
			return "", fmt.Errorf("resolve username failed: %v", resolveErr)
		}
		peer = p
	}

	inputPeer = peer.InputPeer()

	msgs, err := u.Client.API().MessagesGetMessages(ctx, []tg.InputMessageClass{
		&tg.InputMessageID{ID: msgID},
	})
	if err != nil {
		if channel, ok := inputPeer.(*tg.InputPeerChannel); ok {
			channelsMsgs, err2 := u.Client.API().ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
				Channel: &tg.InputChannel{ChannelID: channel.ChannelID, AccessHash: channel.AccessHash},
				ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: msgID}},
			})
			if err2 == nil {
				msgs = channelsMsgs
			} else {
				return "", err2
			}
		} else {
			return "", err
		}
	}

	var media tg.MessageMediaClass
	switch m := msgs.(type) {
	case *tg.MessagesMessages:
		if len(m.Messages) > 0 {
			if msg, ok := m.Messages[0].(*tg.Message); ok {
				media = msg.Media
			}
		}
	case *tg.MessagesChannelMessages:
		if len(m.Messages) > 0 {
			if msg, ok := m.Messages[0].(*tg.Message); ok {
				media = msg.Media
			}
		}
	case *tg.MessagesMessagesSlice:
		if len(m.Messages) > 0 {
			if msg, ok := m.Messages[0].(*tg.Message); ok {
				media = msg.Media
			}
		}
	}

	if media == nil {
		return "", fmt.Errorf("no media found in message")
	}

	dl := downloader.NewDownloader()

	outPath := filepath.Join(outputDir, fmt.Sprintf("userbot_dl_%d", msgID))
	var totalSize int64

	if doc, ok := media.(*tg.MessageMediaDocument); ok {
		if d, ok := doc.Document.(*tg.Document); ok {
			totalSize = d.Size
			for _, attr := range d.Attributes {
				if fn, ok := attr.(*tg.DocumentAttributeFilename); ok {
					outPath = filepath.Join(outputDir, fn.FileName)
					break
				}
			}
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	location, err := getFileLocation(media)
	if err != nil {
		return "", err
	}

	dlCtx := context.Background()

	pw := &ProgressWriter{
		w:          f,
		onProgress: onProgress,
		total:      totalSize,
	}

	if _, err := dl.Download(u.Client.API(), location).Stream(dlCtx, pw); err != nil {
		return "", fmt.Errorf("download failed: %v", err)
	}

	return outPath, nil
}

type ProgressWriter struct {
	w          io.Writer
	onProgress func(downloaded, total int64)
	total      int64
	current    int64
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	if n > 0 {
		pw.current += int64(n)
		if pw.onProgress != nil {
			pw.onProgress(pw.current, pw.total)
		}
	}
	return n, err
}

func getFileLocation(media tg.MessageMediaClass) (tg.InputFileLocationClass, error) {
	switch m := media.(type) {
	case *tg.MessageMediaDocument:
		if doc, ok := m.Document.(*tg.Document); ok {
			return &tg.InputDocumentFileLocation{
				ID:            doc.ID,
				AccessHash:    doc.AccessHash,
				FileReference: doc.FileReference,
				ThumbSize:     "",
			}, nil
		}
	case *tg.MessageMediaPhoto:
		if photo, ok := m.Photo.(*tg.Photo); ok {
			return &tg.InputPhotoFileLocation{
				ID:            photo.ID,
				AccessHash:    photo.AccessHash,
				FileReference: photo.FileReference,
				ThumbSize:     "y",
			}, nil
		}
	}
	return nil, fmt.Errorf("media type not supported for download")
}

func (u *UserBot) JoinChat(hash string) (string, error) {
	u.Mu.RLock()
	if !u.Started {
		u.Mu.RUnlock()
		return "", fmt.Errorf("userbot not started")
	}
	u.Mu.RUnlock()

	ctx, cancel := context.WithTimeout(u.Context, 30*time.Second)
	defer cancel()

	res, err := u.Client.API().MessagesImportChatInvite(ctx, hash)
	if err != nil {
		if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			return "Already joined", nil
		}
		return "", fmt.Errorf("failed to join chat: %w", err)
	}

	var title string
	if result, ok := res.(*tg.MessagesChatInviteJoinResultOk); ok {
		if u, ok := result.Updates.(*tg.Updates); ok {
			for _, chat := range u.Chats {
				switch c := chat.(type) {
				case *tg.Chat:
					title = c.Title
				case *tg.Channel:
					title = c.Title
				}
			}
		}
	}

	if title == "" {
		title = "Unknown Chat"
	}

	return fmt.Sprintf("Successfully joined: %s", title), nil
}
