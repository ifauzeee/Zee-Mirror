package domain

import (
	"database/sql"
	"time"
)

type User struct {
	ID                int64        `json:"id"`
	Username          string       `json:"username"`
	Role              string       `json:"role"`
	MaxDailyTasks     int          `json:"maxDailyTasks"`
	MaxDailyBandwidth int64        `json:"maxDailyBandwidth"`
	ExpiresAt         sql.NullTime `json:"expiresAt"`
	CreatedAt         time.Time    `json:"createdAt"`
	IsActive          bool         `json:"isActive"`
}
