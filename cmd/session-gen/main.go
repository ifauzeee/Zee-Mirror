package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/joho/godotenv"
	"golang.org/x/term"
)

type CapturingStorage struct {
	SessionData []byte
}

func (s *CapturingStorage) LoadSession(_ context.Context) ([]byte, error) {
	if s.SessionData == nil {
		return nil, session.ErrNotFound
	}
	return s.SessionData, nil
}

func (s *CapturingStorage) StoreSession(_ context.Context, data []byte) error {
	s.SessionData = data
	return nil
}

func prompt(msg string) string {
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func promptDefault(msg, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", msg, def)
	} else {
		fmt.Print(msg)
	}
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func promptPassword(msg string) string {
	fmt.Print(msg)
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return ""
	}
	fmt.Println()
	return string(bytePassword)
}

type TermAuth struct {
	PhoneValue string
}

func (a TermAuth) Phone(_ context.Context) (string, error) {
	if a.PhoneValue != "" {
		return a.PhoneValue, nil
	}
	return prompt("Enter Phone Number (with country code, e.g. +62...): "), nil
}

func (a TermAuth) Password(_ context.Context) (string, error) {
	return promptPassword("Enter 2FA Password: "), nil
}

func (a TermAuth) AcceptTermsOfService(_ context.Context, _ tg.HelpTermsOfService) error {
	return nil
}

func (a TermAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	return prompt("Enter Code: "), nil
}

func (a TermAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up is not supported")
}

func main() {
	_ = godotenv.Load()

	fmt.Println("\n🚀 Zee-Mirror Session Generator")
	fmt.Println("===============================")

	envAppID := os.Getenv("APP_ID")
	targetAppIDKey := "APP_ID"
	if envAppID == "" {
		envAppID = os.Getenv("TELEGRAM_API_ID")
		if envAppID != "" {
			targetAppIDKey = "TELEGRAM_API_ID"
		}
	}

	appIDStr := promptDefault("Enter API ID", envAppID)
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		fmt.Println("❌ Invalid API ID (must be integer)")
		return
	}

	envAppHash := os.Getenv("APP_HASH")
	targetAppHashKey := "APP_HASH"
	if envAppHash == "" {
		envAppHash = os.Getenv("TELEGRAM_API_HASH")
		if envAppHash != "" {
			targetAppHashKey = "TELEGRAM_API_HASH"
		}
	}

	appHash := promptDefault("Enter API HASH", envAppHash)
	if appHash == "" {
		fmt.Println("❌ API Hash cannot be empty")
		return
	}

	storage := &CapturingStorage{}

	client := telegram.NewClient(appID, appHash, telegram.Options{
		SessionStorage: storage,
		Logger:         nil,
	})

	flow := auth.NewFlow(TermAuth{}, auth.SendCodeOptions{})

	ctx := context.Background()
	fmt.Println("\n🔄 Connecting to Telegram...")

	if err := client.Run(ctx, func(ctx context.Context) error {
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}

		status, err := client.Auth().Status(ctx)
		if err != nil {
			return err
		}
		if !status.Authorized {
			return fmt.Errorf("login failed: not authorized")
		}

		user, err := client.Self(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("\n✅ Login Successful as: %s (@%s)\n", user.FirstName, user.Username)
		return nil
	}); err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		return
	}

	if storage.SessionData == nil {
		fmt.Println("❌ Failed to capture session data.")
		return
	}

	sessionString := base64.StdEncoding.EncodeToString(storage.SessionData)

	fmt.Println("\n🎉 SESSION STRING GENERATED SUCCESSFULLY!")
	fmt.Println("===========================================")

	envPath := ".env"
	updates := map[string]string{
		targetAppIDKey:        strconv.Itoa(appID),
		targetAppHashKey:      appHash,
		"USER_SESSION_STRING": sessionString,
	}

	if err := updateEnvFile(envPath, updates); err != nil {
		fmt.Printf("\n⚠️  Failed to update .env file automatically: %v\n", err)
		fmt.Println("Please copy the following line manually to your .env file:")
		fmt.Printf("\nUSER_SESSION_STRING=%s\n", sessionString)
	} else {
		fmt.Println("\n✅ Successfully updated .env file!")
		fmt.Println("Please restart your bot to apply changes.")
	}

	fmt.Println("\n===========================================")
	fmt.Println("⚠️  Keep this session string SAFE. It gives full access to your Telegram account.")
}

func updateEnvFile(path string, updates map[string]string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			f, createErr := os.Create(path)
			if createErr != nil {
				return createErr
			}
			defer f.Close()
			for k, v := range updates {
				if _, writeErr := f.WriteString(fmt.Sprintf("%s=%s\n", k, v)); writeErr != nil {
					return writeErr
				}
			}
			return nil
		}
		return err
	}

	lines := strings.Split(string(content), "\n")
	newLines := make([]string, 0, len(lines)+len(updates))
	updatedKeys := make(map[string]bool)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			if val, ok := updates[key]; ok {
				newLines = append(newLines, fmt.Sprintf("%s=%s", key, val))
				updatedKeys[key] = true
				continue
			}
		}
		newLines = append(newLines, line)
	}

	for k, v := range updates {
		if !updatedKeys[k] {
			newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
		}
	}

	output := strings.Join(newLines, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return os.WriteFile(path, []byte(output), 0600)
}
