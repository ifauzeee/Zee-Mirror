package search

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"zee-mirror/internal/service"
	"zee-mirror/pkg/utils"

	"github.com/PuerkitoBio/goquery"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type SearchResult struct {
	Title   string
	Size    string
	Seeders string
	Magnet  string
	Source  string
}

type SearchSession struct {
	CreatedAt time.Time
	Query     string
	Provider  string
	Results   []SearchResult
	Page      int
}

var (
	SearchSessions = make(map[string]*SearchSession)
	SearchMu       sync.RWMutex
)

func HandleSearch(s *service.BotService, message *tgbotapi.Message, args string) {
	if !s.IsAuthorized(message.From.ID) {
		return
	}

	if args == "" {
		s.Reply(message, "⚠️ *Format Salah*\n\nGunakan: `/search <query>`")
		return
	}

	query := args

	sessionID := uuid.New().String()

	SearchMu.Lock()
	SearchSessions[sessionID] = &SearchSession{
		CreatedAt: time.Now(),
		Query:     query,
		Results:   []SearchResult{},
		Page:      0,
	}
	SearchMu.Unlock()

	go func() {
		nyResults := scrapeNyaa(query)
		pbResults := searchPirateBay(query)
		slResults := searchSolidTorrents(query)

		var all []SearchResult
		all = append(all, nyResults...)
		all = append(all, pbResults...)
		all = append(all, slResults...)

		SearchMu.Lock()
		if sess, ok := SearchSessions[sessionID]; ok {
			sess.Results = all
		}
		SearchMu.Unlock()

		showSearchResults(s, message.Chat.ID, message.MessageID, sessionID)
	}()

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("🔍 *Mencari* `%s`\\.\\.\\.", utils.EscapeMarkdownV2(query)))
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	sent, _ := s.Bot.Send(msg)

	_ = sent
}

func HandleSearchCallback(s *service.BotService, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Query tidak valid"))
		return
	}

	provider := parts[1]
	query := strings.Join(parts[2:], ":")

	_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "🔍 Mencari..."))

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, fmt.Sprintf("🔍 Mencari `%s` • *%s*...", utils.EscapeMarkdownV2(query), provider))
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	_, _ = s.Bot.Send(editMsg)

	var results []SearchResult
	slog.Info("Torrent search initiated", "provider", provider, "query", query)
	switch provider {
	case "solid":
		results = searchSolidTorrents(query)
	case "nyaa":
		results = scrapeNyaa(query)
	case "apibay":
		results = searchPirateBay(query)
	}

	if len(results) == 0 {
		sessionID := uuid.New().String()[:8]
		session := &SearchSession{
			Query:     query,
			Provider:  provider,
			Results:   results,
			Page:      0,
			CreatedAt: time.Now(),
		}
		SearchMu.Lock()
		SearchSessions[sessionID] = session
		SearchMu.Unlock()

		text := fmt.Sprintf("📭 *Tidak ada hasil ditemukan di %s*\n\n_Cobalah menggunakan kata kunci lain atau provider yang berbeda\\._", provider)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", fmt.Sprintf("t_back:%s", sessionID)),
				tgbotapi.NewInlineKeyboardButtonData("❌ Tutup", fmt.Sprintf("t_close:%s", sessionID)),
			),
		)

		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdownV2
		editMsg.ReplyMarkup = &keyboard
		_, _ = s.Bot.Send(editMsg)
		return
	}

	sessionID := uuid.New().String()[:8]
	session := &SearchSession{
		Query:     query,
		Provider:  provider,
		Results:   results,
		Page:      0,
		CreatedAt: time.Now(),
	}

	SearchMu.Lock()
	SearchSessions[sessionID] = session
	SearchMu.Unlock()

	showSearchResults(s, callback.Message.Chat.ID, callback.Message.MessageID, sessionID)
}

