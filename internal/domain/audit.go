package domain

import "time"

type AuditEntry struct {
	ID         int64     `json:"id"`
	Action     string    `json:"action"`
	ActorID    int64     `json:"actorId"`
	ActorName  string    `json:"actorName"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
	Details    string    `json:"details"`
	IPAddress  string    `json:"ipAddress"`
	CreatedAt  time.Time `json:"createdAt"`
}
