package domain

import "time"

// AuditLogEntry represents a single audit trail record for entity changes.
type AuditLogEntry struct {
	ID         int64
	EntityType string
	EntityID   int64
	Action     string
	OldValues  string
	NewValues  string
	CreatedAt  time.Time
}
