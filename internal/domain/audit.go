package domain

import "time"

type AuditEntry struct {
	CreatedAt  time.Time `json:"createdAt"`
	Action     string    `json:"action"`
	ActorName  string    `json:"actorName"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
	Details    string    `json:"details"`
	IPAddress  string    `json:"ipAddress"`
	ID         int64     `json:"id"`
	ActorID    int64     `json:"actorId"`
}
