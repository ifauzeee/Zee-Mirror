package queue

import (
	"sync"

	"golang.org/x/time/rate"
)

type UserRateLimiter struct {
	limiters map[int64]*rate.Limiter
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
