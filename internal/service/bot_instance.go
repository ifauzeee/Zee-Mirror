package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotInstance struct {
	Bot   *tgbotapi.BotAPI
	Token string
	Alias string
}

func InitBots(tokens []string, telegramAPI string) ([]*BotInstance, error) {
	var instances []*BotInstance

	httpClient := &http.Client{
		Timeout: 90 * time.Second,
	}

	for i, token := range tokens {
		token := token
		alias := fmt.Sprintf("bot%d", i+1)

		var bot *tgbotapi.BotAPI
		var err error

		if telegramAPI != "" {
			bot, err = tgbotapi.NewBotAPIWithAPIEndpoint(token, telegramAPI)
		} else {
			bot, err = tgbotapi.NewBotAPI(token)
		}

		if err != nil {
			slog.Error("Failed to initialize bot", "index", i, "error", err)
			continue
		}

		bot.Client = httpClient
		bot.Debug = false

		instances = append(instances, &BotInstance{
			Bot:   bot,
			Token: maskToken(token),
			Alias: alias,
		})

		slog.Info("Bot initialized", "alias", alias, "username", bot.Self.UserName)
	}

	return instances, nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