func showSearchResults(s *service.BotService, chatID int64, messageID int, sessionID string) {
	SearchMu.RLock()
	session, exists := SearchSessions[sessionID]
	SearchMu.RUnlock()

	if !exists {
		s.SendOrEditMessage(chatID, messageID, "❌ Sesi pencarian telah berakhir. Silakan cari ulang.", nil)
		return
	}

	perPage := 5
	totalItems := len(session.Results)
	totalPages := (totalItems + perPage - 1) / perPage

	if session.Page >= totalPages {
		session.Page = 0
	}
	if session.Page < 0 {
		session.Page = totalPages - 1
	}

	start := session.Page * perPage
	end := start + perPage
	if end > totalItems {
		end = totalItems
	}

	visibleItems := session.Results[start:end]

	text := fmt.Sprintf("🔍 *Hasil Pencarian \\(%s\\)*\n`%s`\n\n", utils.EscapeMarkdownV2(session.Provider), utils.EscapeMarkdownV2Code(session.Query))
	text += fmt.Sprintf("📄 Halaman %d/%d\n\n", session.Page+1, totalPages)

	var rows [][]tgbotapi.InlineKeyboardButton
	var numberKeyRow []tgbotapi.InlineKeyboardButton

	for i, item := range visibleItems {
		idx := start + i + 1
		text += fmt.Sprintf("%d\\. *%s*\n📦 %s • 👤 %s\n\n",
			idx,
			utils.EscapeMarkdownV2(utils.TruncateString(item.Title, 40)),
			utils.EscapeMarkdownV2(item.Size),
			utils.EscapeMarkdownV2(item.Seeders),
		)

		numberKeyRow = append(numberKeyRow, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d", idx),
			fmt.Sprintf("t_item:%d:%s", start+i, sessionID),
		))
	}
	rows = append(rows, numberKeyRow)

	var navRow []tgbotapi.InlineKeyboardButton
	if totalPages > 1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Prev", fmt.Sprintf("t_page:%d:%s", session.Page-1, sessionID)))
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", session.Page+1, totalPages), "ignore"))
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("t_page:%d:%s", session.Page+1, sessionID)))
	}
	rows = append(rows, navRow)

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", fmt.Sprintf("t_back:%s", sessionID)),
		tgbotapi.NewInlineKeyboardButtonData("❌ Tutup", fmt.Sprintf("t_close:%s", sessionID)),
	))

	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2
	editMsg.ReplyMarkup = &keyboard

	if _, err := s.Bot.Send(editMsg); err != nil {
		slog.Warn("Search results markdown error, using fallback", "error", err)
		fallbackText := fmt.Sprintf("🔍 Hasil Pencarian (%s): %s\n\nHalaman %d/%d\n\n", session.Provider, session.Query, session.Page+1, totalPages)
		for i, item := range visibleItems {
			fallbackText += fmt.Sprintf("%d. %s\nSize: %s | Seeds: %s\n\n", start+i+1, item.Title, item.Size, item.Seeders)
		}
		fallbackEdit := tgbotapi.NewEditMessageText(chatID, messageID, fallbackText)
		fallbackEdit.ReplyMarkup = &keyboard
		_, _ = s.Bot.Send(fallbackEdit)
	}
}

func HandleSearchNavCallback(s *service.BotService, callback *tgbotapi.CallbackQuery, parts []string) {
	action := parts[0]

	if action == "t_back" {
		if len(parts) < 2 {
			return
		}
		sessionID := parts[1]

		SearchMu.RLock()
		session, exists := SearchSessions[sessionID]
		SearchMu.RUnlock()

		if !exists {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Sesi berakhir"))
			return
		}

		query := session.Query
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🌟 Solid (Umum)", fmt.Sprintf("t_search:solid:%s", query)),
				tgbotapi.NewInlineKeyboardButtonData("🏴‍☠️ PirateBay", fmt.Sprintf("t_search:apibay:%s", query)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🌸 Nyaa (Anime)", fmt.Sprintf("t_search:nyaa:%s", query)),
			),
		)

		textRaw := fmt.Sprintf("🔍 Pencarian: %s\n\nPilih provider pencarian yang lebih stabil:", query)
		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, textRaw)
		editMsg.ReplyMarkup = &keyboard

		if _, err := s.Bot.Send(editMsg); err != nil {
			slog.Warn("t_back edit error", "error", err)
		}

		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	if action == "t_close" {
		sessionID := parts[1]
		SearchMu.Lock()
		delete(SearchSessions, sessionID)
		SearchMu.Unlock()
		_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID))
		return
	}

	if len(parts) < 3 {
		return
	}

	if action == "t_page" {
		page, _ := strconv.Atoi(parts[1])
		sessionID := parts[2]

		SearchMu.Lock()
		if session, exists := SearchSessions[sessionID]; exists {
			session.Page = page
		}
		SearchMu.Unlock()

		showSearchResults(s, callback.Message.Chat.ID, callback.Message.MessageID, sessionID)
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	if action == "t_item" {
		index, _ := strconv.Atoi(parts[1])
		sessionID := parts[2]

		SearchMu.RLock()
		session, exists := SearchSessions[sessionID]
		SearchMu.RUnlock()

		if !exists {
			_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Sesi berakhir"))
			return
		}

		if index < 0 || index >= len(session.Results) {
			return
		}

		item := session.Results[index]
		cleanMagnet := CleanMagnetLink(item.Magnet)

		text := fmt.Sprintf("📄 *Detail Torrent*\n\n"+
			"*Judul:* `%s`\n"+
			"*Size:* `%s`\n"+
			"*Seeders:* `%s`\n"+
			"*Source:* `%s`\n\n"+
			"🧲 *Magnet Link:*\n`%s`\n\n"+
			"Klik link di atas untuk menyalin atau reply dengan /mirror",
			utils.EscapeMarkdownV2Code(item.Title),
			utils.EscapeMarkdownV2Code(item.Size),
			utils.EscapeMarkdownV2Code(item.Seeders),
			utils.EscapeMarkdownV2Code(item.Source),
			utils.EscapeMarkdownV2Code(cleanMagnet),
		)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", fmt.Sprintf("t_page:%d:%s", session.Page, sessionID)),
				tgbotapi.NewInlineKeyboardButtonData("❌ Tutup", fmt.Sprintf("t_close:%s", sessionID)),
			),
		)

		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdownV2
		editMsg.ReplyMarkup = &keyboard

		if _, err := s.Bot.Send(editMsg); err != nil {
			slog.Warn("Search item detail markdown error, using fallback", "error", err)
			fallbackText := fmt.Sprintf("Detail Torrent:\n\n%s\nSize: %s\nMagnet:\n%s", item.Title, item.Size, cleanMagnet)
			fallbackEdit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, fallbackText)
			fallbackEdit.ReplyMarkup = &keyboard
			_, _ = s.Bot.Send(fallbackEdit)
		}
		_, _ = s.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

