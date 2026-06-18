package database

import (
	"context"
	"time"
	"zee-mirror/internal/domain"
)

func (db *DB) LogAudit(ctx context.Context, entry domain.AuditEntry) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO audit_logs (action, actor_id, actor_name, resource, resource_id, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.Action, entry.ActorID, entry.ActorName, entry.Resource, entry.ResourceID, entry.Details, entry.IPAddress, time.Now())
	return err
}

func (db *DB) ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AuditEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, action, actor_id, actor_name, resource, resource_id, details, ip_address, created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.AuditEntry
	for rows.Next() {
		var e domain.AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.ActorID, &e.ActorName, &e.Resource, &e.ResourceID, &e.Details, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
