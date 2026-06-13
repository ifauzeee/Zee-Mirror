package queue

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimitRecord struct {
	LastFetch time.Time
	UserID    int64
	Tokens    float64
}

type UserRateLimiter struct {
	limiters map[int64]*rate.Limiter
	db       *sql.DB
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

func NewUserRateLimiter(ratePerMin int, burst int) *UserRateLimiter {
	return &UserRateLimiter{
		limiters: make(map[int64]*rate.Limiter),
		rate:     rate.Limit(float64(ratePerMin) / 60.0),
		burst:    burst,
	}
}

func NewUserRateLimiterWithDB(ratePerMin int, burst int, db *sql.DB) *UserRateLimiter {
	rl := NewUserRateLimiter(ratePerMin, burst)
	rl.db = db
	rl.loadFromDB()
	return rl
}

func (rl *UserRateLimiter) loadFromDB() {
	if rl.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := rl.db.QueryContext(ctx, "SELECT user_id, tokens, last_fetch FROM rate_limits")
	if err != nil {
		slog.Warn("Failed to load rate limits from DB", "error", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var rec RateLimitRecord
		if err := rows.Scan(&rec.UserID, &rec.Tokens, &rec.LastFetch); err != nil {
			continue
		}

		limiter := rate.NewLimiter(rl.rate, rl.burst)
		elapsed := time.Since(rec.LastFetch).Seconds()
		newTokens := rec.Tokens + elapsed*float64(rl.rate)
		if newTokens > float64(rl.burst) {
			newTokens = float64(rl.burst)
		}
		_ = newTokens
		rl.limiters[rec.UserID] = limiter
		count++
	}

	slog.Info("Loaded rate limits from database", "users", count)
}

func (rl *UserRateLimiter) getLimiter(userID int64) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[userID]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[userID] = limiter
	}
	return limiter
}

func (rl *UserRateLimiter) Allow(userID int64) bool {
	return rl.getLimiter(userID).Allow()
}

func (rl *UserRateLimiter) Persist() {
	if rl.db == nil {
		return
	}

	rl.mu.RLock()
	records := make([]RateLimitRecord, 0, len(rl.limiters))
	for userID := range rl.limiters {
		records = append(records, RateLimitRecord{
			UserID:    userID,
			Tokens:    float64(rl.burst),
			LastFetch: time.Now(),
		})
	}
	rl.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := rl.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Warn("Failed to begin rate limit persist transaction", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO rate_limits (user_id, tokens, last_fetch, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			tokens = excluded.tokens,
			last_fetch = excluded.last_fetch,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		slog.Warn("Failed to prepare rate limit persist statement", "error", err)
		return
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.UserID, rec.Tokens, rec.LastFetch, now); err != nil {
			slog.Warn("Failed to persist rate limit", "userID", rec.UserID, "error", err)
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Warn("Failed to commit rate limit persist", "error", err)
	}
}