func searchSolidTorrents(query string) []SearchResult {
	var results []SearchResult
	apiURL := fmt.Sprintf("https://solidtorrents.to/api/v1/search?q=%s&sort=seeders", url.QueryEscape(query))
	slog.Debug("Requesting SolidTorrents", "url", apiURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		slog.Error("SolidTorrents request failed", "error", err)
		return results
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("Error closing SolidTorrents response body", "error", err)
		}
	}()

	slog.Debug("SolidTorrents response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return results
	}

	var data struct {
		Results []struct {
			Title   string `json:"title"`
			Magnet  string `json:"magnet"`
			Size    int64  `json:"size"`
			Seeders int    `json:"seeders"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		slog.Error("SolidTorrents decode error", "error", err)
		return results
	}

	seen := make(map[string]bool)

	for _, item := range data.Results {
		if item.Size < 10*1024*1024 {
			continue
		}

		cleanTitle := strings.ReplaceAll(item.Title, ".", " ")
		cleanTitle = strings.TrimSpace(cleanTitle)

		if seen[cleanTitle] {
			continue
		}
		seen[cleanTitle] = true

		results = append(results, SearchResult{
			Title:   cleanTitle,
			Size:    utils.FormatBytes(item.Size),
			Seeders: fmt.Sprintf("%d", item.Seeders),
			Magnet:  item.Magnet,
			Source:  "Solid",
		})
	}
	return results
}

func scrapeNyaa(query string) []SearchResult {
	var results []SearchResult
	searchURL := fmt.Sprintf("https://nyaa.si/?f=0&c=0_0&q=%s", url.QueryEscape(query))
	slog.Debug("Requesting Nyaa", "url", searchURL)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Nyaa request failed", "error", err)
		return results
	}
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Warn("Error closing Nyaa response body", "error", errClose)
		}
	}()

	slog.Debug("Nyaa response", "status", resp.StatusCode)

	if resp.StatusCode != 200 {
		return results
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		slog.Error("Nyaa parse error", "error", err)
		return results
	}

	doc.Find("table.torrent-list tbody tr").Each(func(_ int, s *goquery.Selection) {
		if len(results) >= 20 {
			return
		}

		title := s.Find("td[colspan='2'] a").Last().AttrOr("title", "")
		if title == "" {
			title = s.Find("td[colspan='2'] a").Last().Text()
		}

		magnet, _ := s.Find("td.text-center a[href^='magnet:']").Attr("href")
		size := s.Find("td.text-center").Eq(1).Text()
		seeders := s.Find("td.text-center").Eq(3).Text()

		if title != "" && magnet != "" {
			results = append(results, SearchResult{
				Title:   title,
				Size:    size,
				Seeders: seeders,
				Magnet:  magnet,
				Source:  "Nyaa",
			})
		}
	})

	return results
}

func searchPirateBay(query string) []SearchResult {
	var results []SearchResult
	apiURL := fmt.Sprintf("https://apibay.org/q.php?q=%s&cat=0", url.QueryEscape(query))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return results
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("Error closing PirateBay response body", "error", err)
		}
	}()

	var data []struct {
		Name     string `json:"name"`
		InfoHash string `json:"info_hash"`
		Seeders  string `json:"seeders"`
		Size     string `json:"size"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return results
	}

	for _, item := range data {
		if len(results) >= 20 {
			break
		}
		if item.InfoHash == "0000000000000000000000000000000000000000" {
			continue
		}

		sizeInt, _ := strconv.ParseInt(item.Size, 10, 64)

		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", item.InfoHash, url.QueryEscape(item.Name))

		results = append(results, SearchResult{
			Title:   item.Name,
			Size:    utils.FormatBytes(sizeInt),
			Seeders: item.Seeders,
			Magnet:  magnet,
			Source:  "TPB",
		})
	}
	return results
}

func CleanMagnetLink(magnet string) string {
	if strings.HasPrefix(magnet, "magnet:?") {
		return magnet
	}
	return fmt.Sprintf("magnet:?xt=urn:btih:%s", magnet)
}
